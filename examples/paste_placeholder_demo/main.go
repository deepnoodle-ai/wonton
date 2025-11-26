package main

import (
	"fmt"
	"image"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// PastePlaceholderApp demonstrates the TextInput widget's paste placeholder mode.
// Multi-line pastes are shown as "[pasted N lines]" placeholders that can be
// deleted atomically with backspace/delete.
type PastePlaceholderApp struct {
	terminal *gooey.Terminal
	input    *gooey.TextInput
	message  string

	// Submitted result
	submitted bool
	result    string
}

func (app *PastePlaceholderApp) Init() error {
	app.terminal.EnableBracketedPaste()
	app.reset()
	return nil
}

func (app *PastePlaceholderApp) reset() {
	app.input = gooey.NewTextInput().
		WithPlaceholder("type or paste here...").
		WithPastePlaceholderMode(true).
		WithMultilineMode(true)
	app.input.SetBounds(image.Rect(0, 0, 50, 1))
	app.input.SetFocused(true)
	app.message = "Paste multi-line text to see the placeholder!"
	app.submitted = false
	app.result = ""
}

func (app *PastePlaceholderApp) Destroy() {
	app.terminal.DisableBracketedPaste()
}

func (app *PastePlaceholderApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if app.submitted {
			// Any key after submit resets for another try
			app.reset()
			return nil
		}

		// Check for paste
		if e.Paste != "" {
			app.input.HandlePaste(e.Paste)
			lines := strings.Split(e.Paste, "\n")
			if len(lines) > 1 {
				app.message = fmt.Sprintf("Pasted %d lines - backspace deletes entire paste", len(lines))
			} else {
				app.message = fmt.Sprintf("Pasted %d chars", len(e.Paste))
			}
			return nil
		}

		switch e.Key {
		case gooey.KeyEscape, gooey.KeyCtrlC:
			return []gooey.Cmd{gooey.Quit()}
		case gooey.KeyEnter:
			if e.Shift {
				// Shift+Enter: let TextInput handle it (inserts newline)
				app.input.HandleKey(e)
				app.message = ""
			} else {
				// Enter: submit
				app.result = app.input.Value()
				app.submitted = true
			}
			return nil
		case gooey.KeyCtrlU:
			app.input.Clear()
			app.message = "Cleared"
			return nil
		default:
			app.input.HandleKey(e)
			app.message = ""
		}
	}
	return nil
}

func (app *PastePlaceholderApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	hintStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

	if app.submitted {
		app.renderResult(frame, width, height, titleStyle, hintStyle)
		return
	}

	app.renderInput(frame, width, titleStyle, hintStyle)
}

func (app *PastePlaceholderApp) renderInput(frame gooey.RenderFrame, width int, titleStyle, hintStyle gooey.Style) {
	successStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)

	y := 0

	// Title
	frame.PrintStyled(0, y, "Paste Placeholder Demo", titleStyle)
	y++
	frame.PrintStyled(0, y, "Multi-line pastes show as '[pasted N lines]' - deleted atomically", hintStyle)
	y++
	frame.PrintStyled(0, y, "Enter: submit | Shift+Enter: newline | ESC: quit | Ctrl+U: clear", hintStyle)
	y += 2

	// Input label
	frame.PrintStyled(0, y, "Input:", gooey.NewStyle().WithBold())
	y++

	// Draw input widget
	inputWidth := width - 4
	if inputWidth > 50 {
		inputWidth = 50
	}
	inputHeight := 5 // Allow multiple lines
	app.input.SetBounds(image.Rect(2, y, 2+inputWidth, y+inputHeight))
	app.input.Draw(frame)
	y += inputHeight + 1

	// Status message
	if app.message != "" {
		frame.PrintStyled(0, y, app.message, successStyle)
		y++
	}

	// Stats
	y++
	statsStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	displayLen := len(app.input.DisplayText())
	actualLen := len(app.input.Value())
	frame.PrintStyled(0, y, fmt.Sprintf("Display: %d chars | Actual: %d chars", displayLen, actualLen), statsStyle)
}

func (app *PastePlaceholderApp) renderResult(frame gooey.RenderFrame, width, height int, titleStyle, hintStyle gooey.Style) {
	y := 0

	frame.PrintStyled(0, y, "Submitted Content", titleStyle)
	y += 2

	// Show result with line numbers
	lines := strings.Split(app.result, "\n")
	lineNumStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	contentStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)

	maxLines := height - 8
	if maxLines > 15 {
		maxLines = 15
	}

	for i, line := range lines {
		if i >= maxLines {
			frame.PrintStyled(0, y, fmt.Sprintf("... (%d more lines)", len(lines)-maxLines), hintStyle)
			y++
			break
		}
		frame.PrintStyled(0, y, fmt.Sprintf("%3d │ ", i+1), lineNumStyle)
		displayLine := line
		if len(displayLine) > width-8 {
			displayLine = displayLine[:width-11] + "..."
		}
		frame.PrintStyled(6, y, displayLine, contentStyle)
		y++
	}

	y++
	statsStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(0, y, fmt.Sprintf("Total: %d chars, %d lines", len(app.result), len(lines)), statsStyle)
	y += 2
	frame.PrintStyled(0, y, "Press any key to try again...", hintStyle)
}

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	app := &PastePlaceholderApp{
		terminal: terminal,
	}

	runtime := gooey.NewRuntime(terminal, app, 30)
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
