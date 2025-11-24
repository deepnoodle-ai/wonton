package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		log.Fatal(err)
	}
	defer terminal.Close()

	// Enable raw mode for key detection
	terminal.EnableRawMode()
	defer terminal.DisableRawMode()

	// Clear screen and show instructions
	terminal.Clear()
	terminal.Println("Shift+Enter Demo")
	terminal.Println("================")
	terminal.Println("")
	terminal.Println("Instructions:")
	terminal.Println("  - Press Shift+Enter to add a newline")
	terminal.Println("  - Press Enter to submit")
	terminal.Println("  - Press Ctrl+C or Esc to cancel")
	terminal.Println("")
	terminal.Flush()

	// Create input handler
	input := gooey.NewInput(terminal)

	// Show cursor
	terminal.ShowCursor()

	// Read input with custom key handling
	result, err := readWithShiftEnter(input, terminal)
	if err != nil {
		terminal.Println(fmt.Sprintf("Error: %v", err))
		terminal.Flush()
		return
	}

	// Display the result
	terminal.Println("")
	terminal.Println("You entered:")
	terminal.Println("------------")
	terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen))
	terminal.Print(result)
	terminal.Reset()
	terminal.Println("")
	terminal.Println("------------")
	terminal.Flush()
}

// readWithShiftEnter implements custom input handling where:
// - Shift+Enter adds a newline
// - Regular Enter submits the input
func readWithShiftEnter(input *gooey.Input, terminal *gooey.Terminal) (string, error) {
	var buffer []rune
	cursorPos := 0
	previousLineCount := 1 // Track how many lines we drew last time

	// Draw initial prompt
	terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan))
	terminal.Print("Enter text: ")
	terminal.Reset()
	terminal.Flush()

	for {
		// Read key event
		event := input.ReadKeyEvent()

		// Handle different key combinations
		switch event.Key {
		case gooey.KeyEnter:
			if event.Shift {
				// Shift+Enter: Add newline
				buffer = insertRune(buffer, cursorPos, '\n')
				cursorPos++
			} else {
				// Regular Enter: Submit input
				terminal.Println("")
				terminal.Flush()
				return string(buffer), nil
			}

		case gooey.KeyBackspace:
			if cursorPos > 0 && len(buffer) > 0 {
				buffer = deleteRune(buffer, cursorPos-1)
				cursorPos--
			}

		case gooey.KeyDelete:
			if cursorPos < len(buffer) {
				buffer = deleteRune(buffer, cursorPos)
			}

		case gooey.KeyArrowLeft:
			if cursorPos > 0 {
				cursorPos--
			}

		case gooey.KeyArrowRight:
			if cursorPos < len(buffer) {
				cursorPos++
			}

		case gooey.KeyHome:
			cursorPos = 0

		case gooey.KeyEnd:
			cursorPos = len(buffer)

		case gooey.KeyEscape, gooey.KeyCtrlC:
			terminal.Println("")
			terminal.Flush()
			return "", fmt.Errorf("input cancelled")

		default:
			// Regular character input
			if event.Rune != 0 {
				buffer = insertRune(buffer, cursorPos, event.Rune)
				cursorPos++
			}
		}

		// Update display
		previousLineCount = updateDisplay(terminal, buffer, cursorPos, previousLineCount)
		terminal.Flush()
	}
}

// updateDisplay redraws the input line with the current buffer and cursor position
// Returns the number of lines drawn so we can clear them next time
func updateDisplay(terminal *gooey.Terminal, buffer []rune, cursorPos, previousLineCount int) int {
	text := string(buffer)
	lines := splitLines(text)

	// Calculate cursor position in the text
	runesBeforeCursor := buffer[:min(cursorPos, len(buffer))]
	cursorLine := countNewlines(string(runesBeforeCursor))

	// Find column position on current line
	lastNewlineIdx := -1
	for i := len(runesBeforeCursor) - 1; i >= 0; i-- {
		if runesBeforeCursor[i] == '\n' {
			lastNewlineIdx = i
			break
		}
	}

	var cursorCol int
	if lastNewlineIdx == -1 {
		cursorCol = len(runesBeforeCursor)
	} else {
		cursorCol = len(runesBeforeCursor) - lastNewlineIdx - 1
	}

	// Clear all previously drawn lines
	// Start by going to beginning of current line
	terminal.Print("\r")
	terminal.ClearToEndOfLine()

	// Clear any additional lines that were drawn before
	for i := 1; i < previousLineCount; i++ {
		terminal.Print("\n")
		terminal.ClearLine()
	}

	// Move back to the start
	for i := 1; i < previousLineCount; i++ {
		terminal.MoveCursorUp(1)
	}
	terminal.Print("\r")

	// Redraw prompt
	terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan))
	terminal.Print("Enter text: ")
	terminal.Reset()

	// Draw all lines
	for i, line := range lines {
		if i > 0 {
			terminal.Print("\n")
			terminal.Print("            ") // Indent continuation lines
		}
		terminal.Print(line)
	}

	// Position cursor at the correct location
	// Move back to first line
	for i := 0; i < len(lines)-1; i++ {
		terminal.MoveCursorUp(1)
	}
	terminal.Print("\r")

	// Move down to cursor line
	for i := 0; i < cursorLine; i++ {
		terminal.MoveCursorDown(1)
	}

	// Move to correct column
	terminal.MoveCursorRight(12 + cursorCol) // "Enter text: " or "            " = 12 chars

	// Return the number of lines we drew
	return len(lines)
}

// Helper functions
func insertRune(buffer []rune, pos int, r rune) []rune {
	if pos < 0 || pos > len(buffer) {
		return buffer
	}
	result := make([]rune, 0, len(buffer)+1)
	result = append(result, buffer[:pos]...)
	result = append(result, r)
	result = append(result, buffer[pos:]...)
	return result
}

func deleteRune(buffer []rune, pos int) []rune {
	if pos < 0 || pos >= len(buffer) {
		return buffer
	}
	result := make([]rune, 0, len(buffer)-1)
	result = append(result, buffer[:pos]...)
	result = append(result, buffer[pos+1:]...)
	return result
}

func splitLines(text string) []string {
	result := []string{""}
	for _, r := range text {
		if r == '\n' {
			result = append(result, "")
		} else {
			result[len(result)-1] += string(r)
		}
	}
	return result
}

func countNewlines(text string) int {
	count := 0
	for _, r := range text {
		if r == '\n' {
			count++
		}
	}
	return count
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
