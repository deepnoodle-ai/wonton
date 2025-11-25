package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// ShiftEnterApp demonstrates Shift+Enter for newlines and Enter to submit using Runtime.
type ShiftEnterApp struct {
	stage  int
	buffer []rune
	cursor int
	result string
}

func (app *ShiftEnterApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		switch app.stage {
		case 0:
			// Instructions screen - any key starts
			app.stage = 1
			app.buffer = []rune{}
			app.cursor = 0

		case 1:
			// Input mode - handle keys
			return app.handleInputKey(e)

		case 2:
			// Result screen - any key quits
			return []gooey.Cmd{gooey.Quit()}
		}
	}

	return nil
}

func (app *ShiftEnterApp) handleInputKey(e gooey.KeyEvent) []gooey.Cmd {
	switch e.Key {
	case gooey.KeyEnter:
		if e.Shift {
			// Shift+Enter: Add newline
			app.buffer = insertRune(app.buffer, app.cursor, '\n')
			app.cursor++
		} else {
			// Regular Enter: Submit input
			app.result = string(app.buffer)
			app.stage = 2
		}

	case gooey.KeyBackspace:
		if app.cursor > 0 && len(app.buffer) > 0 {
			app.buffer = deleteRune(app.buffer, app.cursor-1)
			app.cursor--
		}

	case gooey.KeyDelete:
		if app.cursor < len(app.buffer) {
			app.buffer = deleteRune(app.buffer, app.cursor)
		}

	case gooey.KeyArrowLeft:
		if app.cursor > 0 {
			app.cursor--
		}

	case gooey.KeyArrowRight:
		if app.cursor < len(app.buffer) {
			app.cursor++
		}

	case gooey.KeyHome:
		app.cursor = 0

	case gooey.KeyEnd:
		app.cursor = len(app.buffer)

	case gooey.KeyEscape, gooey.KeyCtrlC:
		return []gooey.Cmd{gooey.Quit()}

	default:
		// Regular character input
		if e.Rune != 0 {
			app.buffer = insertRune(app.buffer, app.cursor, e.Rune)
			app.cursor++
		}
	}

	return nil
}

func (app *ShiftEnterApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()

	y := 0

	switch app.stage {
	case 0:
		// Instructions screen
		frame.PrintStyled(0, y, "Shift+Enter Demo", titleStyle)
		frame.PrintStyled(0, y+1, "================", titleStyle)
		y += 3

		frame.PrintStyled(0, y, "Instructions:", gooey.NewStyle().WithBold())
		y++
		frame.PrintStyled(0, y, "  - Press Shift+Enter to add a newline", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "  - Press Enter to submit", gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, "  - Press Ctrl+C or Esc to cancel", gooey.NewStyle())
		y += 2

		frame.PrintStyled(0, y, "Press any key to start...", gooey.NewStyle().WithForeground(gooey.ColorYellow))

	case 1:
		// Input screen
		frame.PrintStyled(0, y, "Shift+Enter Demo", titleStyle)
		frame.PrintStyled(0, y+1, "================", titleStyle)
		y += 3

		// Show the input prompt and buffer
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
		frame.PrintStyled(0, y, "Enter text: ", promptStyle)
		y++

		// Display the buffer with proper line wrapping
		text := string(app.buffer)
		lines := splitLines(text)

		for i, line := range lines {
			indent := ""
			if i > 0 {
				indent = "            " // 12 spaces to align with prompt
			}
			frame.PrintStyled(0, y+i, indent+line, gooey.NewStyle())
		}

		// Show cursor position info
		y += len(lines) + 2
		infoStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
		frame.PrintStyled(0, y, fmt.Sprintf("Cursor: %d | Buffer length: %d", app.cursor, len(app.buffer)), infoStyle)
		y++
		frame.PrintStyled(0, y, "Shift+Enter for newline, Enter to submit", infoStyle)

	case 2:
		// Result screen
		frame.PrintStyled(0, y, "Shift+Enter Demo - Result", titleStyle)
		frame.PrintStyled(0, y+1, "=========================", titleStyle)
		y += 3

		frame.PrintStyled(0, y, "You entered:", gooey.NewStyle().WithBold())
		y++
		frame.PrintStyled(0, y, "------------", gooey.NewStyle())
		y++

		successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
		lines := strings.Split(app.result, "\n")
		for _, line := range lines {
			frame.PrintStyled(0, y, line, successStyle)
			y++
		}

		y++
		frame.PrintStyled(0, y, "------------", gooey.NewStyle())
		y++

		// Statistics
		frame.PrintStyled(0, y, fmt.Sprintf("Total characters: %d", len(app.result)), gooey.NewStyle())
		y++
		frame.PrintStyled(0, y, fmt.Sprintf("Total lines: %d", len(lines)), gooey.NewStyle())
		y += 2

		frame.PrintStyled(0, y, "Press any key to exit...", gooey.NewStyle().WithForeground(gooey.ColorYellow))
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
	// Create and initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create the application
	app := &ShiftEnterApp{
		stage:  0,
		buffer: []rune{},
		cursor: 0,
	}

	// Create and run the runtime
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
