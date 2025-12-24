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
	case tui.MouseEvent:
		// Handle mouse clicks on grid
		if e.Type == tui.MousePress && e.Button == tui.MouseButtonLeft {
			app.handleGridClick(e.X, e.Y)
		}
		return nil

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

// handleGridClick toggles the color of a grid cell when clicked
func (app *MouseGridApp) handleGridClick(mouseX, mouseY int) {
	// Grid starts at row 3 (after title, description, and spacer)
	// Cell dimensions: 6 wide, 3 tall, with 1 gap between cells
	cellWidth := 6
	cellHeight := 3
	gap := 1

	// Calculate grid starting position (centered horizontally)
	gridWidth := app.gridW*cellWidth + (app.gridW-1)*gap
	startX := (app.width - gridWidth) / 2
	startY := 3

	// Check if click is within grid bounds
	if mouseX < startX || mouseY < startY {
		return
	}

	// Calculate which cell was clicked
	relX := mouseX - startX
	relY := mouseY - startY

	// Account for gaps
	col := relX / (cellWidth + gap)
	row := relY / (cellHeight + gap)

	// Verify click is actually on a cell (not in the gap)
	cellStartX := col * (cellWidth + gap)
	cellStartY := row * (cellHeight + gap)

	if relX >= cellStartX && relX < cellStartX+cellWidth &&
		relY >= cellStartY && relY < cellStartY+cellHeight &&
		row < app.gridH && col < app.gridW {
		// Toggle to next color (cycle through 0-4)
		app.gridState[row][col] = (app.gridState[row][col] + 1) % 5
	}
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
