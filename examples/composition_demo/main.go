package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// Simple composition demo showing nested layouts with buttons and labels.
type App struct {
	counter int
}

func (app *App) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
	}
	return nil
}

func (app *App) View() gooey.View {
	return gooey.VStack(
		gooey.Text("Composition Demo").Bold().Fg(gooey.ColorCyan),

		gooey.Spacer().MinHeight(1),

		gooey.Text("Counter: %d", app.counter).Fg(gooey.ColorYellow),

		gooey.Spacer().MinHeight(1),

		gooey.HStack(
			gooey.Clickable(" + ", func() { app.counter++ }).Fg(gooey.ColorGreen),
			gooey.Clickable(" - ", func() { app.counter-- }).Fg(gooey.ColorRed),
			gooey.Clickable("Reset", func() { app.counter = 0 }),
		).Gap(2),

		gooey.Spacer().MinHeight(1),

		gooey.VStack(
			gooey.Text("Nested VBox").Fg(gooey.ColorMagenta),
			gooey.Text("Item 1"),
			gooey.Text("Item 2"),
			gooey.Text("Item 3"),
		).Bordered().Border(&gooey.SingleBorder),

		gooey.Spacer().MinHeight(1),

		gooey.Text("Click buttons or press 'q' to quit").Dim(),
	).Bordered().Border(&gooey.RoundedBorder)
}

func main() {
	if err := gooey.Run(&App{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
