package main

import (
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// MouseGridApp demonstrates a clickable grid with mouse support using Runtime.
// Click cells to toggle through different colors.
type MouseGridApp struct {
	mouse  *gooey.MouseHandler
	width  int
	height int

	// Grid configuration
	gridW     int
	gridH     int
	cellW     int
	cellH     int
	startX    int
	startY    int
	colors    []gooey.Style
	gridState [][]int
}

// Init initializes the application
func (app *MouseGridApp) Init() error {
	app.mouse = gooey.NewMouseHandler()

	// Grid configuration
	app.gridW, app.gridH = 5, 5
	app.cellW, app.cellH = 6, 3
	app.startX, app.startY = 4, 4

	// Initialize state
	app.gridState = make([][]int, app.gridH)
	for i := range app.gridState {
		app.gridState[i] = make([]int, app.gridW)
	}

	// Define color palette
	app.colors = []gooey.Style{
		gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite), // Off
		gooey.NewStyle().WithBackground(gooey.ColorRed).WithForeground(gooey.ColorWhite),
		gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack),
		gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite),
		gooey.NewStyle().WithBackground(gooey.ColorYellow).WithForeground(gooey.ColorBlack),
	}

	// Setup mouse regions for grid cells
	app.setupGridRegions()

	return nil
}

// setupGridRegions creates mouse regions for each grid cell
func (app *MouseGridApp) setupGridRegions() {
	for y := 0; y < app.gridH; y++ {
		for x := 0; x < app.gridW; x++ {
			screenX := app.startX + (x * (app.cellW + 1))
			screenY := app.startY + (y * (app.cellH + 1))

			// Capture loop variables
			cx, cy := x, y

			mouseRegion := &gooey.MouseRegion{
				X:      screenX,
				Y:      screenY,
				Width:  app.cellW,
				Height: app.cellH,
				ZIndex: 1,
				OnClick: func(e *gooey.MouseEvent) {
					// Toggle color
					app.gridState[cy][cx] = (app.gridState[cy][cx] + 1) % len(app.colors)
				},
			}
			app.mouse.AddRegion(mouseRegion)
		}
	}
}

// HandleEvent processes events
func (app *MouseGridApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.MouseEvent:
		// Forward mouse events to handler
		app.mouse.HandleEvent(&e)
		return nil

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
	return gooey.VStack(
		gooey.Text("Mouse Grid Demo").Bold().Fg(gooey.ColorCyan),
		gooey.Text("Click cells to toggle colors! Press 'q' or Ctrl+C to exit.").Fg(gooey.ColorWhite),
		gooey.Spacer().MinHeight(1),
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			// Render grid
			for y := 0; y < app.gridH; y++ {
				for x := 0; x < app.gridW; x++ {
					screenX := app.startX + (x * (app.cellW + 1))
					screenY := app.startY + (y * (app.cellH + 1))
					colorIdx := app.gridState[y][x]
					style := app.colors[colorIdx]

					// Draw cell
					app.renderCell(frame, screenX, screenY, app.cellW, app.cellH, style)
				}
			}
		}),
	).Align(gooey.AlignCenter)
}

// renderCell draws a single grid cell
func (app *MouseGridApp) renderCell(frame gooey.RenderFrame, x, y, width, height int, style gooey.Style) {
	for row := 0; row < height; row++ {
		frame.FillStyled(x, y+row, width, 1, ' ', style)
	}
}

func main() {
	// Mouse tracking is enabled via WithMouseTracking option, and mouse events are
	// automatically delivered to HandleEvent as gooey.MouseEvent
	if err := gooey.Run(&MouseGridApp{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
