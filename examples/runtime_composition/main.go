package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// ComposedApp demonstrates using the composition system with the Runtime.
// It shows:
// 1. Creating a Container with VBox layout
// 2. Adding ComposableLabel and ComposableButton widgets
// 3. Handling button clicks via keyboard (Enter key)
// 4. Updating label text dynamically
// 5. Handling terminal resize events
// 6. Quitting with the 'q' key
type ComposedApp struct {
	container  *gooey.Container
	titleLabel *gooey.ComposableLabel
	clickLabel *gooey.ComposableLabel
	button     *gooey.ComposableButton
	clicks     int
}

// NewComposedApp creates a new ComposedApp with all UI elements set up.
func NewComposedApp(terminal *gooey.Terminal) *ComposedApp {
	app := &ComposedApp{clicks: 0}

	// Create main container with VBox layout (2 unit spacing between items)
	app.container = gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(2),
		&gooey.SingleBorder,
	)
	app.container.SetStyle(gooey.NewStyle().WithBackground(gooey.ColorBlack))

	// Set padding for the container
	params := gooey.DefaultLayoutParams()
	params.PaddingTop = 1
	params.PaddingBottom = 1
	params.PaddingLeft = 2
	params.PaddingRight = 2
	app.container.SetLayoutParams(params)

	// Create title label
	app.titleLabel = gooey.NewComposableLabel("Gooey Runtime + Composition Demo")
	app.titleLabel.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())
	app.titleLabel.WithAlign(gooey.AlignCenter)
	app.container.AddChild(app.titleLabel)

	// Create description label
	description := gooey.NewComposableMultiLineLabel([]string{
		"This demonstrates the message-driven Runtime",
		"combined with the composition system.",
		"",
		"Press Enter to click the button, or 'q' to quit.",
	})
	description.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorWhite))
	app.container.AddChild(description)

	// Create click counter label (initialize empty, will be updated)
	app.clickLabel = gooey.NewComposableLabel("Click count: 0")
	app.clickLabel.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow).WithBold())
	app.clickLabel.WithAlign(gooey.AlignCenter)
	app.container.AddChild(app.clickLabel)

	// Create a button
	app.button = gooey.NewComposableButton("Click Me!", func() {
		// This callback is called from the event handler
		// when the user presses Enter
		app.clicks++
		app.clickLabel.SetText(fmt.Sprintf("Click count: %d", app.clicks))
	})
	app.button.Style = gooey.NewStyle().
		WithBackground(gooey.ColorGreen).
		WithForeground(gooey.ColorBlack)
	app.button.HoverStyle = gooey.NewStyle().
		WithBackground(gooey.ColorBrightGreen).
		WithForeground(gooey.ColorBlack).
		WithBold()
	app.container.AddChild(app.button)

	// Create instructions label
	instructions := gooey.NewComposableMultiLineLabel([]string{
		"",
		"Controls:",
		"  Enter: Click the button",
		"  q:     Quit the application",
	})
	instructions.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	app.container.AddChild(instructions)

	// Set initial bounds and initialize
	width, height := terminal.Size()
	app.container.SetBounds(image.Rect(0, 0, width, height))
	app.container.Init()

	return app
}

// HandleEvent processes events from the Runtime.
// This runs in a single-threaded event loop (no locks needed!).
func (app *ComposedApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		switch e.Rune {
		case 'q', 'Q':
			// User wants to quit
			return []gooey.Cmd{gooey.Quit()}
		case '\n', '\r': // Enter key
			// Simulate clicking the button
			app.button.OnClick()
		}

	case gooey.ResizeEvent:
		// Update container bounds when terminal is resized
		app.container.SetBounds(image.Rect(0, 0, e.Width, e.Height))
	}

	return nil
}

// Render draws the current application state.
// This uses the composition system and Gooey's double-buffered rendering.
func (app *ComposedApp) Render(frame gooey.RenderFrame) {
	// The container handles all rendering of its children
	app.container.Draw(frame)
}

func main() {
	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create the application
	app := NewComposedApp(terminal)

	// Create and run the Runtime
	// 30 FPS is good for UI responsiveness
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
