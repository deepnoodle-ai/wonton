package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/termtest"
)

// rendererBytesApp is a small app with the parts a real screen has: styled
// header and status rows with background colors, a border with a wide-glyph
// title, an input field, a list, and a progress bar.
type rendererBytesApp struct {
	step   int
	name   string
	sel    int
	status string
}

func (a *rendererBytesApp) View() View {
	items := []string{"alpha", "βeta 中文", "gamma 👍🏽", "delta 🇯🇵"}
	return Stack(
		Text("Inbox 📬 %d", a.step).Bold().FgRGB(250, 250, 250).BgRGB(40, 60, 120).FillBg(),
		Bordered(Stack(
			InputField(&a.name).ID("name"),
			SelectListStrings(items, &a.sel).ID("list"),
		)).Title("日本語 ❤️"),
		Divider(),
		Text("status: %s", a.status).Fg(ColorYellow).Bg(ColorBlue).FillBg(),
		Progress(a.step%11, 10),
	)
}

// TestRendererBytesTUIView runs a tui view through the frame path that
// tui.Run uses (Runtime.render: BeginFrame, Fill, size, render, EndFrame) on
// a test terminal. After each frame it feeds the renderer's output into a
// termtest.Screen and checks that every screen cell matches the back buffer:
// the cells the views set.
func TestRendererBytesTUIView(t *testing.T) {
	const w, h = 36, 14
	var out bytes.Buffer
	term := NewTestTerminal(w, h, &out)
	term.EnableAlternateScreen() // as Run does with the alternate screen on
	app := &rendererBytesApp{status: "ready"}
	r := NewRuntime(term, app, 30)
	screen := termtest.NewScreen(w, h)

	events := []Event{
		nil,
		KeyEvent{Rune: 'h'}, KeyEvent{Rune: 'é'}, KeyEvent{Rune: '中'},
		KeyEvent{Key: KeyBackspace},
		KeyEvent{Key: KeyTab},
		KeyEvent{Key: KeyArrowDown}, KeyEvent{Key: KeyArrowDown},
		KeyEvent{Key: KeyArrowDown}, KeyEvent{Key: KeyArrowUp},
		KeyEvent{Key: KeyTab},
		KeyEvent{Rune: '😀'}, KeyEvent{Rune: 'x'},
	}
	statuses := []string{"ready", "typing…", "日本", "saved ✓", ""}

	for i, ev := range events {
		if ev != nil {
			r.processEvent(ev)
		}
		app.step = i
		app.status = statuses[i%len(statuses)]
		r.render()

		if _, err := screen.Write(out.Bytes()); err != nil {
			t.Fatalf("frame %d: screen write: %v", i, err)
		}
		out.Reset()
		if diff := rbViewCompare(screen, term, w, h); diff != "" {
			t.Fatalf("frame %d (after %#v): the terminal shows a different %s", i, ev, diff)
		}
	}
	// The frames must have reached the screen at all.
	if !screen.Contains("Inbox") || !screen.Contains("gamma") {
		t.Fatalf("screen is missing the view:\n%s", screen.Text())
	}
}

func rbViewCompare(screen *termtest.Screen, term *Terminal, w, h int) string {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := term.GetCell(x, y)
			want := termtest.Cell{
				Char:         c.Char,
				Trailing:     c.Trailing,
				Width:        c.Width,
				Style:        rbViewStyle(c.Style),
				Continuation: c.Continuation,
			}
			got := screen.Cell(x, y)
			if got.Char == want.Char && got.Trailing == want.Trailing &&
				got.Width == want.Width && got.Continuation == want.Continuation &&
				got.Style == want.Style {
				continue
			}
			return fmt.Sprintf("cell (%d,%d):\n  want %s\n  got  %s\n  row %q",
				x, y, rbViewFormat(want), rbViewFormat(got), screen.Row(y))
		}
	}
	return ""
}

func rbViewFormat(c termtest.Cell) string {
	return fmt.Sprintf("glyph=%q width=%d cont=%v style=[%s]",
		string(c.Char)+c.Trailing, c.Width, c.Continuation, rbViewFormatStyle(c.Style))
}

func rbViewFormatStyle(s termtest.Style) string {
	parts := []string{"fg=" + rbViewFormatColor(s.Foreground), "bg=" + rbViewFormatColor(s.Background)}
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

func rbViewFormatColor(c termtest.Color) string {
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

// rbViewStyle converts a terminal style to the style termtest reports for
// text drawn with it. termtest does not track hyperlinks, so URL is dropped.
func rbViewStyle(s Style) termtest.Style {
	return termtest.Style{
		Foreground: rbViewColor(s.Foreground, s.FgRGB),
		Background: rbViewColor(s.Background, s.BgRGB),
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

func rbViewColor(c Color, rgb *RGB) termtest.Color {
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
