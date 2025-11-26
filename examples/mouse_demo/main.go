package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// MouseDemoApp demonstrates mouse support with the Runtime architecture.
// It shows clickable buttons, scroll areas, hover effects, and modifier detection.
type MouseDemoApp struct {
	terminal *gooey.Terminal
	mouse    *gooey.MouseHandler
	width    int
	height   int

	// State
	clickCount   map[string]int
	hoverRegion  string
	scrollOffset int
	modifierInfo string
	lastAction   string

	// Styles
	primaryStyle   gooey.Style
	secondaryStyle gooey.Style
	accentStyle    gooey.Style
	textStyle      gooey.Style
}

// Init initializes the application
func (app *MouseDemoApp) Init() error {
	app.terminal.EnableMouseTracking()
	app.terminal.HideCursor()

	app.width, app.height = app.terminal.Size()
	app.mouse = gooey.NewMouseHandler()
	// app.mouse.EnableDebug() // Uncomment for debug logging

	app.clickCount = make(map[string]int)

	// Define color scheme
	app.primaryStyle = gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)
	app.secondaryStyle = gooey.NewStyle().WithBackground(gooey.ColorMagenta).WithForeground(gooey.ColorWhite)
	app.accentStyle = gooey.NewStyle().WithBackground(gooey.ColorCyan).WithForeground(gooey.ColorBlack)
	app.textStyle = gooey.NewStyle().WithForeground(gooey.ColorWhite)

	app.lastAction = "Ready - interact with buttons below"

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
	// Row 1: Action buttons (centered, starting at y=4)
	buttonY := 4
	buttonSpacing := 3

	// Button 1: Increment counter
	app.mouse.AddRegion(&gooey.MouseRegion{
		X:      10,
		Y:      buttonY,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *gooey.MouseEvent) {
			app.clickCount["increment"]++
			app.lastAction = fmt.Sprintf("Incremented! Count: %d", app.clickCount["increment"])
		},
		OnEnter: func(event *gooey.MouseEvent) {
			app.hoverRegion = "increment"
		},
		OnLeave: func(event *gooey.MouseEvent) {
			app.hoverRegion = ""
		},
	})

	// Button 2: Reset counter
	app.mouse.AddRegion(&gooey.MouseRegion{
		X:      35,
		Y:      buttonY,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *gooey.MouseEvent) {
			app.clickCount["increment"] = 0
			app.lastAction = "Counter reset to 0"
		},
		OnEnter: func(event *gooey.MouseEvent) {
			app.hoverRegion = "reset"
		},
		OnLeave: func(event *gooey.MouseEvent) {
			app.hoverRegion = ""
		},
	})

	// Button 3: Info button
	app.mouse.AddRegion(&gooey.MouseRegion{
		X:      60,
		Y:      buttonY,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *gooey.MouseEvent) {
			app.lastAction = "Info: This demo showcases mouse interactions!"
		},
		OnDoubleClick: func(event *gooey.MouseEvent) {
			app.lastAction = "Info: You double-clicked! Try triple-click too."
		},
		OnTripleClick: func(event *gooey.MouseEvent) {
			app.lastAction = "Info: Triple-click detected!"
		},
		OnEnter: func(event *gooey.MouseEvent) {
			app.hoverRegion = "info"
		},
		OnLeave: func(event *gooey.MouseEvent) {
			app.hoverRegion = ""
		},
	})

	// Scroll area (centered panel)
	scrollY := buttonY + buttonSpacing + 3
	app.mouse.AddRegion(&gooey.MouseRegion{
		X:      10,
		Y:      scrollY,
		Width:  70,
		Height: 8,
		ZIndex: 1,
		OnScroll: func(event *gooey.MouseEvent) {
			if event.DeltaY != 0 {
				app.scrollOffset += event.DeltaY
				if event.DeltaY > 0 {
					app.lastAction = fmt.Sprintf("Scrolled down (offset: %d)", app.scrollOffset)
				} else {
					app.lastAction = fmt.Sprintf("Scrolled up (offset: %d)", app.scrollOffset)
				}
			}
		},
		OnEnter: func(event *gooey.MouseEvent) {
			app.hoverRegion = "scroll"
		},
		OnLeave: func(event *gooey.MouseEvent) {
			app.hoverRegion = ""
		},
	})

	// Modifier detection area
	modY := scrollY + 10
	app.mouse.AddRegion(&gooey.MouseRegion{
		X:      10,
		Y:      modY,
		Width:  70,
		Height: 4,
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
				app.modifierInfo = "none"
				app.lastAction = "Click detected with no modifiers"
			} else {
				app.modifierInfo = strings.Join(mods, "+")
				app.lastAction = fmt.Sprintf("Click with %s modifier(s)", app.modifierInfo)
			}
		},
		OnEnter: func(event *gooey.MouseEvent) {
			app.hoverRegion = "modifier"
		},
		OnLeave: func(event *gooey.MouseEvent) {
			app.hoverRegion = ""
		},
	})
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
	bgStyle := gooey.NewStyle().WithBackground(gooey.ColorBlack)
	frame.FillStyled(0, 0, app.width, app.height, ' ', bgStyle)

	// Header
	app.renderHeader(frame)

	// Action buttons
	app.renderActionButtons(frame)

	// Scroll area
	app.renderScrollArea(frame)

	// Modifier detection area
	app.renderModifierArea(frame)

	// Status footer
	app.renderFooter(frame)
}

func (app *MouseDemoApp) renderHeader(frame gooey.RenderFrame) {
	title := "Gooey Mouse Demo"
	subtitle := "Press 'q' or Ctrl+C to exit"

	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	subtitleStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)

	frame.PrintStyled((app.width-len(title))/2, 1, title, titleStyle)
	frame.PrintStyled((app.width-len(subtitle))/2, 2, subtitle, subtitleStyle)
}

func (app *MouseDemoApp) renderActionButtons(frame gooey.RenderFrame) {
	buttonY := 4

	// Button 1: Increment
	style1 := app.primaryStyle
	if app.hoverRegion == "increment" {
		style1 = app.accentStyle
	}
	app.renderButton(frame, "Increment", 10, buttonY, 20, 3, style1)

	// Button 2: Reset
	style2 := app.secondaryStyle
	if app.hoverRegion == "reset" {
		style2 = app.accentStyle
	}
	app.renderButton(frame, "Reset", 35, buttonY, 20, 3, style2)

	// Button 3: Info
	style3 := app.primaryStyle
	if app.hoverRegion == "info" {
		style3 = app.accentStyle
	}
	app.renderButton(frame, "Info (try clicks)", 60, buttonY, 20, 3, style3)
}

func (app *MouseDemoApp) renderScrollArea(frame gooey.RenderFrame) {
	scrollY := 10
	boxStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack)
	borderStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)

	// Draw border
	app.renderBorder(frame, 9, scrollY-1, 72, 10, borderStyle)

	// Fill background
	app.renderBox(frame, 10, scrollY, 70, 8, boxStyle)

	// Title
	title := "Scrollable Content Area"
	if app.hoverRegion == "scroll" {
		title += " (hovering)"
	}
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(12, scrollY, title, titleStyle)

	// Content with scroll offset
	contentStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	content := []string{
		"Use your mouse wheel to scroll through this area.",
		"",
		"Line 1: The quick brown fox jumps over the lazy dog",
		"Line 2: Lorem ipsum dolor sit amet, consectetur",
		"Line 3: The five boxing wizards jump quickly",
		"Line 4: Pack my box with five dozen liquor jugs",
		"Line 5: How vexingly quick daft zebras jump!",
		"Line 6: The jay, pig, fox, zebra and my wolves",
		"Line 7: Sphinx of black quartz, judge my vow",
		"Line 8: Two driven jocks help fax my big quiz",
	}

	startLine := app.scrollOffset
	if startLine < 0 {
		startLine = 0
	}
	if startLine > len(content)-5 {
		startLine = len(content) - 5
	}
	app.scrollOffset = startLine

	for i := 0; i < 6 && startLine+i < len(content); i++ {
		frame.PrintStyled(12, scrollY+i+1, content[startLine+i], contentStyle)
	}

	// Scroll indicator
	indicatorStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
	indicator := fmt.Sprintf("Scroll: %d/%d", startLine+1, len(content))
	frame.PrintStyled(68, scrollY+7, indicator, indicatorStyle)
}

func (app *MouseDemoApp) renderModifierArea(frame gooey.RenderFrame) {
	modY := 21
	boxStyle := gooey.NewStyle().WithBackground(gooey.ColorBrightBlack)
	borderStyle := gooey.NewStyle().WithForeground(gooey.ColorMagenta)

	// Draw border
	app.renderBorder(frame, 9, modY-1, 72, 6, borderStyle)

	// Fill background
	app.renderBox(frame, 10, modY, 70, 4, boxStyle)

	// Title
	title := "Modifier Key Detection"
	if app.hoverRegion == "modifier" {
		title += " (hovering)"
	}
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(12, modY, title, titleStyle)

	// Instructions
	instructStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	frame.PrintStyled(12, modY+1, "Click here while holding Shift, Ctrl, or Alt keys", instructStyle)

	// Show last detected modifiers
	if app.modifierInfo != "" {
		modStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()
		frame.PrintStyled(12, modY+2, fmt.Sprintf("Last detected: %s", app.modifierInfo), modStyle)
	}
}

func (app *MouseDemoApp) renderFooter(frame gooey.RenderFrame) {
	footerY := app.height - 3
	separatorStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	labelStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
	valueStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	bgStyle := gooey.NewStyle().WithBackground(gooey.ColorBlack)

	// Clear footer area (3 lines) to hide any content behind it
	for row := footerY; row < footerY+3 && row < app.height; row++ {
		frame.FillStyled(0, row, app.width, 1, ' ', bgStyle)
	}

	// Separator
	separator := strings.Repeat("━", app.width)
	frame.PrintStyled(0, footerY, separator, separatorStyle)

	// Counter
	frame.PrintStyled(2, footerY+1, "Counter:", labelStyle)
	frame.PrintStyled(12, footerY+1, fmt.Sprintf("%d", app.clickCount["increment"]), valueStyle)

	// Last action
	frame.PrintStyled(20, footerY+1, "Status:", labelStyle)
	frame.PrintStyled(29, footerY+1, app.lastAction, valueStyle)
}

func (app *MouseDemoApp) renderButton(frame gooey.RenderFrame, text string, x, y, width, height int, style gooey.Style) {
	// Draw button background
	app.renderBox(frame, x, y, width, height, style)

	// Center text
	textX := x + (width-len(text))/2
	textY := y + height/2
	frame.PrintStyled(textX, textY, text, style)
}

func (app *MouseDemoApp) renderBox(frame gooey.RenderFrame, x, y, width, height int, style gooey.Style) {
	for row := 0; row < height; row++ {
		frame.FillStyled(x, y+row, width, 1, ' ', style)
	}
}

func (app *MouseDemoApp) renderBorder(frame gooey.RenderFrame, x, y, width, height int, style gooey.Style) {
	// Top and bottom
	for col := 0; col < width; col++ {
		frame.PrintStyled(x+col, y, "─", style)
		frame.PrintStyled(x+col, y+height-1, "─", style)
	}

	// Left and right
	for row := 0; row < height; row++ {
		frame.PrintStyled(x, y+row, "│", style)
		frame.PrintStyled(x+width-1, y+row, "│", style)
	}

	// Corners
	frame.PrintStyled(x, y, "┌", style)
	frame.PrintStyled(x+width-1, y, "┐", style)
	frame.PrintStyled(x, y+height-1, "└", style)
	frame.PrintStyled(x+width-1, y+height-1, "┘", style)
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

	// Create runtime with 30 FPS
	// Mouse tracking is enabled in Init(), and mouse events are automatically
	// delivered to HandleEvent as gooey.MouseEvent
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run the application (blocks until quit)
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
