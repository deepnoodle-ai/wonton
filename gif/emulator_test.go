package gif

import (
	"testing"
)

func TestEmulator_Basic(t *testing.T) {
	em := NewEmulator(80, 24)

	if em.Screen().Width != 80 {
		t.Errorf("expected width 80, got %d", em.Screen().Width)
	}
	if em.Screen().Height != 24 {
		t.Errorf("expected height 24, got %d", em.Screen().Height)
	}
}

func TestEmulator_WriteText(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("Hello")

	// Check that "Hello" was written at position 0,0
	screen := em.Screen()
	expected := "Hello"
	for i, ch := range expected {
		if screen.Cells[0][i].Char != ch {
			t.Errorf("position %d: expected '%c', got '%c'", i, ch, screen.Cells[0][i].Char)
		}
	}

	// Cursor should be at position 5
	if screen.CursorX != 5 {
		t.Errorf("expected cursor X=5, got %d", screen.CursorX)
	}
}

func TestEmulator_Newline(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("Line1\nLine2")

	screen := em.Screen()

	// Check first line
	if screen.Cells[0][0].Char != 'L' {
		t.Error("expected 'L' at (0,0)")
	}

	// Check second line
	if screen.Cells[1][0].Char != 'L' {
		t.Error("expected 'L' at (0,1)")
	}
}

func TestEmulator_CarriageReturn(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("AAAA\rBB")

	screen := em.Screen()

	// Should have "BBAA" on first line
	if screen.Cells[0][0].Char != 'B' {
		t.Errorf("expected 'B' at (0,0), got '%c'", screen.Cells[0][0].Char)
	}
	if screen.Cells[0][1].Char != 'B' {
		t.Errorf("expected 'B' at (1,0), got '%c'", screen.Cells[0][1].Char)
	}
	if screen.Cells[0][2].Char != 'A' {
		t.Errorf("expected 'A' at (2,0), got '%c'", screen.Cells[0][2].Char)
	}
}

func TestEmulator_CursorMovement(t *testing.T) {
	em := NewEmulator(80, 24)

	// Move cursor to position (10, 5) using CSI H
	em.ProcessOutput("\x1b[6;11H") // Row 6, Col 11 (1-indexed)

	screen := em.Screen()
	if screen.CursorX != 10 {
		t.Errorf("expected cursor X=10, got %d", screen.CursorX)
	}
	if screen.CursorY != 5 {
		t.Errorf("expected cursor Y=5, got %d", screen.CursorY)
	}
}

func TestEmulator_CursorUp(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("\x1b[10;10H") // Move to (9, 9)
	em.ProcessOutput("\x1b[3A")     // Move up 3

	screen := em.Screen()
	if screen.CursorY != 6 {
		t.Errorf("expected cursor Y=6, got %d", screen.CursorY)
	}
}

func TestEmulator_CursorDown(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("\x1b[3B") // Move down 3

	screen := em.Screen()
	if screen.CursorY != 3 {
		t.Errorf("expected cursor Y=3, got %d", screen.CursorY)
	}
}

func TestEmulator_ClearScreen(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("Hello World")
	em.ProcessOutput("\x1b[2J") // Clear screen

	screen := em.Screen()
	if screen.Cells[0][0].Char != ' ' {
		t.Error("expected screen to be cleared")
	}
}

func TestEmulator_ClearLine(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("Hello World")
	em.ProcessOutput("\x1b[5G") // Move to column 5
	em.ProcessOutput("\x1b[0K") // Clear to end of line

	screen := em.Screen()

	// First 4 chars should remain
	if screen.Cells[0][0].Char != 'H' {
		t.Error("expected 'H' at (0,0)")
	}

	// Position 5+ should be cleared
	if screen.Cells[0][5].Char != ' ' {
		t.Errorf("expected space at (5,0), got '%c'", screen.Cells[0][5].Char)
	}
}

func TestEmulator_SGR_Colors(t *testing.T) {
	em := NewEmulator(80, 24)

	// Set red foreground
	em.ProcessOutput("\x1b[31mRed")

	screen := em.Screen()

	// Check that 'R' has red foreground
	cell := screen.Cells[0][0]
	r, g, b, _ := cell.FG.RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

	// ANSI red is RGB(170, 0, 0)
	if r8 != 170 || g8 != 0 || b8 != 0 {
		t.Errorf("expected red (170,0,0), got (%d,%d,%d)", r8, g8, b8)
	}
}

func TestEmulator_SGR_Reset(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("\x1b[31mRed\x1b[0mNormal")

	screen := em.Screen()

	// 'N' should be white (default)
	cell := screen.Cells[0][3]
	r, g, b, _ := cell.FG.RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

	if r8 != 255 || g8 != 255 || b8 != 255 {
		t.Errorf("expected white (255,255,255), got (%d,%d,%d)", r8, g8, b8)
	}
}

func TestEmulator_256Colors(t *testing.T) {
	em := NewEmulator(80, 24)

	// Set 256-color foreground (index 196 = bright red)
	em.ProcessOutput("\x1b[38;5;196mX")

	screen := em.Screen()
	cell := screen.Cells[0][0]
	r, g, b, _ := cell.FG.RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

	// Index 196 in 6x6x6 cube: (196-16) = 180, r=180/36=5, g=(180/6)%6=0, b=180%6=0
	// So RGB = (5*51, 0, 0) = (255, 0, 0)
	if r8 != 255 || g8 != 0 || b8 != 0 {
		t.Errorf("expected (255,0,0), got (%d,%d,%d)", r8, g8, b8)
	}
}

func TestEmulator_TrueColor(t *testing.T) {
	em := NewEmulator(80, 24)

	// Set true color foreground RGB(100, 150, 200)
	em.ProcessOutput("\x1b[38;2;100;150;200mX")

	screen := em.Screen()
	cell := screen.Cells[0][0]
	r, g, b, _ := cell.FG.RGBA()
	r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)

	if r8 != 100 || g8 != 150 || b8 != 200 {
		t.Errorf("expected (100,150,200), got (%d,%d,%d)", r8, g8, b8)
	}
}

func TestEmulator_Write(t *testing.T) {
	em := NewEmulator(80, 24)

	n, err := em.Write([]byte("Hello"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected n=5, got %d", n)
	}

	screen := em.Screen()
	if screen.Cells[0][0].Char != 'H' {
		t.Error("expected 'H' at (0,0)")
	}
}

func TestEmulator_Resize(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("Hello")
	em.Resize(40, 12)

	screen := em.Screen()
	if screen.Width != 40 {
		t.Errorf("expected width 40, got %d", screen.Width)
	}
	if screen.Height != 12 {
		t.Errorf("expected height 12, got %d", screen.Height)
	}

	// Content should be preserved
	if screen.Cells[0][0].Char != 'H' {
		t.Error("expected 'H' at (0,0) after resize")
	}
}

func TestEmulator_Reset(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("Hello")
	em.Reset()

	screen := em.Screen()
	if screen.Cells[0][0].Char != ' ' {
		t.Error("expected screen to be cleared after reset")
	}
	if screen.CursorX != 0 || screen.CursorY != 0 {
		t.Error("expected cursor to be at (0,0) after reset")
	}
}

func TestEmulator_Tab(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("AB\tC")

	screen := em.Screen()
	// Tab should move to column 8
	if screen.Cells[0][8].Char != 'C' {
		t.Errorf("expected 'C' at column 8, got '%c'", screen.Cells[0][8].Char)
	}
}

func TestEmulator_Backspace(t *testing.T) {
	em := NewEmulator(80, 24)

	em.ProcessOutput("ABC\bX")

	screen := em.Screen()
	// Backspace moves cursor back, then X overwrites C
	if screen.Cells[0][2].Char != 'X' {
		t.Errorf("expected 'X' at column 2, got '%c'", screen.Cells[0][2].Char)
	}
}

func TestEmulator_PrivateCSI(t *testing.T) {
	em := NewEmulator(80, 24)

	// Private CSI sequences like ESC[?25l (hide cursor) should be consumed
	// without leaving literal text on screen
	em.ProcessOutput("\x1b[?25lHello\x1b[?25h")

	screen := em.Screen()
	// Should have "Hello" starting at position 0, not "[?25lHello"
	if screen.Cells[0][0].Char != 'H' {
		t.Errorf("expected 'H' at (0,0), got '%c' - private CSI not consumed", screen.Cells[0][0].Char)
	}
	if screen.CursorX != 5 {
		t.Errorf("expected cursor X=5, got %d", screen.CursorX)
	}
}
