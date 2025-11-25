package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// SynchronizedDemoApp demonstrates coordinated updates with animated content and interactive input.
// This shows how the Runtime handles complex UIs with multiple animated regions and user input.
type SynchronizedDemoApp struct {
	// State
	lines        [2]string
	selectedLine int
	cursorPos    [2]int
	counter      int
	width        int
	height       int

	// Animation colors
	colors []gooey.RGB
}

// Init initializes the application state.
func (app *SynchronizedDemoApp) Init() error {
	app.lines = [2]string{"", ""}
	app.selectedLine = 0
	app.cursorPos = [2]int{0, 0}
	app.counter = 0

	// Initialize animation colors
	app.colors = []gooey.RGB{
		gooey.NewRGB(255, 255, 0),
		gooey.NewRGB(255, 165, 0),
		gooey.NewRGB(0, 255, 255),
		gooey.NewRGB(0, 255, 100),
		gooey.NewRGB(255, 0, 255),
		gooey.NewRGB(100, 255, 100),
		gooey.NewRGB(0, 255, 100),
		gooey.NewRGB(0, 255, 100),
	}

	return nil
}

// HandleEvent processes events from the runtime.
func (app *SynchronizedDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle special keys
		if e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

		// Arrow keys for line navigation
		if e.Key == gooey.KeyArrowUp {
			if app.selectedLine > 0 {
				app.selectedLine--
			}
			return nil
		}
		if e.Key == gooey.KeyArrowDown {
			if app.selectedLine < 1 {
				app.selectedLine++
			}
			return nil
		}
		if e.Key == gooey.KeyArrowRight {
			if app.cursorPos[app.selectedLine] < len(app.lines[app.selectedLine]) {
				app.cursorPos[app.selectedLine]++
			}
			return nil
		}
		if e.Key == gooey.KeyArrowLeft {
			if app.cursorPos[app.selectedLine] > 0 {
				app.cursorPos[app.selectedLine]--
			}
			return nil
		}

		// Backspace
		if e.Key == gooey.KeyBackspace {
			line := app.lines[app.selectedLine]
			pos := app.cursorPos[app.selectedLine]
			if pos > 0 && len(line) > 0 {
				app.lines[app.selectedLine] = line[:pos-1] + line[pos:]
				app.cursorPos[app.selectedLine]--
			}
			return nil
		}

		// Enter - move to next line
		if e.Key == gooey.KeyEnter {
			if app.selectedLine < 1 {
				app.selectedLine++
			} else {
				app.selectedLine = 0
			}
			return nil
		}

		// Regular character input
		if e.Rune != 0 && e.Rune >= 32 && e.Rune < 127 {
			line := app.lines[app.selectedLine]
			pos := app.cursorPos[app.selectedLine]
			app.lines[app.selectedLine] = line[:pos] + string(e.Rune) + line[pos:]
			app.cursorPos[app.selectedLine]++
		}

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height

	case gooey.TickEvent:
		// Update animation counter every few ticks (~500ms at 30fps = 15 ticks)
		if e.Frame%15 == 0 {
			app.counter++
		}
	}

	return nil
}

// Render draws the current application state.
func (app *SynchronizedDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()
	app.width = width
	app.height = height

	// Define styles
	rainbowStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	normalStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	highlightStyle := gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack)

	// Header (lines 0-1)
	frame.PrintStyled(0, 0, "🚀 Synchronized Gooey Demo", rainbowStyle)
	frame.PrintStyled(0, 1, "All updates properly synchronized!", normalStyle)

	// Animated content region (lines 2-5)
	statuses := []string{
		"Status: Initializing...",
		"Status: Loading...",
		"Status: Connecting...",
		"Status: Processing...",
		"Status: Optimizing...",
		"Status: Finalizing...",
		"Status: Complete!",
		"Status: Ready",
	}

	statusIdx := app.counter % len(statuses)
	statusText := statuses[statusIdx]

	// Use different colors for different statuses
	var statusColor gooey.RGB
	if statusIdx == 6 { // "Complete!" gets special color
		statusColor = gooey.NewRGB(0, 255, 100)
	} else {
		statusColor = app.colors[statusIdx]
	}
	statusStyle := gooey.NewStyle().WithFgRGB(statusColor).WithBold()
	frame.PrintStyled(0, 2, statusText, statusStyle)

	// Progress bar
	progress := (app.counter % 20) + 1
	progressBar := strings.Repeat("█", progress) + strings.Repeat("░", 20-progress)
	progressText := fmt.Sprintf("Progress: %s %d%%", progressBar, progress*5)
	progressStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
	frame.PrintStyled(0, 3, progressText, progressStyle)

	// Blank line
	frame.PrintStyled(0, 4, "", normalStyle)

	// System info
	connections := 40 + (app.counter % 10)
	systemText := fmt.Sprintf("System: %d connections", connections)
	frame.PrintStyled(0, 5, systemText, normalStyle)

	// Input region (lines 6-7) - Interactive text input
	for i := 0; i < 2; i++ {
		prefix := fmt.Sprintf("Line %d: ", i+1)
		line := prefix + app.lines[i]

		if i == app.selectedLine {
			// Highlight selected line
			frame.PrintStyled(0, 6+i, line, highlightStyle)
		} else {
			// Normal line
			frame.PrintStyled(0, 6+i, line, normalStyle)
		}
	}

	// Footer (lines 8+)
	if height > 8 {
		footerStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
		frame.PrintStyled(0, 8, "↑↓ Switch lines | Type to add text | Ctrl+C to quit", footerStyle)

		if height > 9 {
			successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
			frame.PrintStyled(0, 9, "No more cursor jumping!", successStyle)
		}
	}

	// Note: We're not actually positioning the terminal cursor here since
	// Runtime manages the display. In a real app with text input, you'd
	// want to use a proper input widget or manage cursor visibility differently.
}

func main() {
	fmt.Println("\n🎨 Synchronized Gooey Demo")
	fmt.Println("This demo shows coordinated updates with animations and input:")
	fmt.Println("✨ Smooth animations")
	fmt.Println("🛡️  Protected input regions")
	fmt.Println("🎯 Proper synchronization via Runtime")
	fmt.Println("\nStarting...")

	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create application
	app := &SynchronizedDemoApp{}

	// Create runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run the event loop
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n👋 Thanks for trying the synchronized demo!")
}
