// Example: inline_counter
//
// This example demonstrates InlineApp basics:
// - A live region that updates in place
// - Printing to scrollback history
// - Event handling for keyboard input
// - Graceful quit with Ctrl+C or 'q'
//
// Run with: go run ./examples/inline_counter
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

type CounterApp struct {
	runner *tui.InlineApp
	count  int
}

func (app *CounterApp) LiveView() tui.View {
	return tui.Stack(
		tui.Divider(),
		tui.Text(" Count: %d", app.count).Bold(),
		tui.Text(""),
		tui.Text(" Press +/- to change, q to quit").Dim(),
		tui.Divider(),
	)
}

func (app *CounterApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch {
		case e.Rune == '+' || e.Rune == '=':
			app.count++
			app.runner.Printf("Incremented to %d", app.count)

		case e.Rune == '-' || e.Rune == '_':
			app.count--
			app.runner.Printf("Decremented to %d", app.count)

		case e.Rune == 'q' || e.Key == tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func main() {
	app := &CounterApp{}
	app.runner = tui.NewInlineApp(tui.WithInlineWidth(60))

	if err := app.runner.Run(app); err != nil {
		log.Fatal(err)
	}
}
