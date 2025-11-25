package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// InputAPIDemo demonstrates the three main input methods using Runtime.
// It shows ReadSimple, Read (with suggestions), and ReadPassword.
type InputAPIDemo struct {
	stage    int
	name     string
	command  string
	password string
	message  string
}

func (app *InputAPIDemo) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		switch app.stage {
		case 0:
			// Stage 0: Waiting to start demo
			if e.Rune == '\n' || e.Rune == '\r' || e.Key == gooey.KeyEnter {
				app.stage = 1
				app.message = "Starting demo..."
				return []gooey.Cmd{app.cmdReadName()}
			} else if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape {
				return []gooey.Cmd{gooey.Quit()}
			}

		case 2:
			// Stage 2: After name input, any key continues
			app.stage = 3
			app.message = "Reading command..."
			return []gooey.Cmd{app.cmdReadCommand()}

		case 4:
			// Stage 4: After command input, any key continues
			app.stage = 5
			app.message = "Reading password..."
			return []gooey.Cmd{app.cmdReadPassword()}

		case 6:
			// Stage 6: After password input, any key continues to summary
			app.stage = 7

		case 7:
			// Stage 7: Summary screen, any key quits
			return []gooey.Cmd{gooey.Quit()}
		}

	case InputResultEvent:
		// Handle input results
		if e.Error != nil {
			app.message = fmt.Sprintf("Error: %v", e.Error)
			return []gooey.Cmd{gooey.Quit()}
		}

		switch e.Stage {
		case 1:
			app.name = e.Value
			app.stage = 2
			app.message = "Press any key to continue..."
		case 3:
			app.command = e.Value
			app.stage = 4
			app.message = "Press any key to continue..."
		case 5:
			app.password = e.Value
			app.stage = 6
			app.message = "Press any key to continue..."
		}
	}

	return nil
}

func (app *InputAPIDemo) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	// Title
	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	frame.PrintStyled(0, 0, "╔═══════════════════════════════════════════════╗", titleStyle)
	frame.PrintStyled(0, 1, "║     Gooey Input API Demo                     ║", titleStyle)
	frame.PrintStyled(0, 2, "║     Demonstrating the new simplified API     ║", titleStyle)
	frame.PrintStyled(0, 3, "╚═══════════════════════════════════════════════╝", titleStyle)

	y := 5

	switch app.stage {
	case 0:
		// Welcome screen
		frame.PrintStyled(0, y, "This demo showcases the three main input methods:", gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "1. ReadSimple() - Basic line input", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y+1, "   Use for simple, non-interactive line reading", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		y += 3
		frame.PrintStyled(0, y, "2. Read() - Full-featured input", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y+1, "   Try arrow keys, history, and Tab for autocomplete", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		y += 3
		frame.PrintStyled(0, y, "3. ReadPassword() - Secure password input", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y+1, "   Your input will not be shown on screen", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		y += 3
		frame.PrintStyled(0, y, "Press Enter to start, Q to quit", gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 1:
		// Stage 1: Reading name
		frame.PrintStyled(0, y, "1. ReadSimple() - Basic line input", gooey.NewStyle().WithBold())
		y++
		frame.PrintStyled(0, y, "   Use for simple, non-interactive line reading", gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "Status: Waiting for input (this is simulated in Runtime)", gooey.NewStyle().WithForeground(gooey.ColorYellow))
		y++
		frame.PrintStyled(0, y, "Note: In a real implementation, input would be handled", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		y++
		frame.PrintStyled(0, y, "      via a custom input component.", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 2:
		// Show name result
		frame.PrintStyled(0, y, "1. ReadSimple() - Basic line input", gooey.NewStyle().WithBold())
		y += 2
		resultStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
		frame.PrintStyled(0, y, fmt.Sprintf("   → You entered: %s", app.name), resultStyle)
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 3:
		// Stage 3: Reading command
		frame.PrintStyled(0, y, "2. Read() - Full-featured input", gooey.NewStyle().WithBold())
		y++
		frame.PrintStyled(0, y, "   Try arrow keys, history, and Tab for autocomplete", gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "Status: Waiting for input (this is simulated in Runtime)", gooey.NewStyle().WithForeground(gooey.ColorYellow))
		y++
		frame.PrintStyled(0, y, "Suggestions: start, stop, restart, status, help", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 4:
		// Show command result
		frame.PrintStyled(0, y, "2. Read() - Full-featured input", gooey.NewStyle().WithBold())
		y += 2
		resultStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
		frame.PrintStyled(0, y, fmt.Sprintf("   → You chose: %s", app.command), resultStyle)
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 5:
		// Stage 5: Reading password
		frame.PrintStyled(0, y, "3. ReadPassword() - Secure password input", gooey.NewStyle().WithBold())
		y++
		frame.PrintStyled(0, y, "   Your input will not be shown on screen", gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "Status: Waiting for input (this is simulated in Runtime)", gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 6:
		// Show password result
		frame.PrintStyled(0, y, "3. ReadPassword() - Secure password input", gooey.NewStyle().WithBold())
		y += 2
		resultStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
		frame.PrintStyled(0, y, fmt.Sprintf("   → Password received (%d characters)", len(app.password)), resultStyle)
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 7:
		// Summary
		frame.PrintStyled(0, y, "╔═══════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║     Demo Complete!                           ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚═══════════════════════════════════════════════╝", titleStyle)
		y += 4
		frame.PrintStyled(0, y, "Summary of the new Input API:", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y+1, "  • Read()         - Full-featured interactive input", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y+2, "  • ReadPassword() - Secure password entry", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y+3, "  • ReadSimple()   - Basic line reading", gooey.NewStyle())
		y += 5
		frame.PrintStyled(0, y, "See documentation/input_api_migration.md for more details", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		y += 2
		frame.PrintStyled(0, y, "Press any key to exit", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	}
}

// Commands for async input operations (simulated)
func (app *InputAPIDemo) cmdReadName() gooey.Cmd {
	return func() gooey.Event {
		// In a real implementation, this would block on input
		// For demo purposes, we'll simulate with a default value
		return InputResultEvent{
			Stage: 1,
			Value: "Demo User",
			Error: nil,
		}
	}
}

func (app *InputAPIDemo) cmdReadCommand() gooey.Cmd {
	return func() gooey.Event {
		return InputResultEvent{
			Stage: 3,
			Value: "start",
			Error: nil,
		}
	}
}

func (app *InputAPIDemo) cmdReadPassword() gooey.Cmd {
	return func() gooey.Event {
		return InputResultEvent{
			Stage: 5,
			Value: "********",
			Error: nil,
		}
	}
}

// InputResultEvent is returned when input operations complete
type InputResultEvent struct {
	Stage int
	Value string
	Error error
}

func (e InputResultEvent) Timestamp() time.Time {
	return time.Now()
}

func main() {
	// Create and initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create the application
	app := &InputAPIDemo{
		stage:   0,
		message: "Welcome!",
	}

	// Create and run the runtime
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
