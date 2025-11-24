package main

import (
	"fmt"
	"image"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deepnoodle-ai/gooey"
	"golang.org/x/term"
)

// Minimal test to debug layout issues
func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Setup raw mode for input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enter raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	terminal.EnableAlternateScreen()
	defer terminal.DisableAlternateScreen()
	terminal.HideCursor()
	defer terminal.ShowCursor()
	terminal.EnableMouseTracking()
	defer terminal.DisableMouseTracking()

	width, height := terminal.Size()

	// Debug output
	debugInfo := fmt.Sprintf("Terminal: %dx%d", width, height)

	// Create simple container with border - SMALLER than terminal to see if clipping works
	container := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1).WithAlignment(gooey.LayoutAlignStretch),
		&gooey.SingleBorder,
	)

	// Add some labels
	label1 := gooey.NewComposableLabel(debugInfo)
	label1.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())

	label2 := gooey.NewComposableLabel("Status: Ready")
	label2.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Debug label for button bounds
	buttonDebug := gooey.NewComposableLabel("Button bounds: (waiting...)")
	buttonDebug.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen))

	// Add buttons with explicit callbacks that update label2
	clickCount := 0
	button1 := gooey.NewComposableButton("Click Me!", func() {
		clickCount++
		label2.SetText(fmt.Sprintf("Button 1 clicked %d times!", clickCount))
	})

	button2 := gooey.NewComposableButton("Press Me!", func() {
		label2.SetText("Button 2 was pressed!")
	})

	// Add a longer text to test wrapping
	longLabel := gooey.NewComposableLabel("This is a longer label to test if it gets properly truncated when it exceeds the container width")
	longLabel.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorMagenta))

	// Instructions
	instructions := gooey.NewComposableLabel("Press Ctrl+C to exit")
	instructions.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	container.AddChild(label1)
	container.AddChild(label2)
	container.AddChild(buttonDebug)
	container.AddChild(button1)
	container.AddChild(button2)
	container.AddChild(longLabel)
	container.AddChild(instructions)

	// Set bounds - use full terminal minus some margin
	margin := 2
	container.SetBounds(image.Rect(margin, margin, width-margin, height-margin))
	container.Init()

	// Draw
	drawUI := func() {
		frame, err := terminal.BeginFrame()
		if err != nil {
			return
		}

		// Fill background
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				frame.SetCell(x, y, '.', gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
			}
		}

		// Update button bounds debug
		b1Bounds := button1.GetBounds()
		buttonDebug.SetText(fmt.Sprintf("Button1: (%d,%d)->(%d,%d) W:%d",
			b1Bounds.Min.X, b1Bounds.Min.Y, b1Bounds.Max.X, b1Bounds.Max.Y, b1Bounds.Dx()))

		// Debug: show container bounds at bottom
		bounds := container.GetBounds()
		debugText := fmt.Sprintf("Bounds: (%d,%d)->(%d,%d) Size: %dx%d",
			bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y,
			bounds.Dx(), bounds.Dy())
		frame.PrintStyled(0, height-1, debugText, gooey.NewStyle().WithForeground(gooey.ColorYellow).WithBackground(gooey.ColorBlack))

		container.Draw(frame)
		terminal.EndFrame(frame)
	}

	drawUI()

	// Event loop with input handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	// Channel for input events
	inputChan := make(chan []byte, 10)
	go func() {
		buf := make([]byte, 128)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				inputChan <- data
			}
		}
	}()

	for {
		select {
		case <-sigChan:
			return
		case data := <-inputChan:
			// Check for Ctrl+C
			if len(data) > 0 && data[0] == 3 {
				return
			}

			// Parse mouse events
			if len(data) > 2 && data[0] == 27 && data[1] == '[' && data[2] == '<' {
				event, err := gooey.ParseMouseEvent(data[2:])
				if err == nil {
					// Pass mouse event to container
					if mouseAware, ok := interface{}(container).(gooey.MouseAware); ok {
						mouseAware.HandleMouse(*event)
						drawUI()
					}
				}
			}
		case <-ticker.C:
			if container.NeedsRedraw() {
				drawUI()
			}
		}
	}
}
