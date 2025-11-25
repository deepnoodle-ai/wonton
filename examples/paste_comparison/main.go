package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// PasteComparisonApp demonstrates security benefits of bracketed paste mode using Runtime.
type PasteComparisonApp struct {
	stage   int
	input   string
	message string
}

func (app *PasteComparisonApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if app.stage == 0 {
			// Instructions screen - any key continues
			app.stage = 1
			return []gooey.Cmd{app.cmdReadPasteInput()}
		} else if app.stage == 2 {
			// Result screen - any key quits
			return []gooey.Cmd{gooey.Quit()}
		}

	case PasteInputEvent:
		// Handle paste input result
		if e.Error != nil {
			app.message = fmt.Sprintf("Error: %v", e.Error)
			return []gooey.Cmd{gooey.Quit()}
		}

		app.input = e.Value
		app.stage = 2
	}

	return nil
}

func (app *PasteComparisonApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()

	y := 0

	switch app.stage {
	case 0:
		// Instructions screen
		frame.PrintStyled(0, y, "╔════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║        Bracketed Paste Mode - Security Demonstration      ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚════════════════════════════════════════════════════════════╝", titleStyle)
		y += 4

		frame.PrintStyled(0, y, "This demonstrates the security benefit of bracketed paste.", gooey.NewStyle())
		y += 2

		frame.PrintStyled(0, y, "Try pasting this malicious command:", gooey.NewStyle().WithForeground(gooey.ColorYellow))
		y++
		frame.PrintStyled(0, y, "  echo 'safe command'", gooey.NewStyle().WithForeground(gooey.ColorGreen))
		y++
		frame.PrintStyled(0, y, "  rm -rf /", gooey.NewStyle().WithForeground(gooey.ColorRed))
		y += 2

		frame.PrintStyled(0, y, "WITHOUT bracketed paste: Both commands execute immediately!", gooey.NewStyle().WithForeground(gooey.ColorRed))
		y++
		frame.PrintStyled(0, y, "WITH bracketed paste: Newlines are preserved in input buffer.", gooey.NewStyle().WithForeground(gooey.ColorGreen))
		y++
		frame.PrintStyled(0, y, "                       You can review before pressing Enter.", gooey.NewStyle().WithForeground(gooey.ColorGreen))
		y += 2

		frame.PrintStyled(0, y, "Press any key to continue...", gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 1:
		// Input screen
		frame.PrintStyled(0, y, "╔════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║                    Paste Input                             ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚════════════════════════════════════════════════════════════╝", titleStyle)
		y += 4

		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
		frame.PrintStyled(0, y, "Paste here (Ctrl+Enter to submit): ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated multiline paste in Runtime)", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 2:
		// Result screen
		frame.PrintStyled(0, y, "╔════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║                    Paste Received                          ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚════════════════════════════════════════════════════════════╝", titleStyle)
		y += 4

		frame.PrintStyled(0, y, "The following content was pasted:", gooey.NewStyle())
		y += 2

		lines := strings.Split(app.input, "\n")
		for i, line := range lines {
			lineStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
			if strings.Contains(line, "rm") {
				lineStyle = gooey.NewStyle().WithForeground(gooey.ColorRed).WithBold()
			}
			frame.PrintStyled(0, y+i, fmt.Sprintf("  Line %d: %s", i+1, line), lineStyle)
		}

		y += len(lines) + 2
		frame.PrintStyled(0, y, "✓ Bracketed paste prevented immediate execution!", gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold())
		y++
		frame.PrintStyled(0, y, "  You can now review the content before deciding to execute.", gooey.NewStyle())
		y += 2

		frame.PrintStyled(0, y, "Press any key to exit...", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	}
}

// Command for async paste input (simulated)
func (app *PasteComparisonApp) cmdReadPasteInput() gooey.Cmd {
	return func() gooey.Event {
		// Simulate a potentially malicious paste
		sampleInput := `echo 'safe command'
rm -rf /`

		return PasteInputEvent{
			Value: sampleInput,
			Error: nil,
		}
	}
}

// PasteInputEvent is returned when paste input completes
type PasteInputEvent struct {
	Value string
	Error error
}

func (e PasteInputEvent) Timestamp() time.Time {
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
	app := &PasteComparisonApp{
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
