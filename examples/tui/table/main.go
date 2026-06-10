// Package main demonstrates the declarative Table view: column definitions,
// keyboard navigation, selection styling, and the OnSelect callback.
//
// Run with: go run ./examples/tui/table
package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// TableDemoApp demonstrates the declarative Table view.
type TableDemoApp struct {
	columns   []tui.TableColumn
	rows      [][]string
	selected  int
	activated int // 1-based row confirmed with Enter (0 = none yet)
	height    int
}

// View returns the declarative UI for this app.
func (app *TableDemoApp) View() tui.View {
	// Calculate table height based on terminal size, leaving room for the
	// header, footer, and status lines (including the conditional
	// "last activated" line)
	tableHeight := app.height - 9
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
				// Called when a row is confirmed with Enter (or clicked,
				// if mouse tracking is enabled)
				app.activated = row + 1
			}),
		tui.Spacer().MinHeight(1),
		tui.Text("Selected Row: %d", app.selected+1).Fg(tui.ColorGreen),
		tui.If(app.activated > 0, tui.Text("Last activated row: %d", app.activated).Fg(tui.ColorYellow)),
		tui.Text("Features: Uppercase headers, max column width, color inversion, header border").Dim(),
		tui.Text("Press Arrows to move, Enter to activate a row, q to quit.").Dim(),
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
		// Table component handles navigation internally

	case tui.ResizeEvent:
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
