package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// BracketedPasteDemoApp demonstrates bracketed paste mode using Runtime.
type BracketedPasteDemoApp struct {
	stage   int
	input   string
	message string
}

func (app *BracketedPasteDemoApp) Init() error {
	// Enable bracketed paste mode
	// Note: In a real implementation, this would be handled by the terminal
	return nil
}

func (app *BracketedPasteDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if app.stage == 0 {
			// Instructions screen - any key continues to input
			if e.Key == gooey.KeyEscape || e.Rune == 'q' || e.Rune == 'Q' {
				return []gooey.Cmd{gooey.Quit()}
			}
			app.stage = 1
			return []gooey.Cmd{app.cmdReadMultilineInput()}
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

func (app *BracketedPasteDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	instructionStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	featureStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()

	y := 0

	switch app.stage {
	case 0:
		// Instructions screen
		frame.PrintStyled(0, y, "╔═══════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║          Bracketed Paste Mode Demo - Gooey Library          ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚═══════════════════════════════════════════════════════════════╗", titleStyle)
		y += 4

		frame.PrintStyled(0, y, "This demo shows how bracketed paste mode handles pasted content.", instructionStyle)
		y++
		frame.PrintStyled(0, y, "Try pasting multi-line text, code, or commands with newlines.", instructionStyle)
		y += 2

		frame.PrintStyled(0, y, "Features:", featureStyle)
		y++
		frame.PrintStyled(0, y, "  • Pasted text is treated as a single atomic operation", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • Newlines in pasted content won't execute immediately", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • Prevents security issues from malicious paste attacks", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • ANSI escape codes in pastes are automatically sanitized", instructionStyle)
		y += 2

		frame.PrintStyled(0, y, "Press ESC or Q to exit, any other key to continue", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 1:
		// Input screen
		frame.PrintStyled(0, y, "╔═══════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║                    Paste Input                                ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚═══════════════════════════════════════════════════════════════╗", titleStyle)
		y += 4

		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
		frame.PrintStyled(0, y, "Enter text (type or paste): ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated multiline input in Runtime)", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		y++
		frame.PrintStyled(0, y, "Press Ctrl+Enter or Ctrl+D to submit", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 2:
		// Result screen
		frame.PrintStyled(0, y, "╔═══════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║                        You Entered:                          ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚═══════════════════════════════════════════════════════════════╗", titleStyle)
		y += 4

		// Display the result with line numbers
		lines := strings.Split(app.input, "\n")
		contentStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
		lineNumStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

		maxLines := 15
		displayLines := lines
		if len(lines) > maxLines {
			displayLines = lines[:maxLines]
		}

		for i, line := range displayLines {
			lineNum := fmt.Sprintf("%3d │ ", i+1)
			frame.PrintStyled(0, y+i, lineNum, lineNumStyle)
			frame.PrintStyled(6, y+i, line, contentStyle)
		}

		if len(lines) > maxLines {
			frame.PrintStyled(0, y+maxLines, fmt.Sprintf("... (%d more lines)", len(lines)-maxLines), gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
			y += maxLines + 1
		} else {
			y += len(displayLines)
		}

		// Show statistics
		y += 2
		statsStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
		frame.PrintStyled(0, y, fmt.Sprintf("Total characters: %d", len(app.input)), statsStyle)
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("Total lines: %d", len(lines)), statsStyle)
		y++

		// Show special characters if any
		hasNewlines := strings.Contains(app.input, "\n")
		hasTabs := strings.Contains(app.input, "\t")
		if hasNewlines || hasTabs {
			y++
			frame.PrintStyled(0, y, "Special characters detected:", statsStyle)
			y++
			if hasNewlines {
				frame.PrintStyled(2, y, "• Newlines (\\n)", statsStyle)
				y++
			}
			if hasTabs {
				frame.PrintStyled(2, y, "• Tabs (\\t)", statsStyle)
				y++
			}
		}

		y += 2
		frame.PrintStyled(0, y, "Press any key to exit...", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	}
}

// Command for async multiline input (simulated)
func (app *BracketedPasteDemoApp) cmdReadMultilineInput() gooey.Cmd {
	return func() gooey.Event {
		// Simulate multiline pasted input
		sampleInput := `line 1
line 2
line 3
This is a multi-line paste example.
It demonstrates bracketed paste mode.`

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
	app := &BracketedPasteDemoApp{
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
