package tmux

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/terminal"
)

var testServers atomic.Int64

// testServer returns a private tmux server with no user configuration that
// is killed when the test ends.
func testServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{
		Socket: fmt.Sprintf("wonton-test-%d-%d", os.Getpid(), testServers.Add(1)),
		Config: "/dev/null",
	}
	if !s.Available() {
		t.Skip("tmux is not installed")
	}
	t.Cleanup(func() { _ = s.Kill(context.Background()) })
	return s
}

// waitFor polls until cond holds or the deadline passes.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTerminalMirrorsPane(t *testing.T) {
	s := testServer(t)
	ctx := context.Background()

	updates := make(chan struct{}, 100)
	term, err := s.Open(ctx, TerminalOptions{
		Session: "mirror",
		Command: "printf '\\033[31mred\\033[0m 日本\\n'; cat",
		Width:   30,
		Height:  5,
		OnUpdate: func() {
			select {
			case updates <- struct{}{}:
			default:
			}
		},
	})
	assert.NoError(t, err)
	defer term.Kill()

	assert.True(t, strings.HasPrefix(term.PaneID(), "%"))
	waitFor(t, "printed text", func() bool { return strings.Contains(term.Screen().Text(), "red 日本") })

	screen := term.Screen()
	assert.Equal(t, screen.Width, 30)
	assert.Equal(t, screen.Height, 5)
	assert.Equal(t, screen.Cell(0, 0).Style.Foreground, color.Red)
	assert.Equal(t, screen.Cell(4, 0).Char, '日')
	assert.Equal(t, screen.Cell(4, 0).Width, 2)
	assert.True(t, screen.CursorVisible)
	assert.Equal(t, screen.CursorY, 1)

	// Typing reaches cat, which echoes it back.
	assert.NoError(t, term.SendText("hi there"))
	assert.NoError(t, term.SendKey(terminal.KeyEvent{Key: terminal.KeyEnter}))
	waitFor(t, "echo", func() bool { return strings.Count(term.Screen().Text(), "hi there") == 2 })

	// A paste arrives with its newline as a carriage return.
	assert.NoError(t, term.Paste("pasted\n"))
	waitFor(t, "paste", func() bool { return strings.Count(term.Screen().Text(), "pasted") == 2 })

	term.Resize(40, 8)
	waitFor(t, "resize", func() bool {
		s := term.Screen()
		return s.Width == 40 && s.Height == 8
	})
	assert.Len(t, term.Screen().Cells, 8)
	assert.Len(t, term.Screen().Cells[0], 40)

	select {
	case <-updates:
	default:
		t.Fatal("OnUpdate was never called")
	}
}

func TestTerminalDetachAndReattach(t *testing.T) {
	s := testServer(t)
	ctx := context.Background()

	term, err := s.Open(ctx, TerminalOptions{Session: "keep", Command: "cat", Width: 20, Height: 3})
	assert.NoError(t, err)
	assert.NoError(t, term.SendText("remember me"))
	waitFor(t, "echo", func() bool { return strings.Contains(term.Screen().Text(), "remember me") })

	assert.NoError(t, term.Detach())
	<-term.Done()
	assert.False(t, term.Exited())
	assert.True(t, s.HasSession(ctx, "keep"))

	again, err := s.Open(ctx, TerminalOptions{Session: "keep", Width: 25, Height: 4})
	assert.NoError(t, err)
	defer again.Kill()
	assert.Contains(t, again.Screen().Text(), "remember me")
	assert.Equal(t, again.Screen().Width, 25)

	sessions, err := s.Sessions(ctx)
	assert.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, sessions[0].Name, "keep")
}

func TestTerminalEndsWithItsProgram(t *testing.T) {
	s := testServer(t)
	term, err := s.Open(context.Background(), TerminalOptions{Session: "short", Command: "cat", Width: 20, Height: 3})
	assert.NoError(t, err)
	assert.NoError(t, term.SendKey(terminal.KeyEvent{Key: terminal.KeyCtrlD, Ctrl: true}))
	select {
	case <-term.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("terminal did not end when its program exited")
	}
	assert.True(t, term.Exited())
}

// With detach-on-destroy off, tmux moves a client whose session ends to
// another session instead of detaching it. The terminal must still end.
func TestTerminalEndsWithoutDetachOnDestroy(t *testing.T) {
	s := testServer(t)
	ctx := context.Background()
	assert.NoError(t, s.NewSession(ctx, SessionOptions{Name: "other", Command: "cat"}))
	_, err := s.Run(ctx, "set-option", "-g", "detach-on-destroy", "off")
	assert.NoError(t, err)

	killed, err := s.Open(ctx, TerminalOptions{Session: "killed", Command: "cat", Width: 20, Height: 3})
	assert.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- killed.Kill() }()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Kill did not return")
	}
	assert.True(t, killed.Exited())

	exited, err := s.Open(ctx, TerminalOptions{Session: "exited", Command: "cat", Width: 20, Height: 3})
	assert.NoError(t, err)
	assert.NoError(t, exited.SendKey(terminal.KeyEvent{Key: terminal.KeyCtrlD, Ctrl: true}))
	select {
	case <-exited.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("terminal did not end when its program exited")
	}
	assert.True(t, exited.Exited())
	assert.True(t, s.HasSession(ctx, "other"))
}

func TestTerminalRenameAndKill(t *testing.T) {
	s := testServer(t)
	ctx := context.Background()
	term, err := s.Open(ctx, TerminalOptions{Session: "before", Command: "cat", Width: 20, Height: 3})
	assert.NoError(t, err)

	assert.NoError(t, term.Rename("after"))
	assert.Equal(t, term.Session(), "after")
	assert.True(t, s.HasSession(ctx, "after"))
	assert.Error(t, term.Rename("bad:name"))

	assert.NoError(t, term.Kill())
	assert.True(t, term.Exited())
	assert.False(t, s.HasSession(ctx, "after"))
}

// Another client can rename the session. Kill and Rename target the session
// ID, so they still reach it.
func TestTerminalKillAfterOutsideRename(t *testing.T) {
	s := testServer(t)
	ctx := context.Background()
	term, err := s.Open(ctx, TerminalOptions{Session: "before", Command: "cat", Width: 20, Height: 3})
	assert.NoError(t, err)
	_, err = s.Run(ctx, "rename-session", "-t", "=before", "outside")
	assert.NoError(t, err)

	assert.NoError(t, term.Rename("after"))
	assert.True(t, s.HasSession(ctx, "after"))
	done := make(chan error, 1)
	go func() { done <- term.Kill() }()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Kill did not return")
	}
	assert.False(t, s.HasSession(ctx, "after"))
}

func TestTerminalScrollEntersCopyMode(t *testing.T) {
	s := testServer(t)
	term, err := s.Open(context.Background(), TerminalOptions{
		Session: "scroll",
		Command: "seq 1 50; cat",
		Width:   20,
		Height:  5,
	})
	assert.NoError(t, err)
	defer term.Kill()
	waitFor(t, "output", func() bool { return strings.Contains(term.Screen().Text(), "50") })

	term.Scroll(-10)
	waitFor(t, "copy mode", func() bool {
		s := term.Screen()
		return s.InMode && s.ScrollPosition == 10
	})
	assert.Contains(t, term.Screen().Text(), "40")

	// Scrolling back to the bottom leaves copy mode.
	term.Scroll(10)
	waitFor(t, "leaving copy mode", func() bool { return !term.Screen().InMode })

	// Typing leaves copy mode too. The second key goes out before the screen
	// shows the mode has ended, and must still arrive.
	term.Scroll(-10)
	waitFor(t, "copy mode again", func() bool { return term.Screen().InMode })
	assert.NoError(t, term.SendKey(terminal.KeyEvent{Rune: 'x'}))
	assert.NoError(t, term.SendKey(terminal.KeyEvent{Rune: 'y'}))
	waitFor(t, "both keys", func() bool {
		s := term.Screen()
		return !s.InMode && strings.Contains(s.Text(), "xy")
	})
}

func TestTerminalForwardsMouseOnlyWhenAsked(t *testing.T) {
	s := testServer(t)
	term, err := s.Open(context.Background(), TerminalOptions{
		Session: "mouse",
		// Turn on SGR button tracking, then show what arrives.
		Command: "printf '\\033[?1000h\\033[?1006h'; stty raw -echo; od -c",
		Width:   40,
		Height:  6,
	})
	assert.NoError(t, err)
	defer term.Kill()
	waitFor(t, "mouse mode", func() bool { return term.Screen().MouseReporting })
	assert.True(t, term.Screen().MouseSGR)
	assert.False(t, term.Screen().MouseDrags)

	// Button tracking (1000) asked for no motion, so none is sent.
	drag := terminal.MouseEvent{Type: terminal.MouseDrag, Button: terminal.MouseButtonLeft}
	assert.False(t, term.SendMouse(drag, 2, 3))
	assert.False(t, term.SendMouse(terminal.MouseEvent{Type: terminal.MouseMove}, 2, 3))

	press := terminal.MouseEvent{Type: terminal.MousePress, Button: terminal.MouseButtonLeft}
	assert.True(t, term.SendMouse(press, 2, 3))
	// A paste keeps its tabs and other control characters as they are.
	assert.NoError(t, term.Paste("\t\t\t\t\t\t\t"))
	// od prints a line once it has 16 bytes: 9 of the mouse report and 7
	// tabs. The report arrives as an escape sequence, not as visible text.
	waitFor(t, "mouse bytes", func() bool {
		text := term.Screen().Text()
		return strings.Contains(text, "033   [   <   0   ;   3   ;   4") &&
			strings.Contains(text, "M  \\t  \\t")
	})
}

func TestServerSessionsWithoutServer(t *testing.T) {
	s := &Server{Socket: fmt.Sprintf("wonton-test-none-%d", os.Getpid())}
	if !s.Available() {
		t.Skip("tmux is not installed")
	}
	sessions, err := s.Sessions(context.Background())
	assert.NoError(t, err)
	assert.Len(t, sessions, 0)
	v, err := s.GetOption(context.Background(), "@nothing")
	assert.NoError(t, err)
	assert.Equal(t, v, "")
}

func TestServerOptions(t *testing.T) {
	s := testServer(t)
	ctx := context.Background()
	assert.NoError(t, s.NewSession(ctx, SessionOptions{Name: "opts", Command: "cat"}))
	assert.NoError(t, s.SetOption(ctx, "@layout", `{"a": "b c"}`))
	v, err := s.GetOption(ctx, "@layout")
	assert.NoError(t, err)
	assert.Equal(t, v, `{"a": "b c"}`)
	assert.Error(t, s.NewSession(ctx, SessionOptions{Name: "a.b"}))
}

func TestTerminalReportsMouseMotionModes(t *testing.T) {
	s := testServer(t)
	term, err := s.Open(context.Background(), TerminalOptions{
		Session: "motion",
		Command: "printf '\\033[?1002h'; cat",
		Width:   20,
		Height:  3,
	})
	assert.NoError(t, err)
	defer term.Kill()
	waitFor(t, "mouse mode", func() bool { return term.Screen().MouseDrags })
	assert.False(t, term.Screen().MouseMotion)
	drag := terminal.MouseEvent{Type: terminal.MouseDrag, Button: terminal.MouseButtonLeft}
	assert.True(t, term.SendMouse(drag, 1, 1))
	assert.False(t, term.SendMouse(terminal.MouseEvent{Type: terminal.MouseMove}, 1, 1))
}

func TestTerminalWrappedRows(t *testing.T) {
	s := testServer(t)
	term, err := s.Open(context.Background(), TerminalOptions{
		Session: "wrap",
		Command: "printf 'abcdefghijklmno\\nxy\\n'; cat",
		Width:   10,
		Height:  4,
	})
	assert.NoError(t, err)
	defer term.Kill()
	waitFor(t, "output", func() bool { return strings.Contains(term.Screen().Text(), "xy") })
	wrapped, err := term.WrappedRows()
	assert.NoError(t, err)
	assert.Equal(t, wrapped, []bool{true, false, false, false})
}
