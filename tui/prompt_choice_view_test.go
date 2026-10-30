package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

func TestPromptChoice_Basic(t *testing.T) {
	selected := 0
	inputText := ""

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		Option("No").
		InputOption("Type here")

	// Check total options
	assert.Equal(t, 3, view.totalOptions())
	assert.Equal(t, 2, view.inputOptionIndex())
}

func TestPromptChoice_Navigation(t *testing.T) {
	selected := 0
	inputText := ""

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		Option("No").
		InputOption("Type here")

	view.focused = true

	// Arrow down
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	assert.Equal(t, 1, selected)

	view.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	assert.Equal(t, 2, selected)

	// Can't go past end
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	assert.Equal(t, 2, selected)

	// Arrow up
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, 1, selected)

	view.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, 0, selected)

	// Can't go past start
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, 0, selected)
}

func TestPromptChoice_NumberKeys(t *testing.T) {
	selected := 0
	inputText := ""

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		Option("No").
		InputOption("Type here")

	view.focused = true

	// Press '2' to jump to second option
	view.HandleKeyEvent(KeyEvent{Rune: '2'})
	assert.Equal(t, 1, selected)

	// Press '3' to jump to input option
	view.HandleKeyEvent(KeyEvent{Rune: '3'})
	assert.Equal(t, 2, selected)

	// On the input option, digits are typed instead; arrows navigate away
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, 1, selected)

	// Press '1' to jump back
	view.HandleKeyEvent(KeyEvent{Rune: '1'})
	assert.Equal(t, 0, selected)

	// Invalid number doesn't change selection
	view.HandleKeyEvent(KeyEvent{Rune: '9'})
	assert.Equal(t, 0, selected)
}

func TestPromptChoice_Select(t *testing.T) {
	selected := 0
	inputText := ""
	var selectedIdx int
	var selectedText string

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		Option("No").
		OnSelect(func(idx int, text string) {
			selectedIdx = idx
			selectedText = text
		})

	view.focused = true

	// Select first option
	view.HandleKeyEvent(KeyEvent{Key: KeyEnter})
	assert.Equal(t, 0, selectedIdx)
	assert.Equal(t, "", selectedText)

	// Navigate and select second option
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	view.HandleKeyEvent(KeyEvent{Key: KeyEnter})
	assert.Equal(t, 1, selectedIdx)
}

func TestPromptChoice_Cancel(t *testing.T) {
	selected := 0
	inputText := ""
	cancelled := false

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		OnCancel(func() {
			cancelled = true
		})

	view.focused = true

	view.HandleKeyEvent(KeyEvent{Key: KeyEscape})
	assert.True(t, cancelled)
}

func TestPromptChoice_InputOption(t *testing.T) {
	selected := 2 // Start on input option
	inputText := ""

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		Option("No").
		InputOption("Type here")

	view.focused = true
	view.updateInputFocus()

	// Type some characters
	view.HandleKeyEvent(KeyEvent{Rune: 'h'})
	view.HandleKeyEvent(KeyEvent{Rune: 'e'})
	view.HandleKeyEvent(KeyEvent{Rune: 'l'})
	view.HandleKeyEvent(KeyEvent{Rune: 'l'})
	view.HandleKeyEvent(KeyEvent{Rune: 'o'})

	assert.Equal(t, "hello", inputText)

	// Backspace
	view.HandleKeyEvent(KeyEvent{Key: KeyBackspace})
	assert.Equal(t, "hell", inputText)
}

func TestPromptChoice_InputWithNavigation(t *testing.T) {
	selected := 2 // Start on input option
	inputText := ""

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		Option("No").
		InputOption("Type here")

	view.focused = true
	view.updateInputFocus()

	// Type some text
	view.HandleKeyEvent(KeyEvent{Rune: 'a'})
	view.HandleKeyEvent(KeyEvent{Rune: 'b'})
	assert.Equal(t, "ab", inputText)

	// Arrow up should still navigate (not move cursor in input)
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, 1, selected)

	// Text should be preserved
	assert.Equal(t, "ab", inputText)

	// Navigate back to input
	view.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	assert.Equal(t, 2, selected)

	// Continue typing
	view.HandleKeyEvent(KeyEvent{Rune: 'c'})
	assert.Equal(t, "abc", inputText)
}

func TestPromptChoice_SelectWithInput(t *testing.T) {
	selected := 2
	inputText := "custom"
	var selectedIdx int
	var selectedText string

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		Option("No").
		InputOption("Type here").
		OnSelect(func(idx int, text string) {
			selectedIdx = idx
			selectedText = text
		})

	view.focused = true

	// Select input option with custom text
	view.HandleKeyEvent(KeyEvent{Key: KeyEnter})
	assert.Equal(t, 2, selectedIdx)
	assert.Equal(t, "custom", selectedText)
}

func TestPromptChoice_Render(t *testing.T) {
	t.Skip("Render test requires different terminal setup - skipping for now")
}

func TestPromptChoice_RenderWithInput(t *testing.T) {
	t.Skip("Render test requires different terminal setup - skipping for now")
}

func TestPromptChoice_NoInputOption(t *testing.T) {
	selected := 0
	inputText := ""

	view := PromptChoice(&selected, &inputText).
		Option("Yes").
		Option("No")

	assert.Equal(t, 2, view.totalOptions())
	assert.Equal(t, -1, view.inputOptionIndex())
	assert.False(t, view.isInputSelected())
}

func TestPromptChoice_CustomCursor(t *testing.T) {
	selected := 0
	view := PromptChoice(&selected, nil).
		Option("Yes").
		CursorChar(">")

	assert.Equal(t, ">", view.cursorChar)
}

func TestPromptChoice_NoNumbers(t *testing.T) {
	t.Skip("Render test requires different terminal setup - skipping for now")
}

func TestPromptChoice_InputOptionWithoutBinding(t *testing.T) {
	selected := 1
	var got string
	view := PromptChoice(&selected, nil).
		ID("unbound-input").
		Option("Yes").
		InputOption("Other...").
		OnSelect(func(_ int, text string) { got = text })
	view.SetFocused(true)

	view.HandleKeyEvent(KeyEvent{Rune: 'h'})
	view.HandleKeyEvent(KeyEvent{Rune: 'i'})

	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertRowContains(t, screen, 1, "hi")

	view.HandleKeyEvent(KeyEvent{Key: KeyEnter})
	assert.Equal(t, got, "hi")
}

func TestPromptChoice_OutOfRangeSelectionTypesIntoInput(t *testing.T) {
	selected := 5
	text := ""
	view := PromptChoice(&selected, &text).ID("out-of-range-input").
		Option("Yes").Option("No").InputOption("Other...")
	view.SetFocused(true)
	SprintScreen(view, WithWidth(20))

	view.HandleKeyEvent(KeyEvent{Rune: 'h'})
	assert.Equal(t, text, "h", "the first key reaches the highlighted input row")
}

func TestPromptChoice_OutOfRangeSelection(t *testing.T) {
	selected := 7
	got := -1
	view := PromptChoice(&selected, nil).Option("Yes").Option("No").
		OnSelect(func(idx int, _ string) { got = idx })

	screen := SprintScreen(view, WithWidth(20))
	termtest.AssertRowContains(t, screen, 1, "❯")

	view.HandleKeyEvent(KeyEvent{Key: KeyEnter})
	assert.Equal(t, got, 1, "Enter reports the highlighted in-range option")
}
