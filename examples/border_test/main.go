package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey/tui"
)

// BorderApp demonstrates drawing a border around the terminal
// and handling terminal resize events using the declarative View system.
type BorderApp struct {
	width  int
	height int
}

// HandleEvent processes events from the runtime.
func (app *BorderApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		// Exit on Ctrl+C
		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}

	case tui.ResizeEvent:
		// Update stored dimensions on resize
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// View returns the declarative view structure.
func (app *BorderApp) View() tui.View {
	// Info text at the top
	info := fmt.Sprintf("Terminal: %dx%d (Press Ctrl+C to exit, try resizing the window!)", app.width, app.height)

	return tui.VStack(
		tui.Text("%s", info).Fg(tui.ColorYellow),
		tui.Bordered(
			tui.Spacer(),
		).BorderFg(tui.ColorCyan),
	)
}

func main() {
	if err := tui.Run(&BorderApp{}); err != nil {
		log.Fatal(err)
	}
}
