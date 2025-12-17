package gif

import (
	_ "embed"
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/Inconsolata-Regular.ttf
var defaultFontTTF []byte

// FontFace wraps a TrueType or OpenType font for high-quality text rendering
// to images. It maintains metrics for monospace character cells, making it
// suitable for terminal and grid-based text rendering.
//
// FontFace calculates cell dimensions automatically based on the font metrics,
// ensuring consistent spacing for terminal emulation and text layouts.
type FontFace struct {
	face       font.Face
	cellWidth  int
	cellHeight int
	ascent     int
}

// LoadFontFromBytes loads a TrueType (TTF) or OpenType (OTF) font from raw
// bytes at the specified size in points. The font is configured with standard
// 72 DPI and full hinting for best rendering quality.
//
// This function is useful for embedding custom fonts or loading fonts from
// non-standard locations.
//
// Example:
//
//	fontData, err := os.ReadFile("custom-font.ttf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	font, err := gif.LoadFontFromBytes(fontData, 14)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer font.Close()
func LoadFontFromBytes(data []byte, size float64) (*FontFace, error) {
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	return newFontFace(face), nil
}

// LoadDefaultFont loads the embedded Inconsolata monospace font at the
// specified size in points. Inconsolata is a clean, highly readable monospace
// font ideal for terminal rendering and code display.
//
// This is the recommended font for terminal GIF rendering, as it provides
// excellent readability and proper monospace character metrics.
//
// Example:
//
//	font, err := gif.LoadDefaultFont(14)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer font.Close()
//
//	// Use with terminal renderer
//	opts := gif.DefaultRendererOptions()
//	opts.Font = font
//	renderer := gif.NewTerminalRendererWithOptions(screen, opts)
func LoadDefaultFont(size float64) (*FontFace, error) {
	return LoadFontFromBytes(defaultFontTTF, size)
}

// newFontFace creates a FontFace and calculates cell dimensions.
func newFontFace(face font.Face) *FontFace {
	metrics := face.Metrics()

	// Calculate cell height from font metrics
	cellHeight := (metrics.Ascent + metrics.Descent).Ceil()

	// For monospace fonts, measure 'M' to get the cell width
	adv, _ := face.GlyphAdvance('M')
	cellWidth := adv.Ceil()

	// If GlyphAdvance failed, try measuring a string
	if cellWidth <= 0 {
		bounds, _ := font.BoundString(face, "M")
		cellWidth = (bounds.Max.X - bounds.Min.X).Ceil()
	}

	// Fallback to reasonable defaults
	if cellWidth <= 0 {
		cellWidth = cellHeight / 2
	}

	return &FontFace{
		face:       face,
		cellWidth:  cellWidth,
		cellHeight: cellHeight,
		ascent:     metrics.Ascent.Ceil(),
	}
}

// CellWidth returns the width of a single character cell in pixels. For
// monospace fonts, all characters have the same width. This value is used
// to calculate total image width for terminal rendering.
func (ff *FontFace) CellWidth() int {
	return ff.cellWidth
}

// CellHeight returns the height of a single character cell in pixels. This
// includes the space needed for ascenders, descenders, and line spacing.
// This value is used to calculate total image height for terminal rendering.
func (ff *FontFace) CellHeight() int {
	return ff.cellHeight
}

// Ascent returns the font ascent (distance from baseline to top of tallest
// glyph) in pixels. This is used internally for positioning characters
// correctly on the baseline.
func (ff *FontFace) Ascent() int {
	return ff.ascent
}

// DrawChar draws a single character at the specified pixel position on the
// image. The position (px, py) is the top-left corner of the character cell.
// The character is drawn in the foreground color with proper baseline alignment.
//
// Space characters and null runes are not drawn (optimization).
func (ff *FontFace) DrawChar(img draw.Image, px, py int, char rune, fg color.Color) {
	if char == ' ' || char == 0 {
		return
	}

	// Create a drawer for the character
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(fg),
		Face: ff.face,
		Dot:  fixed.P(px, py+ff.ascent),
	}

	d.DrawString(string(char))
}

// DrawString draws a complete string at the specified pixel position. The
// position (px, py) is the top-left corner of the first character cell.
// Characters advance horizontally according to the font's advance width.
//
// This method is primarily used internally by TerminalRenderer but can be
// used directly for custom text rendering needs.
func (ff *FontFace) DrawString(img draw.Image, px, py int, s string, fg color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(fg),
		Face: ff.face,
		Dot:  fixed.P(px, py+ff.ascent),
	}
	d.DrawString(s)
}

// Close releases resources associated with the font face. While most font
// faces don't require explicit cleanup, this method is provided for
// completeness and should be called when the font is no longer needed,
// typically using defer.
//
// Example:
//
//	font, err := gif.LoadDefaultFont(14)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer font.Close()
func (ff *FontFace) Close() error {
	if closer, ok := ff.face.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
