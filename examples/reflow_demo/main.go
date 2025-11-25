package main

import (
	"fmt"
	"image"

	"github.com/deepnoodle-ai/gooey"
)

// ReflowApp demonstrates the constraint-based layout system with text reflow
// using the Runtime message-driven architecture.
type ReflowApp struct {
	content   *gooey.Container
	width     int
	direction int
	frame     uint64
}

// Init initializes the application widgets.
func (app *ReflowApp) Init() error {
	// Create content container with VBox layout
	// This will use the new ConstraintLayoutManager implementation of VBoxLayout
	app.content = gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1),
		&gooey.RoundedBorder,
	)

	// Add a wrapping label
	label := gooey.NewWrappingLabel("This is a long text that should wrap automatically when the container width decreases. It demonstrates the new constraint-based layout system in Gooey! The height of the box should adjust to fit the text.")
	app.content.AddChild(label)

	// Add another button below to show it moves
	btn := gooey.NewComposableButton("I move down!", nil)
	app.content.AddChild(btn)

	// Initialize with starting width
	app.width = 40
	app.direction = 1

	return nil
}

// HandleEvent processes events from the runtime.
func (app *ReflowApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Exit on any key press or Ctrl+C
		return []gooey.Cmd{gooey.Quit()}

	case gooey.TickEvent:
		// Animate width on each tick
		app.frame = e.Frame
		app.width += app.direction
		if app.width > 60 {
			app.direction = -1
		} else if app.width < 20 {
			app.direction = 1
		}

		// Auto-quit after 200 frames
		if app.frame >= 200 {
			return []gooey.Cmd{gooey.Quit()}
		}
	}

	return nil
}

// Render draws the animated reflow demo.
func (app *ReflowApp) Render(frame gooey.RenderFrame) {
	termW, termH := frame.Size()

	// Clear the frame
	frame.FillStyled(0, 0, termW, termH, ' ', gooey.NewStyle())

	// 1. MEASURE phase
	// We impose a tight width constraint, and loose height constraint
	// This simulates a parent (like Flex or SplitPane) constraining the child
	constraints := gooey.SizeConstraints{
		MinWidth:  app.width,
		MaxWidth:  app.width,
		MinHeight: 0,
		MaxHeight: 0,
	}

	// This triggers the ConstraintLayoutManager logic
	// The Container calls VBoxLayout.Measure
	// VBoxLayout.Measure calls WrappingLabel.Measure
	// WrappingLabel calculates height based on wrapped text
	size := app.content.Measure(constraints)

	// 2. LAYOUT phase
	// We center the box on screen
	x := (termW - size.X) / 2
	y := (termH - size.Y) / 2

	// Set bounds triggers internal layout
	// content.relayout() will use LayoutWithConstraints because we added it
	app.content.SetBounds(image.Rect(x, y, x+size.X, y+size.Y))

	// 3. DRAW phase
	app.content.Draw(frame)

	// Draw debug info
	debugInfo := fmt.Sprintf("Width: %d, Height: %d | Frame: %d/200 | Press any key to exit", size.X, size.Y, app.frame)
	frame.PrintStyled(0, 0, debugInfo, gooey.NewStyle())
}

func main() {
	term, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error creating terminal: %v\n", err)
		return
	}
	defer term.Close()

	// Create the application
	app := &ReflowApp{}

	// Create and run the runtime at 20 FPS (50ms per frame)
	runtime := gooey.NewRuntime(term, app, 20)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Printf("Runtime error: %v\n", err)
		return
	}

	fmt.Println("\nReflow demo complete!")
}
