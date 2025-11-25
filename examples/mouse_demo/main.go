package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// MouseDemoApp demonstrates full mouse support with the Runtime architecture.
// It shows clickable buttons, draggable elements, scroll areas, and modifier detection.
type MouseDemoApp struct {
	terminal *gooey.Terminal
	mouse    *gooey.MouseHandler
	width    int
	height   int

	// State
	clickCount   map[string]int
	dragInfo     string
	hoverInfo    string
	scrollCount  int
	modifierInfo string
	layerInfo    string

	// Styles
	normalStyle gooey.Style
	hoverStyle  gooey.Style
	dragStyle   gooey.Style

	// Mouse regions (need to track for dynamic updates)
	dragRegion *gooey.MouseRegion
}

// Init initializes the application
func (app *MouseDemoApp) Init() error {
	app.terminal.EnableMouseTracking()
	app.terminal.HideCursor()

	app.width, app.height = app.terminal.Size()
	app.mouse = gooey.NewMouseHandler()
	// app.mouse.EnableDebug() // Uncomment for debug logging

	app.clickCount = map[string]int{
		"single": 0,
		"double": 0,
		"triple": 0,
	}

	app.normalStyle = gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)
	app.hoverStyle = gooey.NewStyle().WithBackground(gooey.ColorCyan).WithForeground(gooey.ColorBlack)
	app.dragStyle = gooey.NewStyle().WithBackground(gooey.ColorYellow).WithForeground(gooey.ColorBlack)

	app.setupMouseRegions()

	return nil
}

// Destroy cleans up resources
func (app *MouseDemoApp) Destroy() {
	app.terminal.DisableMouseTracking()
	app.terminal.ShowCursor()
}

// setupMouseRegions creates all mouse regions
func (app *MouseDemoApp) setupMouseRegions() {
	// Clickable button
	buttonRegion := &gooey.MouseRegion{
		X:      5,
		Y:      5,
		Width:  30,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *gooey.MouseEvent) {
			app.clickCount["single"]++
		},
		OnDoubleClick: func(event *gooey.MouseEvent) {
			app.clickCount["double"]++
		},
		OnTripleClick: func(event *gooey.MouseEvent) {
			app.clickCount["triple"]++
		},
		OnEnter: func(event *gooey.MouseEvent) {
			app.hoverInfo = "Button (hover)"
		},
		OnLeave: func(event *gooey.MouseEvent) {
			app.hoverInfo = ""
		},
	}
	app.mouse.AddRegion(buttonRegion)

	// Draggable box
	app.dragRegion = &gooey.MouseRegion{
		X:      40,
		Y:      5,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnDragStart: func(event *gooey.MouseEvent) {
			app.dragInfo = "Dragging..."
		},
		OnDrag: func(event *gooey.MouseEvent) {
			app.dragRegion.X = event.X - 10 // Center on cursor
			app.dragRegion.Y = event.Y - 1
			app.dragInfo = fmt.Sprintf("Dragging... (%d, %d)", app.dragRegion.X, app.dragRegion.Y)
		},
		OnDragEnd: func(event *gooey.MouseEvent) {
			app.dragInfo = fmt.Sprintf("Dropped at (%d, %d)", app.dragRegion.X, app.dragRegion.Y)
		},
		OnEnter: func(event *gooey.MouseEvent) {
			app.hoverInfo = "Draggable box (hover)"
		},
		OnLeave: func(event *gooey.MouseEvent) {
			app.hoverInfo = ""
		},
	}
	app.mouse.AddRegion(app.dragRegion)

	// Scroll area
	scrollRegion := &gooey.MouseRegion{
		X:      5,
		Y:      10,
		Width:  55,
		Height: 5,
		ZIndex: 1,
		OnScroll: func(event *gooey.MouseEvent) {
			if event.DeltaY != 0 {
				app.scrollCount += event.DeltaY
			}
		},
		OnEnter: func(event *gooey.MouseEvent) {
			app.hoverInfo = "Scroll area (use mouse wheel)"
		},
		OnLeave: func(event *gooey.MouseEvent) {
			app.hoverInfo = ""
		},
	}
	app.mouse.AddRegion(scrollRegion)

	// Modifier detection area
	modifierRegion := &gooey.MouseRegion{
		X:      5,
		Y:      17,
		Width:  55,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *gooey.MouseEvent) {
			mods := []string{}
			if event.Modifiers&gooey.ModShift != 0 {
				mods = append(mods, "Shift")
			}
			if event.Modifiers&gooey.ModCtrl != 0 {
				mods = append(mods, "Ctrl")
			}
			if event.Modifiers&gooey.ModAlt != 0 {
				mods = append(mods, "Alt")
			}
			if len(mods) == 0 {
				app.modifierInfo = "No modifiers"
			} else {
				app.modifierInfo = strings.Join(mods, "+")
			}
		},
	}
	app.mouse.AddRegion(modifierRegion)

	// Z-index demonstration - overlapping buttons
	layer1Region := &gooey.MouseRegion{
		X:      5,
		Y:      22,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *gooey.MouseEvent) {
			app.layerInfo = "Layer 1 (z=1)"
		},
	}
	app.mouse.AddRegion(layer1Region)

	layer2Region := &gooey.MouseRegion{
		X:      15,
		Y:      23,
		Width:  20,
		Height: 3,
		ZIndex: 2,
		OnClick: func(event *gooey.MouseEvent) {
			app.layerInfo = "Layer 2 (z=2)"
		},
	}
	app.mouse.AddRegion(layer2Region)
}

// HandleEvent processes events
func (app *MouseDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.MouseEvent:
		// Forward mouse events to the mouse handler
		app.mouse.HandleEvent(&e)
		return nil

	case gooey.KeyEvent:
		// Handle keyboard input
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		// ESC cancels drag
		if e.Key == gooey.KeyEscape {
			app.mouse.CancelDrag()
			app.dragInfo = "Drag cancelled"
		}
		return nil

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
		return nil
	}

	return nil
}

// Render draws the UI
func (app *MouseDemoApp) Render(frame gooey.RenderFrame) {
	// Clear screen
	frame.FillStyled(0, 0, app.width, app.height, ' ', gooey.NewStyle())

	// Title
	title := "🖱️  Full Mouse Support Demo - Press 'q' or Ctrl+C to exit, Esc to cancel drag"
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	frame.PrintStyled((app.width-len(title))/2, 0, title, titleStyle)

	// Render click button
	app.renderButton(frame, "Click Me!", 5, 5, 30, 3, app.normalStyle)

	// Render drag button
	app.renderButton(frame, "Drag Me!", app.dragRegion.X, app.dragRegion.Y, 20, 3, app.dragStyle)

	// Scroll area
	scrollStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite)
	app.renderBox(frame, 5, 10, 55, 5, scrollStyle)
	frame.PrintStyled(7, 11, "Scroll Area - Use mouse wheel", gooey.NewStyle().WithForeground(gooey.ColorWhite))

	// Modifier detection area
	modStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite)
	app.renderBox(frame, 5, 17, 55, 3, modStyle)
	frame.PrintStyled(7, 18, "Click with Shift/Ctrl/Alt modifiers", gooey.NewStyle().WithForeground(gooey.ColorWhite))

	// Z-index layers
	layer1Style := gooey.NewStyle().WithBackground(gooey.ColorMagenta).WithForeground(gooey.ColorWhite)
	layer2Style := gooey.NewStyle().WithBackground(gooey.ColorRed).WithForeground(gooey.ColorWhite)
	app.renderButton(frame, "Layer 1 (z=1)", 5, 22, 20, 3, layer1Style)
	app.renderButton(frame, "Layer 2 (z=2)", 15, 23, 20, 3, layer2Style)

	// Info panel
	app.renderInfo(frame)
}

func (app *MouseDemoApp) renderButton(frame gooey.RenderFrame, text string, x, y, width, height int, style gooey.Style) {
	app.renderBox(frame, x, y, width, height, style)
	textX := x + (width-len(text))/2
	textY := y + height/2
	frame.PrintStyled(textX, textY, text, style)
}

func (app *MouseDemoApp) renderBox(frame gooey.RenderFrame, x, y, width, height int, style gooey.Style) {
	for row := 0; row < height; row++ {
		frame.FillStyled(x, y+row, width, 1, ' ', style)
	}
}

func (app *MouseDemoApp) renderInfo(frame gooey.RenderFrame) {
	infoY := app.height - 12
	infoStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)

	frame.PrintStyled(2, infoY, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", infoStyle)
	frame.PrintStyled(2, infoY+1, "Status:", gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan))

	y := infoY + 2
	frame.PrintStyled(2, y, fmt.Sprintf("Single Clicks: %d", app.clickCount["single"]), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Double Clicks: %d", app.clickCount["double"]), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Triple Clicks: %d", app.clickCount["triple"]), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Scroll Count: %d", app.scrollCount), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Drag: %s", app.dragInfo), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Hover: %s", app.hoverInfo), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Modifiers: %s", app.modifierInfo), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Layer Clicked: %s", app.layerInfo), infoStyle)
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
	app := &MouseDemoApp{
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
	// Create base runtime (which handles keyboard)
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

		// Not a mouse event - reconstruct the bytes and use KeyDecoder
		// We need to handle this more elegantly
		// For simplicity, we'll use a pipe approach or just handle common cases

		// Handle common keyboard events directly
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
