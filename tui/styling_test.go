package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// TestCheckboxListStyling tests the new styling methods for CheckboxList
func TestCheckboxListStyling(t *testing.T) {
	items := []ListItem{
		{Label: "Item 1", Value: "1"},
		{Label: "Item 2", Value: "2"},
		{Label: "Item 3", Value: "3"},
	}
	checked := []bool{false, true, false}
	cursor := 0

	// Test that all new styling methods can be chained without panic
	list := CheckboxList(items, checked, &cursor).
		Fg(ColorWhite).
		Bg(ColorBlack).
		CursorFg(ColorGreen).
		CursorBg(ColorBlue).
		CheckedFg(ColorYellow).
		CheckedBg(ColorMagenta).
		HighlightFg(ColorCyan).
		HighlightBg(ColorRed).
		Style(NewStyle().WithForeground(ColorWhite)).
		CursorStyle(NewStyle().WithBold()).
		CheckedStyle(NewStyle().WithForeground(ColorGreen)).
		HighlightStyle(NewStyle().WithBackground(ColorBlue))

	// Verify the list was created
	assert.NotNil(t, list)
	assert.Equal(t, 3, len(list.items))

	// Verify styles are set
	assert.NotNil(t, list.style)
	assert.NotNil(t, list.cursorStyle)
	assert.NotNil(t, list.checkedStyle)
	assert.NotNil(t, list.highlightStyle)
}

// TestSpinnerStyling tests the new styling methods for Spinner
func TestSpinnerStyling(t *testing.T) {
	// Test that all new styling methods can be chained without panic
	spinner := NewSpinner(SpinnerDots).
		Fg(ColorCyan).
		Bg(ColorBlack).
		Bold().
		Dim().
		WithStyle(NewStyle().WithForeground(ColorGreen)).
		WithMessage("Loading...")

	// Verify the spinner was created
	assert.NotNil(t, spinner)
	assert.NotNil(t, spinner.style)
	assert.Equal(t, "Loading...", spinner.message)
}

// TestCheckboxListDefaultStyles tests that CheckboxList has sensible default styles
func TestCheckboxListDefaultStyles(t *testing.T) {
	items := []ListItem{{Label: "Test", Value: "test"}}
	checked := []bool{false}
	cursor := 0

	list := CheckboxList(items, checked, &cursor)

	// Verify defaults are set
	assert.NotNil(t, list)
	assert.NotNil(t, list.style)
	assert.NotNil(t, list.cursorStyle)
	assert.NotNil(t, list.checkedStyle)
	// highlightStyle is nil by default until set
	assert.Nil(t, list.highlightStyle)
}
