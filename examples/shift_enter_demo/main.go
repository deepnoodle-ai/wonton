package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

type ShiftEnterApp struct {
	lastKey      string
	shiftEntered bool
}

func (app *ShiftEnterApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		app.lastKey = fmt.Sprintf("Key=%d Rune=%q Shift=%v Ctrl=%v Alt=%v",
			e.Key, e.Rune, e.Shift, e.Ctrl, e.Alt)

		if e.Key == gooey.KeyEnter && e.Shift {
			app.shiftEntered = true
		}

		if e.Key == gooey.KeyCtrlC || e.Key == gooey.KeyEscape {
			return []gooey.Cmd{gooey.Quit()}
		}
	}
	return nil
}

func (app *ShiftEnterApp) View() gooey.View {
	var statusText string
	var statusStyle gooey.Style
	if app.shiftEntered {
		statusText = "Shift+Enter detected!"
		statusStyle = gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()
	} else {
		statusText = "Waiting..."
		statusStyle = gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	}

	var lastKeyView gooey.View = gooey.Empty()
	if app.lastKey != "" {
		lastKeyView = gooey.Text("Last key: %s", app.lastKey).Dim()
	}

	return gooey.VStack(
		gooey.Text("Shift+Enter Detection Demo").Fg(gooey.ColorCyan).Bold(),
		gooey.Spacer(),
		gooey.Text("Press Shift+Enter to test detection"),
		gooey.Spacer(),
		gooey.Text("%s", statusText).Style(statusStyle),
		gooey.Spacer(),
		lastKeyView,
		gooey.Spacer(),
		gooey.Text("Press Ctrl+C or Esc to quit").Dim(),
	)
}

func main() {
	if err := gooey.Run(&ShiftEnterApp{}); err != nil {
		log.Fatal(err)
	}
}
