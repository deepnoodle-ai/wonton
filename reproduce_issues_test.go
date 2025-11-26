package gooey

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// Issue 6: Layout Header Cursor Position
func TestLayout_HeaderCursor_Bug(t *testing.T) {
	term := NewTestTerminal(80, 24, &bytes.Buffer{})
	layout := NewLayout(term)

	header := SimpleHeader("Title", NewStyle())
	header.Right = "Right"
	layout.SetHeader(header)

	// Simulate a saved cursor position at Y=10
	term.virtualX = 5
	term.virtualY = 10
	term.SaveCursor()

	// Draw header
	frame, _ := term.BeginFrame()
	layout.drawHeader(frame)
	term.EndFrame(frame)

	// Check buffer at Y=0
	foundAtTop := false
	for x := 0; x < 80; x++ {
		if term.backBuffer[0][x].Char == 'R' {
			foundAtTop = true
			break
		}
	}

	foundAtBottom := false
	for x := 0; x < 80; x++ {
		if term.backBuffer[10][x].Char == 'R' {
			foundAtBottom = true
			break
		}
	}

	assert.True(t, foundAtTop, "Header text should be at Y=0")
	assert.False(t, foundAtBottom, "Header text should NOT be at Y=10")
}

// TestSpinner_PassiveUpdate checks that spinner updates state deterministically
func TestSpinner_PassiveUpdate(t *testing.T) {
	// This replaces the old Animator restart test with a check on the new passive Spinner
	s := NewSpinner(SpinnerDots)

	// Initial state
	assert.Equal(t, 0, s.currentFrame)

	// Update with time < interval
	now := time.Now()
	s.lastUpdate = now
	s.Update(now.Add(10 * time.Millisecond))
	assert.Equal(t, 0, s.currentFrame, "Should not advance frame before interval")

	// Update with time > interval
	s.Update(now.Add(100 * time.Millisecond))
	assert.Equal(t, 1, s.currentFrame, "Should advance frame after interval")
}
