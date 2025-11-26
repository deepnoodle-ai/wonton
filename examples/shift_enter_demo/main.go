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

func (app *ShiftEnterApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	y := 0

	frame.PrintStyled(0, y, "Shift+Enter Detection Demo", gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())
	y += 2

	frame.PrintStyled(0, y, "Press Shift+Enter to test detection", gooey.NewStyle())
	y += 2

	if app.shiftEntered {
		frame.PrintStyled(0, y, "Shift+Enter detected!", gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold())
	} else {
		frame.PrintStyled(0, y, "Waiting...", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	}
	y += 2

	if app.lastKey != "" {
		frame.PrintStyled(0, y, "Last key: "+app.lastKey, gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	}
	y += 2

	frame.PrintStyled(0, y, "Press Ctrl+C or Esc to quit", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
}

func main() {
	if err := gooey.Run(&ShiftEnterApp{}); err != nil {
		log.Fatal(err)
	}
}
