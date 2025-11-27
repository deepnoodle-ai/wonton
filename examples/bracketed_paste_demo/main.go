package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// BracketedPasteDemoApp demonstrates bracketed paste mode using Runtime.
type BracketedPasteDemoApp struct {
	terminal *gooey.Terminal
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

func (app *BracketedPasteDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Check if this is a paste event
		if e.Paste != "" {
			return app.handlePaste(e.Paste)
		}

		// Handle regular key events
		return app.handleKey(e)
	}

	return nil
}

func (app *BracketedPasteDemoApp) handlePaste(content string) []gooey.Cmd {
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

func (app *BracketedPasteDemoApp) handleKey(e gooey.KeyEvent) []gooey.Cmd {
	switch e.Key {
	case gooey.KeyEnter:
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

	case gooey.KeyBackspace:
		if app.cursor > 0 && len(app.buffer) > 0 {
			app.buffer = deleteRune(app.buffer, app.cursor-1)
			app.cursor--
			app.message = ""
		}

	case gooey.KeyDelete:
		if app.cursor < len(app.buffer) {
			app.buffer = deleteRune(app.buffer, app.cursor)
			app.message = ""
		}

	case gooey.KeyArrowLeft:
		if app.cursor > 0 {
			app.cursor--
		}
		app.message = ""

	case gooey.KeyArrowRight:
		if app.cursor < len(app.buffer) {
			app.cursor++
		}
		app.message = ""

	case gooey.KeyArrowUp:
		// Move cursor to previous line
		app.moveCursorUp()
		app.message = ""

	case gooey.KeyArrowDown:
		// Move cursor to next line
		app.moveCursorDown()
		app.message = ""

	case gooey.KeyHome:
		app.cursor = 0
		app.message = ""

	case gooey.KeyEnd:
		app.cursor = len(app.buffer)
		app.message = ""

	case gooey.KeyCtrlU:
		// Clear input
		app.buffer = []rune{}
		app.cursor = 0
		app.message = "Cleared input"

	case gooey.KeyCtrlL:
		// Clear paste history
		app.pastes = nil
		app.message = "Cleared paste history"

	case gooey.KeyEscape, gooey.KeyCtrlC:
		return []gooey.Cmd{gooey.Quit()}

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

func (app *BracketedPasteDemoApp) View() gooey.View {
	// Prepare buffer content
	text := string(app.buffer)
	lines := splitLines(text)
	maxDisplayLines := 8
	displayLines := lines
	if len(lines) > maxDisplayLines {
		displayLines = lines[len(lines)-maxDisplayLines:]
	}

	// Build input box content
	inputBoxLines := make([]gooey.View, 0)
	inputBoxLines = append(inputBoxLines, gooey.Text("┌%s┐", strings.Repeat("─", 68)).Fg(gooey.ColorCyan))

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
		inputBoxLines = append(inputBoxLines, gooey.HStack(
			gooey.Text("│").Fg(gooey.ColorCyan),
			gooey.Text(" %s ", lineContent),
			gooey.Text("│").Fg(gooey.ColorCyan),
		))
	}

	inputBoxLines = append(inputBoxLines, gooey.Text("└%s┘", strings.Repeat("─", 68)).Fg(gooey.ColorCyan))

	// Build status and message section
	statusLine := fmt.Sprintf("Chars: %d | Lines: %d | Cursor: %d", len(app.buffer), len(lines), app.cursor)
	statusSection := []gooey.View{
		gooey.Text("%s", statusLine).Dim(),
	}
	if app.message != "" {
		statusSection = append(statusSection, gooey.Text("%s", app.message).Fg(gooey.ColorGreen))
	} else {
		statusSection = append(statusSection, gooey.Spacer())
	}

	// Build paste history section
	var pasteHistorySection gooey.View
	if len(app.pastes) > 0 {
		// Show last 5 pastes
		startIdx := 0
		if len(app.pastes) > 5 {
			startIdx = len(app.pastes) - 5
		}

		pasteItems := make([]gooey.View, 0)
		for i := startIdx; i < len(app.pastes); i++ {
			p := app.pastes[i]
			preview := p.Content
			if len(preview) > 40 {
				preview = preview[:37] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", "↵")
			preview = strings.ReplaceAll(preview, "\t", "→")
			info := fmt.Sprintf("  #%d: %d chars, %d lines: %q", i+1, p.CharCount, p.LineCount, preview)
			pasteItems = append(pasteItems, gooey.Text("%s", info).Dim())
		}

		if len(app.pastes) > 5 {
			pasteItems = append(pasteItems, gooey.Text("  ... and %d more pastes", len(app.pastes)-5).Dim())
		}

		pasteHistorySection = gooey.VStack(
			gooey.Text("Paste History:").Bold(),
			gooey.VStack(pasteItems...),
		)
	} else {
		pasteHistorySection = gooey.Text("No pastes yet. Try pasting something!").Dim()
	}

	return gooey.VStack(
		// Header
		gooey.Text("╔═══════════════════════════════════════════════════════════════╗").Fg(gooey.ColorCyan),
		gooey.Text("║          Bracketed Paste Mode Demo - Gooey Library            ║").Fg(gooey.ColorCyan).Bold(),
		gooey.Text("╚═══════════════════════════════════════════════════════════════╝").Fg(gooey.ColorCyan),
		gooey.Spacer(),

		// Instructions
		gooey.Text("Try pasting text (Cmd+V / Ctrl+V). Paste events are detected!").Fg(gooey.ColorYellow),
		gooey.Text("Type normally, use arrow keys to navigate, Enter for newlines.").Dim(),
		gooey.Text("Ctrl+U: clear input | Ctrl+L: clear history | Esc/Ctrl+C: quit").Dim(),
		gooey.Spacer(),

		// Input area
		gooey.Text("Input:").Bold(),
		gooey.VStack(inputBoxLines...),
		gooey.Spacer(),

		// Status and message
		gooey.VStack(statusSection...),
		gooey.Spacer(),

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
	// It cannot use the simplified gooey.Run() API until WithBracketedPaste option is added.
	terminal, err := gooey.NewTerminal()
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
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Convert tabs to 2 spaces in pasted content
	runtime.SetPasteTabWidth(2)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		log.Fatalf("Runtime error: %v\n", err)
	}
}
