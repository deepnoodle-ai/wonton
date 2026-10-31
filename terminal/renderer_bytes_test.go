package terminal

import (
	"bytes"
	"fmt"
	"image"
	"math/rand/v2"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/runewidth"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// The tests in this file check the bytes that the diff renderer sends, not
// only the back buffer. Each frame's output goes into a termtest.Screen, which
// interprets it as a terminal would. The test then compares every screen cell
// with a model of the cells that the test set. A renderer change that makes a
// terminal show the wrong frame (a skipped cell, a wrong cursor move, a lost
// style) fails here with the seed and frame number.

const (
	rendererBytesSeeds  = 48
	rendererBytesFrames = 80
)

// Glyph pools. The SetCell pool holds single runes that start a grapheme
// cluster, so a cell never joins the glyph before it on a real terminal.
var (
	rbNarrowRunes = []rune("abcXYZ019#.─│╭é Ω")
	rbWideRunes   = []rune("中日한😀　")
	rbClusters    = []string{
		"a", "Z", "ok", "─", "中", "😀",
		"❤️",      // VS16 emoji presentation
		"👨‍👩‍👧",   // ZWJ sequence
		"👍🏽",      // skin tone modifier
		"🇯🇵",      // regional indicator flag
		"1️⃣",     // keycap
		"e\u0301", // combining mark
	}
)

// rbModel is the test's own record of the cells it set. It applies the same
// wide-glyph rules as the back buffer: a write that covers half of a wide
// glyph blanks the other half with the default style.
type rbModel struct {
	w, h  int
	cells [][]Cell
}

func newRBModel(w, h int) *rbModel {
	m := &rbModel{w: w, h: h, cells: make([][]Cell, h)}
	for y := range m.cells {
		m.cells[y] = make([]Cell, w)
		for x := range m.cells[y] {
			m.cells[y][x] = rbBlank()
		}
	}
	return m
}

func rbBlank() Cell { return Cell{Char: ' ', Style: NewStyle(), Width: 1} }

// put writes a glyph of the given width at (x, y). The caller makes sure the
// glyph fits on the row.
func (m *rbModel) put(x, y int, first rune, trailing string, width int, style Style) {
	row := m.cells[y]
	for p := x; p < x+width; p++ {
		if row[p].Continuation && p == x && x > 0 {
			row[x-1] = rbBlank()
		}
		if row[p].Width == 2 && p+1 == x+width && p+1 < m.w {
			row[p+1] = rbBlank()
		}
	}
	row[x] = Cell{Char: first, Trailing: trailing, Style: style, Width: width}
	if width == 2 {
		row[x+1] = Cell{Style: style, Continuation: true}
	}
}

func (m *rbModel) fill(r image.Rectangle, char rune, style Style) {
	cw := runewidth.RuneWidth(char)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x += max(cw, 1) {
			if x+cw > r.Max.X {
				for ; x < r.Max.X; x++ {
					m.put(x, y, ' ', "", 1, style)
				}
				break
			}
			m.put(x, y, char, "", cw, style)
		}
	}
}

func (m *rbModel) restyle(x, y int, style Style) {
	if m.cells[y][x].Continuation && x > 0 {
		x--
	}
	m.cells[y][x].Style = style
	if m.cells[y][x].Width == 2 && x+1 < m.w {
		m.cells[y][x+1].Style = style
	}
}

// rbScreenStyle converts a terminal style to the style termtest reports for
// text drawn with it. termtest does not track hyperlinks, so URL is dropped.
func rbScreenStyle(s Style) termtest.Style {
	return termtest.Style{
		Foreground: rbScreenColor(s.Foreground, s.FgRGB),
		Background: rbScreenColor(s.Background, s.BgRGB),
		Bold:       s.Bold,
		Dim:        s.Dim,
		Italic:     s.Italic,
		Underline:  s.Underline,
		Blink:      s.Blink,
		Reverse:    s.Reverse,
		Hidden:     s.Hidden,
		Strike:     s.Strikethrough,
	}
}

func rbScreenColor(c Color, rgb *RGB) termtest.Color {
	switch {
	case rgb != nil:
		return termtest.Color{Type: termtest.ColorRGB, R: rgb.R, G: rgb.G, B: rgb.B}
	case c < 0:
		return termtest.Color{}
	case c < 16:
		return termtest.Color{Type: termtest.ColorBasic, Value: uint8(c)}
	default:
		return termtest.Color{Type: termtest.Color256, Value: uint8(c)}
	}
}

// rbScreenCell converts a model cell to the termtest cell a terminal shows.
func rbScreenCell(c Cell) termtest.Cell {
	return termtest.Cell{
		Char:         c.Char,
		Trailing:     c.Trailing,
		Width:        c.Width,
		Style:        rbScreenStyle(c.Style),
		Continuation: c.Continuation,
	}
}

// rbCompare returns a description of the first cell where the screen differs
// from the expected cells, or "" when they match.
func rbCompare(screen *termtest.Screen, w, h int, expected func(x, y int) Cell) string {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			want := rbScreenCell(expected(x, y))
			got := screen.Cell(x, y)
			if got.Char == want.Char && got.Trailing == want.Trailing &&
				got.Width == want.Width && got.Continuation == want.Continuation &&
				got.Style == want.Style {
				continue
			}
			return fmt.Sprintf("cell (%d,%d):\n  want %s\n  got  %s\n  want row %q\n  got row  %q",
				x, y, rbFormatCell(want), rbFormatCell(got), rbRowText(expected, w, y), screen.Row(y))
		}
	}
	return ""
}

// rbModelDiff returns a description of the first cell where the back buffer
// differs from the model, or "" when they match.
func rbModelDiff(m *rbModel, term *Terminal) string {
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			want, got := m.cells[y][x], term.getCellInternal(x, y)
			if !got.Equal(want) {
				return fmt.Sprintf("cell (%d,%d):\n  model       %s url=%q\n  back buffer %s url=%q", x, y,
					rbFormatCell(rbScreenCell(want)), want.Style.URL, rbFormatCell(rbScreenCell(got)), got.Style.URL)
			}
		}
	}
	return ""
}

func rbFormatCell(c termtest.Cell) string {
	return fmt.Sprintf("glyph=%q width=%d cont=%v style=[%s]",
		string(c.Char)+c.Trailing, c.Width, c.Continuation, rbFormatStyle(c.Style))
}

func rbFormatStyle(s termtest.Style) string {
	parts := []string{"fg=" + rbFormatColor(s.Foreground), "bg=" + rbFormatColor(s.Background)}
	for _, a := range []struct {
		on   bool
		name string
	}{
		{s.Bold, "bold"}, {s.Dim, "dim"}, {s.Italic, "italic"}, {s.Underline, "underline"},
		{s.Blink, "blink"}, {s.Reverse, "reverse"}, {s.Hidden, "hidden"}, {s.Strike, "strike"},
	} {
		if a.on {
			parts = append(parts, a.name)
		}
	}
	return strings.Join(parts, " ")
}

func rbFormatColor(c termtest.Color) string {
	switch c.Type {
	case termtest.ColorBasic:
		return fmt.Sprintf("%d", c.Value)
	case termtest.Color256:
		return fmt.Sprintf("256:%d", c.Value)
	case termtest.ColorRGB:
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	default:
		return "default"
	}
}

func rbRowText(expected func(x, y int) Cell, w, y int) string {
	var b strings.Builder
	for x := 0; x < w; x++ {
		c := expected(x, y)
		if c.Continuation {
			continue
		}
		b.WriteRune(c.Char)
		b.WriteString(c.Trailing)
	}
	return strings.TrimRight(b.String(), " ")
}

// rbGen draws random frame content from a seeded generator.
type rbGen struct{ r *rand.Rand }

func (g rbGen) pick(n int) int { return g.r.IntN(n) }
func (g rbGen) chance(p float64) bool {
	return g.r.Float64() < p
}

func (g rbGen) color() (Color, *RGB) {
	switch g.pick(4) {
	case 0:
		return ColorDefault, nil
	case 1:
		return Color(g.pick(16)), nil
	case 2:
		return Color(16 + g.pick(240)), nil
	default:
		rgb := NewRGB(uint8(g.pick(256)), uint8(g.pick(256)), uint8(g.pick(256)))
		return ColorDefault, &rgb
	}
}

func (g rbGen) style() Style {
	s := NewStyle()
	if g.chance(0.3) {
		return s
	}
	s.Foreground, s.FgRGB = g.color()
	s.Background, s.BgRGB = g.color()
	s.Bold = g.chance(0.2)
	s.Dim = g.chance(0.1)
	s.Italic = g.chance(0.15)
	s.Underline = g.chance(0.15)
	s.Blink = g.chance(0.05)
	s.Reverse = g.chance(0.1)
	s.Hidden = g.chance(0.05)
	s.Strikethrough = g.chance(0.1)
	if g.chance(0.1) {
		s.URL = fmt.Sprintf("https://example.com/%d", g.pick(3))
	}
	return s
}

func (g rbGen) cellRune() rune {
	if g.chance(0.3) {
		return rbWideRunes[g.pick(len(rbWideRunes))]
	}
	return rbNarrowRunes[g.pick(len(rbNarrowRunes))]
}

func (g rbGen) rect(w, h int) image.Rectangle {
	x0, y0 := g.pick(w), g.pick(h)
	return image.Rect(x0, y0, x0+1+g.pick(w-x0), y0+1+g.pick(h-y0))
}

// garbage returns bytes that something outside the renderer, such as a
// subprocess, could write to the screen.
func (g rbGen) garbage(w, h int) string {
	var b strings.Builder
	if g.chance(0.5) {
		b.WriteString("\x1b[2J")
	}
	for range 1 + g.pick(6) {
		fmt.Fprintf(&b, "\x1b[%d;%dH", 1+g.pick(h), 1+g.pick(w))
		b.WriteString(g.style().String())
		for range 1 + g.pick(w) {
			b.WriteRune(g.cellRune())
		}
	}
	return b.String()
}

// rbDraw applies random operations to frame and the same changes to the
// model. The frame covers bounds, in terminal coordinates.
func rbDraw(g rbGen, frame RenderFrame, m *rbModel, bounds image.Rectangle) error {
	fw, fh := bounds.Dx(), bounds.Dy()
	for range g.pick(1 + fw*fh/2) {
		lx, ly := g.pick(fw), g.pick(fh)
		x, y := bounds.Min.X+lx, bounds.Min.Y+ly
		switch op := g.pick(10); {
		case op < 5:
			ch, st := g.cellRune(), g.style()
			cw := runewidth.RuneWidth(ch)
			err := frame.SetCell(lx, ly, ch, st)
			if x+cw > bounds.Max.X {
				if err == nil {
					return fmt.Errorf("SetCell(%d,%d,%q) past the frame edge succeeded", lx, ly, ch)
				}
				continue
			}
			if err != nil {
				return fmt.Errorf("SetCell(%d,%d,%q): %w", lx, ly, ch, err)
			}
			m.put(x, y, ch, "", cw, st)
		case op < 8:
			// A run of clusters, cut so it fits on the row: PrintStyled
			// would wrap the rest, and the model does not wrap.
			st := g.style()
			var text strings.Builder
			cx := x
			for range 1 + g.pick(5) {
				cluster := rbClusters[g.pick(len(rbClusters))]
				for c, cw := range runewidth.Graphemes(cluster) {
					if cx+cw > bounds.Max.X {
						break
					}
					first, size := utf8.DecodeRuneInString(c)
					m.put(cx, y, first, c[size:], cw, st)
					text.WriteString(c)
					cx += cw
				}
			}
			if err := frame.PrintStyled(lx, ly, text.String(), st); err != nil {
				return fmt.Errorf("PrintStyled(%d,%d,%q): %w", lx, ly, text.String(), err)
			}
		default:
			st := g.style()
			if err := frame.RestyleCell(lx, ly, st); err != nil {
				return fmt.Errorf("RestyleCell(%d,%d): %w", lx, ly, err)
			}
			m.restyle(x, y, st)
		}
	}
	return nil
}

// TestRendererBytesMatchSetCells drives the diff renderer through random
// frames and checks that a terminal fed its output shows the cells the test
// set. A failure names the seed and frame; rerun one seed with
// -run 'TestRendererBytesMatchSetCells/seed=N'.
func TestRendererBytesMatchSetCells(t *testing.T) {
	for seed := range uint64(rendererBytesSeeds) {
		t.Run(fmt.Sprintf("seed=%d", seed), func(t *testing.T) {
			t.Parallel()
			if err := runRendererBytesSeed(seed); err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
		})
	}
}

func runRendererBytesSeed(seed uint64) error {
	g := rbGen{rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15))}
	// Small sizes reach the edges often; include 1-column and 1-row screens.
	w, h := 1+g.pick(40), 1+g.pick(12)

	var out bytes.Buffer
	term := NewTestTerminal(w, h, &out)
	screen := termtest.NewScreen(w, h)
	model := newRBModel(w, h)
	full := image.Rect(0, 0, w, h)

	// A tui app switches to the alternate screen before its first frame.
	if seed%2 == 0 {
		term.EnableAlternateScreen()
	}

	for frameNo := 0; frameNo < rendererBytesFrames; frameNo++ {
		kind := "partial"
		switch p := g.pick(100); {
		case frameNo == 0 || p < 10:
			kind = "full"
		case p < 20:
			kind = "clear"
		case p < 30:
			kind = "invalidate"
		case p < 35:
			kind = "empty"
		}

		switch kind {
		case "clear":
			st := g.style()
			st.URL = "" // Clear fills with the current style; keep it plain
			term.SetStyle(st)
			term.Clear()
			term.SetStyle(NewStyle())
			for y := range h {
				for x := range w {
					model.cells[y][x] = Cell{Char: ' ', Style: st, Width: 1}
				}
			}
		case "invalidate":
			// Another program draws over the screen; the app invalidates
			// so the next frame repaints every cell.
			out.WriteString(g.garbage(w, h))
			term.Invalidate()
		}

		frame, err := term.BeginFrame()
		if err != nil {
			return fmt.Errorf("frame %d: BeginFrame: %w", frameNo, err)
		}
		var drawErr error
		switch kind {
		case "full":
			// A tui app fills the whole frame, then draws its views.
			ch, st := g.cellRune(), g.style()
			if g.chance(0.5) {
				ch, st = ' ', NewStyle()
			}
			if drawErr = frame.Fill(ch, st); drawErr == nil {
				model.fill(full, ch, st)
				drawErr = rbDraw(g, frame, model, full)
			}
		case "empty":
		default:
			r := g.rect(w, h)
			sub := frame.SubFrame(r)
			if g.chance(0.4) {
				ch, st := g.cellRune(), g.style()
				drawErr = sub.FillStyled(0, 0, r.Dx(), r.Dy(), ch, st)
				model.fill(r, ch, st)
			}
			if drawErr == nil {
				drawErr = rbDraw(g, sub, model, r)
			}
		}
		if err := term.EndFrame(frame); err != nil {
			return fmt.Errorf("frame %d: EndFrame: %w", frameNo, err)
		}
		if drawErr != nil {
			return fmt.Errorf("frame %d (%s): %w", frameNo, kind, drawErr)
		}

		if _, err := screen.Write(out.Bytes()); err != nil {
			return fmt.Errorf("frame %d: screen write: %w", frameNo, err)
		}
		out.Reset()

		// The back buffer must agree with the model, or the model is wrong
		// and the screen check below proves nothing.
		if diff := rbModelDiff(model, term); diff != "" {
			return fmt.Errorf("frame %d (%s): back buffer and model disagree (test bug): %s",
				frameNo, kind, diff)
		}
		at := func(x, y int) Cell { return model.cells[y][x] }
		if diff := rbCompare(screen, w, h, at); diff != "" {
			return fmt.Errorf("frame %d (%s) on a %dx%d screen: the terminal shows a different %s",
				frameNo, kind, w, h, diff)
		}
	}
	return nil
}
