// Package main demonstrates a declarative counter app.
// Compare with examples/runtime_counter which uses the imperative API.
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

type App struct {
	count int
}

func (app *App) View() tui.View {
	return tui.Stack(
		tui.Text("Counter Application").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(2),
		tui.Text("Count: %d", app.count).Bold().Fg(tui.ColorGreen),
		tui.Spacer().MinHeight(2),
		tui.Group(
			tui.Clickable("[ - ]", func() { app.count-- }).Fg(tui.ColorRed),
			tui.Clickable("[ + ]", func() { app.count++ }).Fg(tui.ColorGreen),
			tui.Clickable("[Reset]", func() { app.count = 0 }).Fg(tui.ColorYellow),
		).Gap(2),
		tui.Spacer(),
		tui.Text("[+] increment  [-] decrement  [r] reset  [q] quit").Dim(),
	).Align(tui.AlignCenter).Padding(2)
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch {
		case e.Rune == 'q' || e.Key == tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		case e.Rune == '+' || e.Rune == '=':
			app.count++
		case e.Rune == '-':
			app.count--
		case e.Rune == 'r' || e.Rune == 'R':
			app.count = 0
		}
	}
	return nil
}

func main() {
	if err := tui.Run(&App{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
