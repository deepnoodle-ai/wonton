package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// TableDemoApp demonstrates the declarative Table view.
type TableDemoApp struct {
	columns  []tui.TableColumn
	rows     [][]string
	selected int
	width    int
	height   int
}

// View returns the declarative UI for this app.
func (app *TableDemoApp) View() tui.View {
	// Calculate table height based on terminal size
	tableHeight := app.height - 8
	if tableHeight < 5 {
		tableHeight = 5
	}

	return tui.Stack(
		tui.Text(" Table Demo - Enhanced Features ").Bold().Bg(tui.ColorBlue).Fg(tui.ColorWhite),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
		tui.Table(app.columns, &app.selected).
			Rows(app.rows).
			Height(tableHeight).
			UppercaseHeaders(true).
			MaxColumnWidth(25).
			InvertSelectedColors(true).
			SelectedBg(tui.ColorBlue).
			SelectedFg(tui.ColorWhite).
			OnSelect(func(row int) {
				// Handle row click
			}),
		tui.Spacer().MinHeight(1),
		tui.Text("Selected Row: %d", app.selected+1).Fg(tui.ColorGreen),
		tui.Text("Features: Uppercase headers, max column width, color inversion, header border").Dim(),
		tui.Text("Press Arrows to move, q to quit.").Dim(),
		tui.Spacer(),
		tui.Text(" Press 'q' to quit ").Bg(tui.ColorBrightBlack).Fg(tui.ColorWhite),
	)
}

// HandleEvent processes events from the runtime.
func (app *TableDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

		// Handle keyboard navigation
		switch e.Key {
		case tui.KeyArrowUp:
			if app.selected > 0 {
				app.selected--
			}
		case tui.KeyArrowDown:
			if app.selected < len(app.rows)-1 {
				app.selected++
			}
		case tui.KeyPageUp:
			app.selected -= 10
			if app.selected < 0 {
				app.selected = 0
			}
		case tui.KeyPageDown:
			app.selected += 10
			if app.selected >= len(app.rows) {
				app.selected = len(app.rows) - 1
			}
		case tui.KeyHome:
			app.selected = 0
		case tui.KeyEnd:
			app.selected = len(app.rows) - 1
		}

	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

func main() {
	// Define columns
	columns := []tui.TableColumn{
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
	if err := tui.Run(app); err != nil {
		log.Fatal(err)
	}
}
