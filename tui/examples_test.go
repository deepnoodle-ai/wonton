package tui_test

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/tui"
)

// ExampleApplication demonstrates the basic Application interface
// for building declarative TUI applications.
func ExampleApplication() {
	type CounterApp struct {
		count int
	}

	// View returns the UI tree based on current state
	// func (app *CounterApp) View() tui.View {
	// 	return tui.Stack(
	// 		tui.Text("Count: %d", app.count),
	// 		tui.Button("Increment", func() { app.count++ }),
	// 	)
	// }

	app := &CounterApp{}
	_ = app

	// Run the application
	// tui.Run(app)
}

// ExampleStack demonstrates vertical layout of views.
func ExampleStack() {
	// Stack arranges children vertically
	view := tui.Stack(
		tui.Text("Header").Bold(),
		tui.Text("Body content"),
		tui.Spacer(),
		tui.Text("Footer").Dim(),
	).Gap(1) // 1 row between children

	_ = view
	// Output:
}

// ExampleText demonstrates text rendering with styling.
func ExampleText() {
	// Basic text
	_ = tui.Text("Hello, World!")

	// Text with formatting
	name := "Alice"
	_ = tui.Text("Hello, %s!", name)

	// Text with styling
	_ = tui.Text("Success").Fg(tui.ColorGreen).Bold()

	// Semantic styling
	_ = tui.Text("Error occurred").Error()
	_ = tui.Text("Warning message").Warning()
	_ = tui.Text("Helpful hint").Hint()

	// Output:
}

// ExampleSpacer demonstrates flexible space distribution.
func ExampleSpacer() {
	// Push content to opposite edges
	view := tui.Stack(
		tui.Text("Top"),
		tui.Spacer(), // Fills all available space
		tui.Text("Bottom"),
	)

	_ = view
	// Output:
}

// ExampleButton demonstrates interactive buttons.
func ExampleButton() {
	type App struct {
		count int
	}
	app := &App{}

	// Button with callback
	button := tui.Button("Click Me", func() {
		app.count++
	})

	// Styled button
	_ = tui.Button("Submit", func() {}).
		Fg(tui.ColorGreen).
		Bold()

	_ = button
	// Output:
}

// ExampleInputField demonstrates text input fields.
func ExampleInputField() {
	type App struct {
		name  string
		email string
	}
	app := &App{}

	// Basic input field
	input := tui.InputField(&app.name).
		Label("Name:").
		Placeholder("Enter your name")

	// Input with border and custom styling
	_ = tui.InputField(&app.email).
		Label("Email").
		Placeholder("user@example.com").
		Bordered().
		FocusBorderFg(tui.ColorCyan).
		Width(40)

	_ = input
	// Output:
}

// ExampleCmd demonstrates async command execution.
func ExampleCmd() {
	// type DataEvent struct {
	// 	Time time.Time
	// 	Data string
	// }

	// Custom event type needs Timestamp method
	// func (e DataEvent) Timestamp() time.Time { return e.Time }

	// Command that performs async work
	// fetchData := func() tui.Cmd {
	// 	return func() tui.Event {
	// 		// Simulate async operation
	// 		data := "fetched data"
	// 		return DataEvent{Time: time.Now(), Data: data}
	// 	}
	// }

	// Output:
}

// ExamplePadding demonstrates adding space around views.
func ExamplePadding() {
	content := tui.Text("Padded content")

	// Equal padding on all sides
	_ = tui.Padding(2, content)

	// Different horizontal and vertical padding
	_ = tui.PaddingHV(4, 1, content)

	// Specific padding on each side
	_ = tui.PaddingLTRB(1, 2, 3, 4, content)

	// Output:
}

// ExampleBordered demonstrates bordered views.
func ExampleBordered() {
	content := tui.Text("Inside box")

	// Simple border
	_ = tui.Bordered(content)

	// Border with title and styling
	_ = tui.Bordered(content).
		Border(&tui.RoundedBorder).
		Title("Box Title").
		BorderFg(tui.ColorCyan)

	// Focus-aware border
	_ = tui.Bordered(content).
		FocusID("my-element").
		FocusBorderFg(tui.ColorGreen)

	// Output:
}

// ExampleEventHandler demonstrates event handling.
func ExampleEventHandler() {
	type App struct {
		text string
	}

	// HandleEvent processes events and returns commands
	// func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	// 	switch e := event.(type) {
	// 	case tui.KeyEvent:
	// 		if e.Rune == 'q' {
	// 			return []tui.Cmd{tui.Quit()}
	// 		}
	// 		app.text += string(e.Rune)
	// 	case tui.ResizeEvent:
	// 		// Handle terminal resize
	// 	}
	// 	return nil
	// }

	app := &App{}
	_ = app

	// Output:
}

// ExampleWidth demonstrates size constraints.
func ExampleWidth() {
	content := tui.Text("This is some text content")

	// Fixed width
	_ = tui.Width(40, content)

	// Maximum width (can be smaller)
	_ = tui.MaxWidth(80, content)

	// Fixed width and height
	_ = tui.Size(40, 10, content)

	// Output:
}

// Example demonstrates a complete minimal application.
func Example() {
	type App struct {
		count int
	}

	// View renders the UI
	// func (app *App) View() tui.View {
	// 	return tui.Stack(
	// 		tui.Text("Counter: %d", app.count).Bold(),
	// 		tui.Button("Increment", func() { app.count++ }),
	// 		tui.Text("Press 'q' to quit").Dim(),
	// 	).Gap(1)
	// }

	// HandleEvent processes events
	// func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	// 	if key, ok := event.(tui.KeyEvent); ok && key.Rune == 'q' {
	// 		return []tui.Cmd{tui.Quit()}
	// 	}
	// 	return nil
	// }

	app := &App{count: 0}
	_ = app

	// Run the application
	// err := tui.Run(app)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Output:
}

// ExampleRun demonstrates running a TUI application with options.
func ExampleRun() {
	type App struct{}

	app := &App{}

	// Run with default options
	// tui.Run(app)

	// Run with custom options
	_ = tui.Run(app,
		tui.WithFPS(60),                  // 60 FPS for smooth animations
		tui.WithMouseTracking(true),      // Enable mouse events
		tui.WithAlternateScreen(true),    // Use alternate screen buffer
		tui.WithBracketedPaste(true),     // Handle pasted text properly
	)

	// Output:
}

// ExampleBatch demonstrates running multiple commands.
func ExampleBatch() {
	cmd1 := func() tui.Cmd {
		return func() tui.Event {
			return tui.TickEvent{}
		}
	}

	cmd2 := func() tui.Cmd {
		return func() tui.Event {
			return tui.TickEvent{}
		}
	}

	// Run multiple commands in parallel
	cmds := tui.Batch(
		cmd1(),
		cmd2(),
	)

	fmt.Printf("Queued %d commands", len(cmds))
	// Output: Queued 2 commands
}
