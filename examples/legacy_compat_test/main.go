package main

import (
	"fmt"
	"image"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// LegacyWrapper wraps an old-style Button to be used in the new composition system.
// This simulates how a user might try to reuse old components.
type LegacyWrapper struct {
	gooey.BaseWidget
	btn *gooey.Button
}

func NewLegacyWrapper(label string) *LegacyWrapper {
	w := &LegacyWrapper{BaseWidget: gooey.NewBaseWidget()}
	// Initialize button with arbitrary absolute coordinates that SHOULD be ignored
	// if the compatibility fix is working.
	// We set X,Y to 100,100 - if it draws there, it's broken (or off screen).
	// If it draws at 0,0 relative to the container, it's fixed.
	w.btn = gooey.NewButton(100, 100, label, func() {
		// No-op
	})

	// Set the minimum size of this widget to match the button
	w.SetMinSize(image.Point{X: w.btn.Width, Y: 1})
	return w
}

func (w *LegacyWrapper) Draw(frame gooey.RenderFrame) {
	// Simple passthrough.
	// Before fix: This would draw at 100,100 (clipped/invisible).
	// After fix: Button.Draw detects frame size match and draws at 0,0.
	w.btn.Draw(frame)
}

func (w *LegacyWrapper) HandleKey(event gooey.KeyEvent) bool {
	// No-op for this test
	return false
}

func main() {
	term, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer term.Close()

	// Create a container centered on screen
	width, height := term.Size()

	container := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1),
		&gooey.RoundedBorder,
	)
	container.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen))
	container.SetBounds(image.Rect(width/2-15, height/2-5, width/2+15, height/2+5))

	// Add legacy widgets
	// We wrap them because Container expects ComposableWidget
	btn1 := NewLegacyWrapper("I was at 100,100")
	btn2 := NewLegacyWrapper("Me too!")

	container.AddChild(btn1)
	container.AddChild(btn2)

	container.Init()

	// Draw loop
	for i := 0; i < 50; i++ { // Run for a few seconds
		frame, _ := term.BeginFrame()

		// Clear background
		frame.Fill(' ', gooey.NewStyle())

		// Draw instructions
		frame.PrintStyled(2, 2, "Legacy Compatibility Test", gooey.NewStyle().WithBold())
		frame.PrintStyled(2, 3, "Buttons should appear inside the box.", gooey.NewStyle())
		frame.PrintStyled(2, 4, "If they are missing, the fix failed.", gooey.NewStyle())

		container.Draw(frame)

		term.EndFrame(frame)
		time.Sleep(100 * time.Millisecond)
	}
}
