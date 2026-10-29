// Package tmux runs terminals inside a tmux server and mirrors them into a Go
// program, so a TUI can host real shells and full-screen programs without
// implementing a terminal emulator of its own.
//
// tmux does the emulation. A Terminal attaches to one tmux session over a
// control-mode connection (tmux -C), learns from %output notifications when
// the session's pane has changed, and captures the pane's screen — cells,
// colors, cursor, modes — into a Screen snapshot the program draws however it
// likes. Keys, pastes, and mouse events go back through the same connection.
//
// Because the sessions live in the tmux server and not in the program, they
// outlive it: a program that exits and starts again reattaches to the shells
// it left running, exactly as a tmux client would.
//
// # Running a terminal
//
//	server := &tmux.Server{Socket: "myapp"} // a private server: tmux -L myapp
//	term, err := server.Open(ctx, tmux.TerminalOptions{
//	    Session:  "shell-1",
//	    Width:    80,
//	    Height:   24,
//	    OnUpdate: func() { runtime.SendEvent(redraw{}) },
//	})
//	if err != nil {
//	    return err
//	}
//	defer term.Detach() // the session keeps running
//
//	screen := term.Screen() // an immutable snapshot of the pane
//	term.SendKey(keyEvent)  // a terminal.KeyEvent from the decoder
//	term.Resize(100, 30)
//
// # Lower-level access
//
// Server runs one-shot tmux commands (Run, NewSession, Sessions, ...), and
// Control is a control-mode client that sends commands and reports
// notifications; Terminal is built from the two.
package tmux

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ErrNotInstalled is returned when the tmux binary cannot be found.
var ErrNotInstalled = errors.New("tmux: tmux is not installed or not on PATH")

// Server addresses one tmux server. The zero value is tmux's default server,
// the one a plain "tmux" command talks to. Set Socket to keep a program's
// sessions apart from the user's own.
type Server struct {
	// Socket is the server's socket name, passed to tmux as -L. Empty means
	// tmux's default socket.
	Socket string

	// SocketPath is a full socket path, passed as -S. It wins over Socket.
	SocketPath string

	// Config is the configuration file the server loads when a command starts
	// it, passed as -f. Empty lets tmux load the user's own configuration;
	// "/dev/null" loads none.
	Config string

	// Binary is the tmux executable. Empty means "tmux" found on PATH.
	Binary string
}

// Available reports whether the tmux binary can be found.
func (s *Server) Available() bool {
	_, err := exec.LookPath(s.binary())
	return err == nil
}

func (s *Server) binary() string {
	if s.Binary != "" {
		return s.Binary
	}
	return "tmux"
}

// args prefixes a tmux command line with the server-selection flags. -u makes
// tmux treat the client as UTF-8 whatever the locale says; without it, a
// process started with no LANG gets every wide character back as underscores.
func (s *Server) args(rest ...string) []string {
	args := []string{"-u"}
	switch {
	case s.SocketPath != "":
		args = append(args, "-S", s.SocketPath)
	case s.Socket != "":
		args = append(args, "-L", s.Socket)
	}
	if s.Config != "" {
		args = append(args, "-f", s.Config)
	}
	return append(args, rest...)
}

// Command returns an exec.Cmd that runs tmux against this server with args.
func (s *Server) Command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, s.binary(), s.args(args...)...)
}

// Run runs one tmux command and returns its standard output with the final
// newline removed. A command that fails returns an error carrying tmux's own
// message. Separate several commands with a ";" argument, as on the shell
// command line.
func (s *Server) Run(ctx context.Context, args ...string) (string, error) {
	if !s.Available() {
		return "", ErrNotInstalled
	}
	cmd := s.Command(ctx, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("tmux %s: %s", firstArg(args), msg)
	}
	return strings.TrimSuffix(stdout.String(), "\n"), nil
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// SessionOptions configures NewSession.
type SessionOptions struct {
	// Name is the session name. It must not contain ':' or '.', which tmux
	// reserves for targets.
	Name string

	// Dir is the working directory of the session's first pane. Empty means
	// the directory of the calling process.
	Dir string

	// Command is the shell command the pane runs. Empty runs the user's
	// default shell. When the command exits, the session ends.
	Command string

	// Width and Height are the pane's size in cells. Zero uses tmux's
	// default size.
	Width, Height int

	// Env holds KEY=value pairs set in the session's environment.
	Env []string
}

// NewSession starts a detached session. The session's window is sized
// manually, so attaching clients of any size never resize it, and its status
// line is off, so the pane gets every row of the window.
func (s *Server) NewSession(ctx context.Context, opts SessionOptions) error {
	if opts.Name == "" {
		return errors.New("tmux: session name is required")
	}
	if strings.ContainsAny(opts.Name, ":.") {
		return fmt.Errorf("tmux: session name %q contains ':' or '.'", opts.Name)
	}
	args := []string{"new-session", "-d", "-s", opts.Name}
	if opts.Dir != "" {
		args = append(args, "-c", opts.Dir)
	}
	if opts.Width > 0 && opts.Height > 0 {
		args = append(args, "-x", strconv.Itoa(opts.Width), "-y", strconv.Itoa(opts.Height))
	}
	for _, kv := range opts.Env {
		args = append(args, "-e", kv)
	}
	if opts.Command != "" {
		args = append(args, opts.Command)
	}
	// "=name:" is the session's current window, matched exactly; the
	// window and pane commands do not accept a bare session target.
	target := "=" + opts.Name + ":"
	args = append(args, ";", "set-option", "-t", target, "status", "off")
	if opts.Width > 0 && opts.Height > 0 {
		// resize-window sets the window's window-size option to manual.
		args = append(args, ";", "resize-window", "-t", target,
			"-x", strconv.Itoa(opts.Width), "-y", strconv.Itoa(opts.Height))
	} else {
		args = append(args, ";", "set-option", "-w", "-t", target, "window-size", "manual")
	}
	_, err := s.Run(ctx, args...)
	return err
}

// HasSession reports whether a session with exactly this name exists.
func (s *Server) HasSession(ctx context.Context, name string) bool {
	_, err := s.Run(ctx, "has-session", "-t", "="+name)
	return err == nil
}

// KillSession ends a session and every process running in it.
func (s *Server) KillSession(ctx context.Context, name string) error {
	_, err := s.Run(ctx, "kill-session", "-t", "="+name)
	return err
}

// RenameSession renames a session.
func (s *Server) RenameSession(ctx context.Context, from, to string) error {
	_, err := s.Run(ctx, "rename-session", "-t", "="+from, to)
	return err
}

// Kill stops the server and ends every session in it.
func (s *Server) Kill(ctx context.Context) error {
	_, err := s.Run(ctx, "kill-server")
	return err
}

// SessionInfo describes one session on the server.
type SessionInfo struct {
	Name     string
	ID       string // "$3"
	Created  time.Time
	Attached int // number of attached clients
	Windows  int
	Path     string // the session's working directory
}

// formatFields splits a line of format output on the unit separator that
// separates its fields. tmux before 3.5 prints control characters in command
// output as octal escapes, the separator among them, so a line without a
// separator is decoded first.
func formatFields(line string) []string {
	if !strings.Contains(line, "\x1f") {
		line = string(unescapeOutput(line))
	}
	return strings.Split(line, "\x1f")
}

// Sessions lists the server's sessions. A server that is not running has
// none, which is not an error.
func (s *Server) Sessions(ctx context.Context) ([]SessionInfo, error) {
	const format = "#{session_name}\x1f#{session_id}\x1f#{session_created}\x1f" +
		"#{session_attached}\x1f#{session_windows}\x1f#{session_path}"
	out, err := s.Run(ctx, "list-sessions", "-F", format)
	if err != nil {
		if isNoServer(err) {
			return nil, nil
		}
		return nil, err
	}
	var sessions []SessionInfo
	for line := range strings.SplitSeq(out, "\n") {
		f := formatFields(line)
		if len(f) != 6 {
			continue
		}
		created, _ := strconv.ParseInt(f[2], 10, 64)
		attached, _ := strconv.Atoi(f[3])
		windows, _ := strconv.Atoi(f[4])
		sessions = append(sessions, SessionInfo{
			Name:     f[0],
			ID:       f[1],
			Created:  time.Unix(created, 0),
			Attached: attached,
			Windows:  windows,
			Path:     f[5],
		})
	}
	return sessions, nil
}

// GetOption returns the value of a global user option (one whose name starts
// with "@"), or "" when it is unset or the server is not running.
func (s *Server) GetOption(ctx context.Context, name string) (string, error) {
	out, err := s.Run(ctx, "show-options", "-gqv", name)
	if err != nil && isNoServer(err) {
		return "", nil
	}
	return out, err
}

// SetOption sets a global user option. User options live as long as the
// server, which makes them a place to keep state that belongs with the
// sessions, such as how a program had laid them out.
func (s *Server) SetOption(ctx context.Context, name, value string) error {
	_, err := s.Run(ctx, "set-option", "-g", name, value)
	return err
}

func isNoServer(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "no server running") ||
		strings.Contains(msg, "error connecting to") ||
		strings.Contains(msg, "No such file or directory")
}
