package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey"
	"golang.org/x/term"
)

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		panic(err)
	}

	// Setup raw mode for mouse input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
		terminal.DisableMouseTracking()
		terminal.DisableAlternateScreen()
		terminal.ShowCursor()
	}()

	terminal.EnableAlternateScreen()
	terminal.EnableMouseTracking()
	terminal.HideCursor()

	width, height := terminal.Size()
	mouse := gooey.NewMouseHandler()
	// mouse.EnableDebug() // Uncomment for debug logging (outputs to stderr)

	// State
	clickCount := map[string]int{
		"single": 0,
		"double": 0,
		"triple": 0,
	}
	dragInfo := ""
	hoverInfo := ""
	scrollCount := 0
	modifierInfo := ""
	layerInfo := ""

	// Colors
	normalStyle := gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)
	hoverStyle := gooey.NewStyle().WithBackground(gooey.ColorCyan).WithForeground(gooey.ColorBlack)
	dragStyle := gooey.NewStyle().WithBackground(gooey.ColorYellow).WithForeground(gooey.ColorBlack)

	// Clickable button
	buttonRegion := &gooey.MouseRegion{
		X:      5,
		Y:      5,
		Width:  30,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *gooey.MouseEvent) {
			clickCount["single"]++
			renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnDoubleClick: func(event *gooey.MouseEvent) {
			clickCount["double"]++
			renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnTripleClick: func(event *gooey.MouseEvent) {
			clickCount["triple"]++
			renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnEnter: func(event *gooey.MouseEvent) {
			hoverInfo = "Button (hover)"
			renderButton(terminal, "Click Me!", 5, 5, 30, 3, hoverStyle)
			renderInfo(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnLeave: func(event *gooey.MouseEvent) {
			hoverInfo = ""
			renderButton(terminal, "Click Me!", 5, 5, 30, 3, normalStyle)
			renderInfo(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
	}
	mouse.AddRegion(buttonRegion)

	// Draggable box state
	var dragRegion *gooey.MouseRegion
	dragRegion = &gooey.MouseRegion{
		X:      40,
		Y:      5,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnDragStart: func(event *gooey.MouseEvent) {
			dragInfo = "Dragging..."
			renderInfo(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnDrag: func(event *gooey.MouseEvent) {
			// Update position
			dragRegion.X = event.X - 10 // Center on cursor
			dragRegion.Y = event.Y - 1

			dragInfo = fmt.Sprintf("Dragging... (%d, %d)", dragRegion.X, dragRegion.Y)
			renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnDragEnd: func(event *gooey.MouseEvent) {
			dragInfo = fmt.Sprintf("Dropped at (%d, %d)", dragRegion.X, dragRegion.Y)
			renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnEnter: func(event *gooey.MouseEvent) {
			hoverInfo = "Draggable box (hover)"
			renderButton(terminal, "Drag Me!", dragRegion.X, dragRegion.Y, 20, 3, hoverStyle)
			renderInfo(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnLeave: func(event *gooey.MouseEvent) {
			hoverInfo = ""
			renderButton(terminal, "Drag Me!", dragRegion.X, dragRegion.Y, 20, 3, dragStyle)
			renderInfo(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
	}
	mouse.AddRegion(dragRegion)

	// Scroll area
	scrollRegion := &gooey.MouseRegion{
		X:      5,
		Y:      10,
		Width:  55,
		Height: 5,
		ZIndex: 1,
		OnScroll: func(event *gooey.MouseEvent) {
			if event.DeltaY != 0 {
				scrollCount += event.DeltaY
				renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
			}
		},
		OnEnter: func(event *gooey.MouseEvent) {
			hoverInfo = "Scroll area (use mouse wheel)"
			renderInfo(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
		OnLeave: func(event *gooey.MouseEvent) {
			hoverInfo = ""
			renderInfo(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
	}
	mouse.AddRegion(scrollRegion)

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
				modifierInfo = "No modifiers"
			} else {
				modifierInfo = strings.Join(mods, "+")
			}
			renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
	}
	mouse.AddRegion(modifierRegion)

	// Z-index demonstration - overlapping buttons
	layer1Region := &gooey.MouseRegion{
		X:      5,
		Y:      22,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *gooey.MouseEvent) {
			layerInfo = "Layer 1 (z=1)"
			renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
	}
	mouse.AddRegion(layer1Region)

	layer2Region := &gooey.MouseRegion{
		X:      15,
		Y:      23,
		Width:  20,
		Height: 3,
		ZIndex: 2,
		OnClick: func(event *gooey.MouseEvent) {
			layerInfo = "Layer 2 (z=2)"
			renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
		},
	}
	mouse.AddRegion(layer2Region)

	// Initial render
	renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)

	// Event loop
	buf := make([]byte, 128)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}

		if n > 0 {
			// Check for Ctrl+C or 'q'
			if buf[0] == 3 || buf[0] == 'q' {
				return
			}

			// Check for Escape (to cancel drag)
			if buf[0] == 27 && n == 1 {
				mouse.CancelDrag()
				dragInfo = "Drag cancelled"
				renderUI(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
				continue
			}

			// Handle Mouse events
			if buf[0] == 27 && n > 2 && buf[1] == '[' && buf[2] == '<' {
				event, err := gooey.ParseMouseEvent(buf[2:n])
				if err == nil {
					mouse.HandleEvent(event)
				}
			}
		}
	}
}

func renderUI(terminal *gooey.Terminal, clickCount map[string]int, dragInfo, hoverInfo string, scrollCount int, modifierInfo, layerInfo string, width, height int) {
	terminal.Clear()

	// Title
	title := "🖱️  Full Mouse Support Demo - Press 'q' or Ctrl+C to exit, Esc to cancel drag"
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	frame, _ := terminal.BeginFrame()
	frame.PrintStyled((width-len(title))/2, 0, title, titleStyle)
	terminal.EndFrame(frame)

	// Render all regions
	normalStyle := gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)
	dragStyle := gooey.NewStyle().WithBackground(gooey.ColorYellow).WithForeground(gooey.ColorBlack)
	layer1Style := gooey.NewStyle().WithBackground(gooey.ColorMagenta).WithForeground(gooey.ColorWhite)
	layer2Style := gooey.NewStyle().WithBackground(gooey.ColorRed).WithForeground(gooey.ColorWhite)

	renderButton(terminal, "Click Me!", 5, 5, 30, 3, normalStyle)
	renderButton(terminal, "Drag Me!", 40, 5, 20, 3, dragStyle)

	// Scroll area
	scrollStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite)
	renderBox(terminal, 5, 10, 55, 5, scrollStyle)
	frame, _ = terminal.BeginFrame()
	frame.PrintStyled(7, 11, "Scroll Area - Use mouse wheel", gooey.NewStyle().WithForeground(gooey.ColorWhite))
	terminal.EndFrame(frame)

	// Modifier detection area
	modStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite)
	renderBox(terminal, 5, 17, 55, 3, modStyle)
	frame, _ = terminal.BeginFrame()
	frame.PrintStyled(7, 18, "Click with Shift/Ctrl/Alt modifiers", gooey.NewStyle().WithForeground(gooey.ColorWhite))
	terminal.EndFrame(frame)

	// Z-index layers
	renderButton(terminal, "Layer 1 (z=1)", 5, 22, 20, 3, layer1Style)
	renderButton(terminal, "Layer 2 (z=2)", 15, 23, 20, 3, layer2Style)

	// Info panel
	renderInfo(terminal, clickCount, dragInfo, hoverInfo, scrollCount, modifierInfo, layerInfo, width, height)
}

func renderButton(terminal *gooey.Terminal, text string, x, y, width, height int, style gooey.Style) {
	renderBox(terminal, x, y, width, height, style)
	frame, _ := terminal.BeginFrame()
	textX := x + (width-len(text))/2
	textY := y + height/2
	frame.PrintStyled(textX, textY, text, style)
	terminal.EndFrame(frame)
}

func renderBox(terminal *gooey.Terminal, x, y, width, height int, style gooey.Style) {
	frame, _ := terminal.BeginFrame()
	for row := 0; row < height; row++ {
		frame.FillStyled(x, y+row, width, 1, ' ', style)
	}
	terminal.EndFrame(frame)
}

func renderInfo(terminal *gooey.Terminal, clickCount map[string]int, dragInfo, hoverInfo string, scrollCount int, modifierInfo, layerInfo string, width, height int) {
	infoY := height - 12
	infoStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)

	frame, _ := terminal.BeginFrame()

	frame.PrintStyled(2, infoY, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━", infoStyle)
	frame.PrintStyled(2, infoY+1, "Status:", gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan))

	y := infoY + 2
	frame.PrintStyled(2, y, fmt.Sprintf("Single Clicks: %d", clickCount["single"]), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Double Clicks: %d", clickCount["double"]), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Triple Clicks: %d", clickCount["triple"]), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Scroll Count: %d", scrollCount), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Drag: %s", dragInfo), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Hover: %s", hoverInfo), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Modifiers: %s", modifierInfo), infoStyle)
	y++
	frame.PrintStyled(2, y, fmt.Sprintf("Layer Clicked: %s", layerInfo), infoStyle)

	terminal.EndFrame(frame)
}
