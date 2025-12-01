package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey/tui"
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
func (app *CounterApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
		switch e.Rune {
		case '+':
			app.count++
		case '-':
			app.count--
		case 'q', 'Q':
			return []tui.Cmd{tui.Quit()}
		case 'r', 'R':
			app.count = 0
		}

	case tui.ResizeEvent:
		// Handle terminal resize (we don't need to do anything, View will adapt)
		return nil
	}

	return nil
}

// View returns the declarative view of the application state.
// This is called automatically after each event is processed.
// Uses Gooey's declarative view system for clean, composable UI.
func (app *CounterApp) View() tui.View {
	return tui.VStack(
		tui.Spacer().MinHeight(2),
		tui.Text("Counter Application").Bold().Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(2),
		tui.Text("Count: %d", app.count).Bold().Fg(tui.ColorGreen),
		tui.Spacer().MinHeight(3),
		tui.Text("[+] to increment    [-] to decrement"),
		tui.Text("[R] to reset        [Q] to quit"),
		tui.Spacer(),
		tui.Text("Running...  |  Press q to quit").Dim(),
	).Align(tui.AlignCenter).Padding(2)
}

func main() {
	// Run blocks until the application quits (when 'q' is pressed)
	if err := tui.Run(&CounterApp{}); err != nil {
		log.Fatal(err)
	}
}
