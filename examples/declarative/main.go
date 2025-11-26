// Package main demonstrates the declarative UI API for Gooey.
package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// App holds the application state.
type App struct {
	name string
}

// View returns the declarative UI for this app.
func (app *App) View() gooey.View {
	return gooey.VStack(
		gooey.Text("Declarative Input Demo").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),
		gooey.HStack(
			gooey.Text("Name: "),
			gooey.Input(&app.name).Placeholder("Enter your name...").Width(30),
		).Gap(1),
		gooey.IfElse(app.name != "",
			gooey.Text("Hello, %s!", app.name).Fg(gooey.ColorGreen),
			gooey.Text("Type your name above").Dim(),
		),
		gooey.Spacer(),
		gooey.Text("Press Ctrl+C to quit").Dim(),
	).Padding(1)
}

// HandleEvent processes input events.
func (app *App) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
	}
	return nil
}

func main() {
	if err := gooey.Run(&App{}); err != nil {
		log.Fatal(err)
	}
}
