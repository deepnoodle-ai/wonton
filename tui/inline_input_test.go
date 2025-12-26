package tui

import (
	"errors"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestPrompt_BasicInput(t *testing.T) {
	// TODO: Implement with simulated input
	// This would require refactoring Prompt to accept an io.Reader for input
	t.Skip("Requires terminal input simulation")
}

func TestPrompt_History(t *testing.T) {
	t.Skip("Requires terminal input simulation")
}

func TestPrompt_Autocomplete(t *testing.T) {
	t.Skip("Requires terminal input simulation")
}

func TestPrompt_CtrlC(t *testing.T) {
	t.Skip("Requires terminal input simulation")
}

func TestPrompt_MultiLine(t *testing.T) {
	t.Skip("Requires terminal input simulation")
}

// Test the autocomplete function signature
func TestAutocompleteFunc(t *testing.T) {
	fn := func(input string, cursorPos int) ([]string, int) {
		// Find @ before cursor
		atIdx := strings.LastIndex(input[:cursorPos], "@")
		if atIdx == -1 {
			return nil, 0
		}
		prefix := input[atIdx+1 : cursorPos]

		// Mock file matches
		files := []string{"app.go", "app_test.go", "application.go"}
		var matches []string
		for _, f := range files {
			if strings.HasPrefix(f, prefix) {
				matches = append(matches, f)
			}
		}
		return matches, atIdx
	}

	// Test with @ prefix
	completions, from := fn("hello @ap", 9)
	assert.Equal(t, 3, len(completions))
	assert.Equal(t, 6, from) // Position of @
	assert.Equal(t, "app.go", completions[0])

	// Test without @ prefix
	completions, from = fn("hello", 5)
	assert.Equal(t, 0, len(completions))
}

// Test prompt state operations
func TestPromptState_InsertRune(t *testing.T) {
	cfg := defaultPromptConfig()
	state := &promptState{
		config:    cfg,
		input:     []rune("hello"),
		cursorPos: 5,
	}

	state.insertRune('!')
	assert.Equal(t, "hello!", string(state.input))
	assert.Equal(t, 6, state.cursorPos)

	// Insert in middle
	state.cursorPos = 2
	state.insertRune('X')
	assert.Equal(t, "heXllo!", string(state.input))
	assert.Equal(t, 3, state.cursorPos)
}

func TestPromptState_Backspace(t *testing.T) {
	cfg := defaultPromptConfig()
	state := &promptState{
		config:    cfg,
		input:     []rune("hello"),
		cursorPos: 5,
	}

	state.backspace()
	assert.Equal(t, "hell", string(state.input))
	assert.Equal(t, 4, state.cursorPos)

	// Backspace in middle
	state.cursorPos = 2
	state.backspace()
	assert.Equal(t, "hll", string(state.input))
	assert.Equal(t, 1, state.cursorPos)

	// Backspace at start (no-op)
	state.cursorPos = 0
	state.backspace()
	assert.Equal(t, "hll", string(state.input))
	assert.Equal(t, 0, state.cursorPos)
}

func TestPromptState_DeleteChar(t *testing.T) {
	cfg := defaultPromptConfig()
	state := &promptState{
		config:    cfg,
		input:     []rune("hello"),
		cursorPos: 0,
	}

	state.deleteChar()
	assert.Equal(t, "ello", string(state.input))
	assert.Equal(t, 0, state.cursorPos)

	// Delete in middle
	state.cursorPos = 1
	state.deleteChar()
	assert.Equal(t, "elo", string(state.input))
	assert.Equal(t, 1, state.cursorPos)

	// Delete at end (no-op)
	state.cursorPos = 3
	state.deleteChar()
	assert.Equal(t, "elo", string(state.input))
	assert.Equal(t, 3, state.cursorPos)
}

func TestPromptState_DeleteWordBackward(t *testing.T) {
	cfg := defaultPromptConfig()
	state := &promptState{
		config:    cfg,
		input:     []rune("hello world test"),
		cursorPos: 16,
	}

	state.deleteWordBackward()
	assert.Equal(t, "hello world ", string(state.input))
	assert.Equal(t, 12, state.cursorPos)

	state.deleteWordBackward()
	assert.Equal(t, "hello ", string(state.input))
	assert.Equal(t, 6, state.cursorPos)

	state.deleteWordBackward()
	assert.Equal(t, "", string(state.input))
	assert.Equal(t, 0, state.cursorPos)
}

func TestPromptState_History(t *testing.T) {
	history := []string{"first", "second", "third"}
	cfg := defaultPromptConfig()
	cfg.history = &history

	state := &promptState{
		config:     cfg,
		input:      []rune("current"),
		cursorPos:  7,
		historyIdx: -1,
	}

	// Go up once
	state.historyUp()
	assert.Equal(t, "third", string(state.input))
	assert.Equal(t, 2, state.historyIdx)
	assert.Equal(t, "current", string(state.savedInput))

	// Go up again
	state.historyUp()
	assert.Equal(t, "second", string(state.input))
	assert.Equal(t, 1, state.historyIdx)

	// Go up to first
	state.historyUp()
	assert.Equal(t, "first", string(state.input))
	assert.Equal(t, 0, state.historyIdx)

	// Can't go further up
	state.historyUp()
	assert.Equal(t, "first", string(state.input))
	assert.Equal(t, 0, state.historyIdx)

	// Go down
	state.historyDown()
	assert.Equal(t, "second", string(state.input))
	assert.Equal(t, 1, state.historyIdx)

	// Go to end (restore saved)
	state.historyDown()
	state.historyDown()
	assert.Equal(t, "current", string(state.input))
	assert.Equal(t, -1, state.historyIdx)
}

func TestPromptState_Autocomplete(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.autocomplete = func(input string, cursorPos int) ([]string, int) {
		return []string{"apple", "application", "apply"}, 0
	}

	state := &promptState{
		config:    cfg,
		input:     []rune("ap"),
		cursorPos: 2,
	}

	// Trigger autocomplete
	state.triggerAutocomplete()
	assert.Equal(t, true, state.acActive)
	assert.Equal(t, 3, len(state.acMatches))
	assert.Equal(t, 0, state.acSelected)

	// Apply first match
	state.applyAutocomplete()
	assert.Equal(t, "apple", string(state.input))
	assert.Equal(t, false, state.acActive)
}

func TestPromptState_MaxLength(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.maxLength = 5

	state := &promptState{
		config:    cfg,
		input:     []rune("hello"),
		cursorPos: 5,
	}

	// Can't insert more
	state.insertRune('!')
	assert.Equal(t, "hello", string(state.input))
	assert.Equal(t, 5, state.cursorPos)
}

func TestPromptState_Mask(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.mask = "*"

	state := &promptState{
		config:    cfg,
		input:     []rune("password"),
		cursorPos: 8,
	}

	display := state.getDisplayInput()
	assert.Equal(t, "********", display)
}

func TestPromptState_Validator(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.validator = func(s string) error {
		if len(s) < 3 {
			return errors.New("too short")
		}
		return nil
	}

	state := &promptState{
		config:    cfg,
		input:     []rune("ab"),
		cursorPos: 2,
	}

	// Validator should prevent submission
	err := cfg.validator(string(state.input))
	assert.NotNil(t, err)

	state.input = []rune("abc")
	err = cfg.validator(string(state.input))
	assert.Nil(t, err)
}
