// Package main demonstrates nested Stack/Group layout composition with
// clickable buttons and keyboard shortcuts.
//
// Run with:
//
//	go run ./examples/tui/composition
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

type App struct {
	counter int
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch {
		case e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case e.Rune == '+' || e.Rune == '=':
			app.counter++
		case e.Rune == '-':
			app.counter--
		case e.Rune == 'r' || e.Rune == 'R':
			app.counter = 0
		}
	}
	return nil
}

func (app *App) View() tui.View {
	return tui.Stack(
		tui.Text("Composition Demo").Bold().Fg(tui.ColorCyan),

		tui.Spacer().MinHeight(1),

		tui.Text("Counter: %d", app.counter).Fg(tui.ColorYellow),

		tui.Spacer().MinHeight(1),

		tui.Group(
			tui.Clickable(" + ", func() { app.counter++ }).Fg(tui.ColorGreen),
			tui.Clickable(" - ", func() { app.counter-- }).Fg(tui.ColorRed),
			tui.Clickable("Reset", func() { app.counter = 0 }),
		).Gap(2),

		tui.Spacer().MinHeight(1),

		tui.Stack(
			tui.Text("Nested Stack").Fg(tui.ColorMagenta),
			tui.Text("Item 1"),
			tui.Text("Item 2"),
			tui.Text("Item 3"),
		).Bordered().Border(&tui.SingleBorder),

		tui.Spacer().MinHeight(1),

		tui.Text("Click buttons or +/-/r keys • q to quit").Dim(),
	).Bordered().Border(&tui.RoundedBorder)
}

func main() {
	if err := tui.Run(&App{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
