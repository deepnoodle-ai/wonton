package tui

import (
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// These tests run a full-screen app with tui.Run, with a termtest.Screen in
// place of the user's terminal (WithTerminal), so they check what the user
// sees when the app starts, suspends, resumes and quits.

// lockedScreen lets the runtime's goroutines write to a termtest.Screen while
// the test reads it.
type lockedScreen struct {
	mu     sync.Mutex
	screen *termtest.Screen
}

func (l *lockedScreen) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.screen.Write(p)
}

// read runs fn with the screen locked.
func (l *lockedScreen) read(fn func(s *termtest.Screen)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fn(l.screen)
}

// waitFor polls the screen until cond holds, and fails the test after a
// second.
func (l *lockedScreen) waitFor(t *testing.T, what string, cond func(s *termtest.Screen) bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		ok := false
		l.read(func(s *termtest.Screen) { ok = cond(s) })
		if ok {
			return
		}
		if time.Now().After(deadline) {
			l.read(func(s *termtest.Screen) { t.Fatalf("timed out waiting for %s; screen:\n%s", what, s.Text()) })
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
// out, and waits until cond holds for the first frame. It returns the
// terminal, the keys the app reads, and the channel that receives Run's
// result. The app must quit on 'q'.
func runFullScreen(t *testing.T, app Application, out *lockedScreen, cond func(s *termtest.Screen) bool) (*Terminal, *MockInputSource, chan error) {
	t.Helper()
	var width, height int
	out.read(func(s *termtest.Screen) { width, height = s.Size() })
	term := NewTestTerminal(width, height, out)
	keys := NewMockInputSource()

	done := make(chan error, 1)
	go func() { done <- Run(app, WithTerminal(term), WithInput(keys)) }()
	out.waitFor(t, "the first frame", cond)
	return term, keys, done
}

// quitFullScreen presses 'q' and waits for Run to return.
func quitFullScreen(t *testing.T, keys *MockInputSource, done chan error) {
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
func startAltScreenApp(t *testing.T, app *altScreenApp) (*lockedScreen, *MockInputSource, chan error) {
	t.Helper()
	out := &lockedScreen{screen: termtest.NewScreen(40, 8)}
	_, _ = out.Write([]byte(altScreenPrompt + "\r\n"))
	_, keys, done := runFullScreen(t, app, out, func(s *termtest.Screen) bool {
		return s.IsAlternateScreen() && s.Contains("Inbox (3)")
	})
	return out, keys, done
}

func TestFullScreenAppGivesTheShellScreenBackOnExit(t *testing.T) {
	out, keys, done := startAltScreenApp(t, &altScreenApp{})

	out.read(func(s *termtest.Screen) {
		termtest.AssertRow(t, s, 0, "Inbox (3)")
		termtest.AssertNotContains(t, s, altScreenPrompt)
	})

	quitFullScreen(t, keys, done)

	out.read(func(s *termtest.Screen) {
		assert.False(t, s.IsAlternateScreen())
		termtest.AssertRow(t, s, 0, altScreenPrompt)
		termtest.AssertNotContains(t, s, "Inbox (3)")
		termtest.AssertCursor(t, s, 0, 1)
	})
}

func TestSuspendShowsTheShellScreenAndResumeRedrawsTheApp(t *testing.T) {
	const line = "transcript line"
	app := &altScreenApp{suspendErr: make(chan error, 1)}
	var out *lockedScreen

	// What the user sees inside the callback, read on the event loop.
	var inside struct {
		alt, prompt, frame, line bool
	}
	app.onSuspend = func() {
		out.read(func(s *termtest.Screen) {
			inside.alt = s.IsAlternateScreen()
			inside.prompt = s.Contains(altScreenPrompt)
			inside.frame = s.Contains("Inbox (3)")
		})
		// A real callback writes to the user's terminal: the same writer.
		_, _ = out.Write([]byte(line + "\r\n"))
		out.read(func(s *termtest.Screen) { inside.line = s.Contains(line) })
	}

	out, keys, done := startAltScreenApp(t, app)

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

	out.read(func(s *termtest.Screen) {
		assert.True(t, s.IsAlternateScreen(), "resume returns to the alternate screen")
		termtest.AssertRow(t, s, 0, "Inbox (3)")
		termtest.AssertRow(t, s, 1, "  hello from the app")
		termtest.AssertNotContains(t, s, line)
		termtest.AssertNotContains(t, s, altScreenPrompt)
	})

	quitFullScreen(t, keys, done)

	out.read(func(s *termtest.Screen) {
		assert.False(t, s.IsAlternateScreen())
		termtest.AssertRow(t, s, 0, altScreenPrompt)
		termtest.AssertRow(t, s, 1, line)
		termtest.AssertNotContains(t, s, "Inbox (3)")
		termtest.AssertCursor(t, s, 0, 2)
	})
}

func TestRunClosesTheTerminalItWasGiven(t *testing.T) {
	out := &lockedScreen{screen: termtest.NewScreen(40, 8)}
	term, keys, done := runFullScreen(t, &altScreenApp{}, out, func(s *termtest.Screen) bool {
		return s.Contains("Inbox (3)")
	})
	quitFullScreen(t, keys, done)

	_, err := term.BeginFrame()
	assert.ErrorIs(t, err, ErrClosed)
}
