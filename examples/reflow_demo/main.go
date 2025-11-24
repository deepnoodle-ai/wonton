package main

import (
	"fmt"
	"image"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	term, err := gooey.NewTerminal()
	if err != nil {
		panic(err)
	}
	defer term.Close()

	// Create content container with VBox layout
	// This will use the new ConstraintLayoutManager implementation of VBoxLayout
	content := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1),
		&gooey.RoundedBorder,
	)

	// Add a wrapping label
	label := gooey.NewWrappingLabel("This is a long text that should wrap automatically when the container width decreases. It demonstrates the new constraint-based layout system in Gooey! The height of the box should adjust to fit the text.")
	content.AddChild(label)

	// Add another button below to show it moves
	btn := gooey.NewComposableButton("I move down!", nil)
	content.AddChild(btn)

	width := 40
	direction := 1

	// Run for 200 frames
	for i := 0; i < 200; i++ {
		// Animate width
		width += direction
		if width > 60 {
			direction = -1
		} else if width < 20 {
			direction = 1
		}

		term.Clear()
		frame, _ := term.BeginFrame()

		// 1. MEASURE phase
		// We impose a tight width constraint, and loose height constraint
		// This simulates a parent (like Flex or SplitPane) constraining the child
		constraints := gooey.SizeConstraints{
			MinWidth:  width,
			MaxWidth:  width,
			MinHeight: 0,
			MaxHeight: 0,
		}

		// This triggers the ConstraintLayoutManager logic
		// The Container calls VBoxLayout.Measure
		// VBoxLayout.Measure calls WrappingLabel.Measure
		// WrappingLabel calculates height based on wrapped text
		size := content.Measure(constraints)

		// 2. LAYOUT phase
		// We center the box on screen
		termW, termH := frame.Size()
		x := (termW - size.X) / 2
		y := (termH - size.Y) / 2

		// Set bounds triggers internal layout
		// content.relayout() will use LayoutWithConstraints because we added it
		content.SetBounds(image.Rect(x, y, x+size.X, y+size.Y))

		// 3. DRAW phase
		content.Draw(frame)

		// Draw debug info
		debugInfo := fmt.Sprintf("Width: %d, Height: %d | Frame: %d", size.X, size.Y, i)
		frame.PrintStyled(0, 0, debugInfo, gooey.NewStyle())

		term.EndFrame(frame)
		time.Sleep(50 * time.Millisecond)
	}
}
