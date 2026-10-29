package gif

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

//go:embed fonts/NotoEmoji-Regular.ttf
var emojiFontTTF []byte

// FallbackFont is a font the terminal renderer draws a character with when
// its main font has no glyph for it, such as an emoji or a CJK ideograph.
// The renderer scales each fallback glyph to fit the character's cells, so
// a fallback font has no size of its own.
//
// A fallback font also draws a multi-rune emoji, such as a flag, a keycap,
// or a ZWJ sequence, as one glyph when the font has a ligature for it.
//
// A FallbackFont is safe for concurrent use.
type FallbackFont struct {
	open      func() (*sfnt.Font, []ligatureLookup, error) // nil once loaded
	once      sync.Once
	font      *sfnt.Font
	ligatures []ligatureLookup
}

// ParseFallbackFont parses a TrueType or OpenType font, or the first font of
// a collection (.ttc), for use as a fallback font.
//
// Example:
//
//	data, err := os.ReadFile("NotoSansCJK-Regular.ttc")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	cjk, err := gif.ParseFallbackFont(data)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	opts := gif.DefaultRendererOptions()
//	opts.FallbackFonts = []*gif.FallbackFont{gif.EmojiFallbackFont(), cjk}
func ParseFallbackFont(data []byte) (*FallbackFont, error) {
	f, ligatures, err := parseFallbackFont(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return &FallbackFont{font: f, ligatures: ligatures}, nil
}

var emojiFallback = sync.OnceValue(func() *FallbackFont {
	fb, err := ParseFallbackFont(emojiFontTTF)
	if err != nil {
		panic("gif: embedded emoji font: " + err.Error())
	}
	return fb
})

// EmojiFallbackFont returns the embedded Noto Emoji font, a monochrome emoji
// font. Its glyphs are drawn in the text color, like any other character.
func EmojiFallbackFont() *FallbackFont {
	return emojiFallback()
}

var systemFallbacks = sync.OnceValue(func() []*FallbackFont {
	var fonts []*FallbackFont
	for _, path := range systemFontPaths() {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		fonts = append(fonts, lazyFallbackFont(path))
	}
	return fonts
})

// lazyFallbackFont returns a fallback font that opens and parses the font
// file at path on first use. The file stays open, since the font reads
// glyphs from it on demand. A file that does not parse is closed.
func lazyFallbackFont(path string) *FallbackFont {
	return &FallbackFont{open: func() (*sfnt.Font, []ligatureLookup, error) {
		file, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		f, ligatures, err := parseFallbackFont(file)
		if err != nil {
			file.Close()
			return nil, nil, err
		}
		return f, ligatures, nil
	}}
}

// SystemFallbackFonts returns the installed fonts the renderer knows to cover
// CJK text and symbols, such as Hiragino on macOS, Noto Sans CJK or
// WenQuanYi on Linux, and Microsoft YaHei on Windows. A font is read only
// when a character needs it. The list is empty when none is installed.
func SystemFallbackFonts() []*FallbackFont {
	return systemFallbacks()
}

// DefaultFallbackFonts returns the fallback fonts a renderer uses when its
// options do not name any: the embedded emoji font, then the system fonts
// from SystemFallbackFonts.
func DefaultFallbackFonts() []*FallbackFont {
	return append([]*FallbackFont{EmojiFallbackFont()}, SystemFallbackFonts()...)
}

// systemFontPaths lists fonts with wide CJK and symbol coverage, by
// platform, in order of preference.
func systemFontPaths() []string {
	switch runtime.GOOS {
	case "darwin":
		return []string{
			"/System/Library/Fonts/Hiragino Sans GB.ttc",
			"/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc",
			"/System/Library/Fonts/AppleSDGothicNeo.ttc",
			"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
			"/System/Library/Fonts/Apple Symbols.ttf",
		}
	case "windows":
		dir := os.Getenv("WINDIR")
		if dir == "" {
			dir = `C:\Windows`
		}
		var paths []string
		for _, name := range []string{"msyh.ttc", "YuGothM.ttc", "msgothic.ttc", "malgun.ttf", "seguisym.ttf"} {
			paths = append(paths, filepath.Join(dir, "Fonts", name))
		}
		return paths
	default:
		return []string{
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/google-noto-cjk/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
			"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
			"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		}
	}
}

// parseFallbackFont parses the font, or the first font of a collection, in
// r, with the ligatures of its GSUB table.
func parseFallbackFont(r io.ReaderAt) (*sfnt.Font, []ligatureLookup, error) {
	var header [16]byte
	if _, err := r.ReadAt(header[:], 0); err != nil {
		return nil, nil, err
	}
	var f *sfnt.Font
	var offset int64
	if string(header[:4]) == "ttcf" {
		c, err := sfnt.ParseCollectionReaderAt(r)
		if err != nil {
			return nil, nil, err
		}
		if f, err = c.Font(0); err != nil {
			return nil, nil, err
		}
		offset = int64(binary.BigEndian.Uint32(header[12:]))
	} else {
		var err error
		if f, err = sfnt.ParseReaderAt(r); err != nil {
			return nil, nil, err
		}
	}
	return f, parseLigatures(readTable(r, offset, "GSUB")), nil
}

// maxGSUBSize bounds the GSUB table read into memory.
const maxGSUBSize = 16 << 20

// readTable returns the table with the given tag from the font whose table
// directory starts at offset, or nil if the font has no such table.
func readTable(r io.ReaderAt, offset int64, tag string) []byte {
	var dir [12]byte
	if _, err := r.ReadAt(dir[:], offset); err != nil {
		return nil
	}
	numTables := int(binary.BigEndian.Uint16(dir[4:]))
	records := make([]byte, numTables*16)
	if _, err := r.ReadAt(records, offset+12); err != nil {
		return nil
	}
	for i := 0; i < numTables; i++ {
		rec := records[i*16:]
		if string(rec[:4]) != tag {
			continue
		}
		off := binary.BigEndian.Uint32(rec[8:])
		length := binary.BigEndian.Uint32(rec[12:])
		if length > maxGSUBSize {
			return nil
		}
		table := make([]byte, length)
		if _, err := r.ReadAt(table, int64(off)); err != nil {
			return nil
		}
		return table
	}
	return nil
}

// load returns the parsed font, reading it on first use. It returns nil
// if the font cannot be read.
func (fb *FallbackFont) load() *sfnt.Font {
	fb.once.Do(func() {
		if fb.open != nil {
			fb.font, fb.ligatures, _ = fb.open()
			fb.open = nil
		}
	})
	return fb.font
}

// shape maps a grapheme cluster to the font's glyphs and applies its
// ligatures. A rune the font lacks maps to glyph 0. Variation selectors are
// dropped first: fonts list emoji ligatures without them, as shapers such as
// HarfBuzz skip them when matching.
func (fb *FallbackFont) shape(buf *sfnt.Buffer, cluster string) []sfnt.GlyphIndex {
	f := fb.load()
	if f == nil {
		return nil
	}
	var glyphs []sfnt.GlyphIndex
	for _, r := range cluster {
		if r == '\uFE0E' || r == '\uFE0F' {
			continue
		}
		g, err := f.GlyphIndex(buf, r)
		if err != nil {
			g = 0
		}
		glyphs = append(glyphs, g)
	}
	for _, lookup := range fb.ligatures {
		glyphs = lookup.apply(glyphs)
	}
	return glyphs
}

// glyphChoice is the fallback glyph a renderer draws a cluster with. A nil
// font means the renderer's TTF font draws the cluster.
type glyphChoice struct {
	font  *FallbackFont
	glyph sfnt.GlyphIndex
}

type maskKey struct {
	glyphChoice
	width int
}

// maxGlyphCache bounds each renderer glyph cache. A cache that fills up is
// emptied and filled again.
const maxGlyphCache = 4096

// chooseGlyph picks the fallback glyph for a cell, if any. A fallback font
// draws the cell when the TTF font lacks the cluster's first rune, or when
// the cluster is an emoji sequence that asks for emoji presentation. The
// first fallback font that has a glyph for the cluster's first rune wins.
// If its ligatures do not join the whole cluster, only that glyph is drawn.
func (tr *TerminalRenderer) chooseGlyph(cell TerminalCell) glyphChoice {
	text := cell.Text()
	if c, ok := tr.choices[text]; ok {
		return c
	}
	var c glyphChoice
	if isEmojiSequence(cell.Trailing) || !tr.font.hasGlyph(cell.Char) {
		for _, fb := range tr.fallbacks {
			if glyphs := fb.shape(&tr.glyphBuf, text); len(glyphs) > 0 && glyphs[0] != 0 {
				c = glyphChoice{font: fb, glyph: glyphs[0]}
				break
			}
		}
	}
	if len(tr.choices) >= maxGlyphCache {
		clear(tr.choices)
	}
	tr.choices[text] = c
	return c
}

// glyphMask returns the cached mask of a fallback glyph drawn w pixels wide.
func (tr *TerminalRenderer) glyphMask(c glyphChoice, w int) *image.Alpha {
	key := maskKey{c, w}
	if m, ok := tr.masks[key]; ok {
		return m
	}
	m := c.font.mask(&tr.glyphBuf, c.glyph, w, tr.cellHeight)
	if len(tr.masks) >= maxGlyphCache {
		clear(tr.masks)
	}
	tr.masks[key] = m
	return m
}

// isEmojiSequence reports whether the trailing runes of a cluster make it an
// emoji sequence: a variation selector, ZWJ, keycap, regional indicator,
// skin tone modifier, or tag.
func isEmojiSequence(trailing string) bool {
	for _, r := range trailing {
		switch {
		case r == '️', r == '‍', r == '⃣',
			r >= 0x1F1E6 && r <= 0x1F1FF, // regional indicators
			r >= 0x1F3FB && r <= 0x1F3FF, // skin tone modifiers
			r >= 0xE0020 && r <= 0xE007F: // tags
			return true
		}
	}
	return false
}

// mask rasterizes glyph g as an alpha mask of size w×h. The glyph is scaled
// so that the font's line height fits h and the glyph's advance fits w, and
// it is centered in the box.
func (fb *FallbackFont) mask(buf *sfnt.Buffer, g sfnt.GlyphIndex, w, h int) *image.Alpha {
	m := image.NewAlpha(image.Rect(0, 0, w, h))
	f := fb.load()
	if f == nil || w <= 0 || h <= 0 {
		return m
	}
	// Read metrics at a ppem of unitsPerEm. upem, and every metric read at
	// that size, is a 26.6 value: 64 times the value in font units.
	upem := fixed.I(int(f.UnitsPerEm()))
	metrics, err := f.Metrics(buf, upem, font.HintingNone)
	if err != nil {
		return m
	}
	advance, err := f.GlyphAdvance(buf, g, upem, font.HintingNone)
	if err != nil || advance <= 0 {
		advance = upem
	}
	lineHeight := metrics.Ascent + metrics.Descent
	if lineHeight <= 0 {
		lineHeight = upem
	}
	// scale converts those 26.6 values to pixels, so upem*scale is the
	// size in pixels per em. LoadGlyph takes that size in 26.6.
	scale := min(float32(h)/float32(lineHeight), float32(w)/float32(advance))
	pixelsPerEm := float32(upem) * scale
	segments, err := f.LoadGlyph(buf, g, fixed.Int26_6(pixelsPerEm*64), nil)
	if err != nil {
		return m
	}
	originX := (float32(w) - float32(advance)*scale) / 2
	originY := (float32(h)-float32(lineHeight)*scale)/2 + float32(metrics.Ascent)*scale

	z := vector.NewRasterizer(w, h)
	pt := func(p fixed.Point26_6) (float32, float32) {
		return originX + float32(p.X)/64, originY + float32(p.Y)/64
	}
	for _, seg := range segments {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			z.MoveTo(pt(seg.Args[0]))
		case sfnt.SegmentOpLineTo:
			z.LineTo(pt(seg.Args[0]))
		case sfnt.SegmentOpQuadTo:
			x1, y1 := pt(seg.Args[0])
			x2, y2 := pt(seg.Args[1])
			z.QuadTo(x1, y1, x2, y2)
		case sfnt.SegmentOpCubeTo:
			x1, y1 := pt(seg.Args[0])
			x2, y2 := pt(seg.Args[1])
			x3, y3 := pt(seg.Args[2])
			z.CubeTo(x1, y1, x2, y2, x3, y3)
		}
	}
	z.ClosePath()
	z.Draw(m, m.Bounds(), image.Opaque, image.Point{})
	return m
}
