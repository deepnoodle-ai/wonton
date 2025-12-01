package main

import (
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// PastePlaceholderApp demonstrates the TextInput widget's paste placeholder mode.
// Multi-line pastes are shown as "[pasted N lines]" placeholders that can be
// deleted atomically with backspace/delete.
type PastePlaceholderApp struct {
	terminal *tui.Terminal
	input    *tui.TextInput
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
	app.input = tui.NewTextInput().
		WithPlaceholder("type or paste here...").
		WithPastePlaceholderMode(true).
		WithMultilineMode(true)
	app.input.SetFocused(true)
	app.message = "Paste multi-line text to see the placeholder!"
	app.submitted = false
	app.result = ""
}

func (app *PastePlaceholderApp) Destroy() {
	app.terminal.DisableBracketedPaste()
}

func (app *PastePlaceholderApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
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
		case tui.KeyEscape, tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case tui.KeyEnter:
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
		case tui.KeyCtrlU:
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

func (app *PastePlaceholderApp) View() tui.View {
	if app.submitted {
		return app.resultView()
	}
	return app.inputView()
}

func (app *PastePlaceholderApp) inputView() tui.View {
	// Stats
	displayLen := len(app.input.DisplayText())
	actualLen := len(app.input.Value())
	statsText := fmt.Sprintf("Display: %d chars | Actual: %d chars", displayLen, actualLen)

	var children []tui.View

	// Title
	children = append(children,
		tui.Text("Paste Placeholder Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Multi-line pastes show as '[pasted N lines]' - deleted atomically").Fg(tui.ColorBrightBlack),
		tui.Text("Enter: submit | Shift+Enter: newline | ESC: quit | Ctrl+U: clear").Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),
	)

	// Input label and widget
	children = append(children,
		tui.Text("Input:").Bold(),
		tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
			app.input.SetBounds(bounds)
			app.input.Draw(frame)
		}).Height(5).Width(50),
		tui.Spacer().MinHeight(1),
	)

	// Status message
	if app.message != "" {
		children = append(children, tui.Text("%s", app.message).Fg(tui.ColorGreen))
		children = append(children, tui.Spacer().MinHeight(1))
	}

	// Stats
	children = append(children, tui.Text("%s", statsText).Fg(tui.ColorYellow))

	return tui.VStack(children...)
}

func (app *PastePlaceholderApp) resultView() tui.View {
	lines := strings.Split(app.result, "\n")
	maxLines := 15

	var children []tui.View

	// Title
	children = append(children,
		tui.Text("Submitted Content").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
	)

	// Show result with line numbers using Canvas for custom formatting
	children = append(children,
		tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
			lineNumStyle := tui.NewStyle().WithForeground(tui.ColorBrightBlack)
			contentStyle := tui.NewStyle().WithForeground(tui.ColorWhite)
			hintStyle := tui.NewStyle().WithForeground(tui.ColorBrightBlack)
			width := bounds.Dx()

			y := 0
			for i, line := range lines {
				if i >= maxLines {
					frame.PrintStyled(0, y, fmt.Sprintf("... (%d more lines)", len(lines)-maxLines), hintStyle)
					break
				}
				if y >= bounds.Dy() {
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
		}).Height(maxLines+1).Width(80),
		tui.Spacer().MinHeight(1),
	)

	// Stats
	statsText := fmt.Sprintf("Total: %d chars, %d lines", len(app.result), len(lines))
	children = append(children,
		tui.Text("%s", statsText).Fg(tui.ColorYellow),
		tui.Spacer().MinHeight(1),
		tui.Text("Press any key to try again...").Fg(tui.ColorBrightBlack),
	)

	return tui.VStack(children...)
}

func main() {
	// Note: This example uses the old pattern (NewTerminal + NewRuntime) instead of
	// tui.Run() because it needs to call terminal.EnableBracketedPaste() in Init(),
	// which requires access to the terminal instance. There is currently no
	// tui.WithBracketedPaste() option.

	terminal, err := tui.NewTerminal()
	if err != nil {
		log.Fatalf("Failed to create terminal: %v\n", err)
	}
	defer terminal.Close()

	app := &PastePlaceholderApp{
		terminal: terminal,
	}

	runtime := tui.NewRuntime(terminal, app, 30)
	if err := runtime.Run(); err != nil {
		log.Fatalf("Runtime error: %v\n", err)
	}
}
