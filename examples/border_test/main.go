package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// BorderApp demonstrates drawing a border around the terminal
// and handling terminal resize events using the declarative View system.
type BorderApp struct {
	width  int
	height int
}

// HandleEvent processes events from the runtime.
func (app *BorderApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Exit on Ctrl+C
		if e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		// Update stored dimensions on resize
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// View returns the declarative view structure.
func (app *BorderApp) View() gooey.View {
	// Info text at the top
	info := fmt.Sprintf("Terminal: %dx%d (Press Ctrl+C to exit, try resizing the window!)", app.width, app.height)

	return gooey.VStack(
		gooey.Text("%s", info).Fg(gooey.ColorYellow),
		gooey.Bordered(
			gooey.Spacer(),
		).BorderFg(gooey.ColorCyan),
	)
}

func main() {
	if err := gooey.Run(&BorderApp{}); err != nil {
		log.Fatal(err)
	}
}
