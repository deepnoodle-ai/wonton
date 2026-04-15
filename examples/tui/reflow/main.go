package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/runewidth"
	"github.com/deepnoodle-ai/wonton/tui"
)

// ReflowApp animates a container width and reflows a paragraph packed with
// Unicode clusters that expose grapheme-segmentation bugs in other terminal
// libraries. As the box shrinks and grows, cluster boundaries should always
// hold: no emoji should split across a wrap, no column should jitter, and
// the border should stay flush.
//
// The demo runs with runewidth.SetTerminalCompatMode(true) so the layout
// math matches what real terminals draw — see main() and the runewidth
// package docs for what the mode adjusts.
type ReflowApp struct {
	width     int
	direction int
	frame     uint64
}

// showcase is a single paragraph that packs together the grapheme-cluster
// shapes that other libraries break: VS16 emoji promotion, regional-
// indicator flag pairs, ZWJ sequences (rainbow flag, family of four),
// skin-toned and multi-person ZWJ, Indic combining marks, and CJK. With
// terminal-compat mode enabled the reported widths match what real
// terminals will draw, so the box never has to lie about cell math.
const showcase = "Wonton reflows tricky Unicode cleanly. " +
	"Keycaps #\uFE0F\u20E3 *\uFE0F\u20E3 0\uFE0F\u20E3 9\uFE0F\u20E3 stay one cluster each — " +
	"width follows the terminal, but base + VS16 + U+20E3 never splits. " +
	"VS16 emoji \u2764\uFE0F \u26A0\uFE0F \u2600\uFE0F promote to width 2 " +
	"while the text-default glyphs \u2764 \u26A0 \u2600 stay width 1. " +
	"Regional-indicator flags \U0001F1FA\U0001F1F8 \U0001F1EF\U0001F1F5 \U0001F1EA\U0001F1FA " +
	"hold together as one cluster of width 2 each. " +
	"ZWJ sequences like \U0001F3F3\uFE0F\u200D\U0001F308 (rainbow flag) and " +
	"\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466 (family of four) collapse to one grapheme. " +
	"Skin-toned \U0001F44B\U0001F3FD and multi-person \U0001F9D1\U0001F3FD\u200D\U0001F91D\u200D\U0001F9D1\U0001F3FE never split. " +
	"Indic clusters \u0BA4\u0BAE\u0BBF\u0BB4\u0BCD, \u09AC\u09BE\u0982\u09B2\u09BE, and \u0939\u093F\u0928\u094D\u0926\u0940 " +
	"are sized by their base consonant, not the sum of every combining mark. " +
	"And the usual CJK 中文 日本語 한국어 still lines up exactly two columns per ideograph."

// caption describes specific clusters and their wonton widths, computed live
// via runewidth.StringWidth so the numbers stay honest if the tables (or the
// active compat mode) change.
func caption() string {
	// Print the cluster bare (not %q) so ZWJ joiners stay zero-width and the
	// terminal assembles family / rainbow-flag glyphs correctly. Brackets
	// delimit the cluster so zero-width or single-cell shapes are still
	// visible in the legend.
	row := func(label, s string) string {
		return fmt.Sprintf("  %-22s [%s]  w=%d", label, s, runewidth.StringWidth(s))
	}
	mode := "strict"
	if runewidth.TerminalCompatMode() {
		mode = "terminal-compat"
	}
	return fmt.Sprintf("Widths reported by runewidth.StringWidth (%s mode):\n", mode) +
		row("hash keycap", "#\uFE0F\u20E3") + "\n" +
		row("heart (VS16)", "\u2764\uFE0F") + "\n" +
		row("JP flag", "\U0001F1EF\U0001F1F5") + "\n" +
		row("rainbow flag", "\U0001F3F3\uFE0F\u200D\U0001F308") + "\n" +
		row("family of four", "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466") + "\n" +
		row("tamil", "\u0BA4\u0BAE\u0BBF\u0BB4\u0BCD") + "\n" +
		row("two-em dash", "a\u2E3Ab")
}

// HandleEvent processes events from the runtime.
func (app *ReflowApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		return []tui.Cmd{tui.Quit()}

	case tui.TickEvent:
		app.frame = e.Frame
		// Advance the width once every 3 ticks (~0.33 cells/tick) so the
		// reflow is slow enough to eyeball individual cluster boundaries.
		if app.frame%3 == 0 {
			app.width += app.direction
			if app.width > 70 {
				app.direction = -1
			} else if app.width < 24 {
				app.direction = 1
			}
		}

		if app.frame >= 900 {
			return []tui.Cmd{tui.Quit()}
		}
	}

	return nil
}

// View returns the declarative view tree.
func (app *ReflowApp) View() tui.View {
	debugInfo := fmt.Sprintf("Width: %d | Frame: %d/900 | Press any key to exit", app.width, app.frame)

	borderedBox := tui.Bordered(
		tui.MaxWidth(app.width-4, tui.Text("%s", showcase).Wrap().Flex(1)),
	).
		Border(&tui.RoundedBorder).
		BorderFg(tui.ColorCyan)

	legend := tui.Text("%s", caption()).Fg(tui.ColorBrightBlack)

	footer := tui.Text("Cluster boundaries hold; wraps land between graphemes, not inside them.").
		Fg(tui.ColorGreen).
		Bold()

	return tui.Stack(
		tui.Text("%s", debugInfo),
		tui.Stack(
			borderedBox,
			legend,
			footer,
		).Gap(1),
	).Gap(1)
}

func main() {
	// Match what the host terminal will actually draw. Most terminals don't
	// composite emoji keycap sequences (#⃣ etc.) into a width-2 glyph and
	// don't honor the typographic widths for U+2E3A / U+2E3B. With compat
	// mode on, runewidth.StringWidth reports widths that match the screen,
	// so wrap math and the bordered box stay flush. Run examples/termprobe
	// to see which clusters your terminal differs on.
	runewidth.SetTerminalCompatMode(true)

	app := &ReflowApp{
		width:     48,
		direction: 1,
	}
	if err := tui.Run(app, tui.WithFPS(20)); err != nil {
		log.Fatal(err)
	}
}
