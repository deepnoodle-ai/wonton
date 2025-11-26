package main

import (
	"fmt"
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// TableDemoApp demonstrates the Table widget using the declarative View system.
// It shows how to display tabular data with scrolling and selection.
type TableDemoApp struct {
	table  *gooey.Table
	width  int
	height int
}

// View returns the declarative UI for this app.
func (app *TableDemoApp) View() gooey.View {
	var tableView gooey.View
	var infoView gooey.View

	// Create table view if table is initialized
	if app.table != nil {
		tableView = gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			app.table.Draw(frame)
		})

		infoView = gooey.VStack(
			gooey.Text("Selected Row: %d", app.table.SelectedRow).Fg(gooey.ColorGreen),
			gooey.Text("Press Arrows to move, q to quit.").Dim(),
		)
	} else {
		tableView = gooey.Spacer()
		infoView = gooey.Text("Initializing...").Dim()
	}

	return gooey.VStack(
		gooey.Text(" Table Demo ").Bold().Bg(gooey.ColorBlue).Fg(gooey.ColorWhite),
		gooey.Text(repeatRune('─', app.width)).Fg(gooey.ColorBrightBlack),
		gooey.Spacer().MinHeight(1),
		tableView,
		gooey.Spacer().MinHeight(1),
		infoView,
		gooey.Spacer(),
		gooey.Text(" Press 'q' to quit ").Bg(gooey.ColorBrightBlack).Fg(gooey.ColorWhite),
	)
}

// repeatRune creates a string by repeating a rune n times.
func repeatRune(r rune, n int) string {
	if n < 0 {
		n = 0
	}
	runes := make([]rune, n)
	for i := range runes {
		runes[i] = r
	}
	return string(runes)
}

// HandleEvent processes events from the runtime.
func (app *TableDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		if app.table != nil {
			app.table.HandleKey(e)
		}

	case gooey.ResizeEvent:
		// Update dimensions
		app.width = e.Width
		app.height = e.Height

		// Create table on first resize if needed
		if app.table == nil {
			tableWidth := e.Width - 4
			tableHeight := e.Height - 6
			if tableWidth < 10 {
				tableWidth = 10
			}
			if tableHeight < 5 {
				tableHeight = 5
			}

			app.table = gooey.NewTable(2, 3, tableWidth, tableHeight)
			app.table.SetColumns([]gooey.Column{
				{Title: "ID", Width: 5},
				{Title: "Name", Width: 20},
				{Title: "Role", Width: 15},
				{Title: "Status", Width: 10},
			})

			// Generate dummy data
			rows := make([][]string, 50)
			for i := 0; i < 50; i++ {
				rows[i] = []string{
					fmt.Sprintf("%d", i+1),
					fmt.Sprintf("User %d", i+1),
					"Developer",
					"Active",
				}
			}
			app.table.SetRows(rows)
		} else {
			// Update table dimensions on subsequent resizes
			tableWidth := e.Width - 4
			tableHeight := e.Height - 6
			if tableWidth < 10 {
				tableWidth = 10
			}
			if tableHeight < 5 {
				tableHeight = 5
			}
			app.table.Width = tableWidth
			app.table.Height = tableHeight
		}
	}

	return nil
}

func main() {
	// Create the application
	app := &TableDemoApp{}

	// Run the application (30 FPS is default)
	if err := gooey.Run(app); err != nil {
		log.Fatal(err)
	}
}
