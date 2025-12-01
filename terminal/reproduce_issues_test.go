package terminal

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/gooey/assert"
	"github.com/deepnoodle-ai/gooey/require"
)

// Issue 2: Terminal Clear Style Logic
func TestTerminal_Clear_Style(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})

	// Set a style (e.g., red background)
	redBg := NewStyle().WithBgRGB(NewRGB(255, 0, 0))
	term.SetStyle(redBg)

	// Clear the screen
	term.Clear()

	// Check if the buffer cells have the style
	cell := term.backBuffer[0][0]

	// Now it should PASS
	assert.Equal(t, redBg, cell.Style, "Clear() should respect the currently set style")
}

// Issue 3: Terminal Negative Cursor Panic
func TestTerminal_NegativeCursor_Panic(t *testing.T) {
	term := NewTestTerminal(10, 5, &bytes.Buffer{})

	// Move to negative coordinates
	term.MoveCursor(-5, -5)

	// Try to perform an operation that iterates based on cursor
	require.NotPanics(t, func() {
		term.ClearToEndOfLine()
	}, "ClearToEndOfLine should not panic with negative cursor")

	// Try to print
	require.NotPanics(t, func() {
		term.Print("Hello")
	}, "Print should not panic with negative cursor")
}
