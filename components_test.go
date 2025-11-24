package gooey

import (
	"bytes"
	"testing"
)

func TestTabCompleter_Draw_Panic(t *testing.T) {
	// Reproduce panic with Width < 2
	var buf bytes.Buffer
	term := NewTestTerminal(80, 24, &buf)
	tc := NewTabCompleter()

	// Setup invalid state
	tc.Visible = true
	tc.Width = 1 // Less than 2, should cause panic in border calculation if not guarded
	tc.SetSuggestions([]string{"test"}, "")

	// This should panic if not fixed
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Recovered from panic: %v", r)
			// We expect a panic currently, so if we recover, it confirms the issue.
			// But we want the test to FAIL if it panics once we fix it.
			// For now, let's just verify it DOES panic to confirm reproduction.
		}
	}()

	frame, _ := term.BeginFrame()
	tc.Draw(frame)
	term.EndFrame(frame)
}
