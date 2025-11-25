package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// MouseGridApp demonstrates a clickable grid with mouse support using Runtime.
// Click cells to toggle through different colors.
type MouseGridApp struct {
	terminal *gooey.Terminal
	mouse    *gooey.MouseHandler
	width    int
	height   int

	// Grid configuration
	gridW   int
	gridH   int
	cellW   int
	cellH   int
	startX  int
	startY  int
	colors  []gooey.Style
	gridState [][]int
}

// Init initializes the application
func (app *MouseGridApp) Init() error {
	app.terminal.EnableMouseTracking()
	app.terminal.HideCursor()

	app.width, app.height = app.terminal.Size()
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

// Destroy cleans up resources
func (app *MouseGridApp) Destroy() {
	app.terminal.DisableMouseTracking()
	app.terminal.ShowCursor()
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

// Render draws the grid
func (app *MouseGridApp) Render(frame gooey.RenderFrame) {
	// Clear screen
	frame.FillStyled(0, 0, app.width, app.height, ' ', gooey.NewStyle())

	// Title
	title := "🖱️  Mouse Grid Demo"
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	frame.PrintStyled((app.width-len(title))/2, 0, title, titleStyle)

	subtitle := "Click cells to toggle colors! Press 'q' or Ctrl+C to exit."
	subtitleStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	frame.PrintStyled((app.width-len(subtitle))/2, 1, subtitle, subtitleStyle)

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
}

// renderCell draws a single grid cell
func (app *MouseGridApp) renderCell(frame gooey.RenderFrame, x, y, width, height int, style gooey.Style) {
	for row := 0; row < height; row++ {
		frame.FillStyled(x, y+row, width, 1, ' ', style)
	}
}

func main() {
	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create application
	app := &MouseGridApp{
		terminal: terminal,
	}

	// Create runtime with mouse support
	runtime := NewMouseRuntime(terminal, app, 30)

	// Run the application
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}

// MouseRuntime extends Runtime with mouse event support.
// It handles both keyboard and mouse input from stdin.
type MouseRuntime struct {
	*gooey.Runtime
	terminal *gooey.Terminal
}

// NewMouseRuntime creates a runtime that handles both keyboard and mouse events
func NewMouseRuntime(terminal *gooey.Terminal, app gooey.Application, fps int) *MouseRuntime {
	baseRuntime := gooey.NewRuntime(terminal, app, fps)

	return &MouseRuntime{
		Runtime:  baseRuntime,
		terminal: terminal,
	}
}

// Run starts the mouse-aware runtime
func (r *MouseRuntime) Run() error {
	// Start a custom input reader goroutine for mouse+keyboard
	go r.mouseInputReader()

	// Run the base runtime (which will handle events we send)
	return r.Runtime.Run()
}

// mouseInputReader reads both keyboard and mouse events from stdin.
// This replaces the Runtime's standard inputReader to handle both types of events.
func (r *MouseRuntime) mouseInputReader() {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Peek at the first byte to determine event type
		firstByte, err := reader.ReadByte()
		if err != nil {
			return
		}

		// Check if this is the start of a mouse event (ESC [ <)
		if firstByte == 27 {
			// Peek ahead to see if it's a mouse event
			next, err := reader.Peek(2)
			if err == nil && len(next) >= 2 && next[0] == '[' && next[1] == '<' {
				// Read the mouse sequence
				reader.ReadByte() // consume '['
				reader.ReadByte() // consume '<'

				buf := make([]byte, 20)
				i := 0
				buf[i] = '<'
				i++

				// Read until we find M or m
				for {
					b, err := reader.ReadByte()
					if err != nil {
						break
					}
					buf[i] = b
					i++
					if b == 'M' || b == 'm' {
						break
					}
					if i >= len(buf) {
						break
					}
				}

				// Parse and send mouse event
				event, err := gooey.ParseMouseEvent(buf[:i])
				if err == nil {
					r.SendEvent(*event)
				}
				continue
			}
		}

		// Handle keyboard events
		switch firstByte {
		case 'q', 'Q':
			r.SendEvent(gooey.KeyEvent{Rune: rune(firstByte)})
		case 3: // Ctrl+C
			r.SendEvent(gooey.KeyEvent{Key: gooey.KeyCtrlC})
		case 27: // ESC (not followed by mouse sequence)
			r.SendEvent(gooey.KeyEvent{Key: gooey.KeyEscape})
		default:
			// For other keys, create a KeyEvent
			if firstByte >= 32 && firstByte < 127 {
				r.SendEvent(gooey.KeyEvent{Rune: rune(firstByte)})
			}
		}
	}
}
