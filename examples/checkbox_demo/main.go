package main

import (
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey"
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
func (app *CheckboxDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		// Handle keyboard navigation
		switch e.Key {
		case gooey.KeyArrowUp:
			if app.cursor > 0 {
				app.cursor--
			}
		case gooey.KeyArrowDown:
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
func (app *CheckboxDemoApp) View() gooey.View {
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
	items := make([]gooey.ListItem, len(app.items))
	for i, item := range app.items {
		items[i] = gooey.ListItem{Label: item, Value: item}
	}

	return gooey.VStack(
		// Header
		gooey.HeaderBar("Checkbox Demo").Bg(gooey.ColorBlue).Fg(gooey.ColorWhite),

		// Divider
		gooey.Divider(),

		// Spacing
		gooey.Spacer().MinHeight(1),

		// Checkbox list using new declarative view
		gooey.CheckboxList(items, app.checked, &app.cursor).
			Fg(gooey.ColorWhite).
			CursorFg(gooey.ColorGreen),

		// Spacing
		gooey.Spacer().MinHeight(1),

		// Selected items display
		gooey.Text("%s", selectedMsg).Fg(gooey.ColorGreen),

		// Spacing
		gooey.Spacer().MinHeight(1),

		// Help text
		gooey.Text("Press Space to toggle, Arrows to move, q to quit.").Dim(),

		// Flexible spacer to push footer to bottom
		gooey.Spacer(),

		// Footer
		gooey.StatusBar("Press 'q' to quit"),
	).Padding(2)
}

func main() {
	if err := gooey.Run(&CheckboxDemoApp{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
