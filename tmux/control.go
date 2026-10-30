package tmux

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// ErrClosed is returned by Control.Command after the connection has ended.
var ErrClosed = errors.New("tmux: control connection closed")

// Notification is an asynchronous message from a control-mode connection:
// every line tmux sends that is not part of a command's reply.
type Notification struct {
	// Name is the notification without its leading '%': "output", "exit",
	// "window-renamed", "session-changed", and so on.
	Name string

	// Args is the rest of the line after the name, unparsed. For "output" it
	// is empty; see Pane and Data instead.
	Args string

	// Pane is the pane ID ("%3") of an "output" notification.
	Pane string

	// Data is the decoded output of an "output" notification: the bytes the
	// pane's program wrote, escape sequences and all.
	Data []byte
}

// CommandError is the error a command returns when tmux reports that it
// failed. Output holds tmux's message lines.
type CommandError struct {
	Command string
	Output  []string
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("tmux: %s: %s", e.Command, strings.Join(e.Output, "; "))
}

// Control is a control-mode client (tmux -C): a connection that runs
// commands and reports notifications without drawing anything. It is safe
// for concurrent use.
//
// Commands are answered in the order they were sent, so Command can be called
// from several goroutines at once.
type Control struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	notify func(Notification)

	mu      sync.Mutex
	pending []pendingCommand
	closed  bool

	done chan struct{}
	err  error
}

type pendingCommand struct {
	command   string
	reply     chan reply
	remaining int      // replies still to come: one per command in the line
	output    []string // the replies so far, concatenated
}

type reply struct {
	output []string
	err    error
}

// Attach opens a control-mode connection to an existing session. notify is
// called on the connection's reader goroutine for every notification, in the
// order tmux sent them; it must not block for long, since the next line is
// not read until it returns. It may be nil.
//
// The connection ends when the session does, when Close is called, or when
// ctx is canceled.
func (s *Server) Attach(ctx context.Context, session string, notify func(Notification)) (*Control, error) {
	if !s.Available() {
		return nil, ErrNotInstalled
	}
	cmd := s.Command(ctx, "-C", "attach-session", "-t", "="+session)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c := newControl(stdin, stdout, notify)
	c.cmd = cmd
	go func() {
		<-c.done
		_ = cmd.Wait()
	}()
	return c, nil
}

// newControl runs the protocol over an arbitrary pipe pair. Attach supplies
// a tmux process; tests supply a fake.
func newControl(stdin io.WriteCloser, stdout io.Reader, notify func(Notification)) *Control {
	c := &Control{
		stdin:  stdin,
		notify: notify,
		done:   make(chan struct{}),
	}
	go c.read(stdout)
	return c
}

// Command runs a tmux command and returns its output lines. A command tmux
// rejects returns a *CommandError. The command is written in tmux's command
// language; use Quote for arguments that may contain spaces, quotes, or
// control characters. Several commands separated by " ; " run in order and
// their output is concatenated; the error is the first one that failed.
func (c *Control) Command(command string) ([]string, error) {
	ch, err := c.send(command)
	if err != nil {
		return nil, err
	}
	select {
	case r := <-ch:
		return r.output, r.err
	case <-c.done:
		// The reply may have been delivered just before the connection
		// ended.
		select {
		case r := <-ch:
			return r.output, r.err
		default:
			return nil, ErrClosed
		}
	}
}

// Go sends a command without waiting for its reply. The returned channel
// receives the reply's error (nil on success) and is closed. It is closed
// without a value if the connection ends first.
func (c *Control) Go(command string) <-chan error {
	out := make(chan error, 1)
	ch, err := c.send(command)
	if err != nil {
		out <- err
		close(out)
		return out
	}
	go func() {
		defer close(out)
		select {
		case r := <-ch:
			out <- r.err
		case <-c.done:
		}
	}()
	return out
}

func (c *Control) send(command string) (chan reply, error) {
	// An empty line detaches a control client, and a newline would split
	// the command in two: neither may reach tmux.
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, errors.New("tmux: empty command")
	}
	if strings.ContainsAny(command, "\r\n") {
		return nil, errors.New("tmux: command contains a newline; quote it with Quote")
	}
	// tmux sends no reply to a line of only comments and separators, so
	// waiting for one would wait forever.
	count := countCommands(command)
	if count == 0 {
		return nil, errors.New("tmux: command line has no commands")
	}
	ch := make(chan reply, 1)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, ErrClosed
	}
	// Queue before writing, so the reply can never arrive first.
	c.pending = append(c.pending, pendingCommand{
		command:   command,
		reply:     ch,
		remaining: count,
	})
	if _, err := io.WriteString(c.stdin, command+"\n"); err != nil {
		c.pending = c.pending[:len(c.pending)-1]
		return nil, err
	}
	return ch, nil
}

// Close detaches the client. The session keeps running.
func (c *Control) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		<-c.done
		return nil
	}
	c.closed = true
	err := c.stdin.Close()
	c.mu.Unlock()
	<-c.done
	return err
}

// Done is closed when the connection has ended.
func (c *Control) Done() <-chan struct{} { return c.done }

// Err returns why the connection ended: nil for a normal detach or exit.
// It is only meaningful after Done is closed.
func (c *Control) Err() error {
	<-c.done
	return c.err
}

// read is the connection's only reader. It pairs command replies with the
// pending queue and hands every other line to notify.
func (c *Control) read(stdout io.Reader) {
	defer c.finish()
	r := bufio.NewReaderSize(stdout, 64*1024)

	var (
		inBlock bool
		ours    bool     // the block answers one of our commands
		endTag  string   // " <time> <number> <flags>" that closes the block
		output  []string // lines of the block so far
	)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				c.err = err
			}
			return
		}
		line = strings.TrimSuffix(line, "\n")

		if inBlock {
			if tag, ok := strings.CutPrefix(line, "%end"); ok && tag == endTag {
				inBlock = false
				if ours {
					c.deliver(output, nil)
				}
				continue
			}
			if tag, ok := strings.CutPrefix(line, "%error"); ok && tag == endTag {
				inBlock = false
				if ours {
					c.deliver(nil, &CommandError{Command: c.peek(), Output: output})
				}
				continue
			}
			output = append(output, line)
			continue
		}

		if tag, ok := strings.CutPrefix(line, "%begin"); ok {
			inBlock = true
			endTag = tag
			output = nil
			// The flags field is 1 for a command this client sent. The
			// attach that opened the connection replies with 0.
			ours = strings.HasSuffix(tag, " 1")
			continue
		}

		// %exit comes last; tmux closes the connection right after it.
		if n, ok := parseNotification(line); ok && c.notify != nil {
			c.notify(n)
		}
	}
}

func (c *Control) peek() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.pending) == 0 {
		return ""
	}
	return c.pending[0].command
}

// deliver records one command's reply against the oldest pending line, and
// answers the line once every command in it has replied. A failed command
// answers the line at once: tmux skips the rest of a list after an error,
// and a line that does not parse gets a single error reply.
func (c *Control) deliver(output []string, err error) {
	c.mu.Lock()
	if len(c.pending) == 0 {
		c.mu.Unlock()
		return
	}
	p := &c.pending[0]
	p.output = append(p.output, output...)
	p.remaining--
	if p.remaining > 0 && err == nil {
		c.mu.Unlock()
		return
	}
	done := *p
	c.pending = c.pending[1:]
	c.mu.Unlock()
	if err != nil {
		done.output = nil
	}
	done.reply <- reply{output: done.output, err: err}
}

// countCommands counts the commands in a command line: tmux answers each
// with its own reply. Any ';' outside quotes and not escaped ends a command;
// a command with no words gets no reply. An unquoted '#' at the start of a
// word comments out the rest of the line.
func countCommands(line string) int {
	n := 0
	inCommand := false // the current command has a word
	wordStart := true
	var quote byte
	for i := 0; i < len(line); i++ {
		ch := line[i]
		atWordStart := wordStart
		wordStart = false
		switch {
		case quote == '\'':
			if ch == '\'' {
				quote = 0
			}
			continue
		case quote == '"' && ch == '"':
			quote = 0
			continue
		case quote == '"' && ch != '\\':
			continue
		case ch == '#' && atWordStart:
			return n
		case ch == ';':
			inCommand = false
			wordStart = true
			continue
		case ch == ' ' || ch == '\t':
			wordStart = true
			continue
		case ch == '\\':
			i++ // the escaped character is never special
		case ch == '\'' || ch == '"':
			quote = ch
		}
		if !inCommand {
			inCommand = true
			n++
		}
	}
	return n
}

func (c *Control) finish() {
	c.mu.Lock()
	c.closed = true
	c.pending = nil
	_ = c.stdin.Close()
	c.mu.Unlock()
	close(c.done)
}

// parseNotification splits a "%name args" line. %output payloads are
// decoded from tmux's octal escaping.
func parseNotification(line string) (Notification, bool) {
	if !strings.HasPrefix(line, "%") {
		return Notification{}, false
	}
	name, args, _ := strings.Cut(line[1:], " ")
	n := Notification{Name: name, Args: args}
	if name == "output" {
		pane, data, _ := strings.Cut(args, " ")
		n.Args = ""
		n.Pane = pane
		n.Data = unescapeOutput(data)
	}
	return n, true
}

// unescapeOutput reverses the escaping tmux applies to %output data: bytes
// below 32 and the backslash are written as a backslash and three octal
// digits.
func unescapeOutput(s string) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) && isOctal(s[i+1]) && isOctal(s[i+2]) && isOctal(s[i+3]) {
			out = append(out, (s[i+1]-'0')<<6|(s[i+2]-'0')<<3|(s[i+3]-'0'))
			i += 3
			continue
		}
		out = append(out, s[i])
	}
	return out
}

func isOctal(b byte) bool { return b >= '0' && b <= '7' }

// Quote returns s as a double-quoted string in tmux's command language, safe
// to splice into a command for Control.Command: quotes, backslashes, and '$'
// are escaped, and control characters (newlines included) become octal
// escapes, which tmux decodes back into the same bytes.
func Quote(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '"' || ch == '\\' || ch == '$':
			b.WriteByte('\\')
			b.WriteByte(ch)
		case ch < 0x20 || ch == 0x7f:
			fmt.Fprintf(&b, "\\%03o", ch)
		default:
			b.WriteByte(ch)
		}
	}
	b.WriteByte('"')
	return b.String()
}
