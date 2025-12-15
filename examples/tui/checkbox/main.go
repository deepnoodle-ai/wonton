package main

import (
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// CheckboxDemoApp demonstrates the CheckboxList declarative view.
type CheckboxDemoApp struct {
	items   []string
	checked []bool
	cursor  int
}

// Init initializes the application.
func (app *CheckboxDemoApp) Init() error {
	app.items = []string{
		"Apple",
		"Banana",
		"Cherry",
		"Date",
		"Elderberry",
	}
	app.checked = make([]bool, len(app.items))
	return nil
}

// HandleEvent processes events from the runtime.
func (app *CheckboxDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		// Handle keyboard navigation
		switch e.Key {
		case tui.KeyArrowUp:
			if app.cursor > 0 {
				app.cursor--
			}
		case tui.KeyArrowDown:
			if app.cursor < len(app.items)-1 {
				app.cursor++
			}
		}
		if e.Rune == ' ' {
			app.checked[app.cursor] = !app.checked[app.cursor]
		}
	}

	return nil
}

// View returns the declarative view structure.
func (app *CheckboxDemoApp) View() tui.View {
	// Get selected items for display
	var selected []string
	for i, item := range app.items {
		if app.checked[i] {
			selected = append(selected, item)
		}
	}
	selectedMsg := "Selected: " + strings.Join(selected, ", ")
	if len(selected) == 0 {
		selectedMsg = "Selected: (none)"
	}

	// Build checkbox items
	items := make([]tui.ListItem, len(app.items))
	for i, item := range app.items {
		items[i] = tui.ListItem{Label: item, Value: item}
	}

	return tui.VStack(
		// Header
		tui.HeaderBar("Checkbox Demo").Bg(tui.ColorBlue).Fg(tui.ColorWhite),

		// Divider
		tui.Divider(),

		// Spacing
		tui.Spacer().MinHeight(1),

		// Checkbox list using new declarative view
		tui.CheckboxList(items, app.checked, &app.cursor).
			Fg(tui.ColorWhite).
			CursorFg(tui.ColorGreen),

		// Spacing
		tui.Spacer().MinHeight(1),

		// Selected items display
		tui.Text("%s", selectedMsg).Fg(tui.ColorGreen),

		// Spacing
		tui.Spacer().MinHeight(1),

		// Help text
		tui.Text("Press Space to toggle, Arrows to move, q to quit.").Dim(),

		// Flexible spacer to push footer to bottom
		tui.Spacer(),

		// Footer
		tui.StatusBar("Press 'q' to quit"),
	).Padding(2)
}

func main() {
	if err := tui.Run(&CheckboxDemoApp{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
