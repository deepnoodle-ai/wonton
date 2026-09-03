package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// MouseDemoApp demonstrates mouse support using TUI components.
// It shows clickable buttons and a scroll area driven by the wheel.
type MouseDemoApp struct {
	clickCount    int
	scrollOffset  int
	lastAction    string
	pointerX      int
	pointerY      int
	width, height int
}

// scrollContent is the body of the scroll area.
var scrollContent = []string{
	"Use your mouse wheel or the arrow keys to scroll through this area.",
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
	"Line 13: We promptly judged antique ivory buckles for the next prize",
	"Line 14: Jinxed wizards pluck ivy from the big quilt",
	"Line 15: Blowzy red vixens fight for a quick jump",
	"Line 16: Quick zephyrs blow, vexing daft Jim",
	"Line 17: Jackdaws love my big sphinx of quartz",
	"Line 18: The public was amazed to view the quickness of the fox",
	"Line 19: A quivering Texas zombie fought republic linked jewels",
	"Line 20: Sixty zippers were quickly picked from the woven jute bag",
	"",
	"You have reached the bottom.",
}

// HandleEvent processes events.
//
// Clicks on the buttons never reach here: the runtime dispatches those to the
// Clickable callbacks in View. What is left for the app is the wheel, which no
// view consumes on its own, and the keyboard.
func (app *MouseDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.ResizeEvent:
		app.width, app.height = e.Width, e.Height

	case tui.MouseEvent:
		app.pointerX, app.pointerY = e.X, e.Y
		switch e.Button {
		case tui.MouseButtonWheelUp:
			app.scrollBy(-1)
			app.lastAction = "Wheel up"
		case tui.MouseButtonWheelDown:
			app.scrollBy(1)
			app.lastAction = "Wheel down"
		}

	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyCtrlC || e.Key == tui.KeyEscape {
			return []tui.Cmd{tui.Quit()}
		}
		page := max(app.viewHeight()-1, 1)
		switch e.Key {
		case tui.KeyArrowUp:
			app.scrollBy(-1)
		case tui.KeyArrowDown:
			app.scrollBy(1)
		case tui.KeyPageUp:
			app.scrollBy(-page)
		case tui.KeyPageDown:
			app.scrollBy(page)
		case tui.KeyHome:
			app.scrollOffset = 0
		case tui.KeyEnd:
			app.scrollOffset = app.maxScroll()
		}
	}

	return nil
}

// viewHeight is what the chrome leaves for the scrolling content: three header
// lines, a blank, the button row, the footer, the stack's own top and bottom
// padding, and the two border rows.
const chromeHeight = 10

func (app *MouseDemoApp) viewHeight() int { return max(app.height-chromeHeight, 1) }

// maxScroll is how far the content can move before its last line reaches the
// bottom. It is an upper bound — ScrollView clamps scrollOffset exactly when it
// renders, and writes the clamped value back.
func (app *MouseDemoApp) maxScroll() int {
	if app.width == 0 {
		return 0
	}
	_, contentHeight := tui.Measure(app.scrollBody(), app.width-2, 0)
	return max(contentHeight-app.viewHeight(), 0)
}

func (app *MouseDemoApp) scrollBy(lines int) {
	app.scrollOffset = min(max(app.scrollOffset+lines, 0), app.maxScroll())
}

func (app *MouseDemoApp) scrollBody() tui.View {
	lines := make([]tui.View, 0, len(scrollContent))
	for _, line := range scrollContent {
		lines = append(lines, tui.Text("%s", line))
	}
	return tui.Stack(lines...).Padding(1)
}

// View returns the declarative view structure using TUI components.
func (app *MouseDemoApp) View() tui.View {
	// Action buttons using Clickable components. Spacer is flexible by default,
	// so a fixed gap needs Flex(0) — otherwise the buttons spread to the edges.
	gap := func() tui.View { return tui.Spacer().MinWidth(2).Flex(0) }
	buttonRow := tui.Group(
		tui.Clickable("[ Increment ]", func() {
			app.clickCount++
			app.lastAction = fmt.Sprintf("Incremented! Count: %d", app.clickCount)
		}).Fg(tui.ColorBlue),

		gap(),

		tui.Clickable("[ Reset ]", func() {
			app.clickCount = 0
			app.lastAction = "Counter reset to 0"
		}).Fg(tui.ColorMagenta),

		gap(),

		tui.Clickable("[ Info ]", func() {
			app.lastAction = "Info: This demo showcases mouse interactions with TUI components!"
		}).Fg(tui.ColorGreen),
	)

	scrollArea := tui.Bordered(tui.Scroll(app.scrollBody(), &app.scrollOffset)).
		BorderFg(tui.ColorCyan).
		Title("Scrollable Content Area")

	footer := tui.Group(
		tui.Text("Counter:").Fg(tui.ColorCyan),
		tui.Text(" %d", app.clickCount),
		tui.Text("  Pointer:").Fg(tui.ColorCyan),
		tui.Text(" %d,%d", app.pointerX, app.pointerY),
		tui.Text("  Status:").Fg(tui.ColorCyan),
		tui.Text(" %s", app.lastAction),
	)

	return tui.Stack(
		tui.Text("Wonton Mouse Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Click the buttons; scroll the box with the wheel or ↑↓ PgUp/PgDn Home/End").Dim(),
		tui.Text("Press 'q', Esc, or Ctrl+C to exit").Dim(),
		tui.Spacer().MinHeight(1).Flex(0),
		buttonRow,
		scrollArea,
		footer,
	).Padding(1)
}

func main() {
	// Mouse tracking is enabled via WithMouseTracking option. Hover reporting
	// is what fills in the pointer readout; WithMouseButtons would be enough
	// for the clicks and the wheel alone.
	app := &MouseDemoApp{
		lastAction: "Ready - click buttons or scroll the content area",
	}

	if err := tui.Run(app, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
