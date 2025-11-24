package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		log.Fatal(err)
	}
	defer terminal.Close()

	// Enable alternate screen and raw mode
	terminal.EnableAlternateScreen()
	terminal.EnableRawMode()
	terminal.HideCursor()
	defer terminal.ShowCursor()

	// Enable bracketed paste mode
	terminal.EnableBracketedPaste()
	defer terminal.DisableBracketedPaste()

	// Clear screen
	terminal.Clear()

	// Create title
	titleStyle := gooey.NewStyle().
		WithForeground(gooey.ColorCyan).
		WithBold()

	frame, _ := terminal.BeginFrame()
	frame.PrintStyled(0, 0, "╔═══════════════════════════════════════════════════════════════╗", titleStyle)
	frame.PrintStyled(0, 1, "║          Bracketed Paste Mode Demo - Gooey Library          ║", titleStyle)
	frame.PrintStyled(0, 2, "╚═══════════════════════════════════════════════════════════════╝", titleStyle)

	instructionStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(0, 4, "This demo shows how bracketed paste mode handles pasted content.", instructionStyle)
	frame.PrintStyled(0, 5, "Try pasting multi-line text, code, or commands with newlines.", instructionStyle)
	frame.PrintStyled(0, 6, "", instructionStyle)
	frame.PrintStyled(0, 7, "Features:", gooey.NewStyle().WithBold().WithForeground(gooey.ColorGreen))
	frame.PrintStyled(0, 8, "  • Pasted text is treated as a single atomic operation", instructionStyle)
	frame.PrintStyled(0, 9, "  • Newlines in pasted content won't execute immediately", instructionStyle)
	frame.PrintStyled(0, 10, "  • Prevents security issues from malicious paste attacks", instructionStyle)
	frame.PrintStyled(0, 11, "  • ANSI escape codes in pastes are automatically sanitized", instructionStyle)
	frame.PrintStyled(0, 12, "", instructionStyle)
	frame.PrintStyled(0, 13, "Press ESC or Ctrl+C to exit", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	terminal.EndFrame(frame)

	// Create input handler
	input := gooey.NewInput(terminal)
	input.WithPrompt("Enter text (type or paste): ", gooey.NewStyle().WithForeground(gooey.ColorGreen))
	input.EnableMultiline()

	// Position input area
	terminal.MoveCursor(0, 15)

	// Read input
	result, err := input.Read()
	if err != nil {
		// User pressed ESC
		return
	}

	// Show result
	terminal.Clear()
	frame, _ = terminal.BeginFrame()

	resultStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	frame.PrintStyled(0, 0, "╔═══════════════════════════════════════════════════════════════╗", resultStyle)
	frame.PrintStyled(0, 1, "║                        You Entered:                          ║", resultStyle)
	frame.PrintStyled(0, 2, "╚═══════════════════════════════════════════════════════════════╝", resultStyle)
	frame.PrintStyled(0, 3, "", resultStyle)

	// Display the result with line numbers
	lines := strings.Split(result, "\n")
	contentStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	lineNumStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

	for i, line := range lines {
		lineNum := fmt.Sprintf("%3d │ ", i+1)
		frame.PrintStyled(0, 4+i, lineNum, lineNumStyle)
		frame.PrintStyled(6, 4+i, line, contentStyle)
	}

	// Show statistics
	statsY := 4 + len(lines) + 2
	statsStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(0, statsY, fmt.Sprintf("Total characters: %d", len(result)), statsStyle)
	frame.PrintStyled(0, statsY+1, fmt.Sprintf("Total lines: %d", len(lines)), statsStyle)

	// Show special characters if any
	hasNewlines := strings.Contains(result, "\n")
	hasTabs := strings.Contains(result, "\t")
	if hasNewlines || hasTabs {
		frame.PrintStyled(0, statsY+2, "Special characters detected:", statsStyle)
		if hasNewlines {
			frame.PrintStyled(2, statsY+3, "• Newlines (\\n)", statsStyle)
		}
		if hasTabs {
			frame.PrintStyled(2, statsY+4, "• Tabs (\\t)", statsStyle)
		}
	}

	terminal.EndFrame(frame)

	// Wait for user to press any key
	pressKeyStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	frame2, _ := terminal.BeginFrame()
	frame2.PrintStyled(0, statsY+6, "Press any key to continue...", pressKeyStyle)
	terminal.EndFrame(frame2)

	input.ReadKeyEvent()
}
