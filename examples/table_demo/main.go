package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// TableDemoApp demonstrates the declarative Table view.
type TableDemoApp struct {
	columns  []gooey.TableColumn
	rows     [][]string
	selected int
	width    int
	height   int
}

// View returns the declarative UI for this app.
func (app *TableDemoApp) View() gooey.View {
	// Calculate table height based on terminal size
	tableHeight := app.height - 8
	if tableHeight < 5 {
		tableHeight = 5
	}

	return gooey.VStack(
		gooey.Text(" Table Demo ").Bold().Bg(gooey.ColorBlue).Fg(gooey.ColorWhite),
		gooey.Divider(),
		gooey.Spacer().MinHeight(1),
		gooey.Table(app.columns, &app.selected).
			Rows(app.rows).
			Height(tableHeight).
			OnSelect(func(row int) {
				// Handle row click
			}),
		gooey.Spacer().MinHeight(1),
		gooey.Text("Selected Row: %d", app.selected+1).Fg(gooey.ColorGreen),
		gooey.Text("Press Arrows to move, q to quit.").Dim(),
		gooey.Spacer(),
		gooey.Text(" Press 'q' to quit ").Bg(gooey.ColorBrightBlack).Fg(gooey.ColorWhite),
	)
}

// HandleEvent processes events from the runtime.
func (app *TableDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

		// Handle keyboard navigation
		switch e.Key {
		case gooey.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
			}
		case gooey.KeyArrowDown:
			if app.selected < len(app.rows)-1 {
				app.selected++
			}
		case gooey.KeyPageUp:
			app.selected -= 10
			if app.selected < 0 {
				app.selected = 0
			}
		case gooey.KeyPageDown:
			app.selected += 10
			if app.selected >= len(app.rows) {
				app.selected = len(app.rows) - 1
			}
		case gooey.KeyHome:
			app.selected = 0
		case gooey.KeyEnd:
			app.selected = len(app.rows) - 1
		}

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

func main() {
	// Define columns
	columns := []gooey.TableColumn{
		{Title: "ID", Width: 5},
		{Title: "Name", Width: 20},
		{Title: "Role", Width: 15},
		{Title: "Status", Width: 10},
	}

	// Generate sample data
	rows := make([][]string, 50)
	for i := 0; i < 50; i++ {
		rows[i] = []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("User %d", i+1),
			"Developer",
			"Active",
		}
	}

	// Create the application
	app := &TableDemoApp{
		columns: columns,
		rows:    rows,
	}

	// Run the application
	if err := gooey.Run(app); err != nil {
		log.Fatal(err)
	}
}
