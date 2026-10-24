package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// WithMask("") is documented as "show nothing".
func TestPromptState_EmptyMaskShowsNothing(t *testing.T) {
	cfg := defaultPromptConfig()
	WithMask("")(&cfg)
	state := &promptState{config: cfg, input: []rune("secret"), cursorPos: 6}
	assert.Equal(t, "", state.getDisplayInput())
}

// The line printed after submit must use the masked display, not the secret.
func TestPromptState_FinalLineMasksInput(t *testing.T) {
	cfg := defaultPromptConfig()
	WithMask("*")(&cfg)
	state := &promptState{config: cfg, input: []rune("my-secret"), cursorPos: 9}

	var buf bytes.Buffer
	state.printFinal(&buf, "Password: ")
	out := buf.String()
	if strings.Contains(out, "my-secret") {
		t.Fatalf("final line leaks the secret: %q", out)
	}
	assert.True(t, strings.Contains(out, "*********"))
}

// AutocompleteFunc works in byte offsets, as its doc example shows; the
// prompt must convert to and from rune positions.
func TestPromptState_AutocompleteMultiByte(t *testing.T) {
	cfg := defaultPromptConfig()
	var gotCursor int
	cfg.autocomplete = func(input string, cursorPos int) ([]string, int) {
		gotCursor = cursorPos
		at := strings.LastIndex(input[:cursorPos], "@")
		return []string{"file.go"}, at + 1
	}
	state := &promptState{config: cfg, input: []rune("日本 @fi"), cursorPos: 6}

	state.triggerAutocomplete()
	assert.Equal(t, len("日本 @fi"), gotCursor)
	state.applyAutocomplete()
	assert.Equal(t, "日本 @file.go", string(state.input))
	assert.Equal(t, len([]rune("日本 @file.go")), state.cursorPos)
}
