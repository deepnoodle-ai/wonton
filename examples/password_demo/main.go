package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// PasswordDemoApp demonstrates various password input modes using Runtime.
type PasswordDemoApp struct {
	stage       int
	currentDemo int
	passwords   [5]string
	message     string
}

func (app *PasswordDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if app.stage == 0 {
			// Intro screen - any key starts
			app.stage = 1
			app.currentDemo = 1
			return []gooey.Cmd{app.cmdReadPassword(1)}
		} else if app.stage == app.currentDemo*2 && app.currentDemo < 5 {
			// After showing result, move to next demo
			app.currentDemo++
			app.stage = app.currentDemo*2 - 1
			return []gooey.Cmd{app.cmdReadPassword(app.currentDemo)}
		} else if app.stage == 10 {
			// After last demo result, show summary
			app.stage = 11
		} else if app.stage == 11 {
			// Summary screen - any key quits
			return []gooey.Cmd{gooey.Quit()}
		}

	case PasswordResultEvent:
		// Handle password input results
		if e.Error != nil {
			app.message = fmt.Sprintf("Error: %v", e.Error)
			return []gooey.Cmd{gooey.Quit()}
		}

		app.passwords[e.DemoNum-1] = e.Value
		app.stage = e.DemoNum * 2
		app.message = "Press any key to continue..."
	}

	return nil
}

func (app *PasswordDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	headerStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	infoStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

	// Title
	frame.PrintStyled(0, 0, "🔒 Gooey Secure Password Input Demo", headerStyle)
	frame.PrintStyled(0, 1, "====================================", headerStyle)

	y := 3

	// Detect terminal type
	termProgram := os.Getenv("TERM_PROGRAM")
	if termProgram != "" && app.stage == 0 {
		frame.PrintStyled(0, y, fmt.Sprintf("Detected terminal: %s", termProgram), infoStyle)
		y++
		if termProgram == "iTerm.app" {
			frame.PrintStyled(0, y, "✓ iTerm2 secure input mode will be enabled", infoStyle)
		} else if termProgram == "vscode" {
			frame.PrintStyled(0, y, "✓ VS Code terminal detected", infoStyle)
		} else {
			frame.PrintStyled(0, y, "ℹ Using generic secure mode", infoStyle)
		}
		y += 2
	}

	switch app.stage {
	case 0:
		// Welcome screen
		frame.PrintStyled(0, y, "This demo showcases secure password input modes:", gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "1. Standard secure password (no echo)", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "2. Password with masked characters (*)", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "3. Password with custom mask (•)", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "4. Password with max length (16 chars)", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "5. Password with placeholder", gooey.NewStyle())
		y += 2
		frame.PrintStyled(0, y, "Press any key to start", gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 1:
		// Demo 1: Reading password (no echo)
		frame.PrintStyled(0, y, "Demo 1: Standard secure password (no echo)", gooey.NewStyle().WithBold())
		frame.PrintStyled(0, y+1, "-------------------------------------------", gooey.NewStyle())
		y += 3
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
		frame.PrintStyled(0, y, "Enter password: ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated input - no echo)", infoStyle)

	case 2:
		// Show Demo 1 result
		frame.PrintStyled(0, y, "Demo 1: Standard secure password (no echo)", gooey.NewStyle().WithBold())
		y += 2
		frame.PrintStyled(0, y, fmt.Sprintf("✓ Password received (length: %d)", len(app.passwords[0])), successStyle)
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 3:
		// Demo 2: Reading password (masked)
		frame.PrintStyled(0, y, "Demo 2: Password with masked characters (*)", gooey.NewStyle().WithBold())
		frame.PrintStyled(0, y+1, "--------------------------------------------", gooey.NewStyle())
		y += 3
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
		frame.PrintStyled(0, y, "Enter password: ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated input - shows asterisks)", infoStyle)

	case 4:
		// Show Demo 2 result
		frame.PrintStyled(0, y, "Demo 2: Password with masked characters (*)", gooey.NewStyle().WithBold())
		y += 2
		frame.PrintStyled(0, y, fmt.Sprintf("✓ Password received (length: %d)", len(app.passwords[1])), successStyle)
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 5:
		// Demo 3: Reading password (custom mask)
		frame.PrintStyled(0, y, "Demo 3: Password with custom mask (•)", gooey.NewStyle().WithBold())
		frame.PrintStyled(0, y+1, "---------------------------------------", gooey.NewStyle())
		y += 3
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
		frame.PrintStyled(0, y, "Enter password: ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated input - shows bullet points)", infoStyle)

	case 6:
		// Show Demo 3 result
		frame.PrintStyled(0, y, "Demo 3: Password with custom mask (•)", gooey.NewStyle().WithBold())
		y += 2
		frame.PrintStyled(0, y, fmt.Sprintf("✓ Password received (length: %d)", len(app.passwords[2])), successStyle)
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 7:
		// Demo 4: Reading password (max length)
		frame.PrintStyled(0, y, "Demo 4: Password with max length (16 chars)", gooey.NewStyle().WithBold())
		frame.PrintStyled(0, y+1, "--------------------------------------------", gooey.NewStyle())
		y += 3
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
		frame.PrintStyled(0, y, "Enter password: ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated input - 16 char limit)", infoStyle)

	case 8:
		// Show Demo 4 result
		frame.PrintStyled(0, y, "Demo 4: Password with max length (16 chars)", gooey.NewStyle().WithBold())
		y += 2
		frame.PrintStyled(0, y, fmt.Sprintf("✓ Password received (length: %d)", len(app.passwords[3])), successStyle)
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 9:
		// Demo 5: Reading password (with placeholder)
		frame.PrintStyled(0, y, "Demo 5: Password with placeholder", gooey.NewStyle().WithBold())
		frame.PrintStyled(0, y+1, "----------------------------------", gooey.NewStyle())
		y += 3
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
		frame.PrintStyled(0, y, "Enter password: ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated input - with placeholder)", infoStyle)

	case 10:
		// Show Demo 5 result
		frame.PrintStyled(0, y, "Demo 5: Password with placeholder", gooey.NewStyle().WithBold())
		y += 2
		frame.PrintStyled(0, y, fmt.Sprintf("✓ Password received (length: %d)", len(app.passwords[4])), successStyle)
		y += 2
		frame.PrintStyled(0, y, app.message, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 11:
		// Summary
		frame.PrintStyled(0, y, "=====================================", headerStyle)
		frame.PrintStyled(0, y+1, "✓ All demos completed successfully!", successStyle)
		frame.PrintStyled(0, y+2, "=====================================", headerStyle)
		y += 4

		// Security tips
		frame.PrintStyled(0, y, "Security Features:", infoStyle)
		y++
		frame.PrintStyled(0, y, "  • iTerm2 secure input mode (prevents keylogging)", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "  • Memory is zeroed after use (via defer password.Clear())", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "  • Clipboard disable option available", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "  • Paste confirmation (can be enabled)", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "  • Max length enforcement", gooey.NewStyle())
		y += 2

		frame.PrintStyled(0, y, "Press any key to exit", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	}
}

// Commands for async password input operations (simulated)
func (app *PasswordDemoApp) cmdReadPassword(demoNum int) gooey.Cmd {
	return func() gooey.Event {
		// Simulate reading password based on demo number
		var password string
		switch demoNum {
		case 1:
			password = "password1"
		case 2:
			password = "password2"
		case 3:
			password = "password3"
		case 4:
			password = "pass4567890123" // 14 chars (under 16 limit)
		case 5:
			password = "password5"
		}

		return PasswordResultEvent{
			DemoNum: demoNum,
			Value:   password,
			Error:   nil,
		}
	}
}

// PasswordResultEvent is returned when password input completes
type PasswordResultEvent struct {
	DemoNum int
	Value   string
	Error   error
}

func (e PasswordResultEvent) Timestamp() time.Time {
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
	app := &PasswordDemoApp{
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
