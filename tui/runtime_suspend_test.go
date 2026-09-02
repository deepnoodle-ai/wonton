package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestSuspendLeavesAndRestoresTheScreen(t *testing.T) {
	var buf bytes.Buffer
	term := NewTestTerminal(80, 24, &buf)
	term.EnableAlternateScreen()
	term.HideCursor()
	term.EnableMouseDrag()
	r := NewRuntime(term, &testRuntimeModel{}, 30)

	buf.Reset()
	ran := false
	assert.NoError(t, r.Suspend(func(keys <-chan Event) {
		ran = true
		// Inside fn the terminal is back to normal: an app can print here and
		// the user's own terminal handles the scrollback, search and copy.
		assert.False(t, term.IsAlternateScreen(), "alternate screen must be released")
		assert.False(t, term.IsCursorHidden(), "cursor must be visible")
		assert.False(t, term.IsMouseEnabled(), "mouse reporting must be released")
	}))
	assert.True(t, ran)

	assert.True(t, term.IsAlternateScreen(), "alternate screen must come back")
	assert.True(t, term.IsCursorHidden(), "cursor must be hidden again")
	assert.True(t, term.IsMouseEnabled(), "mouse reporting must come back")

	out := buf.String()
	assert.Contains(t, out, "\033[?1049l") // left the alternate screen
	assert.Contains(t, out, "\033[?1049h") // and came back
}

func TestSuspendLeavesAlreadyDisabledModesAlone(t *testing.T) {
	var buf bytes.Buffer
	term := NewTestTerminal(80, 24, &buf)
	r := NewRuntime(term, &testRuntimeModel{}, 30)

	buf.Reset()
	assert.NoError(t, r.Suspend(func(keys <-chan Event) {}))

	assert.False(t, term.IsAlternateScreen())
	assert.False(t, term.IsMouseEnabled())
	assert.False(t, strings.Contains(buf.String(), "\033[?1049"),
		"an app that never entered the alternate screen should not be put into it")
}

func TestSuspendDeliversKeysToFnNotTheApplication(t *testing.T) {
	var buf bytes.Buffer
	term := NewTestTerminal(80, 24, &buf)
	r := NewRuntime(term, &testRuntimeModel{}, 30)

	got := make(chan Event, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, r.Suspend(func(keys <-chan Event) {
			select {
			case ev := <-keys:
				got <- ev
			case <-time.After(2 * time.Second):
			}
		}))
	}()

	// Wait for the suspend to take effect, then deliver a key the way the
	// input reader would.
	deadline := time.Now().Add(2 * time.Second)
	for r.suspendChannel() == nil && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	assert.NotNil(t, r.suspendChannel(), "Suspend should have opened a key channel")

	r.forwardEvent(KeyEvent{Rune: 'x'})
	select {
	case ev := <-got:
		key, ok := ev.(KeyEvent)
		assert.True(t, ok, "expected a KeyEvent, got %T", ev)
		assert.Equal(t, 'x', key.Rune)
	case <-time.After(2 * time.Second):
		t.Fatal("fn never received the key")
	}
	<-done

	assert.Equal(t, 0, len(r.events), "the application must not also see the key")
}

func TestSuspendRefusesToNest(t *testing.T) {
	var buf bytes.Buffer
	r := NewRuntime(NewTestTerminal(80, 24, &buf), &testRuntimeModel{}, 30)

	var inner error
	assert.NoError(t, r.Suspend(func(keys <-chan Event) {
		inner = r.Suspend(func(keys <-chan Event) {})
	}))
	assert.Equal(t, ErrSuspendReentrant, inner)
}

func TestSuspendWithNilFunctionDoesNothing(t *testing.T) {
	var buf bytes.Buffer
	term := NewTestTerminal(80, 24, &buf)
	term.EnableAlternateScreen()
	r := NewRuntime(term, &testRuntimeModel{}, 30)

	buf.Reset()
	assert.NoError(t, r.Suspend(nil))
	assert.True(t, term.IsAlternateScreen())
	assert.Equal(t, "", buf.String())
}
