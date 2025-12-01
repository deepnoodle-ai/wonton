package main

import (
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey/tui"
)

// MouseDemoApp demonstrates mouse support using the declarative View system.
// It shows clickable buttons, scroll areas, hover effects, and modifier detection.
type MouseDemoApp struct {
	mouse  *tui.MouseHandler
	width  int
	height int

	// State
	clickCount   map[string]int
	hoverRegion  string
	scrollOffset int
	modifierInfo string
	lastAction   string

	// Styles
	primaryStyle   tui.Style
	secondaryStyle tui.Style
	accentStyle    tui.Style
	textStyle      tui.Style
}

// Init initializes the application
func (app *MouseDemoApp) Init() error {
	app.mouse = tui.NewMouseHandler()
	// app.mouse.EnableDebug() // Uncomment for debug logging

	app.clickCount = make(map[string]int)

	// Define color scheme
	app.primaryStyle = tui.NewStyle().WithBackground(tui.ColorBlue).WithForeground(tui.ColorWhite)
	app.secondaryStyle = tui.NewStyle().WithBackground(tui.ColorMagenta).WithForeground(tui.ColorWhite)
	app.accentStyle = tui.NewStyle().WithBackground(tui.ColorCyan).WithForeground(tui.ColorBlack)
	app.textStyle = tui.NewStyle().WithForeground(tui.ColorWhite)

	app.lastAction = "Ready - interact with buttons below"

	app.setupMouseRegions()

	return nil
}

// setupMouseRegions creates all mouse regions
func (app *MouseDemoApp) setupMouseRegions() {
	// Row 1: Action buttons (centered, starting at y=4)
	buttonY := 4
	buttonSpacing := 3

	// Button 1: Increment counter
	app.mouse.AddRegion(&tui.MouseRegion{
		X:      10,
		Y:      buttonY,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *tui.MouseEvent) {
			app.clickCount["increment"]++
			app.lastAction = fmt.Sprintf("Incremented! Count: %d", app.clickCount["increment"])
		},
		OnEnter: func(event *tui.MouseEvent) {
			app.hoverRegion = "increment"
		},
		OnLeave: func(event *tui.MouseEvent) {
			app.hoverRegion = ""
		},
	})

	// Button 2: Reset counter
	app.mouse.AddRegion(&tui.MouseRegion{
		X:      35,
		Y:      buttonY,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *tui.MouseEvent) {
			app.clickCount["increment"] = 0
			app.lastAction = "Counter reset to 0"
		},
		OnEnter: func(event *tui.MouseEvent) {
			app.hoverRegion = "reset"
		},
		OnLeave: func(event *tui.MouseEvent) {
			app.hoverRegion = ""
		},
	})

	// Button 3: Info button
	app.mouse.AddRegion(&tui.MouseRegion{
		X:      60,
		Y:      buttonY,
		Width:  20,
		Height: 3,
		ZIndex: 1,
		OnClick: func(event *tui.MouseEvent) {
			app.lastAction = "Info: This demo showcases mouse interactions!"
		},
		OnDoubleClick: func(event *tui.MouseEvent) {
			app.lastAction = "Info: You double-clicked! Try triple-click too."
		},
		OnTripleClick: func(event *tui.MouseEvent) {
			app.lastAction = "Info: Triple-click detected!"
		},
		OnEnter: func(event *tui.MouseEvent) {
			app.hoverRegion = "info"
		},
		OnLeave: func(event *tui.MouseEvent) {
			app.hoverRegion = ""
		},
	})

	// Scroll area (centered panel)
	scrollY := buttonY + buttonSpacing + 3
	app.mouse.AddRegion(&tui.MouseRegion{
		X:      10,
		Y:      scrollY,
		Width:  70,
		Height: 8,
		ZIndex: 1,
		OnScroll: func(event *tui.MouseEvent) {
			if event.DeltaY != 0 {
				app.scrollOffset += event.DeltaY
				if event.DeltaY > 0 {
					app.lastAction = fmt.Sprintf("Scrolled down (offset: %d)", app.scrollOffset)
				} else {
					app.lastAction = fmt.Sprintf("Scrolled up (offset: %d)", app.scrollOffset)
				}
			}
		},
		OnEnter: func(event *tui.MouseEvent) {
			app.hoverRegion = "scroll"
		},
		OnLeave: func(event *tui.MouseEvent) {
			app.hoverRegion = ""
		},
	})

	// Modifier detection area
	modY := scrollY + 10
	app.mouse.AddRegion(&tui.MouseRegion{
		X:      10,
		Y:      modY,
		Width:  70,
		Height: 4,
		ZIndex: 1,
		OnClick: func(event *tui.MouseEvent) {
			mods := []string{}
			if event.Modifiers&tui.ModShift != 0 {
				mods = append(mods, "Shift")
			}
			if event.Modifiers&tui.ModCtrl != 0 {
				mods = append(mods, "Ctrl")
			}
			if event.Modifiers&tui.ModAlt != 0 {
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
		OnEnter: func(event *tui.MouseEvent) {
			app.hoverRegion = "modifier"
		},
		OnLeave: func(event *tui.MouseEvent) {
			app.hoverRegion = ""
		},
	})
}

// HandleEvent processes events
func (app *MouseDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.MouseEvent:
		// Forward mouse events to the mouse handler
		app.mouse.HandleEvent(&e)
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

// View returns the declarative view structure
func (app *MouseDemoApp) View() tui.View {
	return tui.VStack(
		tui.Spacer(),
		tui.Text("Gooey Mouse Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Press 'q' or Ctrl+C to exit"),
		tui.Spacer(),

		// Action buttons using Canvas for custom drawing
		app.renderActionButtonsView(),

		tui.Spacer(),

		// Scroll area
		app.renderScrollAreaView(),

		tui.Spacer(),

		// Modifier detection area
		app.renderModifierAreaView(),

		tui.Spacer(),

		// Status footer
		app.renderFooterView(),
	)
}

func (app *MouseDemoApp) renderActionButtonsView() tui.View {
	return tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
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
	})
}

func (app *MouseDemoApp) renderScrollAreaView() tui.View {
	return tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
		scrollY := 10
		boxStyle := tui.NewStyle().WithBackground(tui.ColorBrightBlack)
		borderStyle := tui.NewStyle().WithForeground(tui.ColorCyan)

		// Draw border
		app.renderBorder(frame, 9, scrollY-1, 72, 10, borderStyle)

		// Fill background
		app.renderBox(frame, 10, scrollY, 70, 8, boxStyle)

		// Title
		title := "Scrollable Content Area"
		if app.hoverRegion == "scroll" {
			title += " (hovering)"
		}
		titleStyle := tui.NewStyle().WithBold().WithForeground(tui.ColorYellow)
		frame.PrintStyled(12, scrollY, title, titleStyle)

		// Content with scroll offset
		contentStyle := tui.NewStyle().WithForeground(tui.ColorWhite)
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
		indicatorStyle := tui.NewStyle().WithForeground(tui.ColorCyan)
		indicator := fmt.Sprintf("Scroll: %d/%d", startLine+1, len(content))
		frame.PrintStyled(68, scrollY+7, indicator, indicatorStyle)
	})
}

func (app *MouseDemoApp) renderModifierAreaView() tui.View {
	return tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
		modY := 21
		boxStyle := tui.NewStyle().WithBackground(tui.ColorBrightBlack)
		borderStyle := tui.NewStyle().WithForeground(tui.ColorMagenta)

		// Draw border
		app.renderBorder(frame, 9, modY-1, 72, 6, borderStyle)

		// Fill background
		app.renderBox(frame, 10, modY, 70, 4, boxStyle)

		// Title
		title := "Modifier Key Detection"
		if app.hoverRegion == "modifier" {
			title += " (hovering)"
		}
		titleStyle := tui.NewStyle().WithBold().WithForeground(tui.ColorYellow)
		frame.PrintStyled(12, modY, title, titleStyle)

		// Instructions
		instructStyle := tui.NewStyle().WithForeground(tui.ColorWhite)
		frame.PrintStyled(12, modY+1, "Click here while holding Shift, Ctrl, or Alt keys", instructStyle)

		// Show last detected modifiers
		if app.modifierInfo != "" {
			modStyle := tui.NewStyle().WithForeground(tui.ColorGreen).WithBold()
			frame.PrintStyled(12, modY+2, fmt.Sprintf("Last detected: %s", app.modifierInfo), modStyle)
		}
	})
}

func (app *MouseDemoApp) renderFooterView() tui.View {
	sep := strings.Repeat("━", 80)
	return tui.VStack(
		tui.Text("%s", sep).Dim(),
		tui.HStack(
			tui.Text("Counter:").Fg(tui.ColorCyan),
			tui.Text("%d", app.clickCount["increment"]),
			tui.Text("  Status:").Fg(tui.ColorCyan),
			tui.Text("%s", app.lastAction),
		),
	)
}

func (app *MouseDemoApp) renderButton(frame tui.RenderFrame, text string, x, y, width, height int, style tui.Style) {
	// Draw button background
	app.renderBox(frame, x, y, width, height, style)

	// Center text
	textX := x + (width-len(text))/2
	textY := y + height/2
	frame.PrintStyled(textX, textY, text, style)
}

func (app *MouseDemoApp) renderBox(frame tui.RenderFrame, x, y, width, height int, style tui.Style) {
	for row := 0; row < height; row++ {
		frame.FillStyled(x, y+row, width, 1, ' ', style)
	}
}

func (app *MouseDemoApp) renderBorder(frame tui.RenderFrame, x, y, width, height int, style tui.Style) {
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
	// Mouse tracking is enabled via WithMouseTracking option, and mouse events are
	// automatically delivered to HandleEvent as tui.MouseEvent
	if err := tui.Run(&MouseDemoApp{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
