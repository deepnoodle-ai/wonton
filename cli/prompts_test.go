package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

// TestInputPromptSubmitsOnEnter verifies that Enter reaches the input
// prompt and submits, even though the focused tui.Input sees it first.
func TestInputPromptSubmitsOnEnter(t *testing.T) {
	var buf bytes.Buffer
	terminal := tui.NewTestTerminal(80, 24, &buf)

	value := ""
	done := false
	app := &inputPrompt{prompt: "Name:", value: &value, done: &done}
	runtime := tui.NewRuntime(terminal, app, 30)

	go func() {
		time.Sleep(50 * time.Millisecond)
		runtime.SendEvent(tui.KeyEvent{Rune: 'A'})
		runtime.SendEvent(tui.KeyEvent{Key: tui.KeyEnter})
		time.Sleep(200 * time.Millisecond)
		runtime.SendEvent(tui.KeyEvent{Key: tui.KeyCtrlC})
	}()

	if err := runtime.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !done {
		t.Fatal("Enter did not submit the input prompt")
	}
	if value != "A" {
		t.Fatalf("value = %q, want %q", value, "A")
	}
}
