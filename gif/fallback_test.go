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

// TestCoverageCountsGlyphsAsWork checks that a format 2 range charges
// work per glyph. Six bytes can cover 65,536 glyphs, so counting reads
// alone let a crafted table run for billions of steps.
func TestCoverageCountsGlyphsAsWork(t *testing.T) {
	// Format 2, one range: glyphs 0 to 999 at coverage index 0.
	b := []byte{0, 2, 0, 1, 0, 0, 0x03, 0xE7, 0, 0}
	r := &gsubReader{b: b}
	if got := r.coverage(0, 1000); len(got) != 1000 {
		t.Fatalf("coverage returned %d glyphs, want 1000", len(got))
	}
	if r.work < 1000 {
		t.Errorf("work = %d after covering 1000 glyphs, want at least 1000", r.work)
	}
}

// TestLigatureSkipsUncoveredSlots checks that a coverage table shorter
// than the ligature set count adds no ligature on glyph 0, .notdef.
func TestLigatureSkipsUncoveredSlots(t *testing.T) {
	b := []byte{
		0, 1, // format 1
		0, 10, // coverage offset
		0, 2, // ligature set count
		0, 16, 0, 16, // both sets at offset 16
		0, 1, 0, 1, 0, 5, // coverage format 1: one glyph, 5
		0, 1, 0, 4, // set: one ligature at offset 4
		0, 9, 0, 2, 0, 7, // ligature: glyph 9 replaces the first glyph, then 7
	}
	table := ligatureLookup{}
	(&gsubReader{b: b}).readLigatureSubst(table, 0)
	if len(table[5]) != 1 {
		t.Fatalf("glyph 5 has %d ligatures, want 1", len(table[5]))
	}
	if len(table[0]) != 0 {
		t.Errorf("glyph 0 (.notdef) has %d ligatures, want 0", len(table[0]))
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

// litBounds returns the bounding box of the pixels that are not black
// background in the given cells of a rendered row, with no padding. It
// counts gray pixels too, since thin outlines anti-alias to gray.
func litBounds(tr *TerminalRenderer, firstCell, cells int) image.Rectangle {
	img := tr.GIF().images[len(tr.GIF().images)-1]
	var box image.Rectangle
	for y := 0; y < tr.cellHeight; y++ {
		for x := firstCell * tr.cellWidth; x < (firstCell+cells)*tr.cellWidth; x++ {
			if r, g, b, _ := img.At(x, y).RGBA(); r+g+b > 0 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
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

// TestTerminalRenderer_FallbackGlyphFillsItsCells checks that a fallback
// glyph is scaled to its cells: its lit area spans most of their height
// and more than one of their two columns.
func TestTerminalRenderer_FallbackGlyphFillsItsCells(t *testing.T) {
	for _, text := range []string{"\U0001F680", "\U0001F1EF\U0001F1F5", "❤️"} {
		tr := renderRow(text, []*FallbackFont{EmojiFallbackFont()})
		box := litBounds(tr, 0, 2)
		if box.Dy() < tr.cellHeight/2 || box.Dx() <= tr.cellWidth {
			t.Errorf("%+q: lit area %dx%d, want at least %dx%d of the %dx%d cells",
				text, box.Dx(), box.Dy(), tr.cellWidth+1, tr.cellHeight/2, 2*tr.cellWidth, tr.cellHeight)
		}
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
// fails to parse draws nothing and is closed, and the next fallback font is
// tried.
func TestFallbackFont_UnreadableFileIsSkipped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.ttf")
	if err := os.WriteFile(path, []byte("not a font"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := lazyFallbackFont(path)
	var buf sfnt.Buffer
	if glyphs := broken.shape(&buf, "\U0001F680"); glyphs != nil {
		t.Errorf("unreadable font shaped to %v, want nil", glyphs)
	}
	// Windows cannot remove a file that is still open.
	if err := os.Remove(path); err != nil {
		t.Errorf("unreadable font file left open: %v", err)
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
