package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
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
func (app *MouseGridApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// Handle keyboard input
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		return nil

	case tui.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
		return nil
	}

	return nil
}

// View returns the declarative view structure
func (app *MouseGridApp) View() tui.View {
	// Color palette for the grid
	colors := []tui.Color{
		tui.ColorBrightBlack, // Off
		tui.ColorRed,
		tui.ColorGreen,
		tui.ColorBlue,
		tui.ColorYellow,
	}

	return tui.VStack(
		tui.Text("Mouse Grid Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Click cells to toggle colors! Press 'q' or Ctrl+C to exit.").Fg(tui.ColorWhite),
		tui.Spacer().MinHeight(1),
		tui.ColorGrid(app.gridW, app.gridH, app.gridState, colors).
			CellSize(6, 3).Gap(1),
		tui.Spacer(),
	).Align(tui.AlignCenter)
}

func main() {
	// Mouse tracking is enabled via WithMouseTracking option, and mouse events are
	// automatically delivered to HandleEvent as tui.MouseEvent
	if err := tui.Run(&MouseGridApp{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
