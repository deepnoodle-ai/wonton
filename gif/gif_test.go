package gif

import (
	"bytes"
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
		name string
		c    color.RGBA
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
