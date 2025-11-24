package main

import (
	"github.com/deepnoodle-ai/gooey"
)

type MainWidget struct {
	screen *gooey.Screen
}

func (w *MainWidget) Draw(frame gooey.RenderFrame) {
	width, height := frame.Size()
	msg := "Press 'm' to open modal. Press 'q' to quit."
	x := (width - len(msg)) / 2
	y := height / 2
	frame.PrintStyled(x, y, msg, gooey.NewStyle())
}

func (w *MainWidget) HandleKey(event gooey.KeyEvent) bool {
	if event.Rune == 'm' {
		modal := gooey.NewModal("Confirm Action", "Are you sure you want to perform this action?\nIt cannot be undone.", []string{"OK", "Cancel"})
		modal.SetCallback(func(index int) {
			w.screen.CloseModal()
		})
		w.screen.ShowModal(modal)
		return true
	}
	if event.Rune == 'q' {
		w.screen.Stop()
		return true
	}
	return false
}

func main() {
	term, err := gooey.NewTerminal()
	if err != nil {
		panic(err)
	}
	defer term.Close()

	screen := gooey.NewScreen(term)

	header := gooey.SimpleHeader("Modal Demo", gooey.NewStyle().WithBold().WithForeground(gooey.ColorWhite).WithBackground(gooey.ColorBlue))
	footer := gooey.SimpleFooter("Left", "Center", "Right", gooey.NewStyle().WithForeground(gooey.ColorBlack).WithBackground(gooey.ColorWhite))

	layout := gooey.NewLayout(term).SetHeader(header).SetFooter(footer)
	screen.SetLayout(layout)

	widget := &MainWidget{screen: screen}
	screen.AddWidget(widget)

	if err := screen.Run(); err != nil {
		panic(err)
	}
}
