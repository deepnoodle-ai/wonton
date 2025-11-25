package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// CounterApp is a simple application that counts up and down.
// It demonstrates basic keyboard input handling and state management without locks.
//
// This example shows:
// - Basic keyboard input handling ('+', '-', 'r' for reset, 'q' for quit)
// - State management without any locks (single-threaded event loop guarantees)
// - Simple rendering using frame primitives
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
		switch e.Rune {
		case '+':
			// Increment counter
			app.count++
			return nil

		case '-':
			// Decrement counter
			app.count--
			return nil

		case 'q', 'Q':
			// Quit application
			return []gooey.Cmd{gooey.Quit()}

		case 'r', 'R':
			// Reset counter
			app.count = 0
			return nil
		}

	case gooey.ResizeEvent:
		// Handle terminal resize (we don't need to do anything, Render will adapt)
		return nil
	}

	return nil
}

// Render draws the current application state to the terminal.
// This is called automatically after each event is processed.
// Uses Gooey's double-buffered rendering for flicker-free updates.
func (app *CounterApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Create styles
	headerStyle := gooey.NewStyle().
		WithForeground(gooey.ColorCyan).
		WithBold()

	counterStyle := gooey.NewStyle().
		WithForeground(gooey.ColorGreen).
		WithBold()

	helpStyle := gooey.NewStyle().
		WithForeground(gooey.ColorWhite)

	dimStyle := gooey.NewStyle().
		WithForeground(gooey.ColorBrightBlack)

	// Clear screen by filling with spaces
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	// Title
	title := "Counter Application"
	titleX := (width - len(title)) / 2
	if titleX < 0 {
		titleX = 0
	}
	frame.PrintStyled(titleX, 2, title, headerStyle)

	// Counter display (big and prominent)
	counterText := fmt.Sprintf("Count: %d", app.count)
	counterX := (width - len(counterText)) / 2
	if counterX < 0 {
		counterX = 0
	}
	frame.PrintStyled(counterX, 5, counterText, counterStyle)

	// Help text
	helpLines := []string{
		"[+] to increment    [-] to decrement",
		"[R] to reset        [Q] to quit",
	}

	startY := 9
	for i, line := range helpLines {
		x := (width - len(line)) / 2
		if x < 0 {
			x = 0
		}
		frame.PrintStyled(x, startY+i, line, helpStyle)
	}

	// Footer with status
	if height > 0 {
		statusText := "Running...  |  Press q to quit"
		if len(statusText) <= width {
			x := (width - len(statusText)) / 2
			if x < 0 {
				x = 0
			}
			frame.PrintStyled(x, height-1, statusText, dimStyle)
		}
	}
}

func main() {
	// Create and initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create the counter application
	app := &CounterApp{
		count: 0,
	}

	// Create and run the runtime with 30 FPS
	// This starts the event loop which is single-threaded and race-free!
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits (when 'q' is pressed)
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
