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
		// Handle quit
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

		// Handle keyboard navigation
		switch e.Key {
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
				app.lastAction = "Moved up"
			}
		case tui.KeyArrowDown:
			app.selected++
			app.lastAction = "Moved down"
		case tui.KeyHome:
			app.selected = 0
			app.lastAction = "Jumped to top"
		case tui.KeyEnd:
			// This will be clamped by the list view
			app.selected = 9999
			app.lastAction = "Jumped to bottom"
		case tui.KeyPageUp:
			app.selected -= 5
			if app.selected < 0 {
				app.selected = 0
			}
			app.lastAction = "Page up"
		case tui.KeyPageDown:
			app.selected += 5
			app.lastAction = "Page down"
		case tui.KeyBackspace:
			if len(app.filterText) > 0 {
				app.filterText = app.filterText[:len(app.filterText)-1]
				app.lastAction = fmt.Sprintf("Filter: '%s'", app.filterText)
			}
		default:
			// Handle text input for filtering
			if e.Rune >= 32 && e.Rune < 127 {
				app.filterText += string(e.Rune)
				app.selected = 0 // Reset selection when filtering
				app.lastAction = fmt.Sprintf("Filter: '%s'", app.filterText)
			}
		}
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
			tui.Text("Type to filter • Arrows to navigate • Enter to select • q to quit").Dim(),
			tui.Text("Filter: %s", app.filterText).Fg(tui.ColorCyan),
		).Padding(1),

		// Divider
		tui.Divider(),

		// Main content area with list
		tui.Group(
			// Left padding
			tui.Spacer().MinWidth(2),

			// List with border
			tui.Bordered(
				tui.FilterableList(app.items, &app.selected).
					Filter(&app.filterText).
					FilterPlaceholder("Start typing to filter...").
					Height(15).
					Width(40).
					SelectedFg(tui.ColorWhite).
					SelectedBg(tui.ColorBlue).
					ScrollOffset(&app.scrollOffset).
					OnSelect(func(item tui.ListItem, idx int) {
						app.lastAction = fmt.Sprintf("Selected: %s (index %d)", item.Label, idx)
					}),
			).Title("Fruits").Border(&tui.RoundedBorder),

			// Right padding
			tui.Spacer().MinWidth(2),

			// Info panel
			tui.Stack(
				tui.Text("Selection Info").Bold(),
				tui.Divider(),
				tui.Spacer().MinHeight(1),
				tui.Text("Index: %d", app.selected),
				tui.Text("Scroll: %d", app.scrollOffset),
				tui.Spacer().MinHeight(1),
				tui.Text("Last Action:").Dim(),
				tui.Text("%s", app.lastAction).Fg(tui.ColorGreen),
				tui.Spacer(),
			).Padding(1),

			// Right padding
			tui.Spacer().MinWidth(2),
		),

		// Bottom spacer
		tui.Spacer(),

		// Footer
		tui.StatusBar("Press 'q' to quit"),
	).Padding(1)
}

func main() {
	if err := tui.Run(&ListDemoApp{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
