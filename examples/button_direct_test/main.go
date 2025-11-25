package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// ButtonApp demonstrates composable button with mouse interaction
// using the Runtime message-driven architecture.
type ButtonApp struct {
	width        int
	height       int
	clickCount   int
	statusLabel  *gooey.ComposableLabel
	button       *gooey.ComposableButton
	instructions *gooey.ComposableLabel
}

// Init initializes the application widgets.
func (app *ButtonApp) Init() error {
	// Create a status label that will show click count
	app.statusLabel = gooey.NewComposableLabel("Status: Ready - Click the button!")
	app.statusLabel.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Create a button and set its bounds manually
	app.button = gooey.NewComposableButton("Click Me!", func() {
		app.clickCount++
		app.statusLabel.SetText(fmt.Sprintf("Button clicked %d times!", app.clickCount))
	})

	// Instructions
	app.instructions = gooey.NewComposableLabel("Press Ctrl+C or Q to exit")
	app.instructions.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	// Initialize widgets
	app.button.Init()
	app.statusLabel.Init()
	app.instructions.Init()

	return nil
}

// HandleEvent processes events from the runtime.
func (app *ButtonApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Exit on Ctrl+C or Q
		if e.Key == gooey.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.MouseEvent:
		// Pass mouse event to button directly
		if mouseAware, ok := interface{}(app.button).(gooey.MouseAware); ok {
			mouseAware.HandleMouse(e)
		}

	case gooey.ResizeEvent:
		// Update stored dimensions and widget bounds
		app.width = e.Width
		app.height = e.Height
		app.updateBounds()
	}

	return nil
}

// updateBounds updates widget bounds based on terminal size.
func (app *ButtonApp) updateBounds() {
	app.statusLabel.SetBounds(image.Rect(5, 2, app.width-5, 3))
	app.button.SetBounds(image.Rect(5, 5, app.width-5, 6))
	app.instructions.SetBounds(image.Rect(5, app.height-2, app.width-5, app.height-1))
}

// Render draws the current application state.
func (app *ButtonApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Update dimensions if not set
	if app.width == 0 || app.height == 0 {
		app.width = width
		app.height = height
		app.updateBounds()
	}

	// Fill background with dots
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			frame.SetCell(x, y, '.', gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		}
	}

	// Draw debug info
	bounds := app.button.GetBounds()
	info := fmt.Sprintf("Terminal: %dx%d | Button bounds: (%d,%d)->(%d,%d) W:%d",
		width, height,
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, bounds.Dx())
	frame.PrintStyled(0, 0, info, gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBackground(gooey.ColorBlack))

	// Draw widgets
	app.statusLabel.Draw(frame)
	app.button.Draw(frame)
	app.instructions.Draw(frame)
}

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	terminal.EnableAlternateScreen()
	defer terminal.DisableAlternateScreen()
	terminal.HideCursor()
	defer terminal.ShowCursor()
	terminal.EnableMouseTracking()
	defer terminal.DisableMouseTracking()

	// Get initial terminal size
	width, height := terminal.Size()

	// Create the application
	app := &ButtonApp{
		width:  width,
		height: height,
	}

	// Create and run the runtime
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
