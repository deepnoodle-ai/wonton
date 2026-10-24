package gif

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/termsession"
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
	// xterm cube levels are 0, 95, 135, 175, 215, 255, so RGB = (255, 0, 0)
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

func rowText(s *TerminalScreen, y int) string {
	var b strings.Builder
	for _, c := range s.Cells[y] {
		b.WriteRune(c.Char)
	}
	return strings.TrimRight(b.String(), " ")
}

func TestEmulator_UnrecognizedEscapeKeepsText(t *testing.T) {
	cases := []struct {
		name, input, want string
	}{
		{"cursor style with intermediate", "\x1b[0 qHello, World!\x1b[0m", "Hello, World!"},
		{"colon SGR", "\x1b[4:3mHello\x1b[0m", "Hello"},
		{"DECSC/DECRC", "\x1b7Hello\x1b8\x1b[0m", "Hello"},
		{"keypad mode", "\x1b=Hello\x1b>\x1b[0m", "Hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEmulator(40, 3)
			em.ProcessOutput(tc.input)
			if got := rowText(em.Screen(), 0); got != tc.want {
				t.Errorf("row 0 = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEmulator_PrivateSGRKeepsColors(t *testing.T) {
	em := NewEmulator(20, 3)
	em.ProcessOutput("\x1b[31m\x1b[>4;2mRed")
	if got := em.Screen().Cells[0][0].FG; got != ansiColor(1) {
		t.Errorf("fg = %v, want red", got)
	}
}

func TestEmulator_ColonSGRColors(t *testing.T) {
	em := NewEmulator(20, 3)
	em.ProcessOutput("\x1b[38:2::10:20:30mA\x1b[38:5:196mB\x1b[31;4:3mC")
	cells := em.Screen().Cells[0]
	if cells[0].FG != RGB(10, 20, 30) {
		t.Errorf("A fg = %v, want (10,20,30)", cells[0].FG)
	}
	if cells[1].FG != color256(196) {
		t.Errorf("B fg = %v, want color 196", cells[1].FG)
	}
	if cells[2].FG != ansiColor(1) {
		t.Errorf("C fg = %v, want red (4:3 must not reset)", cells[2].FG)
	}
}

func TestEmulator_PrivateEraseDisplay(t *testing.T) {
	em := NewEmulator(10, 3)
	em.ProcessOutput("abc\r\ndef\x1b[1;1H\x1b[?2J")
	if got := rowText(em.Screen(), 1); got != "" {
		t.Errorf("row 1 = %q after ESC[?2J, want empty", got)
	}
}

func TestColor256MatchesPalette(t *testing.T) {
	p := Terminal256()
	for i := 0; i < 256; i++ {
		if got := color256(i); got != p[i] {
			t.Errorf("color256(%d) = %v, palette has %v", i, got, p[i])
		}
	}
	// Out-of-range indexes must not panic or wrap.
	_ = color256(-1)
	_ = color256(300)
}

func TestEmulator_ResizeFromZero(t *testing.T) {
	em := NewEmulator(10, 5)
	em.Resize(0, 0)
	em.Resize(10, 5)
	em.Screen().WriteString("A", nil, nil) // must not panic
	em.ProcessOutput("B")
	if got := rowText(em.Screen(), 0); got != "AB" {
		t.Errorf("row 0 = %q, want %q", got, "AB")
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

// renderScreenImage renders the output of a fresh emulator to one frame with
// the same renderer settings RenderCastEvents uses by default.
func renderScreenImage(cols, rows int, output string) *image.Paletted {
	em := NewEmulator(cols, rows)
	em.ProcessOutput(output)
	opts := DefaultCastOptions()
	r := NewTerminalRendererWithOptions(em.Screen(), RendererOptions{
		FontSize: opts.FontSize, BitmapFont: BitmapFont8x16, Padding: opts.Padding,
	})
	r.RenderFrame(1)
	return r.GIF().images[0]
}

func TestRenderCastEvents_FrameShowsScreenBeforeEvent(t *testing.T) {
	header := &termsession.RecordingHeader{Version: 2, Width: 20, Height: 5}
	events := []termsession.RecordingEvent{
		{Time: 0.0, Type: "o", Data: "Screen 1"},
		{Time: 1.5, Type: "o", Data: "\rScreen 2"},
	}
	g, err := RenderCastEvents(header, events, DefaultCastOptions())
	if err != nil {
		t.Fatal(err)
	}
	if g.FrameCount() != 2 {
		t.Fatalf("frames = %d, want 2", g.FrameCount())
	}
	if !bytes.Equal(g.images[0].Pix, renderScreenImage(20, 5, "Screen 1").Pix) {
		t.Error("frame 0 does not show Screen 1")
	}
	if g.delays[0] != 150 {
		t.Errorf("frame 0 delay = %d, want 150 (Screen 1 is visible for 1.5s)", g.delays[0])
	}
	if !bytes.Equal(g.images[1].Pix, renderScreenImage(20, 5, "Screen 1\rScreen 2").Pix) {
		t.Error("frame 1 does not show Screen 2")
	}
}

func TestRenderCastEvents_NilHeader(t *testing.T) {
	events := []termsession.RecordingEvent{{Time: 0.1, Type: "o", Data: "hi"}}
	opts := DefaultCastOptions()
	opts.Cols, opts.Rows = 10, 3
	g, err := RenderCastEvents(nil, events, opts)
	if err != nil {
		t.Fatal(err)
	}
	if g.FrameCount() == 0 {
		t.Error("no frames rendered")
	}
}

func TestRenderCastEvents_ResizeEvent(t *testing.T) {
	header := &termsession.RecordingHeader{Version: 2, Width: 5, Height: 2}
	events := []termsession.RecordingEvent{
		{Time: 0.0, Type: "o", Data: "ab"},
		{Time: 0.5, Type: "r", Data: "10x3"},
		{Time: 1.0, Type: "o", Data: "cdefghij"},
	}
	g, err := RenderCastEvents(header, events, DefaultCastOptions())
	if err != nil {
		t.Fatal(err)
	}
	// The canvas fits the largest size in the recording.
	want := renderScreenImage(10, 3, "abcdefghij")
	if g.Width() != want.Bounds().Dx() || g.Height() != want.Bounds().Dy() {
		t.Fatalf("canvas = %dx%d, want %dx%d", g.Width(), g.Height(), want.Bounds().Dx(), want.Bounds().Dy())
	}
	last := g.images[g.FrameCount()-1]
	if !bytes.Equal(last.Pix, want.Pix) {
		t.Error("last frame does not show the text on one 10-column row")
	}
}

func TestFrameDrawRectEmpty(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		f.Fill(Black)
		f.DrawRect(5, 5, 0, 0, White)
	})
	for _, px := range g.images[0].Pix {
		if px != g.images[0].Pix[0] {
			t.Fatal("DrawRect with zero size drew pixels")
		}
	}
}
