package gif

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"testing"
)

func TestNew(t *testing.T) {
	g := New(100, 100)
	if g.Width() != 100 {
		t.Errorf("expected width 100, got %d", g.Width())
	}
	if g.Height() != 100 {
		t.Errorf("expected height 100, got %d", g.Height())
	}
	if g.FrameCount() != 0 {
		t.Errorf("expected 0 frames, got %d", g.FrameCount())
	}
}

func TestNewWithPalette(t *testing.T) {
	palette := Palette{White, Black, Red}
	g := NewWithPalette(50, 50, palette)
	if g.Width() != 50 {
		t.Errorf("expected width 50, got %d", g.Width())
	}
}

func TestAddFrame(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.SetPixel(5, 5, Black)
	})
	if g.FrameCount() != 1 {
		t.Errorf("expected 1 frame, got %d", g.FrameCount())
	}
}

func TestAddFrameWithDelay(t *testing.T) {
	g := New(10, 10)
	g.AddFrameWithDelay(nil, 50)
	if g.FrameCount() != 1 {
		t.Errorf("expected 1 frame, got %d", g.FrameCount())
	}
}

func TestSetLoopCount(t *testing.T) {
	// Test loop forever (0) - requires multiple frames for NETSCAPE extension
	g := New(10, 10)
	g.SetLoopCount(0)
	g.AddFrame(nil)
	g.AddFrame(nil)

	data, err := g.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error: %v", err)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeAll() error: %v", err)
	}
	if decoded.LoopCount != 0 {
		t.Errorf("expected loop count 0 (infinite), got %d", decoded.LoopCount)
	}
}

func TestFrameFill(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		f.Fill(Black)
	})

	img := g.images[0]
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if img.ColorIndexAt(x, y) != 1 { // Black is index 1 in DefaultPalette
				t.Errorf("expected black at (%d, %d)", x, y)
			}
		}
	}
}

func TestFrameFillRect(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.FillRect(2, 2, 3, 3, Black)
	})

	img := g.images[0]
	// Check that the rectangle is filled
	for y := 2; y < 5; y++ {
		for x := 2; x < 5; x++ {
			if img.ColorIndexAt(x, y) != 1 { // Black
				t.Errorf("expected black at (%d, %d)", x, y)
			}
		}
	}
	// Check that outside the rectangle is white
	if img.ColorIndexAt(0, 0) != 0 { // White
		t.Error("expected white at (0, 0)")
	}
}

func TestFrameDrawLine(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.DrawLine(0, 0, 9, 9, Black)
	})

	img := g.images[0]
	// Check diagonal pixels
	for i := 0; i < 10; i++ {
		if img.ColorIndexAt(i, i) != 1 { // Black
			t.Errorf("expected black at (%d, %d)", i, i)
		}
	}
}

func TestFrameDrawRect(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.DrawRect(2, 2, 5, 5, Black)
	})

	img := g.images[0]
	// Check corners
	corners := []struct{ x, y int }{
		{2, 2}, {6, 2}, {2, 6}, {6, 6},
	}
	for _, c := range corners {
		if img.ColorIndexAt(c.x, c.y) != 1 {
			t.Errorf("expected black at corner (%d, %d)", c.x, c.y)
		}
	}
	// Check center is white
	if img.ColorIndexAt(4, 4) != 0 {
		t.Error("expected white at center (4, 4)")
	}
}

func TestFrameDrawCircle(t *testing.T) {
	g := New(20, 20)
	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.DrawCircle(10, 10, 5, Black)
	})

	img := g.images[0]
	// Check some points on the circle
	if img.ColorIndexAt(15, 10) != 1 { // Right
		t.Error("expected black at right of circle")
	}
	if img.ColorIndexAt(5, 10) != 1 { // Left
		t.Error("expected black at left of circle")
	}
}

func TestFrameFillCircle(t *testing.T) {
	g := New(20, 20)
	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.FillCircle(10, 10, 3, Black)
	})

	img := g.images[0]
	// Check center
	if img.ColorIndexAt(10, 10) != 1 {
		t.Error("expected black at center")
	}
	// Check outside circle
	if img.ColorIndexAt(0, 0) != 0 {
		t.Error("expected white outside circle")
	}
}

func TestFrameSetPixelIndex(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		f.SetPixelIndex(5, 5, 1) // Black
	})

	img := g.images[0]
	if img.ColorIndexAt(5, 5) != 1 {
		t.Error("expected palette index 1 at (5, 5)")
	}
}

func TestFrameBoundsCheck(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		// These should not panic
		f.SetPixel(-1, 0, Black)
		f.SetPixel(0, -1, Black)
		f.SetPixel(10, 0, Black)
		f.SetPixel(0, 10, Black)
		f.SetPixelIndex(-1, 0, 1)
		f.SetPixelIndex(0, 100, 1)
		f.SetPixelIndex(0, 0, 255) // Invalid index
	})
}

func TestEncode(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		f.Fill(White)
	})
	g.AddFrame(func(f *Frame) {
		f.Fill(Black)
	})

	var buf bytes.Buffer
	if err := g.Encode(&buf); err != nil {
		t.Fatalf("Encode() error: %v", err)
	}

	// Verify it's a valid GIF
	decoded, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatalf("DecodeAll() error: %v", err)
	}
	if len(decoded.Image) != 2 {
		t.Errorf("expected 2 frames, got %d", len(decoded.Image))
	}
}

func TestBytes(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(nil)

	data, err := g.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
	// Check GIF magic number
	if string(data[:3]) != "GIF" {
		t.Error("expected GIF magic number")
	}
}

func TestRGB(t *testing.T) {
	c := RGB(255, 128, 64)
	if c.R != 255 || c.G != 128 || c.B != 64 || c.A != 255 {
		t.Errorf("unexpected color: %v", c)
	}
}

func TestRGBA(t *testing.T) {
	c := RGBA(255, 128, 64, 128)
	if c.R != 255 || c.G != 128 || c.B != 64 || c.A != 128 {
		t.Errorf("unexpected color: %v", c)
	}
}

func TestGrayscale(t *testing.T) {
	p := Grayscale(16)
	if len(p) != 16 {
		t.Errorf("expected 16 colors, got %d", len(p))
	}

	// Check first is black
	r, g, b, _ := p[0].RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Error("expected first color to be black")
	}

	// Check last is white
	r, g, b, _ = p[15].RGBA()
	if r != 0xffff || g != 0xffff || b != 0xffff {
		t.Error("expected last color to be white")
	}
}

func TestGrayscaleBounds(t *testing.T) {
	// Test minimum
	p := Grayscale(1)
	if len(p) != 2 {
		t.Errorf("expected minimum 2 colors, got %d", len(p))
	}

	// Test maximum
	p = Grayscale(500)
	if len(p) != 256 {
		t.Errorf("expected maximum 256 colors, got %d", len(p))
	}
}

func TestNewDimensionValidation(t *testing.T) {
	tests := []struct {
		name             string
		width, height    int
		expectW, expectH int
	}{
		{"normal", 100, 100, 100, 100},
		{"zero width", 0, 100, 1, 100},
		{"zero height", 100, 0, 100, 1},
		{"negative width", -5, 100, 1, 100},
		{"negative height", 100, -10, 100, 1},
		{"both zero", 0, 0, 1, 1},
		{"both negative", -1, -1, 1, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := New(tc.width, tc.height)
			if g.Width() != tc.expectW {
				t.Errorf("width: got %d, want %d", g.Width(), tc.expectW)
			}
			if g.Height() != tc.expectH {
				t.Errorf("height: got %d, want %d", g.Height(), tc.expectH)
			}
		})
	}
}

func TestNewWithPaletteValidation(t *testing.T) {
	// Empty palette should use default
	g := NewWithPalette(10, 10, Palette{})
	if len(g.palette) != len(DefaultPalette) {
		t.Errorf("empty palette: got %d colors, want %d", len(g.palette), len(DefaultPalette))
	}

	// Oversized palette should be truncated
	bigPalette := make(Palette, 300)
	for i := range bigPalette {
		bigPalette[i] = color.RGBA{uint8(i % 256), 0, 0, 255}
	}
	g = NewWithPalette(10, 10, bigPalette)
	if len(g.palette) != 256 {
		t.Errorf("big palette: got %d colors, want 256", len(g.palette))
	}

	// Valid palette should work as-is
	smallPalette := Palette{White, Black, Red}
	g = NewWithPalette(10, 10, smallPalette)
	if len(g.palette) != 3 {
		t.Errorf("small palette: got %d colors, want 3", len(g.palette))
	}
}

func TestNegativeDelayValidation(t *testing.T) {
	g := New(10, 10)
	g.AddFrameWithDelay(nil, -100)

	data, err := g.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error: %v", err)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeAll() error: %v", err)
	}
	if decoded.Delay[0] != 0 {
		t.Errorf("expected delay 0 (clamped from negative), got %d", decoded.Delay[0])
	}
}

func TestAddImageNilValidation(t *testing.T) {
	g := New(10, 10)
	g.AddImage(nil, 10) // Should not panic or add a frame
	if g.FrameCount() != 0 {
		t.Errorf("nil image should not add frame, got %d frames", g.FrameCount())
	}
}

func TestAddImageNegativeDelay(t *testing.T) {
	g := New(10, 10)
	img := image.NewPaletted(image.Rect(0, 0, 10, 10), g.palette)
	g.AddImage(img, -50)

	data, err := g.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error: %v", err)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeAll() error: %v", err)
	}
	if decoded.Delay[0] != 0 {
		t.Errorf("expected delay 0 (clamped from negative), got %d", decoded.Delay[0])
	}
}

func TestWebSafe(t *testing.T) {
	p := WebSafe()
	if len(p) != 216 {
		t.Errorf("expected 216 colors, got %d", len(p))
	}
}

func TestFrameImage(t *testing.T) {
	g := New(10, 10)
	g.AddFrame(func(f *Frame) {
		img := f.Image()
		if img == nil {
			t.Error("expected non-nil image")
		}
		if img.Bounds().Dx() != 10 || img.Bounds().Dy() != 10 {
			t.Error("unexpected image bounds")
		}
	})
}

func TestFrameDimensions(t *testing.T) {
	g := New(20, 30)
	g.AddFrame(func(f *Frame) {
		if f.Width() != 20 {
			t.Errorf("expected width 20, got %d", f.Width())
		}
		if f.Height() != 30 {
			t.Errorf("expected height 30, got %d", f.Height())
		}
	})
}

func TestMultipleFrameAnimation(t *testing.T) {
	g := New(50, 50)
	for i := 0; i < 10; i++ {
		offset := i * 4
		g.AddFrameWithDelay(func(f *Frame) {
			f.Fill(White)
			f.FillCircle(25, 25, 5+offset, Red)
		}, 5)
	}

	data, err := g.Bytes()
	if err != nil {
		t.Fatalf("Bytes() error: %v", err)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeAll() error: %v", err)
	}
	if len(decoded.Image) != 10 {
		t.Errorf("expected 10 frames, got %d", len(decoded.Image))
	}
	for i, delay := range decoded.Delay {
		if delay != 5 {
			t.Errorf("frame %d: expected delay 5, got %d", i, delay)
		}
	}
}

func TestPredefinedColors(t *testing.T) {
	colors := []struct {
		name       string
		c          color.RGBA
		r, g, b, a uint8
	}{
		{"Black", Black, 0, 0, 0, 255},
		{"White", White, 255, 255, 255, 255},
		{"Red", Red, 255, 0, 0, 255},
		{"Green", Green, 0, 255, 0, 255},
		{"Blue", Blue, 0, 0, 255, 255},
		{"Yellow", Yellow, 255, 255, 0, 255},
		{"Cyan", Cyan, 0, 255, 255, 255},
		{"Magenta", Magenta, 255, 0, 255, 255},
		{"Transparent", Transparent, 0, 0, 0, 0},
	}

	for _, tc := range colors {
		if tc.c.R != tc.r || tc.c.G != tc.g || tc.c.B != tc.b || tc.c.A != tc.a {
			t.Errorf("%s: expected (%d,%d,%d,%d), got (%d,%d,%d,%d)",
				tc.name, tc.r, tc.g, tc.b, tc.a, tc.c.R, tc.c.G, tc.c.B, tc.c.A)
		}
	}
}

func TestChainedAPI(t *testing.T) {
	g := New(10, 10).
		SetLoopCount(3).
		AddFrame(nil).
		AddFrameWithDelay(nil, 20)

	if g.FrameCount() != 2 {
		t.Errorf("expected 2 frames, got %d", g.FrameCount())
	}
}

// Example demonstrates creating a simple animated GIF.
func Example() {
	// Create a 100x100 pixel GIF
	g := New(100, 100)

	// Add 10 frames showing a moving circle
	for i := 0; i < 10; i++ {
		g.AddFrame(func(f *Frame) {
			f.Fill(White)
			f.FillCircle(20+i*8, 50, 10, Red)
		})
	}

	// Save to file
	if err := g.Save("animation.gif"); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// ExampleNew demonstrates creating a basic GIF with the default palette.
func ExampleNew() {
	g := New(200, 100)

	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.FillRect(50, 25, 100, 50, Blue)
		f.DrawRect(49, 24, 102, 52, Black)
	})

	g.Save("simple.gif")
}

// ExampleNewWithPalette demonstrates creating a GIF with a custom palette.
func ExampleNewWithPalette() {
	// Create a custom palette with grayscale colors
	palette := Grayscale(16)

	g := NewWithPalette(100, 100, palette)

	// Create frames with different shades
	for i := 0; i < 16; i++ {
		g.AddFrame(func(f *Frame) {
			f.Fill(palette[i])
		})
	}

	g.Save("grayscale.gif")
}

// ExampleGIF_AddFrame demonstrates adding frames with drawing operations.
func ExampleGIF_AddFrame() {
	g := New(150, 150)

	// Add frames showing geometric shapes
	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.FillCircle(75, 75, 50, Red)
	})

	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.FillRect(25, 25, 100, 100, Blue)
	})

	g.AddFrame(func(f *Frame) {
		f.Fill(White)
		f.DrawLine(0, 0, 150, 150, Green)
		f.DrawLine(150, 0, 0, 150, Green)
	})

	g.Save("shapes.gif")
}

// ExampleGIF_SetLoopCount demonstrates controlling animation looping.
func ExampleGIF_SetLoopCount() {
	g := New(100, 50)

	// Set to loop 3 times (not infinite)
	g.SetLoopCount(3)

	for i := 0; i < 5; i++ {
		g.AddFrameWithDelay(func(f *Frame) {
			f.Fill(White)
			f.FillCircle(10+i*20, 25, 10, Red)
		}, 20) // 200ms delay
	}

	g.Save("limited-loop.gif")
}

// ExampleFrame_DrawLine demonstrates drawing lines on a frame.
func ExampleFrame_DrawLine() {
	g := New(200, 200)

	g.AddFrame(func(f *Frame) {
		f.Fill(White)

		// Draw a grid
		for i := 0; i <= 200; i += 20 {
			f.DrawLine(i, 0, i, 200, RGB(200, 200, 200))
			f.DrawLine(0, i, 200, i, RGB(200, 200, 200))
		}

		// Draw diagonal lines
		f.DrawLine(0, 0, 200, 200, Red)
		f.DrawLine(200, 0, 0, 200, Blue)
	})

	g.Save("lines.gif")
}

// ExampleFrame_DrawCircle demonstrates drawing circles.
func ExampleFrame_DrawCircle() {
	g := New(200, 200)

	g.AddFrame(func(f *Frame) {
		f.Fill(White)

		// Draw concentric circles
		for r := 20; r <= 100; r += 20 {
			f.DrawCircle(100, 100, r, Black)
		}
	})

	g.Save("circles.gif")
}

// ExampleGrayscale demonstrates creating a grayscale palette.
func ExampleGrayscale() {
	// Create a 32-shade grayscale palette
	palette := Grayscale(32)

	g := NewWithPalette(320, 100, palette)

	g.AddFrame(func(f *Frame) {
		// Draw a gradient bar showing all shades
		for i := 0; i < 32; i++ {
			f.FillRect(i*10, 0, 10, 100, palette[i])
		}
	})

	g.Save("gradient.gif")
}

func TestTerminal256(t *testing.T) {
	palette := Terminal256()

	// Should have exactly 256 colors
	if len(palette) != 256 {
		t.Errorf("Terminal256: got %d colors, want 256", len(palette))
	}

	// Check standard ANSI colors (0-15)
	// Color 0 should be black
	if c, ok := palette[0].(color.RGBA); !ok || c.R != 0 || c.G != 0 || c.B != 0 {
		t.Errorf("Terminal256[0]: expected black, got %v", palette[0])
	}
	// Color 15 should be bright white
	if c, ok := palette[15].(color.RGBA); !ok || c.R != 255 || c.G != 255 || c.B != 255 {
		t.Errorf("Terminal256[15]: expected white, got %v", palette[15])
	}

	// Check color cube (16-231)
	// Color 16 should be RGB(0, 0, 0) - start of cube
	if c, ok := palette[16].(color.RGBA); !ok || c.R != 0 || c.G != 0 || c.B != 0 {
		t.Errorf("Terminal256[16]: expected (0,0,0), got %v", palette[16])
	}
	// Color 231 should be RGB(255, 255, 255) - end of cube
	if c, ok := palette[231].(color.RGBA); !ok || c.R != 255 || c.G != 255 || c.B != 255 {
		t.Errorf("Terminal256[231]: expected (255,255,255), got %v", palette[231])
	}

	// Check grayscale ramp (232-255)
	// Color 232 should be dark gray (8, 8, 8)
	if c, ok := palette[232].(color.RGBA); !ok || c.R != 8 || c.G != 8 || c.B != 8 {
		t.Errorf("Terminal256[232]: expected (8,8,8), got %v", palette[232])
	}
	// Color 255 should be light gray (238, 238, 238)
	if c, ok := palette[255].(color.RGBA); !ok || c.R != 238 || c.G != 238 || c.B != 238 {
		t.Errorf("Terminal256[255]: expected (238,238,238), got %v", palette[255])
	}
}

// ExampleTerminal256 demonstrates creating a terminal-style palette.
func ExampleTerminal256() {
	// Use the standard xterm 256-color palette for terminal rendering
	palette := Terminal256()

	g := NewWithPalette(400, 300, palette)

	g.AddFrame(func(f *Frame) {
		// Dark background like a terminal
		f.Fill(palette[0]) // Black

		// The palette includes all colors needed for anti-aliased text
	})

	g.Save("terminal.gif")
}
