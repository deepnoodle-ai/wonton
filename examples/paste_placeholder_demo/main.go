package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// PastePlaceholderApp demonstrates advanced paste handling with placeholders using Runtime.
type PastePlaceholderApp struct {
	stage          int
	input          string
	message        string
	pasteModified  bool
	originalLength int
}

func (app *PastePlaceholderApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if app.stage == 0 {
			// Instructions screen - any key continues
			if e.Key == gooey.KeyEscape || e.Rune == 'q' || e.Rune == 'Q' {
				return []gooey.Cmd{gooey.Quit()}
			}
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
		app.pasteModified = e.WasModified
		app.originalLength = e.OriginalLength
		app.stage = 2
	}

	return nil
}

func (app *PastePlaceholderApp) Render(frame gooey.RenderFrame) {
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
		frame.PrintStyled(0, y, "╔═════════════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║     Placeholder Paste Mode Demo - Advanced Paste Handling          ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚═════════════════════════════════════════════════════════════════════╝", titleStyle)
		y += 4

		frame.PrintStyled(0, y, "This demo shows advanced paste handling with placeholders and callbacks.", instructionStyle)
		y += 2

		frame.PrintStyled(0, y, "Features Demonstrated:", featureStyle)
		y++
		frame.PrintStyled(0, y, "  • Placeholder display: Shows '[pasted 27 lines]' instead of content", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • Paste handlers: Inspect, modify, or reject pasted content", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • ANSI stripping: Automatically clean malicious escape codes", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • Size limits: Reject pastes that are too large", instructionStyle)
		y += 2

		frame.PrintStyled(0, y, "Try pasting:", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • Multi-line text (will show '[pasted N lines]')", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • Text with ANSI codes (will be stripped automatically)", instructionStyle)
		y++
		frame.PrintStyled(0, y, "  • Very large content (over 5000 chars will be rejected)", instructionStyle)
		y += 2

		frame.PrintStyled(0, y, "Press ESC or Q to exit, any other key to continue", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 1:
		// Input screen
		frame.PrintStyled(0, y, "╔═════════════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║                          Paste Input                                ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚═════════════════════════════════════════════════════════════════════╝", titleStyle)
		y += 4

		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
		frame.PrintStyled(0, y, "Paste or type text: ", promptStyle)
		y++
		frame.PrintStyled(0, y, "(Simulated multiline paste in Runtime)", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	case 2:
		// Result screen
		frame.PrintStyled(0, y, "╔═════════════════════════════════════════════════════════════════════╗", titleStyle)
		frame.PrintStyled(0, y+1, "║                          Result                                     ║", titleStyle)
		frame.PrintStyled(0, y+2, "╚═════════════════════════════════════════════════════════════════════╝", titleStyle)
		y += 4

		// Display the result with line numbers
		lines := strings.Split(app.input, "\n")
		contentStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
		lineNumStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

		maxLinesToShow := 20
		displayLines := lines
		if len(lines) > maxLinesToShow {
			displayLines = lines[:maxLinesToShow]
		}

		for i, line := range displayLines {
			lineNum := fmt.Sprintf("%3d │ ", i+1)
			frame.PrintStyled(0, y+i, lineNum, lineNumStyle)

			// Truncate long lines for display
			displayLine := line
			if len(line) > 60 {
				displayLine = line[:60] + "..."
			}
			frame.PrintStyled(6, y+i, displayLine, contentStyle)
		}

		if len(lines) > maxLinesToShow {
			moreStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack).WithItalic()
			frame.PrintStyled(0, y+maxLinesToShow, fmt.Sprintf("... (%d more lines)", len(lines)-maxLinesToShow), moreStyle)
			y += maxLinesToShow + 1
		} else {
			y += len(displayLines)
		}

		// Show statistics
		y += 2
		statsStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
		frame.PrintStyled(0, y, "Statistics:", gooey.NewStyle().WithBold().WithForeground(gooey.ColorGreen))
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("  Total characters: %d", len(app.input)), statsStyle)
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("  Total lines: %d", len(lines)), statsStyle)
		y++

		if app.pasteModified {
			frame.PrintStyled(0, y, fmt.Sprintf("  Content was sanitized (original: %d chars)", app.originalLength), statsStyle)
			y++
		}

		// Show special characters if any
		hasNewlines := strings.Contains(app.input, "\n")
		hasTabs := strings.Contains(app.input, "\t")
		if hasNewlines || hasTabs {
			y++
			frame.PrintStyled(0, y, "  Special characters:", statsStyle)
			y++
			if hasNewlines {
				frame.PrintStyled(4, y, "• Newlines (\\n)", statsStyle)
				y++
			}
			if hasTabs {
				frame.PrintStyled(4, y, "• Tabs (\\t)", statsStyle)
				y++
			}
		}

		y += 2
		frame.PrintStyled(0, y, "Press any key to exit...", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	}
}

// Command for async paste input with validation (simulated)
func (app *PastePlaceholderApp) cmdReadPasteInput() gooey.Cmd {
	return func() gooey.Event {
		// Simulate multiline paste with potential ANSI codes
		sampleInput := `This is line 1
This is line 2
This is line 3
This is a multi-line paste example.
It demonstrates placeholder display mode.

With some extra content to show line numbering.
And more lines...
And even more lines...
To demonstrate the truncation feature.`

		// Simulate ANSI stripping (in real implementation, this would be done by paste handler)
		cleanedInput := sampleInput
		wasModified := false

		// Check size limit
		if len(cleanedInput) > 5000 {
			return PasteInputEvent{
				Value: "",
				Error: fmt.Errorf("paste too large: %d chars exceeds 5000 char limit", len(cleanedInput)),
			}
		}

		return PasteInputEvent{
			Value:          cleanedInput,
			WasModified:    wasModified,
			OriginalLength: len(sampleInput),
			Error:          nil,
		}
	}
}

// PasteInputEvent is returned when paste input completes
type PasteInputEvent struct {
	Value          string
	WasModified    bool
	OriginalLength int
	Error          error
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
	app := &PastePlaceholderApp{
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
