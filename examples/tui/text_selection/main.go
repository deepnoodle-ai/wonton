package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// TextSelectionApp demonstrates text selection and copy functionality.
//
// Features:
//   - Click and drag to select text (auto-copies to clipboard)
//   - Double-click to select a word
//   - Triple-click to select a line
//   - Cmd+V / Ctrl+V to paste (handled by terminal)
//   - Escape to clear selection
type TextSelectionApp struct {
	content   string
	selection tui.TextSelection
	status    string
}

func (app *TextSelectionApp) Init() error {
	app.content = `The quick brown fox jumps over the lazy dog.

This is a demonstration of text selection in Wonton TUI.
You can select text by clicking and dragging with your mouse.

Features:
  - Click and drag to select text
  - Double-click to select a word
  - Triple-click to select an entire line
  - Press Ctrl+C to copy the selection
  - Press Escape to clear the selection

Try selecting some of this text!

func example() {
    fmt.Println("Code can be selected too!")
    for i := 0; i < 10; i++ {
        fmt.Printf("Line %d\n", i)
    }
}

The selection works across multiple lines and handles
scrolling when the content exceeds the visible area.`

	app.status = "Click and drag to select text"
	return nil
}

func (app *TextSelectionApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyCtrlC, tui.KeyCtrlQ:
			return []tui.Cmd{tui.Quit()}
		case tui.KeyEscape:
			app.status = "Selection cleared"
		}

	case tui.MouseEvent:
		// Forward mouse events to the textarea
		if tui.TextAreaHandleMouseEvent("content", &e) {
			// Update status when selection changes
			if app.selection.Active && !app.selection.IsEmpty() {
				start, end := app.selection.Normalized()
				lines := end.Line - start.Line + 1
				app.status = fmt.Sprintf("Selected %d line(s) - copied to clipboard! (Cmd+V to paste)", lines)
			} else {
				app.status = "Click and drag to select text"
			}
		}
	}
	return nil
}

func (app *TextSelectionApp) View() tui.View {
	return tui.Stack(
		// Header
		tui.Text("Text Selection Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Ctrl+C/Q: quit | Escape: clear selection").Fg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),

		// Status line
		tui.Text("%s", app.status).Fg(tui.ColorYellow),
		tui.Spacer().MinHeight(1),

		// Main content area with selection enabled
		tui.TextArea(&app.content).
			ID("content").
			Title("Selectable Text").
			Bordered().
			FocusBorderFg(tui.ColorCyan).
			EnableSelection().
			Selection(&app.selection).
			Size(70, 20),
	)
}

func main() {
	err := tui.Run(&TextSelectionApp{}, tui.WithMouseTracking(true))
	if err != nil {
		log.Fatal(err)
	}
}
