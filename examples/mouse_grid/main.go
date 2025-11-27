package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// MouseGridApp demonstrates a clickable grid with mouse support using Runtime.
// Click cells to toggle through different colors.
type MouseGridApp struct {
	width  int
	height int

	// Grid configuration
	gridW     int
	gridH     int
	gridState [][]int
}

// Init initializes the application
func (app *MouseGridApp) Init() error {
	// Grid configuration
	app.gridW, app.gridH = 5, 5

	// Initialize state
	app.gridState = make([][]int, app.gridH)
	for i := range app.gridState {
		app.gridState[i] = make([]int, app.gridW)
	}

	return nil
}

// HandleEvent processes events
func (app *MouseGridApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle keyboard input
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		return nil

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
		return nil
	}

	return nil
}

// View returns the declarative view structure
func (app *MouseGridApp) View() gooey.View {
	// Color palette for the grid
	colors := []gooey.Color{
		gooey.ColorBrightBlack, // Off
		gooey.ColorRed,
		gooey.ColorGreen,
		gooey.ColorBlue,
		gooey.ColorYellow,
	}

	return gooey.VStack(
		gooey.Text("Mouse Grid Demo").Bold().Fg(gooey.ColorCyan),
		gooey.Text("Click cells to toggle colors! Press 'q' or Ctrl+C to exit.").Fg(gooey.ColorWhite),
		gooey.Spacer().MinHeight(1),
		gooey.ColorGrid(app.gridW, app.gridH, app.gridState, colors).
			CellSize(6, 3).Gap(1),
		gooey.Spacer(),
	).Align(gooey.AlignCenter)
}

func main() {
	// Mouse tracking is enabled via WithMouseTracking option, and mouse events are
	// automatically delivered to HandleEvent as gooey.MouseEvent
	if err := gooey.Run(&MouseGridApp{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
