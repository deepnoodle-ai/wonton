package gooey

import (
	"image"
	"io"
	"testing"
)

func TestButtonDrawInSubFrame(t *testing.T) {
	// Create a button at (10, 10)
	btn := NewButton(10, 10, "Test", nil)
	// Button size: width=8 (len("Test")+4), height=1

	// Create a mock terminal (large enough)
	term := NewTestTerminal(80, 24, io.Discard)

	// 1. Draw in full frame (should use absolute coordinates)
	frame, _ := term.BeginFrame()
	btn.Draw(frame)
	term.EndFrame(frame) // End frame to flush changes if needed, but we check backBuffer

	// Verify it drew at (10, 10)
	// backBuffer is [y][x]
	cell := term.backBuffer[10][10]
	if cell.Char != '[' {
		t.Errorf("Expected '[' at (10,10), got %v", string(cell.Char))
	}

	// 2. Draw in a SubFrame matching the button size (should use relative coordinates 0,0)
	// Clear terminal first
	term.Clear()

	frame, _ = term.BeginFrame()
	// Create a SubFrame at (20, 20) with size 8x1
	subFrame := frame.SubFrame(image.Rect(20, 20, 20+8, 20+1))

	// Verify subframe size
	w, h := subFrame.Size()
	if w != 8 || h != 1 {
		t.Fatalf("SubFrame size incorrect: %dx%d", w, h)
	}

	// Draw the button into the subframe.
	// The button has X=10, Y=10.
	// If it didn't detect SubFrame, it would try to draw at (10, 10) inside the subframe.
	// Since subframe is 8x1, (10, 10) is out of bounds and nothing would be drawn (clipped).
	// If it DOES detect SubFrame, it draws at (0, 0).
	btn.Draw(subFrame)

	term.EndFrame(frame)

	// Verify it drew at the SubFrame's absolute position (20, 20)
	// (0,0) in subframe maps to (20, 20) in terminal.
	cell = term.backBuffer[20][20]
	if cell.Char != '[' {
		t.Errorf("Expected '[' at (20,20) when drawn in SubFrame, got %v", string(cell.Char))
	}
}

func TestRadioGroupDrawInSubFrame(t *testing.T) {
	rg := NewRadioGroup(10, 10, []string{"Opt1", "Opt2"})
	// RadioGroup has 2 options, so height is 2.

	term := NewTestTerminal(80, 24, io.Discard)
	frame, _ := term.BeginFrame()

	// Create subframe at (30, 10) with height 2
	subFrame := frame.SubFrame(image.Rect(30, 10, 50, 12)) // 20x2 (width doesn't match, but height does)

	rg.Draw(subFrame)

	term.EndFrame(frame)

	// Check (30, 10) for '○' (Space then circle)
	// Format is " ○ Opt1" -> Space, Circle, Space, Opt1
	// index 0: ' '
	// index 1: '○'

	// (0,0) in subframe is (30,10) in absolute.
	// so (1,0) in subframe is (31,10) absolute.

	cell := term.backBuffer[10][31] // y=10, x=31
	if cell.Char != '●' {
		t.Errorf("Expected '●' at (31,10), got %c", cell.Char)
	}
}
