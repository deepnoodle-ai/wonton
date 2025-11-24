package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

func stripANSI(s string) string {
	// Remove ANSI escape codes
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

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
	frame.PrintStyled(0, 0, "╔═════════════════════════════════════════════════════════════════════╗", titleStyle)
	frame.PrintStyled(0, 1, "║     Placeholder Paste Mode Demo - Advanced Paste Handling          ║", titleStyle)
	frame.PrintStyled(0, 2, "╚═════════════════════════════════════════════════════════════════════╝", titleStyle)

	instructionStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(0, 4, "This demo shows advanced paste handling with placeholders and callbacks.", instructionStyle)
	frame.PrintStyled(0, 5, "", instructionStyle)
	frame.PrintStyled(0, 6, "Features Demonstrated:", gooey.NewStyle().WithBold().WithForeground(gooey.ColorGreen))
	frame.PrintStyled(0, 7, "  • Placeholder display: Shows '[pasted 27 lines]' instead of content", instructionStyle)
	frame.PrintStyled(0, 8, "  • Paste handlers: Inspect, modify, or reject pasted content", instructionStyle)
	frame.PrintStyled(0, 9, "  • ANSI stripping: Automatically clean malicious escape codes", instructionStyle)
	frame.PrintStyled(0, 10, "  • Size limits: Reject pastes that are too large", instructionStyle)
	frame.PrintStyled(0, 11, "", instructionStyle)
	frame.PrintStyled(0, 12, "Try pasting:", instructionStyle)
	frame.PrintStyled(0, 13, "  • Multi-line text (will show '[pasted N lines]')", instructionStyle)
	frame.PrintStyled(0, 14, "  • Text with ANSI codes (will be stripped automatically)", instructionStyle)
	frame.PrintStyled(0, 15, "  • Very large content (over 5000 chars will be rejected)", instructionStyle)
	frame.PrintStyled(0, 16, "", instructionStyle)
	frame.PrintStyled(0, 17, "Press ESC or Ctrl+C to exit", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	terminal.EndFrame(frame)

	// Create input handler with advanced paste handling
	input := gooey.NewInput(terminal)
	input.WithPrompt("Paste or type text: ", gooey.NewStyle().WithForeground(gooey.ColorGreen))
	input.EnableMultiline()

	// Configure placeholder display mode
	input.WithPasteDisplayMode(gooey.PasteDisplayPlaceholder)

	// Customize placeholder style
	placeholderStyle := gooey.NewStyle().
		WithForeground(gooey.ColorMagenta).
		WithItalic()
	input.WithPlaceholderStyle(placeholderStyle)

	// Add paste handler for validation and sanitization
	var pasteWasModified bool
	input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
		// Check size limit
		if info.ByteCount > 5000 {
			terminal.MoveCursor(0, 19)
			warningStyle := gooey.NewStyle().WithForeground(gooey.ColorRed).WithBold()
			terminal.PrintStyled(fmt.Sprintf("⚠ Paste rejected: %d chars exceeds 5000 char limit", info.ByteCount), warningStyle)
			terminal.Flush()
			return gooey.PasteReject, ""
		}

		// Strip ANSI codes
		cleaned := stripANSI(info.Content)
		if cleaned != info.Content {
			pasteWasModified = true
			terminal.MoveCursor(0, 19)
			infoStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
			terminal.PrintStyled("ℹ ANSI escape codes were stripped from paste", infoStyle)
			terminal.Flush()
			return gooey.PasteModified, cleaned
		}

		return gooey.PasteAccept, ""
	})

	// Position input area
	terminal.MoveCursor(0, 20)

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
	frame.PrintStyled(0, 0, "╔═════════════════════════════════════════════════════════════════════╗", resultStyle)
	frame.PrintStyled(0, 1, "║                          Result                                     ║", resultStyle)
	frame.PrintStyled(0, 2, "╚═════════════════════════════════════════════════════════════════════╝", resultStyle)
	frame.PrintStyled(0, 3, "", resultStyle)

	// Display the result with line numbers
	lines := strings.Split(result, "\n")
	contentStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	lineNumStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

	maxLinesToShow := 20
	for i, line := range lines {
		if i >= maxLinesToShow {
			moreStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack).WithItalic()
			frame.PrintStyled(0, 4+i, fmt.Sprintf("... (%d more lines)", len(lines)-maxLinesToShow), moreStyle)
			break
		}
		lineNum := fmt.Sprintf("%3d │ ", i+1)
		frame.PrintStyled(0, 4+i, lineNum, lineNumStyle)

		// Truncate long lines for display
		displayLine := line
		if len(line) > 60 {
			displayLine = line[:60] + "..."
		}
		frame.PrintStyled(6, 4+i, displayLine, contentStyle)
	}

	// Show statistics
	statsY := 4 + min(len(lines), maxLinesToShow) + 2
	statsStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(0, statsY, "Statistics:", gooey.NewStyle().WithBold().WithForeground(gooey.ColorGreen))
	frame.PrintStyled(0, statsY+1, fmt.Sprintf("  Total characters: %d", len(result)), statsStyle)
	frame.PrintStyled(0, statsY+2, fmt.Sprintf("  Total lines: %d", len(lines)), statsStyle)

	if pasteWasModified {
		frame.PrintStyled(0, statsY+3, "  Content was sanitized (ANSI codes removed)", statsStyle)
	}

	// Show special characters if any
	hasNewlines := strings.Contains(result, "\n")
	hasTabs := strings.Contains(result, "\t")
	if hasNewlines || hasTabs {
		frame.PrintStyled(0, statsY+4, "  Special characters:", statsStyle)
		if hasNewlines {
			frame.PrintStyled(4, statsY+5, "• Newlines (\\n)", statsStyle)
		}
		if hasTabs {
			frame.PrintStyled(4, statsY+6, "• Tabs (\\t)", statsStyle)
		}
	}

	terminal.EndFrame(frame)

	// Wait for user to press any key
	pressKeyStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	frame2, _ := terminal.BeginFrame()
	frame2.PrintStyled(0, statsY+8, "Press any key to exit...", pressKeyStyle)
	terminal.EndFrame(frame2)

	input.ReadKeyEvent()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
