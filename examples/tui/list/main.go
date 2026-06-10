package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// ListDemoApp demonstrates the List declarative view with filtering and keyboard navigation.
type ListDemoApp struct {
	items        []tui.ListItem
	selected     int
	chosen       []int // tracks which items have been chosen (Enter key)
	filterText   string
	scrollOffset int
	lastAction   string
}

// Init initializes the application.
func (app *ListDemoApp) Init() error {
	// Create sample items with icons
	app.items = []tui.ListItem{
		{Label: "Apple", Icon: "🍎", Value: "apple"},
		{Label: "Banana", Icon: "🍌", Value: "banana"},
		{Label: "Cherry", Icon: "🍒", Value: "cherry"},
		{Label: "Date", Icon: "🌴", Value: "date"},
		{Label: "Elderberry", Icon: "🫐", Value: "elderberry"},
		{Label: "Fig", Icon: "🌿", Value: "fig"},
		{Label: "Grape", Icon: "🍇", Value: "grape"},
		{Label: "Honeydew", Icon: "🍈", Value: "honeydew"},
		{Label: "Kiwi", Icon: "🥝", Value: "kiwi"},
		{Label: "Lemon", Icon: "🍋", Value: "lemon"},
		{Label: "Mango", Icon: "🥭", Value: "mango"},
		{Label: "Orange", Icon: "🍊", Value: "orange"},
		{Label: "Papaya", Icon: "🌺", Value: "papaya"},
		{Label: "Raspberry", Icon: "🍓", Value: "raspberry"},
		{Label: "Strawberry", Icon: "🍓", Value: "strawberry"},
		{Label: "Tangerine", Icon: "🍊", Value: "tangerine"},
		{Label: "Watermelon", Icon: "🍉", Value: "watermelon"},
	}
	app.lastAction = "Use arrows to navigate, type to filter, Enter to select"
	return nil
}

// HandleEvent processes events from the runtime.
func (app *ListDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// The focused FilterableList consumes printable keys for its filter
		// (typing 'q' filters, it doesn't quit), so quit on Escape/Ctrl+C.
		if e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		// FilterableList component handles navigation and filtering internally
	}

	return nil
}

// View returns the declarative view structure.
func (app *ListDemoApp) View() tui.View {
	return tui.Stack(
		// Header
		tui.HeaderBar("List Component Demo").Bg(tui.ColorBlue).Fg(tui.ColorWhite),

		// Divider
		tui.Divider(),

		// Instructions
		tui.Stack(
			tui.Text("Type to filter • Arrows to navigate • Enter to select • Esc to quit").Dim(),
			tui.Text("Filter: %s", app.filterText).Fg(tui.ColorCyan),
		).Padding(1),

		// Divider
		tui.Divider(),

		// Main content area with list
		tui.Group(

			// List with border
			tui.Bordered(
				tui.FilterableList(app.items, &app.selected).
					Filter(&app.filterText).
					FilterPlaceholder("Start typing to filter...").
					Height(15).
					Width(40).
					SelectedFg(tui.ColorWhite).
					SelectedBg(tui.ColorBlue).
					ChosenFg(tui.ColorGreen).
					MultiSelect(true).
					Chosen(&app.chosen).
					Markers("[ ]", "[x]").
					ScrollOffset(&app.scrollOffset).
					OnSelect(func(item tui.ListItem, idx int) {
						app.lastAction = fmt.Sprintf("Toggled: %s (index %d)", item.Label, idx)
					}),
			).Title("Fruits").Border(&tui.RoundedBorder),

			// Gap between list and info panel
			// tui.Spacer().MinWidth(1),

			// Info panel
			tui.Stack(
				tui.Text("Selection Info").Bold(),
				tui.Divider(),
				tui.Text("Index: %d", app.selected),
				tui.Text("Scroll: %d", app.scrollOffset),
				tui.Text(""),
				tui.Text("Last Action:").Dim(),
				tui.Text("%s", app.lastAction).Fg(tui.ColorGreen),
				tui.Spacer(),
			).Padding(1),

			// Right padding
			tui.Spacer().MinWidth(1),
		),

		// Bottom spacer
		tui.Spacer(),

		// Footer
		tui.StatusBar("Press Esc or Ctrl+C to quit"),
	).Padding(1)
}

func main() {
	if err := tui.Run(&ListDemoApp{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
