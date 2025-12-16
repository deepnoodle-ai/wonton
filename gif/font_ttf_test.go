package gif

import (
	"image"
	"image/color"
	"testing"
)

func TestLoadDefaultFont(t *testing.T) {
	font, err := LoadDefaultFont(14)
	if err != nil {
		t.Fatalf("failed to load default font: %v", err)
	}
	defer font.Close()

	if font.CellWidth() <= 0 {
		t.Error("expected positive cell width")
	}
	if font.CellHeight() <= 0 {
		t.Error("expected positive cell height")
	}
	if font.Ascent() <= 0 {
		t.Error("expected positive ascent")
	}

	// Font should be approximately 2:1 height:width ratio for monospace
	ratio := float64(font.CellHeight()) / float64(font.CellWidth())
	if ratio < 1.0 || ratio > 3.0 {
		t.Errorf("unexpected aspect ratio %.2f", ratio)
	}
}

func TestLoadDefaultFont_DifferentSizes(t *testing.T) {
	sizes := []float64{10, 12, 14, 16, 20}

	var prevWidth, prevHeight int
	for _, size := range sizes {
		font, err := LoadDefaultFont(size)
		if err != nil {
			t.Fatalf("failed to load font at size %.0f: %v", size, err)
		}

		// Larger sizes should produce larger dimensions
		if prevWidth > 0 && font.CellWidth() < prevWidth {
			t.Errorf("size %.0f should be wider than previous", size)
		}
		if prevHeight > 0 && font.CellHeight() < prevHeight {
			t.Errorf("size %.0f should be taller than previous", size)
		}

		prevWidth = font.CellWidth()
		prevHeight = font.CellHeight()
		font.Close()
	}
}

func TestFontFace_DrawChar(t *testing.T) {
	font, err := LoadDefaultFont(14)
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}
	defer font.Close()

	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Draw a character - shouldn't panic
	font.DrawChar(img, 10, 10, 'A', color.White)

	// Check that some pixels were drawn (not all black)
	hasWhite := false
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				hasWhite = true
				break
			}
		}
	}

	if !hasWhite {
		t.Error("expected some pixels to be drawn")
	}
}

func TestFontFace_DrawString(t *testing.T) {
	font, err := LoadDefaultFont(14)
	if err != nil {
		t.Fatalf("failed to load font: %v", err)
	}
	defer font.Close()

	img := image.NewRGBA(image.Rect(0, 0, 200, 50))

	// Draw a string
	font.DrawString(img, 10, 10, "Hello", color.White)

	// Check that some pixels were drawn
	hasContent := false
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 {
				hasContent = true
				break
			}
		}
	}

	if !hasContent {
		t.Error("expected some pixels to be drawn")
	}
}

func TestBitmapFont8x8(t *testing.T) {
	if BitmapFont8x8.Width != 8 {
		t.Errorf("expected width 8, got %d", BitmapFont8x8.Width)
	}
	if BitmapFont8x8.Height != 8 {
		t.Errorf("expected height 8, got %d", BitmapFont8x8.Height)
	}
}

func TestBitmapFont8x16(t *testing.T) {
	if BitmapFont8x16.Width != 8 {
		t.Errorf("expected width 8, got %d", BitmapFont8x16.Width)
	}
	if BitmapFont8x16.Height != 16 {
		t.Errorf("expected height 16, got %d", BitmapFont8x16.Height)
	}
}

func TestGetGlyph8x16(t *testing.T) {
	glyph := getGlyph8x16('A')

	// Check that it's 16 rows
	if len(glyph) != 16 {
		t.Errorf("expected 16 rows, got %d", len(glyph))
	}

	// Check that it has some content (not all zeros)
	hasContent := false
	for _, row := range glyph {
		if row != 0 {
			hasContent = true
			break
		}
	}
	if !hasContent {
		t.Error("expected glyph to have some content")
	}
}

func TestDrawBitmapChar(t *testing.T) {
	g := New(20, 20)

	g.AddFrame(func(f *Frame) {
		f.Fill(Black)
		DrawBitmapChar(f, 5, 5, 'X', White, BitmapFont8x8)
	})

	// Verify some white pixels were drawn
	img := g.images[0]
	hasWhite := false
	for y := 5; y < 13; y++ {
		for x := 5; x < 13; x++ {
			if img.ColorIndexAt(x, y) != 0 { // Not black
				hasWhite = true
				break
			}
		}
	}

	if !hasWhite {
		t.Error("expected some character pixels to be drawn")
	}
}

func TestDrawBitmapChar_8x16(t *testing.T) {
	g := NewWithPalette(30, 30, Palette{Black, White})

	g.AddFrame(func(f *Frame) {
		f.Fill(Black)
		DrawBitmapChar(f, 5, 5, 'A', White, BitmapFont8x16)
	})

	// Verify some white pixels were drawn in 8x16 area
	img := g.images[0]
	hasWhite := false
	for y := 5; y < 21; y++ {
		for x := 5; x < 13; x++ {
			if img.ColorIndexAt(x, y) == 1 { // White
				hasWhite = true
				break
			}
		}
	}

	if !hasWhite {
		t.Error("expected some character pixels to be drawn")
	}
}
