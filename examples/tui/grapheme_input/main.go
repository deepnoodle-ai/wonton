// Package main demonstrates grapheme-aware text input.
//
// The input is pre-populated with Unicode clusters that are easy to break in
// other terminal libraries: ZWJ family emoji, regional-indicator flags, VS16
// emoji, skin-toned hands, and CJK ideographs. The live counters show how
// many bytes, runes, grapheme clusters, and display cells the current value
// takes — the numbers should stay in sync as you edit.
//
// Things to poke at:
//
//   - Arrow left/right should move one user-perceived character per press.
//     A family emoji (👨‍👩‍👧‍👦) is seven code points but one keystroke.
//   - Backspace on the Japan flag (🇯🇵) should remove both regional-indicator
//     code points in one press, not leave a dangling half-flag.
//   - Typing more emoji should never split an existing cluster.
//
// If any of those misbehave, it's a regression in runewidth.Graphemes /
// WidthIndex or the tui/text_input grapheme wiring.
package main

import (
	"log"
	"strings"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/runewidth"
	"github.com/deepnoodle-ai/wonton/tui"
)

type App struct {
	value string
}

func (app *App) View() tui.View {
	return tui.Stack(
		tui.Text("Grapheme-aware Input Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Edit the text below. Arrow keys move one cluster at a time; "+
			"backspace removes whole clusters.").Dim(),
		tui.Spacer().MinHeight(1),

		tui.InputField(&app.value).
			Label("Text: ").
			Width(50).
			Bordered(),

		tui.Spacer().MinHeight(1),
		tui.Text("bytes=%d  runes=%d  graphemes=%d  width=%d",
			len(app.value),
			utf8.RuneCountInString(app.value),
			graphemeCount(app.value),
			runewidth.StringWidth(app.value),
		).Fg(tui.ColorGreen),

		tui.Spacer().MinHeight(1),
		tui.Text("Cluster breakdown:").Bold(),
		tui.Text("%s", clusterBreakdown(app.value)).Fg(tui.ColorBrightBlack),

		tui.Spacer(),
		tui.Text("Esc or Ctrl+C to quit").Dim(),
	).Padding(1)
}

func (app *App) Init() error {
	// Pre-populate with the clusters most likely to break in other libraries.
	app.value = "Hi \U0001F44B\U0001F3FD " + // waving hand, medium skin tone
		"\U0001F1EF\U0001F1F5 " + // Japan flag
		"\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466 " + // family of four
		"\u2764\uFE0F " + // red heart (VS16)
		"中文"
	return nil
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	if e, ok := event.(tui.KeyEvent); ok {
		if e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func graphemeCount(s string) int {
	n := 0
	for range runewidth.Graphemes(s) {
		n++
	}
	return n
}

func clusterBreakdown(s string) string {
	var b strings.Builder
	i := 0
	for cluster := range runewidth.Graphemes(s) {
		if i > 0 {
			b.WriteString(" · ")
		}
		b.WriteString(cluster)
		i++
		if i >= 16 {
			b.WriteString(" …")
			break
		}
	}
	return b.String()
}

func main() {
	if err := tui.Run(&App{}); err != nil {
		log.Fatal(err)
	}
}
