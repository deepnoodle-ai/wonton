package main

import (
	"fmt"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// BorderApp demonstrates drawing a border around the terminal
// and handling terminal resize events using the Runtime.
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

// Render draws the border and info text.
func (app *BorderApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Update dimensions if not set
	if app.width == 0 || app.height == 0 {
		app.width = width
		app.height = height
	}

	// Clear the frame
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	// Draw info at top
	info := fmt.Sprintf("Terminal: %dx%d (Press Ctrl+C to exit, try resizing the window!)", width, height)
	frame.PrintStyled(0, 0, info, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Draw a border around the entire terminal
	style := gooey.NewStyle().WithForeground(gooey.ColorCyan)

	// Top border
	for x := 0; x < width; x++ {
		frame.SetCell(x, 2, '═', style)
	}

	// Bottom border
	for x := 0; x < width; x++ {
		frame.SetCell(x, height-1, '═', style)
	}

	// Left and right borders
	for y := 3; y < height-1; y++ {
		frame.SetCell(0, y, '║', style)
		frame.SetCell(width-1, y, '║', style)
	}

	// Corners
	frame.SetCell(0, 2, '╔', style)
	frame.SetCell(width-1, 2, '╗', style)
	frame.SetCell(0, height-1, '╚', style)
	frame.SetCell(width-1, height-1, '╝', style)
}

func main() {
	if err := gooey.Run(&BorderApp{}); err != nil {
		log.Fatal(err)
	}
}
