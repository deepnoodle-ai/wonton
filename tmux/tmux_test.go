package tmux

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/terminal"
)

func TestQuote(t *testing.T) {
	assert.Equal(t, Quote(`plain text`), `"plain text"`)
	assert.Equal(t, Quote(`say "hi" \ $HOME`), `"say \"hi\" \\ \$HOME"`)
	assert.Equal(t, Quote("a\nb\x1b[0m\x7f"), `"a\012b\033[0m\177"`)
	assert.Equal(t, Quote("日本; #x"), `"日本; #x"`)
}

func TestUnescapeOutput(t *testing.T) {
	assert.Equal(t, string(unescapeOutput(`\033[1mhi\015\012`)), "\x1b[1mhi\r\n")
	assert.Equal(t, string(unescapeOutput(`a\134b`)), `a\b`)
	// A backslash that does not start an escape is kept.
	assert.Equal(t, string(unescapeOutput(`a\9`)), `a\9`)
	assert.Equal(t, string(unescapeOutput(`end\03`)), `end\03`)
}

func TestParseNotification(t *testing.T) {
	n, ok := parseNotification(`%output %12 hello\015\012`)
	assert.True(t, ok)
	assert.Equal(t, n.Name, "output")
	assert.Equal(t, n.Pane, "%12")
	assert.Equal(t, string(n.Data), "hello\r\n")

	n, ok = parseNotification(`%window-renamed @1 zsh`)
	assert.True(t, ok)
	assert.Equal(t, n.Name, "window-renamed")
	assert.Equal(t, n.Args, "@1 zsh")

	_, ok = parseNotification("plain line")
	assert.False(t, ok)
}

func TestParseCaptureText(t *testing.T) {
	rows := ParseCapture([]string{"hello", "日本x"}, 6, 3)
	assert.Len(t, rows, 3)
	assert.Len(t, rows[0], 6)
	assert.Equal(t, rows[0][0].Char, 'h')
	assert.Equal(t, rows[0][5].Char, ' ')

	// A wide character fills two columns: the glyph, then a continuation.
	assert.Equal(t, rows[1][0].Char, '日')
	assert.Equal(t, rows[1][0].Width, 2)
	assert.True(t, rows[1][1].Continuation)
	assert.Equal(t, rows[1][2].Char, '本')
	assert.Equal(t, rows[1][4].Char, 'x')

	// Rows past the captured lines are blank.
	assert.Equal(t, rows[2][0].Char, ' ')

	s := &Screen{Cells: rows}
	assert.Equal(t, s.Text(), "hello\n日本x\n")
}

func TestParseCaptureTabs(t *testing.T) {
	rows := ParseCapture([]string{"ab\tX\t\tY|", "\t\t\t\tend"}, 30, 2)
	s := &Screen{Cells: rows}
	assert.Equal(t, s.Text(), "ab      X               Y|\n")
	assert.Equal(t, rows[0][8].Char, 'X')
	// A tab past the last stop stops at the right edge.
	assert.Equal(t, rows[1][29].Char, ' ')
}

func TestParseCaptureClipsLongLines(t *testing.T) {
	rows := ParseCapture([]string{"abcdef", "ab日"}, 3, 2)
	assert.Equal(t, string([]rune{rows[0][0].Char, rows[0][1].Char, rows[0][2].Char}), "abc")
	// A wide character that would straddle the right edge is dropped.
	assert.Equal(t, rows[1][2].Char, ' ')
	assert.False(t, rows[1][2].Continuation)
}

func TestParseCaptureStyles(t *testing.T) {
	lines := []string{
		"\x1b[38;2;255;10;10mr\x1b[39m \x1b[38;5;200mp\x1b[39m \x1b[1;41mb\x1b[0m",
		"\x1b[44m  ",
		"x",
		"\x1b[0m\x1b]8;;http://x.y\x1b\\L\x1b]8;;\x1b\\n",
	}
	rows := ParseCapture(lines, 5, 4)

	r := rows[0][0].Style
	assert.NotNil(t, r.FgRGB)
	assert.Equal(t, *r.FgRGB, color.NewRGB(255, 10, 10))
	assert.Equal(t, rows[0][1].Style.FgRGB == nil, true)
	assert.Equal(t, rows[0][2].Style.Foreground, color.Palette(200))
	b := rows[0][4].Style
	assert.True(t, b.Bold)
	assert.Equal(t, b.Background, color.Red)

	// Style carries from one line to the next, as tmux writes it...
	assert.Equal(t, rows[1][0].Style.Background, color.Blue)
	assert.Equal(t, rows[2][0].Style.Background, color.Blue)
	// ...but the blank cells past a line's end are default.
	assert.Equal(t, rows[2][1].Style.Background, color.Default)

	// OSC 8 hyperlinks.
	assert.Equal(t, rows[3][0].Style.URL, "http://x.y")
	assert.Equal(t, rows[3][1].Style.URL, "")
}

func TestApplySGR(t *testing.T) {
	s := applySGR(terminal.NewStyle(), "1;2;3;4;7;9")
	assert.True(t, s.Bold && s.Dim && s.Italic && s.Underline && s.Reverse && s.Strikethrough)
	s = applySGR(s, "22;23;24;27;29")
	assert.False(t, s.Bold || s.Dim || s.Italic || s.Underline || s.Reverse || s.Strikethrough)

	s = applySGR(terminal.NewStyle(), "92;104")
	assert.Equal(t, s.Foreground, color.BrightGreen)
	assert.Equal(t, s.Background, color.BrightBlue)

	s = applySGR(terminal.NewStyle(), "48:2::1:2:3;4:3")
	assert.Equal(t, *s.BgRGB, color.NewRGB(1, 2, 3))
	assert.True(t, s.Underline)

	// Underline color is skipped without disturbing what follows.
	s = applySGR(terminal.NewStyle(), "58;2;1;2;3;1")
	assert.True(t, s.Bold)

	s = applySGR(terminal.NewStyle().WithBold(), "")
	assert.False(t, s.Bold)
}

func TestKeyName(t *testing.T) {
	cases := []struct {
		ev      terminal.KeyEvent
		name    string
		literal bool
	}{
		{terminal.KeyEvent{Rune: 'a'}, "a", true},
		{terminal.KeyEvent{Rune: 'A', Shift: true}, "A", true},
		{terminal.KeyEvent{Rune: 'x', Alt: true}, "M-x", false},
		{terminal.KeyEvent{Rune: 'R', Ctrl: true, Shift: true}, "C-r", false},
		{terminal.KeyEvent{Rune: ' ', Ctrl: true}, "C-Space", false},
		{terminal.KeyEvent{Key: terminal.KeyEnter}, "Enter", false},
		{terminal.KeyEvent{Key: terminal.KeyEnter, Shift: true}, "S-Enter", false},
		{terminal.KeyEvent{Key: terminal.KeyTab, Shift: true}, "BTab", false},
		{terminal.KeyEvent{Key: terminal.KeyBackspace, Alt: true}, "M-BSpace", false},
		{terminal.KeyEvent{Key: terminal.KeyArrowLeft, Ctrl: true}, "C-Left", false},
		{terminal.KeyEvent{Key: terminal.KeyPageUp}, "PPage", false},
		{terminal.KeyEvent{Key: terminal.KeyDelete}, "DC", false},
		{terminal.KeyEvent{Key: terminal.KeyF5}, "F5", false},
		{terminal.KeyEvent{Key: terminal.KeyCtrlC, Ctrl: true}, "C-c", false},
		{terminal.KeyEvent{Key: terminal.KeyCtrlB, Ctrl: true, Alt: true}, "M-C-b", false},
		{terminal.KeyEvent{Paste: "a\nb"}, "a\nb", true},
	}
	for _, c := range cases {
		name, literal, ok := KeyName(c.ev)
		assert.True(t, ok, "%+v", c.ev)
		assert.Equal(t, name, c.name, "%+v", c.ev)
		assert.Equal(t, literal, c.literal, "%+v", c.ev)
	}
	_, _, ok := KeyName(terminal.KeyEvent{})
	assert.False(t, ok)
}

func TestMouseSequence(t *testing.T) {
	press := terminal.MouseEvent{Type: terminal.MousePress, Button: terminal.MouseButtonLeft}
	seq, ok := MouseSequence(press, 4, 9, true)
	assert.True(t, ok)
	assert.Equal(t, seq, "\x1b[<0;5;10M")

	release := terminal.MouseEvent{Type: terminal.MouseRelease, Button: terminal.MouseButtonLeft}
	seq, _ = MouseSequence(release, 4, 9, true)
	assert.Equal(t, seq, "\x1b[<0;5;10m")

	drag := terminal.MouseEvent{Type: terminal.MouseDrag, Button: terminal.MouseButtonLeft, Modifiers: terminal.ModCtrl}
	seq, _ = MouseSequence(drag, 0, 0, true)
	assert.Equal(t, seq, "\x1b[<48;1;1M")

	wheel := terminal.MouseEvent{Type: terminal.MousePress, Button: terminal.MouseButtonWheelDown}
	seq, _ = MouseSequence(wheel, 1, 1, true)
	assert.Equal(t, seq, "\x1b[<65;2;2M")

	seq, _ = MouseSequence(press, 0, 0, false)
	assert.Equal(t, seq, "\x1b[M !!")
	seq, _ = MouseSequence(release, 0, 0, false)
	assert.Equal(t, seq, "\x1b[M#!!")
	_, ok = MouseSequence(press, 300, 0, false)
	assert.False(t, ok)
	_, ok = MouseSequence(press, -5, 0, true)
	assert.False(t, ok)
	_, ok = MouseSequence(press, 0, -1, false)
	assert.False(t, ok)

	_, ok = MouseSequence(terminal.MouseEvent{Type: terminal.MouseClick}, 0, 0, true)
	assert.False(t, ok)
}

// fakeTmux plays the server side of a control-mode connection.
type fakeTmux struct {
	in    *io.PipeReader // what the client wrote
	out   *io.PipeWriter // what the client reads
	lines chan string
}

func newFakeControl(t *testing.T, notify func(Notification)) (*Control, *fakeTmux) {
	t.Helper()
	cr, cw := io.Pipe() // client -> fake
	fr, fw := io.Pipe() // fake -> client
	f := &fakeTmux{in: cr, out: fw, lines: make(chan string, 16)}
	go func() {
		buf := make([]byte, 4096)
		var pending string
		for {
			n, err := cr.Read(buf)
			if err != nil {
				close(f.lines)
				return
			}
			pending += string(buf[:n])
			for {
				line, rest, ok := strings.Cut(pending, "\n")
				if !ok {
					break
				}
				f.lines <- line
				pending = rest
			}
		}
	}()
	c := newControl(cw, fr, notify)
	t.Cleanup(func() { _ = fw.Close(); _ = c.Close() })
	return c, f
}

func (f *fakeTmux) send(lines ...string) {
	for _, l := range lines {
		_, _ = io.WriteString(f.out, l+"\n")
	}
}

func TestControlCommandReplies(t *testing.T) {
	var mu sync.Mutex
	var got []Notification
	c, f := newFakeControl(t, func(n Notification) {
		mu.Lock()
		got = append(got, n)
		mu.Unlock()
	})

	// The attach reply (flags 0) must not be taken for a command's.
	f.send("%begin 1 100 0", "%end 1 100 0", "%session-changed $1 s")

	type result struct {
		out []string
		err error
	}
	first := make(chan result, 1)
	second := make(chan result, 1)
	go func() { out, err := c.Command("list-windows"); first <- result{out, err} }()
	assert.Equal(t, <-f.lines, "list-windows")
	go func() { out, err := c.Command("bogus"); second <- result{out, err} }()
	assert.Equal(t, <-f.lines, "bogus")

	f.send(
		"%begin 2 101 1",
		"0: zsh",
		"%end 2 101", // not this block's tag: still output
		"%output %1 x\\015",
		"%end 2 101 1",
		"%begin 2 102 1",
		"unknown command: bogus",
		"%error 2 102 1",
	)
	r := <-first
	assert.NoError(t, r.err)
	assert.Equal(t, r.out, []string{"0: zsh", "%end 2 101", "%output %1 x\\015"})
	r = <-second
	var ce *CommandError
	assert.ErrorAs(t, r.err, &ce)
	assert.Equal(t, ce.Command, "bogus")
	assert.Equal(t, ce.Output, []string{"unknown command: bogus"})

	f.send("%exit")
	_ = f.out.Close()
	<-c.Done()
	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, got, 2)
	assert.Equal(t, got[0].Name, "session-changed")
	assert.Equal(t, got[1].Name, "exit")

	_, err := c.Command("list-windows")
	assert.ErrorIs(t, err, ErrClosed)
}

func TestControlRejectsUnsafeCommands(t *testing.T) {
	c, _ := newFakeControl(t, nil)
	_, err := c.Command("   ")
	assert.Error(t, err)
	_, err = c.Command("send-keys a\nkill-server")
	assert.Error(t, err)
}

func TestControlPendingFailOnClose(t *testing.T) {
	c, f := newFakeControl(t, nil)
	done := make(chan error, 1)
	go func() { _, err := c.Command("list-windows"); done <- err }()
	<-f.lines
	_ = f.out.Close()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, ErrClosed)
	case <-time.After(2 * time.Second):
		t.Fatal("pending command did not fail when the connection closed")
	}
}

func TestCountCommands(t *testing.T) {
	assert.Equal(t, countCommands("list-windows"), 1)
	assert.Equal(t, countCommands("set-buffer x ; paste-buffer"), 2)
	assert.Equal(t, countCommands(`send-keys -l ";" ; send-keys ' ; ' ; a \; b`), 3)
	assert.Equal(t, countCommands(`send-keys -l "\" ; x"`), 1)
	assert.Equal(t, countCommands("send-keys a;b"), 2)
	assert.Equal(t, countCommands("display -p a; display -p b"), 2)
	assert.Equal(t, countCommands("display -p a ;"), 1)
	assert.Equal(t, countCommands("display -p a ; ; "), 1)
	// tmux skips empty commands and comments, and replies to neither.
	assert.Equal(t, countCommands("display -p one ; ; display -p two"), 2)
	assert.Equal(t, countCommands("display -p a ; # comment ; display -p b"), 1)
	assert.Equal(t, countCommands("display -p a #b ; display -p c"), 1)
	assert.Equal(t, countCommands("display -p a ;# x"), 1)
	assert.Equal(t, countCommands("display -p a#b ; display -p c"), 2)
	assert.Equal(t, countCommands(`display -p "#{session_name}" ; display -p '#'`), 2)
	assert.Equal(t, countCommands("# only a comment"), 0)
	assert.Equal(t, countCommands(" ; "), 0)
}

// tmux skips the rest of a list after a failed command, so the error
// answers the whole line and the next line's reply pairs correctly.
func TestControlFailedCommandEndsList(t *testing.T) {
	c, f := newFakeControl(t, nil)
	first := make(chan error, 1)
	go func() { _, err := c.Command("send-keys -X cancel ; send-keys -l b"); first <- err }()
	<-f.lines
	f.send("%begin 1 1 1", "not in a mode", "%error 1 1 1")
	err := <-first
	var cerr *CommandError
	assert.True(t, errors.As(err, &cerr))
	assert.Equal(t, cerr.Output, []string{"not in a mode"})

	second := make(chan []string, 1)
	go func() { out, _ := c.Command("display -p hello"); second <- out }()
	<-f.lines
	f.send("%begin 1 2 1", "hello", "%end 1 2 1")
	select {
	case out := <-second:
		assert.Equal(t, out, []string{"hello"})
	case <-time.After(2 * time.Second):
		t.Fatal("the next command was not answered")
	}
}

func TestControlMultiCommandReply(t *testing.T) {
	c, f := newFakeControl(t, nil)
	done := make(chan []string, 1)
	go func() { out, _ := c.Command("display -p x ; capture-pane -p"); done <- out }()
	<-f.lines
	f.send("%begin 1 1 1", "x", "%end 1 1 1")
	select {
	case <-done:
		t.Fatal("answered before the second command replied")
	case <-time.After(20 * time.Millisecond):
	}
	f.send("%begin 1 2 1", "line 1", "line 2", "%end 1 2 1")
	assert.Equal(t, <-done, []string{"x", "line 1", "line 2"})
}

func TestStripPasteEnd(t *testing.T) {
	assert.Equal(t, stripPasteEnd("a\x1b[201~b"), "ab")
	assert.Equal(t, stripPasteEnd("X\x1b[20\x1b[201~1~Y"), "XY")
	assert.Equal(t, stripPasteEnd("\x1b[200~ok"), "\x1b[200~ok")
}

func TestWrappedRows(t *testing.T) {
	rows := []string{
		"klmno     ", // the end of a line that scrolled off the top
		"xy",
		"hello wor ", // wraps at a space
		"ld",
		"日本語日本",
		"語x  ",
		"",
	}
	joined := []string{"klmno", "xy", "hello wor ld", "日本語日本語x", ""}
	assert.Equal(t, wrappedRows(rows, joined), []bool{false, false, true, false, true, false, false})

	// As tmux captures a 9-column pane: a wide character that does not fit
	// leaves a padding cell, and trailing spaces wrap onto further rows.
	rows = []string{"日本語日 ", "本語日本 ", "語  ", "hello    ", "         ", "    ", "ab  "}
	joined = []string{"日本語日本語日本語", "hello               ", "ab"}
	assert.Equal(t, wrappedRows(rows, joined), []bool{true, true, false, true, true, false, false})
}

func TestParseSnapshotEscapedHeader(t *testing.T) {
	// tmux 3.4 writes the separator as the text \037.
	header := strings.Join([]string{"%3", "20", "2", "4", "1", "1", "0", "0", "0", "0", "", "0", "", "", "0", "0", "cat", "/tmp", "host"}, `\037`)
	screen, err := parseSnapshot([]string{header, "hi", ""})
	assert.NoError(t, err)
	assert.Equal(t, screen.PaneID, "%3")
	assert.Equal(t, screen.Width, 20)
	assert.Equal(t, screen.CursorX, 4)
	assert.Equal(t, screen.Command, "cat")
	assert.Equal(t, screen.Path, "/tmp")
	assert.Equal(t, screen.Title, "host")
	assert.Equal(t, screen.Text(), "hi\n")
}

// Output that arrives while Open takes its first capture gets a capture of
// its own once Open is done, rather than waiting for the pane's next write.
func TestTerminalCapturesOutputMissedDuringOpen(t *testing.T) {
	term := &Terminal{pane: "%1", interval: time.Millisecond, done: make(chan struct{})}
	c, f := newFakeControl(t, term.notified)
	term.ctl = c

	f.send("%output %1 hello")
	waitFor(t, "the output to be noticed", func() bool {
		term.mu.Lock()
		defer term.mu.Unlock()
		return term.dirty
	})
	term.start()
	select {
	case line := <-f.lines:
		assert.Contains(t, line, "capture-pane")
	case <-time.After(2 * time.Second):
		t.Fatal("no capture after start")
	}
}

// A resize tmux refuses is tried again the next time the size is asked for.
func TestTerminalRetriesFailedResize(t *testing.T) {
	term := &Terminal{pane: "%1", done: make(chan struct{})}
	c, f := newFakeControl(t, nil)
	term.ctl = c

	term.Resize(30, 10)
	assert.Contains(t, <-f.lines, "resize-window")
	f.send("%begin 1 1 1", "no space for new pane", "%error 1 1 1")
	waitFor(t, "the refusal", func() bool {
		term.mu.Lock()
		defer term.mu.Unlock()
		return !term.retryAt.IsZero()
	})

	// Asking again at once sends nothing, so resizing on every frame does
	// not flood tmux. Once the delay is over, it tries again.
	term.Resize(30, 10)
	select {
	case line := <-f.lines:
		t.Fatalf("resent at once: %q", line)
	case <-time.After(50 * time.Millisecond):
	}
	term.mu.Lock()
	term.retryAt = time.Now().Add(-time.Millisecond)
	term.mu.Unlock()
	term.Resize(30, 10)
	assert.Contains(t, <-f.lines, "resize-window")
}
