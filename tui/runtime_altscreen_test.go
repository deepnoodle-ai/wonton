package tui

import (
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// These tests run a full-screen app with tui.Run, with a termtest.Screen in
// place of the user's terminal (WithTerminal), so they check what the user
// sees when the app starts, suspends, resumes and quits.

// waitFor polls screen until cond holds, and fails the test after a second.
func waitFor(t *testing.T, screen *termtest.Screen, what string, cond func(s *termtest.Screen) bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !cond(screen) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s; screen:\n%s", what, screen.Text())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

const altScreenPrompt = "$ myapp"

// altScreenApp shows one frame. 'q' quits; 's' suspends and runs onSuspend.
type altScreenApp struct {
	rt         *Runtime
	onSuspend  func()
	suspendErr chan error
}

func (a *altScreenApp) SetRuntime(r *Runtime) { a.rt = r }

func (a *altScreenApp) View() View {
	return Stack(Text("Inbox (3)"), Text("  hello from the app"))
}

func (a *altScreenApp) HandleEvent(event Event) []Cmd {
	if key, ok := event.(KeyEvent); ok {
		switch key.Rune {
		case 'q':
			return []Cmd{Quit()}
		case 's':
			a.suspendErr <- a.rt.Suspend(func(<-chan Event) { a.onSuspend() })
		}
	}
	return nil
}

// runFullScreen starts app with tui.Run on a test terminal that writes to
// screen, and waits until cond holds for the first frame. It returns the
// terminal, the keys the app reads, and the channel that receives Run's
// result. The app must quit on 'q'.
func runFullScreen(t *testing.T, app Application, screen *termtest.Screen, cond func(s *termtest.Screen) bool) (*Terminal, *TestInput, chan error) {
	t.Helper()
	width, height := screen.Size()
	term := NewTestTerminal(width, height, screen)
	keys := NewTestInput()
	t.Cleanup(keys.Close)

	done := make(chan error, 1)
	go func() { done <- Run(app, WithTerminal(term), WithInput(keys)) }()
	waitFor(t, screen, "the first frame", cond)
	return term, keys, done
}

// quitFullScreen presses 'q' and waits for Run to return.
func quitFullScreen(t *testing.T, keys *TestInput, done chan error) {
	t.Helper()
	keys.Send(KeyEvent{Rune: 'q', Time: time.Now()})
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("the app did not quit")
	}
}

// startAltScreenApp writes a shell prompt, then runs the app with tui.Run and
// waits for its first frame.
func startAltScreenApp(t *testing.T, app *altScreenApp) (*termtest.Screen, *TestInput, chan error) {
	t.Helper()
	screen := termtest.NewScreen(40, 8)
	screen.WriteString(altScreenPrompt + "\r\n")
	_, keys, done := runFullScreen(t, app, screen, func(s *termtest.Screen) bool {
		return s.IsAlternateScreen() && s.Contains("Inbox (3)")
	})
	return screen, keys, done
}

func TestFullScreenAppGivesTheShellScreenBackOnExit(t *testing.T) {
	screen, keys, done := startAltScreenApp(t, &altScreenApp{})

	termtest.AssertRow(t, screen, 0, "Inbox (3)")
	termtest.AssertNotContains(t, screen, altScreenPrompt)

	quitFullScreen(t, keys, done)

	assert.False(t, screen.IsAlternateScreen())
	termtest.AssertRow(t, screen, 0, altScreenPrompt)
	termtest.AssertNotContains(t, screen, "Inbox (3)")
	termtest.AssertCursor(t, screen, 0, 1)
}

func TestSuspendShowsTheShellScreenAndResumeRedrawsTheApp(t *testing.T) {
	const line = "transcript line"
	app := &altScreenApp{suspendErr: make(chan error, 1)}
	var screen *termtest.Screen

	// What the user sees inside the callback, read on the event loop.
	var inside struct {
		alt, prompt, frame, line bool
	}
	app.onSuspend = func() {
		inside.alt = screen.IsAlternateScreen()
		inside.prompt = screen.Contains(altScreenPrompt)
		inside.frame = screen.Contains("Inbox (3)")
		// A real callback writes to the user's terminal: the same writer.
		screen.WriteString(line + "\r\n")
		inside.line = screen.Contains(line)
	}

	screen, keys, done := startAltScreenApp(t, app)

	keys.Send(KeyEvent{Rune: 's', Time: time.Now()})
	select {
	case err := <-app.suspendErr:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Suspend did not return")
	}

	assert.False(t, inside.alt, "suspend leaves the alternate screen")
	assert.True(t, inside.prompt, "suspend shows the shell screen")
	assert.False(t, inside.frame, "no app frame on the shell screen")
	assert.True(t, inside.line, "the callback's output is visible")

	assert.True(t, screen.IsAlternateScreen(), "resume returns to the alternate screen")
	termtest.AssertRow(t, screen, 0, "Inbox (3)")
	termtest.AssertRow(t, screen, 1, "  hello from the app")
	termtest.AssertNotContains(t, screen, line)
	termtest.AssertNotContains(t, screen, altScreenPrompt)

	quitFullScreen(t, keys, done)

	assert.False(t, screen.IsAlternateScreen())
	termtest.AssertRow(t, screen, 0, altScreenPrompt)
	termtest.AssertRow(t, screen, 1, line)
	termtest.AssertNotContains(t, screen, "Inbox (3)")
	termtest.AssertCursor(t, screen, 0, 2)
}

func TestRunClosesTheTerminalItWasGiven(t *testing.T) {
	screen := termtest.NewScreen(40, 8)
	term, keys, done := runFullScreen(t, &altScreenApp{}, screen, func(s *termtest.Screen) bool {
		return s.Contains("Inbox (3)")
	})
	quitFullScreen(t, keys, done)

	_, err := term.BeginFrame()
	assert.ErrorIs(t, err, ErrClosed)
}

func TestRunRefusesAClosedTerminal(t *testing.T) {
	screen := termtest.NewScreen(40, 8)
	term := NewTestTerminal(40, 8, screen)
	assert.NoError(t, term.Close())
	before := screen.Text()

	keys := NewTestInput()
	keys.Close()
	err := Run(&altScreenApp{}, WithTerminal(term), WithInput(keys))
	assert.ErrorIs(t, err, ErrClosed)
	assert.Equal(t, screen.Text(), before, "Run drew nothing")
}
