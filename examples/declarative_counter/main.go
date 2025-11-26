// Package main demonstrates a declarative counter app.
// Compare with examples/runtime_counter which uses the imperative API.
package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

type App struct {
	count int
}

func (app *App) View() gooey.View {
	return gooey.VStack(
		gooey.Text("Counter Application").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(2),
		gooey.Text("Count: %d", app.count).Bold().Fg(gooey.ColorGreen),
		gooey.Spacer().MinHeight(2),
		gooey.HStack(
			gooey.Clickable("[ - ]", func() { app.count-- }).Fg(gooey.ColorRed),
			gooey.Clickable("[ + ]", func() { app.count++ }).Fg(gooey.ColorGreen),
			gooey.Clickable("[Reset]", func() { app.count = 0 }).Fg(gooey.ColorYellow),
		).Gap(2),
		gooey.Spacer(),
		gooey.Text("[+] increment  [-] decrement  [r] reset  [q] quit").Dim(),
	).Align(gooey.AlignCenter).Padding(2)
}

func (app *App) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		switch {
		case e.Rune == 'q' || e.Key == gooey.KeyCtrlC:
			return []gooey.Cmd{gooey.Quit()}
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
	if err := gooey.Run(&App{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
