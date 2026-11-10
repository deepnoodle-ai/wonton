package tui

import (
	"bytes"
	"testing"
	"time"
)

// keyRoutingApp has a focused InputField; it records which KeyEvents reach
// HandleEvent and quits on Ctrl+C or on a custom doneEvent.
type keyRoutingApp struct {
	text     string
	received []rune
	gotCtrlC bool
}

func (a *keyRoutingApp) View() View {
	return Stack(
		InputField(&a.text).ID("field"),
	)
}

func (a *keyRoutingApp) HandleEvent(event Event) []Cmd {
	switch e := event.(type) {
	case KeyEvent:
		if isInterruptKey(e) {
			a.gotCtrlC = true
			return []Cmd{Quit()}
		}
		a.received = append(a.received, e.Rune)
	case doneEvent:
		return []Cmd{Quit()}
	}
	return nil
}

type doneEvent struct{}

func (e doneEvent) Timestamp() time.Time { return time.Now() }

// TestConsumedKeysNotDeliveredToApp verifies that keys consumed by a focused
// input do not reach the app's HandleEvent (so typing 'q' into a field can't
// trigger a 'q'-to-quit shortcut), while Ctrl+C is always delivered.
func TestConsumedKeysNotDeliveredToApp(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	app := &keyRoutingApp{}
	runtime := NewRuntime(terminal, app, 30)

	go func() {
		// Let the first render register and focus the input field.
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(KeyEvent{Rune: 'q', Key: KeyUnknown})
		runtime.SendEvent(KeyEvent{Rune: 'x', Key: KeyUnknown})
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(KeyEvent{Key: KeyCtrlC})
	}()

	if err := runtime.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if app.text != "qx" {
		t.Errorf("input field should have received typed keys, got %q", app.text)
	}
	if len(app.received) != 0 {
		t.Errorf("consumed keys should not reach HandleEvent, got %q", string(app.received))
	}
	if !app.gotCtrlC {
		t.Error("Ctrl+C should always be delivered to HandleEvent")
	}
}

// Keys that arrive before the first frame still reach the focused input.
// Before the first render, the focus manager knows no inputs, so without a
// first frame these keys would fall through to HandleEvent.
func TestKeysBeforeFirstFrameReachFocusedInput(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	app := &keyRoutingApp{}
	runtime := NewRuntime(terminal, app, 30)

	// Queue the keys before Run, so the loop takes them before it has
	// rendered anything.
	runtime.SendEvent(KeyEvent{Rune: 'h', Key: KeyUnknown})
	runtime.SendEvent(KeyEvent{Rune: 'i', Key: KeyUnknown})
	runtime.SendEvent(doneEvent{})

	if err := runtime.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if app.text != "hi" {
		t.Errorf("input field should have received the early keys, got %q", app.text)
	}
	if len(app.received) != 0 {
		t.Errorf("early keys should not reach HandleEvent, got %q", string(app.received))
	}
}

// unconsumedKeyApp has no focusable elements, so every key reaches the app.
type unconsumedKeyApp struct {
	received []rune
}

func (a *unconsumedKeyApp) View() View { return Text("static") }

func (a *unconsumedKeyApp) HandleEvent(event Event) []Cmd {
	if e, ok := event.(KeyEvent); ok {
		a.received = append(a.received, e.Rune)
		if e.Rune == 'q' {
			return []Cmd{Quit()}
		}
	}
	return nil
}

// TestUnconsumedKeysDeliveredToApp verifies the common case: with no focused
// input consuming keys, the app receives them as before.
func TestUnconsumedKeysDeliveredToApp(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)

	app := &unconsumedKeyApp{}
	runtime := NewRuntime(terminal, app, 30)

	go func() {
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(KeyEvent{Rune: 'a', Key: KeyUnknown})
		runtime.SendEvent(KeyEvent{Rune: 'q', Key: KeyUnknown})
	}()

	if err := runtime.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if string(app.received) != "aq" {
		t.Errorf("expected app to receive %q, got %q", "aq", string(app.received))
	}
}
