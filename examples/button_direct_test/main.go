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

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %v\n", err)
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

	// Create a status label that will show click count
	clickCount := 0
	statusLabel := gooey.NewComposableLabel("Status: Ready - Click the button!")
	statusLabel.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))
	statusLabel.SetBounds(image.Rect(5, 2, width-5, 3))

	// Create a button and set its bounds manually
	button := gooey.NewComposableButton("Click Me!", func() {
		clickCount++
		statusLabel.SetText(fmt.Sprintf("Button clicked %d times!", clickCount))
	})

	// Give it explicit bounds - full width of terminal minus margin
	button.SetBounds(image.Rect(5, 5, width-5, 6))

	// Instructions
	instructions := gooey.NewComposableLabel("Press Ctrl+C to exit")
	instructions.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	instructions.SetBounds(image.Rect(5, height-2, width-5, height-1))

	// Initialize widgets
	button.Init()
	statusLabel.Init()
	instructions.Init()

	// Draw function
	drawUI := func() {
		frame, err := terminal.BeginFrame()
		if err != nil {
			return
		}

		// Fill background with dots
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				frame.SetCell(x, y, '.', gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
			}
		}

		// Draw debug info
		bounds := button.GetBounds()
		info := fmt.Sprintf("Terminal: %dx%d | Button bounds: (%d,%d)->(%d,%d) W:%d",
			width, height,
			bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y, bounds.Dx())
		frame.PrintStyled(0, 0, info, gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBackground(gooey.ColorBlack))

		// Draw widgets
		statusLabel.Draw(frame)
		button.Draw(frame)
		instructions.Draw(frame)

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
					// Pass mouse event to button directly
					if mouseAware, ok := interface{}(button).(gooey.MouseAware); ok {
						mouseAware.HandleMouse(*event)
						drawUI()
					}
				}
			}
		case <-ticker.C:
			if button.NeedsRedraw() || statusLabel.NeedsRedraw() {
				drawUI()
			}
		}
	}
}
