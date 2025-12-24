package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// MouseDemoApp demonstrates mouse support using TUI components.
// It shows clickable buttons and scrollable content using the declarative component system.
type MouseDemoApp struct {
	// State
	clickCount   int
	scrollOffset int
	lastAction   string
}

// HandleEvent processes events
func (app *MouseDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// Handle keyboard input
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyCtrlC || e.Key == tui.KeyEscape {
			return []tui.Cmd{tui.Quit()}
		}
		return nil
	}

	return nil
}

// View returns the declarative view structure using TUI components
func (app *MouseDemoApp) View() tui.View {
	// Action buttons using Clickable components
	buttonRow := tui.Group(
		tui.Clickable("[ Increment ]", func() {
			app.clickCount++
			app.lastAction = fmt.Sprintf("Incremented! Count: %d", app.clickCount)
		}).Fg(tui.ColorBlue),

		tui.Spacer().MinWidth(2),

		tui.Clickable("[ Reset ]", func() {
			app.clickCount = 0
			app.lastAction = "Counter reset to 0"
		}).Fg(tui.ColorMagenta),

		tui.Spacer().MinWidth(2),

		tui.Clickable("[ Info ]", func() {
			app.lastAction = "Info: This demo showcases mouse interactions with TUI components!"
		}).Fg(tui.ColorGreen),
	)

	// Scrollable content area using Scroll component
	scrollContent := []string{
		"Use your mouse wheel or arrow keys to scroll through this area.",
		"",
		"Line 1: The quick brown fox jumps over the lazy dog",
		"Line 2: Lorem ipsum dolor sit amet, consectetur adipiscing elit",
		"Line 3: The five boxing wizards jump quickly",
		"Line 4: Pack my box with five dozen liquor jugs",
		"Line 5: How vexingly quick daft zebras jump!",
		"Line 6: The jay, pig, fox, zebra and my wolves quack!",
		"Line 7: Sphinx of black quartz, judge my vow",
		"Line 8: Two driven jocks help fax my big quiz",
		"Line 9: Mr Jock, TV quiz PhD, bags few lynx",
		"Line 10: Waltz, bad nymph, for quick jigs vex",
		"Line 11: How razorback-jumping frogs can level six piqued gymnasts",
		"Line 12: Crazy Fredrick bought many very exquisite opal jewels",
	}

	var scrollLines []tui.View
	for _, line := range scrollContent {
		scrollLines = append(scrollLines, tui.Text("%s", line))
	}

	scrollArea := tui.Bordered(
		tui.Scroll(
			tui.Stack(scrollLines...).Padding(1),
			&app.scrollOffset,
		),
	).
		BorderFg(tui.ColorCyan).
		Title("Scrollable Content Area")

	// Footer with status
	sep := strings.Repeat("━", 80)
	footer := tui.Stack(
		tui.Text("%s", sep).Dim(),
		tui.Group(
			tui.Text("Counter:").Fg(tui.ColorCyan),
			tui.Text(" %d", app.clickCount),
			tui.Text("  Status:").Fg(tui.ColorCyan),
			tui.Text(" %s", app.lastAction),
		),
	)

	return tui.Stack(
		tui.Spacer().MinHeight(1),
		tui.Text("Wonton Mouse Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Click buttons and scroll content with mouse or keyboard").Dim(),
		tui.Text("Press 'q', Esc, or Ctrl+C to exit").Dim(),
		tui.Spacer().MinHeight(1),

		// Action buttons
		buttonRow,

		tui.Spacer().MinHeight(1),

		// Scroll area
		scrollArea,

		tui.Spacer(),

		// Status footer
		footer,
	).Padding(1)
}

func main() {
	// Mouse tracking is enabled via WithMouseTracking option
	app := &MouseDemoApp{
		lastAction: "Ready - click buttons or scroll the content area",
	}

	if err := tui.Run(app, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
