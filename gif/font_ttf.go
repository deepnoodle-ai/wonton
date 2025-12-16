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

// FontFace wraps an opentype font for rendering text to images.
type FontFace struct {
	face       font.Face
	cellWidth  int
	cellHeight int
	ascent     int
}

// LoadFontFromBytes loads a TTF/OTF font from raw bytes at the specified size.
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

// LoadDefaultFont loads the embedded Inconsolata font at the specified size.
// This is the recommended default for terminal rendering.
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

// CellWidth returns the width of a single character cell in pixels.
func (ff *FontFace) CellWidth() int {
	return ff.cellWidth
}

// CellHeight returns the height of a single character cell in pixels.
func (ff *FontFace) CellHeight() int {
	return ff.cellHeight
}

// Ascent returns the font ascent (baseline to top) in pixels.
func (ff *FontFace) Ascent() int {
	return ff.ascent
}

// DrawChar draws a single character at the given pixel position on the image.
// The position (px, py) is the top-left corner of the character cell.
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

// DrawString draws a string at the given pixel position.
func (ff *FontFace) DrawString(img draw.Image, px, py int, s string, fg color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(fg),
		Face: ff.face,
		Dot:  fixed.P(px, py+ff.ascent),
	}
	d.DrawString(s)
}

// Close releases resources associated with the font face.
func (ff *FontFace) Close() error {
	if closer, ok := ff.face.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}
