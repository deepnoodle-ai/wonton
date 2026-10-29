package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// MouseGridApp demonstrates a clickable grid with mouse support using Runtime.
// Click cells to toggle through different colors.
type MouseGridApp struct {
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
	// Clicks on the grid never need handling here: ColorGrid registers a
	// click region for each cell and cycles that cell's color itself.
	if e, ok := event.(tui.KeyEvent); ok {
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
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

	return tui.Stack(
		tui.Text("Mouse Grid Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Click cells to toggle colors! Press 'q' or Ctrl+C to exit.").Fg(tui.ColorWhite),
		tui.Text(""),
		tui.ColorGrid(app.gridW, app.gridH, app.gridState, colors).
			CellSize(6, 3).Gap(1),
		tui.Spacer(),
	).Align(tui.AlignCenter)
}

func main() {
	// Mouse tracking is enabled via WithMouseTracking option; the runtime routes
	// clicks to the regions ColorGrid registers for its cells
	if err := tui.Run(&MouseGridApp{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
