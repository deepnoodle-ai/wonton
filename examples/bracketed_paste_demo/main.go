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

func (app *BracketedPasteDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	instructionStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	dimStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	borderStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)

	y := 0

	// Header
	frame.PrintStyled(0, y, "╔═══════════════════════════════════════════════════════════════╗", borderStyle)
	y++
	frame.PrintStyled(0, y, "║          Bracketed Paste Mode Demo - Gooey Library            ║", titleStyle)
	y++
	frame.PrintStyled(0, y, "╚═══════════════════════════════════════════════════════════════╝", borderStyle)
	y += 2

	// Instructions
	frame.PrintStyled(0, y, "Try pasting text (Cmd+V / Ctrl+V). Paste events are detected!", instructionStyle)
	y++
	frame.PrintStyled(0, y, "Type normally, use arrow keys to navigate, Enter for newlines.", dimStyle)
	y++
	frame.PrintStyled(0, y, "Ctrl+U: clear input | Ctrl+L: clear history | Esc/Ctrl+C: quit", dimStyle)
	y += 2

	// Input area
	frame.PrintStyled(0, y, "Input:", gooey.NewStyle().WithBold())
	y++

	// Draw input box border
	inputBoxWidth := width - 4
	if inputBoxWidth > 70 {
		inputBoxWidth = 70
	}
	frame.PrintStyled(2, y, "┌"+strings.Repeat("─", inputBoxWidth-2)+"┐", borderStyle)
	y++

	// Display the buffer content (limited height)
	text := string(app.buffer)
	lines := splitLines(text)
	maxDisplayLines := 8
	displayLines := lines
	if len(lines) > maxDisplayLines {
		displayLines = lines[len(lines)-maxDisplayLines:]
	}

	contentWidth := inputBoxWidth - 4 // Space between │ and │
	for i := 0; i < maxDisplayLines; i++ {
		frame.PrintStyled(2, y+i, "│", borderStyle)
		// Clear the line content area
		frame.PrintStyled(3, y+i, strings.Repeat(" ", contentWidth), gooey.NewStyle())
		// Draw content if we have a line for this row
		if i < len(displayLines) {
			displayLine := displayLines[i]
			if len(displayLine) > contentWidth {
				displayLine = displayLine[:contentWidth-3] + "..."
			}
			frame.PrintStyled(4, y+i, displayLine, gooey.NewStyle())
		}
		frame.PrintStyled(2+inputBoxWidth-1, y+i, "│", borderStyle)
	}
	y += maxDisplayLines

	frame.PrintStyled(2, y, "└"+strings.Repeat("─", inputBoxWidth-2)+"┘", borderStyle)
	y += 2

	// Status line
	statusLine := fmt.Sprintf("Chars: %d | Lines: %d | Cursor: %d", len(app.buffer), len(lines), app.cursor)
	frame.PrintStyled(0, y, statusLine, dimStyle)
	y++

	// Message
	if app.message != "" {
		frame.PrintStyled(0, y, app.message, successStyle)
	}
	y += 2

	// Paste history
	if len(app.pastes) > 0 {
		frame.PrintStyled(0, y, "Paste History:", gooey.NewStyle().WithBold())
		y++

		// Show last 5 pastes
		startIdx := 0
		if len(app.pastes) > 5 {
			startIdx = len(app.pastes) - 5
		}
		for i := startIdx; i < len(app.pastes); i++ {
			p := app.pastes[i]
			preview := p.Content
			if len(preview) > 40 {
				preview = preview[:37] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", "↵")
			preview = strings.ReplaceAll(preview, "\t", "→")
			info := fmt.Sprintf("  #%d: %d chars, %d lines: %q", i+1, p.CharCount, p.LineCount, preview)
			frame.PrintStyled(0, y, info, dimStyle)
			y++
		}
		if len(app.pastes) > 5 {
			frame.PrintStyled(0, y, fmt.Sprintf("  ... and %d more pastes", len(app.pastes)-5), dimStyle)
			y++
		}
	} else {
		frame.PrintStyled(0, y, "No pastes yet. Try pasting something!", dimStyle)
	}
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
