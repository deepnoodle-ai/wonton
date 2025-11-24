package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/deepnoodle-ai/gooey"
)

func drawBorder(terminal *gooey.Terminal) error {
	width, height := terminal.Size()

	// Clear the terminal to remove any old content from previous size
	terminal.Clear()

	frame, err := terminal.BeginFrame()
	if err != nil {
		return err
	}

	// Draw info at top
	info := fmt.Sprintf("Terminal: %dx%d (Press Ctrl+C to exit, try resizing the window!)", width, height)
	frame.PrintStyled(0, 0, info, gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Draw a border around the entire terminal
	style := gooey.NewStyle().WithForeground(gooey.ColorCyan)

	// Top border
	for x := 0; x < width; x++ {
		frame.SetCell(x, 2, '═', style)
	}

	// Bottom border
	for x := 0; x < width; x++ {
		frame.SetCell(x, height-1, '═', style)
	}

	// Left and right borders
	for y := 3; y < height-1; y++ {
		frame.SetCell(0, y, '║', style)
		frame.SetCell(width-1, y, '║', style)
	}

	// Corners
	frame.SetCell(0, 2, '╔', style)
	frame.SetCell(width-1, 2, '╗', style)
	frame.SetCell(0, height-1, '╚', style)
	frame.SetCell(width-1, height-1, '╝', style)

	return terminal.EndFrame(frame)
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

	// Enable automatic resize detection
	terminal.WatchResize()
	defer terminal.StopWatchResize()

	// Channel to signal when to redraw
	redrawChan := make(chan bool, 1)

	// Register resize callback
	unregister := terminal.OnResize(func(width, height int) {
		// Signal redraw on resize
		select {
		case redrawChan <- true:
		default:
			// Channel already has a pending redraw, skip
		}
	})
	defer unregister()

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initial draw
	if err := drawBorder(terminal); err != nil {
		fmt.Fprintf(os.Stderr, "Draw error: %v\n", err)
		return
	}

	// Main event loop - wait for resize or exit signal
	for {
		select {
		case <-redrawChan:
			// Redraw on resize
			if err := drawBorder(terminal); err != nil {
				fmt.Fprintf(os.Stderr, "Draw error: %v\n", err)
				return
			}

		case <-sigChan:
			// Exit on Ctrl+C
			return
		}
	}
}
