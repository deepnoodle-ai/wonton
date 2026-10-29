package gif

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/sfnt"
)

// TestEmojiFallbackFont_Shape checks that emoji sequences join into the
// font's ligature glyph, named in the font's post table.
func TestEmojiFallbackFont_Shape(t *testing.T) {
	fb := EmojiFallbackFont()
	var buf sfnt.Buffer
	cases := []struct {
		name  string
		input string
		want  []string // glyph names; "" for a rune the font lacks
	}{
		{"flag", "\U0001F1EF\U0001F1F5", []string{"flagJapan"}},
		{"keycap", "#️⃣", []string{"keycap_hash"}},
		{"heart VS16", "❤️", []string{"redHeart"}},
		{"ZWJ ligature", "❤️‍\U0001F525", []string{"heartOnFire"}},
		{"unknown ZWJ sequence", "\U0001F680‍\U0001F680", []string{"u1F680", "uni200D", "u1F680"}},
		{"missing rune", "A", []string{""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			glyphs := fb.shape(&buf, tc.input)
			got := make([]string, len(glyphs))
			for i, g := range glyphs {
				if g != 0 {
					got[i], _ = fb.font.GlyphName(&buf, g)
				}
			}
			if len(got) != len(tc.want) {
				t.Fatalf("glyphs = %q, want %q", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("glyphs = %q, want %q", got, tc.want)
				}
			}
		})
	}
}

func TestParseFallbackFont_Invalid(t *testing.T) {
	if _, err := ParseFallbackFont([]byte("not a font")); err == nil {
		t.Error("ParseFallbackFont accepted invalid data")
	}
}

// TestParseFallbackFont_Collection parses a .ttc built by wrapping the
// emoji font in a collection header.
func TestParseFallbackFont_Collection(t *testing.T) {
	header := []byte{'t', 't', 'c', 'f', 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 16}
	data := append(header, emojiFontTTF...)
	// Table offsets in a collection are from the start of the file.
	data = shiftTableOffsets(t, data, 16)

	fb, err := ParseFallbackFont(data)
	if err != nil {
		t.Fatal(err)
	}
	var buf sfnt.Buffer
	if glyphs := fb.shape(&buf, "\U0001F1EF\U0001F1F5"); len(glyphs) != 1 {
		t.Errorf("flag shaped to %d glyphs, want 1: ligatures not read from the collection", len(glyphs))
	}
}

// shiftTableOffsets adds delta to each table offset of the font whose
// table directory starts at delta.
func shiftTableOffsets(t *testing.T, data []byte, delta int) []byte {
	t.Helper()
	numTables := int(data[delta+4])<<8 | int(data[delta+5])
	for i := 0; i < numTables; i++ {
		rec := data[delta+12+i*16:]
		off := uint32(rec[8])<<24 | uint32(rec[9])<<16 | uint32(rec[10])<<8 | uint32(rec[11])
		off += uint32(delta)
		rec[8], rec[9], rec[10], rec[11] = byte(off>>24), byte(off>>16), byte(off>>8), byte(off)
	}
	return data
}

func FuzzParseLigatures(f *testing.F) {
	gsub := readTable(bytes.NewReader(emojiFontTTF), 0, "GSUB")
	f.Add(gsub)
	f.Add(gsub[:len(gsub)/2])
	f.Add([]byte{0, 1, 0, 0, 0, 10, 0, 10, 0, 10})
	f.Fuzz(func(t *testing.T, b []byte) {
		for _, lookup := range parseLigatures(b) {
			lookup.apply([]sfnt.GlyphIndex{1, 2, 3})
		}
	})
}

func TestParseLigatures_Truncated(t *testing.T) {
	gsub := readTable(bytes.NewReader(emojiFontTTF), 0, "GSUB")
	if len(parseLigatures(gsub)) == 0 {
		t.Fatal("no ligature lookups in the emoji font")
	}
	for n := 0; n < len(gsub); n += 97 {
		parseLigatures(gsub[:n])
	}
}

// litPixels counts the pixels brighter than mid-gray in the given cells of
// a rendered row, with no padding.
func litPixels(tr *TerminalRenderer, firstCell, cells int) int {
	img := tr.GIF().images[len(tr.GIF().images)-1]
	n := 0
	for y := 0; y < tr.cellHeight; y++ {
		for x := firstCell * tr.cellWidth; x < (firstCell+cells)*tr.cellWidth; x++ {
			if r, g, b, _ := img.At(x, y).RGBA(); r+g+b > 3*0x8000 {
				n++
			}
		}
	}
	return n
}

func renderRow(text string, fallbacks []*FallbackFont) *TerminalRenderer {
	screen := NewTerminalScreen(6, 1)
	screen.WriteString(text, White, Black)
	tr := NewTerminalRendererWithOptions(screen, RendererOptions{FontSize: 14, FallbackFonts: fallbacks})
	tr.RenderFrame(10)
	return tr
}

func TestTerminalRenderer_FallbackDrawsEmoji(t *testing.T) {
	if n := litPixels(renderRow("\U0001F680", nil), 0, 2); n == 0 {
		t.Error("default fallback fonts drew no pixels for an emoji")
	}
	if n := litPixels(renderRow("\U0001F680", []*FallbackFont{}), 0, 2); n != 0 {
		t.Errorf("with fallback off, the emoji lit %d pixels, want 0", n)
	}
}

// TestTerminalRenderer_FlagStaysInItsCells checks that a flag is drawn as
// one glyph inside its two cells.
func TestTerminalRenderer_FlagStaysInItsCells(t *testing.T) {
	tr := renderRow("\U0001F1EF\U0001F1F5", []*FallbackFont{EmojiFallbackFont()})
	if n := litPixels(tr, 0, 2); n == 0 {
		t.Error("flag drew no pixels")
	}
	if n := litPixels(tr, 2, 4); n != 0 {
		t.Errorf("flag lit %d pixels outside its cells", n)
	}
}

// TestTerminalRenderer_KeycapUsesEmojiFont checks that an emoji sequence
// is drawn by the fallback font even though the main font has its base.
func TestTerminalRenderer_KeycapUsesEmojiFont(t *testing.T) {
	keycap := litPixels(renderRow("#️⃣", nil), 0, 2)
	plain := litPixels(renderRow("#", nil), 0, 2)
	if keycap == plain {
		t.Errorf("keycap lit %d pixels, the same as a plain #", keycap)
	}
}

func TestTerminalRenderer_MainFontWinsForItsGlyphs(t *testing.T) {
	with := renderRow("A#", nil).GIF().images[0]
	without := renderRow("A#", []*FallbackFont{}).GIF().images[0]
	if !samePixels(with, without) {
		t.Error("fallback fonts changed text the main font can draw")
	}
}

func TestTerminalRenderer_SystemFontDrawsCJK(t *testing.T) {
	var buf sfnt.Buffer
	found := false
	for _, fb := range SystemFallbackFonts() {
		if glyphs := fb.shape(&buf, "世"); len(glyphs) > 0 && glyphs[0] != 0 {
			found = true
			break
		}
	}
	if !found {
		t.Skip("no installed fallback font has CJK glyphs")
	}
	if n := litPixels(renderRow("世", nil), 0, 2); n == 0 {
		t.Error("default fallback fonts drew no pixels for 世")
	}
}

// TestFallbackFont_UnreadableFileIsSkipped checks that a system font that
// fails to parse draws nothing, and the next fallback font is tried.
func TestFallbackFont_UnreadableFileIsSkipped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.ttf")
	if err := os.WriteFile(path, []byte("not a font"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := &FallbackFont{open: func() (*sfnt.Font, []ligatureLookup, error) {
		file, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		return parseFallbackFont(file)
	}}
	var buf sfnt.Buffer
	if glyphs := broken.shape(&buf, "\U0001F680"); glyphs != nil {
		t.Errorf("unreadable font shaped to %v, want nil", glyphs)
	}
	tr := renderRow("\U0001F680", []*FallbackFont{broken, EmojiFallbackFont()})
	if n := litPixels(tr, 0, 2); n == 0 {
		t.Error("the font after the unreadable one drew nothing")
	}
}

func samePixels(a, b *image.Paletted) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	for i := range a.Pix {
		if a.Pix[i] != b.Pix[i] {
			return false
		}
	}
	return true
}
