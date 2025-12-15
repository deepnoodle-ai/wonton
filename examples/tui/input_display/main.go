// Package main demonstrates the declarative UI API for Wonton.
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

// App holds the application state.
type App struct {
	name string
}

// View returns the declarative UI for this app.
func (app *App) View() tui.View {
	return tui.VStack(
		tui.Text("Declarative Input Demo").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
		tui.HStack(
			tui.Text("Name: "),
			tui.Input(&app.name).Placeholder("Enter your name...").Width(30),
		).Gap(1),
		tui.IfElse(app.name != "",
			tui.Text("Hello, %s!", app.name).Fg(tui.ColorGreen),
			tui.Text("Type your name above").Dim(),
		),
		tui.Spacer(),
		tui.Text("Press Ctrl+C to quit").Dim(),
	).Padding(1)
}

// HandleEvent processes input events.
func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func main() {
	if err := tui.Run(&App{}); err != nil {
		log.Fatal(err)
	}
}
