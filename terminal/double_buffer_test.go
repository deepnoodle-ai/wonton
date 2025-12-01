package terminal

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/gooey/require"
)

// Helper to create a test terminal
func createTestTerminal(width, height int) (*Terminal, *bytes.Buffer) {
	out := new(bytes.Buffer)
	term := NewTestTerminal(width, height, out)
	return term, out
}

func TestResizeBuffer(t *testing.T) {
	term, _ := createTestTerminal(10, 10)
	term.PrintAt(0, 0, "Hello")
	term.Flush()

	require.Equal(t, 'H', term.backBuffer[0][0].Char)

	// Resize smaller
	term.resizeBuffers(5, 5)
	term.width = 5
	term.height = 5

	require.Equal(t, 5, len(term.backBuffer))
	require.Equal(t, 5, len(term.backBuffer[0]))
	require.Equal(t, 'H', term.backBuffer[0][0].Char)

	// Resize larger
	term.resizeBuffers(10, 10)
	term.width = 10
	term.height = 10

	require.Equal(t, 10, len(term.backBuffer))
	require.Equal(t, 'H', term.backBuffer[0][0].Char) // Should persist
	require.Equal(t, ' ', term.backBuffer[9][9].Char) // New space

	// Check virtual cursor
	term.MoveCursor(8, 8)
	term.resizeBuffers(5, 5) // Cursor now OOB
	term.width = 5
	term.height = 5
	// Write should be safe (ignored) or handled
	term.Print("X")
	// virtualX is 8, width is 5.
	// Expectation: Should not panic.
}

func TestMultiWidthChar(t *testing.T) {
	term, _ := createTestTerminal(20, 5)

	// Wide character 'Ａ' (U+FF21) - should take 2 cells
	term.Print("Ａ")

	// Check buffer - wide char should be at [0][0] with width 2
	require.Equal(t, 'Ａ', term.backBuffer[0][0].Char)
	require.Equal(t, 2, term.backBuffer[0][0].Width, "Wide character should have width 2")

	// The second cell should be a continuation cell
	require.Equal(t, true, term.backBuffer[0][1].Continuation, "Second cell should be continuation")

	// Next character 'B' should be at position [0][2] since wide char takes 2 cells
	term.Print("B")
	require.Equal(t, 'B', term.backBuffer[0][2].Char, "Next char should be after wide char (at position 2)")
}

func TestScrollUp(t *testing.T) {
	term, _ := createTestTerminal(10, 3)

	term.Println("Line 1")
	term.Println("Line 2")
	term.Println("Line 3") // Cursor at (0, 3) -> Scroll -> (0, 2)

	// Buffer should contain:
	// Line 2
	// Line 3
	// (empty) -> Wait, Println does: Print("Line 3"), then \n (scrolls if at bottom)

	// Check line 0
	// "Line 2 " (padded)
	require.Equal(t, 'L', term.backBuffer[0][0].Char)
	require.Equal(t, 'i', term.backBuffer[0][1].Char)
	require.Equal(t, '2', term.backBuffer[0][5].Char)

	// Check line 1
	// "Line 3 "
	require.Equal(t, '3', term.backBuffer[1][5].Char)

	// Check line 2 (cursor is here now)
	require.Equal(t, ' ', term.backBuffer[2][0].Char)
}

func TestColorsAndStyles(t *testing.T) {
	term, out := createTestTerminal(20, 5)

	style := NewStyle().WithForeground(ColorRed)
	term.SetStyle(style)
	term.Print("Red")
	term.Reset()
	term.Print("Default")

	term.Flush()

	output := out.String()
	require.Contains(t, output, "31m") // Red code
	require.Contains(t, output, "Red")
	require.Contains(t, output, "Default")
	// Should contain reset code \033[0m
	require.Contains(t, output, "\033[0m")
}

func TestClearLine_Background(t *testing.T) {
	term, _ := createTestTerminal(20, 5)

	// Set Blue BG
	bg := NewStyle().WithBackground(ColorBlue)
	term.SetStyle(bg)
	term.Print("Test") // Sets "Test" with Blue BG

	// Clear line
	term.ClearLine()

	// Check first cell style.
	// ClearLine now uses current style (Blue BG), so it should be Blue BG.
	cell := term.backBuffer[0][0]
	require.Equal(t, ColorBlue, cell.Style.Background)

	// Verify current style is still Blue BG?
	// ClearLine doesn't change current style.
	// But ClearToEndOfLine MIGHT use it?

	term.SetStyle(bg)
	term.PrintAt(0, 1, "Test2")
	term.MoveCursor(0, 1)
	term.ClearToEndOfLine()

	// ClearToEndOfLine uses current style if BG is set
	cell2 := term.backBuffer[1][0]
	require.Equal(t, ColorBlue, cell2.Style.Background)
}

func TestOutOfBounds(t *testing.T) {
	term, _ := createTestTerminal(10, 10)

	require.NotPanics(t, func() {
		term.PrintAt(-1, -1, "OutOfBounds")
		term.PrintAt(100, 100, "WayOut")
	})

	// Verify nothing written to (0,0)
	require.Equal(t, ' ', term.backBuffer[0][0].Char)
}
