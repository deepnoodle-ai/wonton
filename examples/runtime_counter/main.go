package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// CounterApp is a simple application that counts up and down.
// It demonstrates basic keyboard input handling and state management without locks.
//
// This example shows:
// - Basic keyboard input handling ('+', '-', 'r' for reset, 'q' for quit)
// - State management without any locks (single-threaded event loop guarantees)
// - Declarative view rendering using the View API
// - Terminal resize handling
type CounterApp struct {
	count int
}

// HandleEvent processes events from the runtime.
// This method is called in a single-threaded context, so no locks are needed.
// Return commands for any async work (HTTP requests, timers, etc.).
func (app *CounterApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		switch e.Rune {
		case '+':
			app.count++
		case '-':
			app.count--
		case 'q', 'Q':
			return []gooey.Cmd{gooey.Quit()}
		case 'r', 'R':
			app.count = 0
		}

	case gooey.ResizeEvent:
		// Handle terminal resize (we don't need to do anything, View will adapt)
		return nil
	}

	return nil
}

// View returns the declarative view of the application state.
// This is called automatically after each event is processed.
// Uses Gooey's declarative view system for clean, composable UI.
func (app *CounterApp) View() gooey.View {
	return gooey.VStack(
		gooey.Spacer().MinHeight(2),
		gooey.Text("Counter Application").Bold().Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(2),
		gooey.Text("Count: %d", app.count).Bold().Fg(gooey.ColorGreen),
		gooey.Spacer().MinHeight(3),
		gooey.Text("[+] to increment    [-] to decrement"),
		gooey.Text("[R] to reset        [Q] to quit"),
		gooey.Spacer(),
		gooey.Text("Running...  |  Press q to quit").Dim(),
	).Align(gooey.AlignCenter).Padding(2)
}

func main() {
	// Run blocks until the application quits (when 'q' is pressed)
	if err := gooey.Run(&CounterApp{}); err != nil {
		log.Fatal(err)
	}
}
