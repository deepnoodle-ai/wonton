package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Moving the cursor left over multi-byte input must not panic on render.
func TestPromptChoice_RenderMultiByteCursor(t *testing.T) {
	selected := 1
	inputText := ""
	view := PromptChoice(&selected, &inputText).Option("Yes").InputOption("Other...")
	view.focused = true
	view.updateInputFocus()

	view.HandleKeyEvent(KeyEvent{Rune: '中'})
	view.HandleKeyEvent(KeyEvent{Rune: 'a'})
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowLeft})
	assert.Equal(t, "中a", inputText)

	renderWithRegistries(t, view, view.reg)
}

// A nil selected pointer must not panic on navigation keys. The selection
// has nowhere to persist, so it stays at option 0 (see PromptChoice docs).
func TestPromptChoice_NilSelected(t *testing.T) {
	view := PromptChoice(nil, nil).Option("A").Option("B")
	view.focused = true
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	view.HandleKeyEvent(KeyEvent{Rune: '2'})
}

// Digits typed into the custom input go to the input, not to quick-select.
func TestPromptChoice_DigitsTypeIntoInput(t *testing.T) {
	selected := 2
	inputText := ""
	view := PromptChoice(&selected, &inputText).Option("Yes").Option("No").InputOption("Other...")
	view.focused = true
	view.updateInputFocus()

	view.HandleKeyEvent(KeyEvent{Rune: '1'})
	view.HandleKeyEvent(KeyEvent{Rune: '2'})
	assert.Equal(t, 2, selected)
	assert.Equal(t, "12", inputText)
}
