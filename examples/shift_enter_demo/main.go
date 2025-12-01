package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey/tui"
)

type ShiftEnterApp struct {
	lastKey      string
	shiftEntered bool
}

func (app *ShiftEnterApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		app.lastKey = fmt.Sprintf("Key=%d Rune=%q Shift=%v Ctrl=%v Alt=%v",
			e.Key, e.Rune, e.Shift, e.Ctrl, e.Alt)

		if e.Key == tui.KeyEnter && e.Shift {
			app.shiftEntered = true
		}

		if e.Key == tui.KeyCtrlC || e.Key == tui.KeyEscape {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func (app *ShiftEnterApp) View() tui.View {
	var statusText string
	var statusStyle tui.Style
	if app.shiftEntered {
		statusText = "Shift+Enter detected!"
		statusStyle = tui.NewStyle().WithForeground(tui.ColorGreen).WithBold()
	} else {
		statusText = "Waiting..."
		statusStyle = tui.NewStyle().WithForeground(tui.ColorBrightBlack)
	}

	var lastKeyView tui.View = tui.Empty()
	if app.lastKey != "" {
		lastKeyView = tui.Text("Last key: %s", app.lastKey).Dim()
	}

	return tui.VStack(
		tui.Text("Shift+Enter Detection Demo").Fg(tui.ColorCyan).Bold(),
		tui.Spacer(),
		tui.Text("Press Shift+Enter to test detection"),
		tui.Spacer(),
		tui.Text("%s", statusText).Style(statusStyle),
		tui.Spacer(),
		lastKeyView,
		tui.Spacer(),
		tui.Text("Press Ctrl+C or Esc to quit").Dim(),
	)
}

func main() {
	if err := tui.Run(&ShiftEnterApp{}); err != nil {
		log.Fatal(err)
	}
}
