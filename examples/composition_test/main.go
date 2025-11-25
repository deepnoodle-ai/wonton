package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// CompositionTestApp is a minimal test application for debugging layout issues.
// It demonstrates:
// 1. Container with border and margins
// 2. Label updates from button clicks
// 3. Text truncation for long labels
// 4. Background fill pattern
// 5. Debug info display
type CompositionTestApp struct {
	container   *gooey.Container
	label2      *gooey.ComposableLabel
	buttonDebug *gooey.ComposableLabel
	button1     *gooey.ComposableButton
	clickCount  int
	width       int
	height      int
}

// NewCompositionTestApp creates and initializes the test application.
func NewCompositionTestApp(terminal *gooey.Terminal) *CompositionTestApp {
	app := &CompositionTestApp{clickCount: 0}

	app.width, app.height = terminal.Size()

	// Debug output
	debugInfo := fmt.Sprintf("Terminal: %dx%d", app.width, app.height)

	// Create simple container with border - SMALLER than terminal to see if clipping works
	app.container = gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1).WithAlignment(gooey.LayoutAlignStretch),
		&gooey.SingleBorder,
	)

	// Add some labels
	label1 := gooey.NewComposableLabel(debugInfo)
	label1.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())

	app.label2 = gooey.NewComposableLabel("Status: Ready")
	app.label2.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Debug label for button bounds
	app.buttonDebug = gooey.NewComposableLabel("Button bounds: (waiting...)")
	app.buttonDebug.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen))

	// Add buttons with explicit callbacks that update label2
	app.button1 = gooey.NewComposableButton("Click Me!", func() {
		app.clickCount++
		app.label2.SetText(fmt.Sprintf("Button 1 clicked %d times!", app.clickCount))
	})

	button2 := gooey.NewComposableButton("Press Me!", func() {
		app.label2.SetText("Button 2 was pressed!")
	})

	// Add a longer text to test wrapping
	longLabel := gooey.NewComposableLabel("This is a longer label to test if it gets properly truncated when it exceeds the container width")
	longLabel.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorMagenta))

	// Instructions
	instructions := gooey.NewComposableLabel("Press 'q' to quit")
	instructions.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	app.container.AddChild(label1)
	app.container.AddChild(app.label2)
	app.container.AddChild(app.buttonDebug)
	app.container.AddChild(app.button1)
	app.container.AddChild(button2)
	app.container.AddChild(longLabel)
	app.container.AddChild(instructions)

	// Set bounds - use full terminal minus some margin
	margin := 2
	app.container.SetBounds(image.Rect(margin, margin, app.width-margin, app.height-margin))
	app.container.Init()

	return app
}

// HandleEvent processes events from the Runtime.
func (app *CompositionTestApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle 'q' to quit
		if e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.MouseEvent:
		// Pass mouse events to the container
		if mouseAware, ok := interface{}(app.container).(gooey.MouseAware); ok {
			mouseAware.HandleMouse(e)
		}

	case gooey.ResizeEvent:
		// Update size and container bounds when terminal is resized
		app.width = e.Width
		app.height = e.Height
		margin := 2
		app.container.SetBounds(image.Rect(margin, margin, app.width-margin, app.height-margin))
	}

	return nil
}

// Render draws the current application state.
func (app *CompositionTestApp) Render(frame gooey.RenderFrame) {
	// Fill background with dots to make clipping visible
	for y := 0; y < app.height; y++ {
		for x := 0; x < app.width; x++ {
			frame.SetCell(x, y, '.', gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
		}
	}

	// Update button bounds debug info
	b1Bounds := app.button1.GetBounds()
	app.buttonDebug.SetText(fmt.Sprintf("Button1: (%d,%d)->(%d,%d) W:%d",
		b1Bounds.Min.X, b1Bounds.Min.Y, b1Bounds.Max.X, b1Bounds.Max.Y, b1Bounds.Dx()))

	// Debug: show container bounds at bottom
	bounds := app.container.GetBounds()
	debugText := fmt.Sprintf("Bounds: (%d,%d)->(%d,%d) Size: %dx%d",
		bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y,
		bounds.Dx(), bounds.Dy())
	frame.PrintStyled(0, app.height-1, debugText, gooey.NewStyle().WithForeground(gooey.ColorYellow).WithBackground(gooey.ColorBlack))

	// Draw the container
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
	app := NewCompositionTestApp(terminal)

	// Create and run the Runtime
	// 30 FPS is good for UI responsiveness
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
