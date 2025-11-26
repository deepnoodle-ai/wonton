package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// TableDemoApp demonstrates the Table widget using the Runtime architecture.
// It shows how to display tabular data with scrolling and selection.
type TableDemoApp struct {
	table  *gooey.Table
	width  int
	height int
}

// Init initializes the application by creating the table widget.
func (app *TableDemoApp) Init() error {
	// Table will be created on first ResizeEvent with actual terminal dimensions
	return nil
}

// HandleEvent processes events from the runtime.
func (app *TableDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		app.table.HandleKey(e)

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

// Render draws the current application state.
func (app *TableDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Draw header
	headerStyle := gooey.NewStyle().WithBold().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)
	headerText := " Table Demo "
	for i := 0; i < width; i++ {
		if i >= (width-len(headerText))/2 && i < (width-len(headerText))/2+len(headerText) {
			frame.SetCell(i, 0, rune(headerText[i-(width-len(headerText))/2]), headerStyle)
		} else {
			frame.SetCell(i, 0, ' ', headerStyle)
		}
	}

	// Draw separator
	separatorStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	for i := 0; i < width; i++ {
		frame.SetCell(i, 1, '─', separatorStyle)
	}

	// Draw table (if initialized)
	if app.table != nil {
		app.table.Draw(frame)
	}

	// Draw info below table (if table exists)
	if app.table != nil {
		infoY := app.table.Y + app.table.Height + 1
		if infoY < height-2 {
			msg := fmt.Sprintf("Selected Row: %d", app.table.SelectedRow)
			msgStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
			frame.PrintStyled(2, infoY, msg, msgStyle)

			helpStyle := gooey.NewStyle().WithDim()
			frame.PrintStyled(2, infoY+1, "Press Arrows to move, q to quit.", helpStyle)
		}
	}

	// Draw footer
	if height > 0 {
		footerStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite)
		footerText := " Press 'q' to quit "
		footerX := (width - len(footerText)) / 2
		if footerX < 0 {
			footerX = 0
		}
		for i := 0; i < width; i++ {
			if i >= footerX && i < footerX+len(footerText) {
				frame.SetCell(i, height-1, rune(footerText[i-footerX]), footerStyle)
			} else {
				frame.SetCell(i, height-1, ' ', footerStyle)
			}
		}
	}
}

func main() {
	// Create the application
	app := &TableDemoApp{}

	// Run the application (30 FPS is default)
	if err := gooey.Run(app); err != nil {
		log.Fatal(err)
	}
}
