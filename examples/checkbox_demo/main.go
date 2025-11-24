package main

import (
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

type CheckboxDemo struct {
	screen   *gooey.Screen
	checkbox *gooey.CheckboxGroup
}

func (d *CheckboxDemo) Draw(frame gooey.RenderFrame) {
	d.checkbox.Draw(frame)

	// Draw selected items below
	selected := d.checkbox.GetSelectedItems()
	msg := "Selected: " + strings.Join(selected, ", ")
	frame.PrintStyled(2, 10, msg, gooey.NewStyle())

	frame.PrintStyled(2, 12, "Press Space to toggle, Arrows to move, q to quit.", gooey.NewStyle().WithDim())
}

func (d *CheckboxDemo) HandleKey(event gooey.KeyEvent) bool {
	if event.Rune == 'q' {
		d.screen.Stop()
		return true
	}

	// Pass to checkbox
	if d.checkbox.HandleKey(event) {
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

	header := gooey.SimpleHeader("Checkbox Demo", gooey.NewStyle().WithBold().WithBackground(gooey.ColorBlue))
	screen.SetLayout(gooey.NewLayout(term).SetHeader(header))

	checkbox := gooey.NewCheckboxGroup(2, 2, []string{"Apple", "Banana", "Cherry", "Date", "Elderberry"})

	demo := &CheckboxDemo{
		screen:   screen,
		checkbox: checkbox,
	}

	screen.AddWidget(demo)

	if err := screen.Run(); err != nil {
		panic(err)
	}
}
