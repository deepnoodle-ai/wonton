package tui

import (
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// These tests run a full-screen Runtime the way Run does, but with a
// termtest.Screen in place of the user's terminal, so they check what the user
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

// startAltScreenApp writes a shell prompt, then starts the app on the
// alternate screen as Run does, and waits for its first frame.
func startAltScreenApp(t *testing.T, app *altScreenApp) (*lockedScreen, *Terminal, *MockInputSource, chan error) {
	t.Helper()
	out := &lockedScreen{screen: termtest.NewScreen(40, 8)}
	_, _ = out.Write([]byte(altScreenPrompt + "\r\n"))

	term := NewTestTerminal(40, 8, out)
	term.EnableAlternateScreen()
	term.HideCursor()
	rt := NewRuntime(term, app, 30)
	keys := NewMockInputSource()
	rt.SetInputSource(keys)

	done := make(chan error, 1)
	go func() { done <- rt.Run() }()
	out.waitFor(t, "the app frame", func(s *termtest.Screen) bool {
		return s.IsAlternateScreen() && s.Contains("Inbox (3)")
	})
	return out, term, keys, done
}

// quitAltScreenApp quits the app and does the cleanup that Run defers.
func quitAltScreenApp(t *testing.T, term *Terminal, keys *MockInputSource, done chan error) {
	t.Helper()
	keys.Send(KeyEvent{Rune: 'q', Time: time.Now()})
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("the app did not quit")
	}
	assert.NoError(t, term.Close())
}

func TestFullScreenAppGivesTheShellScreenBackOnExit(t *testing.T) {
	out, term, keys, done := startAltScreenApp(t, &altScreenApp{})

	out.read(func(s *termtest.Screen) {
		termtest.AssertRow(t, s, 0, "Inbox (3)")
		termtest.AssertNotContains(t, s, altScreenPrompt)
	})

	quitAltScreenApp(t, term, keys, done)

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

	out, term, keys, done := startAltScreenApp(t, app)

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

	quitAltScreenApp(t, term, keys, done)

	out.read(func(s *termtest.Screen) {
		assert.False(t, s.IsAlternateScreen())
		termtest.AssertRow(t, s, 0, altScreenPrompt)
		termtest.AssertRow(t, s, 1, line)
		termtest.AssertNotContains(t, s, "Inbox (3)")
		termtest.AssertCursor(t, s, 0, 2)
	})
}
