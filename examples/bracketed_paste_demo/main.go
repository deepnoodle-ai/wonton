package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// BracketedPasteDemoApp demonstrates bracketed paste mode using Runtime.
type BracketedPasteDemoApp struct {
	terminal *tui.Terminal
	buffer   []rune
	cursor   int
	pastes   []PasteRecord // History of pastes
	message  string        // Status message
}

// PasteRecord stores information about a paste event
type PasteRecord struct {
	Content   string
	LineCount int
	CharCount int
}

func (app *BracketedPasteDemoApp) Init() error {
	// Enable bracketed paste mode
	app.terminal.EnableBracketedPaste()
	return nil
}

func (app *BracketedPasteDemoApp) Destroy() {
	app.terminal.DisableBracketedPaste()
}

func (app *BracketedPasteDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// Check if this is a paste event
		if e.Paste != "" {
			return app.handlePaste(e.Paste)
		}

		// Handle regular key events
		return app.handleKey(e)
	}

	return nil
}

func (app *BracketedPasteDemoApp) handlePaste(content string) []tui.Cmd {
	// Line endings are already normalized by the library (\r\n and \r -> \n)

	// Record the paste
	lines := strings.Split(content, "\n")
	record := PasteRecord{
		Content:   content,
		LineCount: len(lines),
		CharCount: len(content),
	}
	app.pastes = append(app.pastes, record)

	// Insert pasted content into buffer
	for _, r := range content {
		app.buffer = insertRune(app.buffer, app.cursor, r)
		app.cursor++
	}

	app.message = fmt.Sprintf("Pasted %d chars (%d lines)", record.CharCount, record.LineCount)
	return nil
}

func (app *BracketedPasteDemoApp) handleKey(e tui.KeyEvent) []tui.Cmd {
	switch e.Key {
	case tui.KeyEnter:
		if e.Shift {
			// Shift+Enter: Add newline
			app.buffer = insertRune(app.buffer, app.cursor, '\n')
			app.cursor++
			app.message = "Added newline (Shift+Enter)"
		} else {
			// Regular Enter: Also add newline (for multiline support)
			app.buffer = insertRune(app.buffer, app.cursor, '\n')
			app.cursor++
			app.message = "Added newline"
		}

	case tui.KeyBackspace:
		if app.cursor > 0 && len(app.buffer) > 0 {
			app.buffer = deleteRune(app.buffer, app.cursor-1)
			app.cursor--
			app.message = ""
		}

	case tui.KeyDelete:
		if app.cursor < len(app.buffer) {
			app.buffer = deleteRune(app.buffer, app.cursor)
			app.message = ""
		}

	case tui.KeyArrowLeft:
		if app.cursor > 0 {
			app.cursor--
		}
		app.message = ""

	case tui.KeyArrowRight:
		if app.cursor < len(app.buffer) {
			app.cursor++
		}
		app.message = ""

	case tui.KeyArrowUp:
		// Move cursor to previous line
		app.moveCursorUp()
		app.message = ""

	case tui.KeyArrowDown:
		// Move cursor to next line
		app.moveCursorDown()
		app.message = ""

	case tui.KeyHome:
		app.cursor = 0
		app.message = ""

	case tui.KeyEnd:
		app.cursor = len(app.buffer)
		app.message = ""

	case tui.KeyCtrlU:
		// Clear input
		app.buffer = []rune{}
		app.cursor = 0
		app.message = "Cleared input"

	case tui.KeyCtrlL:
		// Clear paste history
		app.pastes = nil
		app.message = "Cleared paste history"

	case tui.KeyEscape, tui.KeyCtrlC:
		return []tui.Cmd{tui.Quit()}

	default:
		// Regular character input
		if e.Rune != 0 {
			app.buffer = insertRune(app.buffer, app.cursor, e.Rune)
			app.cursor++
			app.message = ""
		}
	}

	return nil
}

func (app *BracketedPasteDemoApp) moveCursorUp() {
	text := string(app.buffer[:app.cursor])
	lastNewline := strings.LastIndex(text, "\n")
	if lastNewline == -1 {
		return // Already on first line
	}
	colPos := app.cursor - lastNewline - 1
	prevText := text[:lastNewline]
	prevNewline := strings.LastIndex(prevText, "\n")
	lineStart := prevNewline + 1
	lineLen := lastNewline - lineStart
	newCol := colPos
	if newCol > lineLen {
		newCol = lineLen
	}
	app.cursor = lineStart + newCol
}

func (app *BracketedPasteDemoApp) moveCursorDown() {
	text := string(app.buffer)
	if app.cursor >= len(app.buffer) {
		return
	}
	// Find current line start and column
	textBefore := text[:app.cursor]
	lastNewline := strings.LastIndex(textBefore, "\n")
	colPos := app.cursor - lastNewline - 1

	// Find next line start
	textAfter := text[app.cursor:]
	nextNewline := strings.Index(textAfter, "\n")
	if nextNewline == -1 {
		return // Already on last line
	}
	nextLineStart := app.cursor + nextNewline + 1
	if nextLineStart >= len(app.buffer) {
		app.cursor = len(app.buffer)
		return
	}

	// Find next line length
	remainingText := text[nextLineStart:]
	nextNextNewline := strings.Index(remainingText, "\n")
	var nextLineLen int
	if nextNextNewline == -1 {
		nextLineLen = len(remainingText)
	} else {
		nextLineLen = nextNextNewline
	}

	newCol := colPos
	if newCol > nextLineLen {
		newCol = nextLineLen
	}
	app.cursor = nextLineStart + newCol
}

func (app *BracketedPasteDemoApp) View() tui.View {
	// Prepare buffer content
	text := string(app.buffer)
	lines := splitLines(text)
	maxDisplayLines := 8
	displayLines := lines
	if len(lines) > maxDisplayLines {
		displayLines = lines[len(lines)-maxDisplayLines:]
	}

	// Build input box content
	inputBoxLines := make([]tui.View, 0)
	inputBoxLines = append(inputBoxLines, tui.Text("┌%s┐", strings.Repeat("─", 68)).Fg(tui.ColorCyan))

	for i := 0; i < maxDisplayLines; i++ {
		var lineContent string
		if i < len(displayLines) {
			lineContent = displayLines[i]
			if len(lineContent) > 64 {
				lineContent = lineContent[:61] + "..."
			}
		}
		// Pad to consistent width
		lineContent = fmt.Sprintf("%-64s", lineContent)
		inputBoxLines = append(inputBoxLines, tui.HStack(
			tui.Text("│").Fg(tui.ColorCyan),
			tui.Text(" %s ", lineContent),
			tui.Text("│").Fg(tui.ColorCyan),
		))
	}

	inputBoxLines = append(inputBoxLines, tui.Text("└%s┘", strings.Repeat("─", 68)).Fg(tui.ColorCyan))

	// Build status and message section
	statusLine := fmt.Sprintf("Chars: %d | Lines: %d | Cursor: %d", len(app.buffer), len(lines), app.cursor)
	statusSection := []tui.View{
		tui.Text("%s", statusLine).Dim(),
	}
	if app.message != "" {
		statusSection = append(statusSection, tui.Text("%s", app.message).Fg(tui.ColorGreen))
	} else {
		statusSection = append(statusSection, tui.Spacer())
	}

	// Build paste history section
	var pasteHistorySection tui.View
	if len(app.pastes) > 0 {
		// Show last 5 pastes
		startIdx := 0
		if len(app.pastes) > 5 {
			startIdx = len(app.pastes) - 5
		}

		pasteItems := make([]tui.View, 0)
		for i := startIdx; i < len(app.pastes); i++ {
			p := app.pastes[i]
			preview := p.Content
			if len(preview) > 40 {
				preview = preview[:37] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", "↵")
			preview = strings.ReplaceAll(preview, "\t", "→")
			info := fmt.Sprintf("  #%d: %d chars, %d lines: %q", i+1, p.CharCount, p.LineCount, preview)
			pasteItems = append(pasteItems, tui.Text("%s", info).Dim())
		}

		if len(app.pastes) > 5 {
			pasteItems = append(pasteItems, tui.Text("  ... and %d more pastes", len(app.pastes)-5).Dim())
		}

		pasteHistorySection = tui.VStack(
			tui.Text("Paste History:").Bold(),
			tui.VStack(pasteItems...),
		)
	} else {
		pasteHistorySection = tui.Text("No pastes yet. Try pasting something!").Dim()
	}

	return tui.VStack(
		// Header
		tui.Text("╔═══════════════════════════════════════════════════════════════╗").Fg(tui.ColorCyan),
		tui.Text("║          Bracketed Paste Mode Demo - Wonton Library            ║").Fg(tui.ColorCyan).Bold(),
		tui.Text("╚═══════════════════════════════════════════════════════════════╝").Fg(tui.ColorCyan),
		tui.Spacer(),

		// Instructions
		tui.Text("Try pasting text (Cmd+V / Ctrl+V). Paste events are detected!").Fg(tui.ColorYellow),
		tui.Text("Type normally, use arrow keys to navigate, Enter for newlines.").Dim(),
		tui.Text("Ctrl+U: clear input | Ctrl+L: clear history | Esc/Ctrl+C: quit").Dim(),
		tui.Spacer(),

		// Input area
		tui.Text("Input:").Bold(),
		tui.VStack(inputBoxLines...),
		tui.Spacer(),

		// Status and message
		tui.VStack(statusSection...),
		tui.Spacer(),

		// Paste history
		pasteHistorySection,
	)
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
	if text == "" {
		return []string{""}
	}
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

func main() {
	// Note: This example uses bracketed paste mode which requires direct terminal access.
	// It cannot use the simplified tui.Run() API until WithBracketedPaste option is added.
	terminal, err := tui.NewTerminal()
	if err != nil {
		log.Fatalf("Failed to create terminal: %v\n", err)
	}
	defer terminal.Close()

	// Create the application
	app := &BracketedPasteDemoApp{
		terminal: terminal,
		buffer:   []rune{},
		cursor:   0,
		pastes:   nil,
		message:  "Ready! Paste something to see bracketed paste in action.",
	}

	// Create and run the runtime
	runtime := tui.NewRuntime(terminal, app, 30)

	// Convert tabs to 2 spaces in pasted content
	runtime.SetPasteTabWidth(2)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		log.Fatalf("Runtime error: %v\n", err)
	}
}
