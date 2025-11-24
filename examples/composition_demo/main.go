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

// This example demonstrates the new composition system in Gooey.
// It shows:
// 1. Nested containers with different layout managers
// 2. VBox and HBox layouts
// 3. FlexLayout with various settings
// 4. Composable buttons and labels
// 5. Parent-child relationships and event handling

func main() {
	// Initialize terminal
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

	// Get terminal size
	width, height := terminal.Size()

	// Create the main container with VBox layout
	// Use LayoutAlignStretch to make all children fill the width
	mainContainer := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1).WithAlignment(gooey.LayoutAlignStretch),
		&gooey.SingleBorder,
	)
	mainContainer.SetStyle(gooey.NewStyle().WithBackground(gooey.ColorBlack))

	// Set layout params for padding
	params := gooey.DefaultLayoutParams()
	params.PaddingTop = 1
	params.PaddingBottom = 1
	params.PaddingLeft = 2
	params.PaddingRight = 2
	mainContainer.SetLayoutParams(params)

	// Create header
	header := gooey.NewComposableLabel("Gooey Composition Demo")
	header.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())
	header.WithAlign(gooey.AlignCenter)

	// Create description
	description := gooey.NewComposableMultiLineLabel([]string{
		"This showcases the composition system:",
		"• Nested containers with layouts",
		"• VBox, HBox, and Flex layouts",
		"• Bounds-based positioning",
		"• Parent-child event delegation",
	})
	description.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorWhite))

	// Create a horizontal button bar using HBox
	buttonBar := gooey.NewContainer(gooey.NewHBoxLayout(2))
	buttonBarParams := gooey.DefaultLayoutParams()
	buttonBarParams.MarginTop = 1
	buttonBar.SetLayoutParams(buttonBarParams)

	clickCount := 0
	statusLabel := gooey.NewComposableLabel("Status: Ready")
	statusLabel.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Create buttons with different styles
	btn1 := gooey.NewComposableButton("Click Me!", func() {
		clickCount++
		statusLabel.SetText(fmt.Sprintf("Status: Button 1 clicked! (Count: %d)", clickCount))
	})

	btn2 := gooey.NewComposableButton("Press Here", func() {
		statusLabel.SetText("Status: Button 2 pressed!")
	})
	btn2.Style = gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack)
	btn2.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorBrightGreen).WithForeground(gooey.ColorBlack).WithBold()

	btn3 := gooey.NewComposableButton("Exit Demo", func() {
		statusLabel.SetText("Status: Exiting...")
		os.Exit(0)
	})
	btn3.Style = gooey.NewStyle().WithBackground(gooey.ColorRed).WithForeground(gooey.ColorWhite)
	btn3.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorBrightRed).WithForeground(gooey.ColorWhite).WithBold()

	buttonBar.AddChild(btn1)
	buttonBar.AddChild(btn2)
	buttonBar.AddChild(btn3)

	// Create a flex layout demo section
	flexSection := gooey.NewContainerWithBorder(
		gooey.NewFlexLayout().
			WithDirection(gooey.FlexRow).
			WithJustify(gooey.FlexJustifySpaceBetween).
			WithAlignItems(gooey.FlexAlignItemsCenter).
			WithSpacing(1),
		&gooey.RoundedBorder,
	)
	flexSectionParams := gooey.DefaultLayoutParams()
	flexSectionParams.MarginTop = 1
	flexSectionParams.PaddingTop = 1
	flexSectionParams.PaddingBottom = 1
	flexSectionParams.PaddingLeft = 2
	flexSectionParams.PaddingRight = 2
	flexSection.SetLayoutParams(flexSectionParams)

	flexLabel1 := gooey.NewComposableLabel("Flex Item 1")
	flexLabel1.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorMagenta))

	flexLabel2 := gooey.NewComposableLabel("Flex Item 2")
	flexLabel2.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan))
	flexLabel2Params := gooey.DefaultLayoutParams()
	flexLabel2Params.Grow = 1 // This will take extra space
	flexLabel2.SetLayoutParams(flexLabel2Params)

	flexLabel3 := gooey.NewComposableLabel("Flex Item 3")
	flexLabel3.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	flexSection.AddChild(flexLabel1)
	flexSection.AddChild(flexLabel2)
	flexSection.AddChild(flexLabel3)

	// Create a nested container demo
	nestedSection := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(0),
		&gooey.DoubleBorder,
	)
	nestedSectionParams := gooey.DefaultLayoutParams()
	nestedSectionParams.MarginTop = 1
	nestedSectionParams.PaddingTop = 0
	nestedSectionParams.PaddingBottom = 0
	nestedSectionParams.PaddingLeft = 1
	nestedSectionParams.PaddingRight = 1
	nestedSection.SetLayoutParams(nestedSectionParams)

	nestedTitle := gooey.NewComposableLabel("Nested Container")
	nestedTitle.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold())
	nestedTitle.WithAlign(gooey.AlignCenter)

	nestedContent := gooey.NewComposableLabel("Container inside container!")
	nestedContent.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorWhite))
	nestedContent.WithAlign(gooey.AlignCenter)

	nestedSection.AddChild(nestedTitle)
	nestedSection.AddChild(nestedContent)

	// Add instructions
	instructions := gooey.NewComposableMultiLineLabel([]string{
		"",
		"Instructions:",
		"• Click buttons with your mouse!",
		"• Watch the status label update",
		"• Press Ctrl+C to exit",
	})
	instructions.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	// Add all sections to main container
	mainContainer.AddChild(header)
	mainContainer.AddChild(description)
	mainContainer.AddChild(buttonBar)
	mainContainer.AddChild(statusLabel)
	mainContainer.AddChild(flexSection)
	mainContainer.AddChild(nestedSection)
	mainContainer.AddChild(instructions)

	// Set the main container bounds to fill the terminal
	mainContainer.SetBounds(image.Rect(0, 0, width, height))

	// Initialize the container (calls Init on all children)
	mainContainer.Init()

	// Enable automatic resize handling
	terminal.WatchResize()
	defer terminal.StopWatchResize()
	mainContainer.WatchResize(terminal)

	// Draw function
	drawUI := func() {
		frame, err := terminal.BeginFrame()
		if err != nil {
			return
		}

		// Debug: Print bounds info
		bounds := mainContainer.GetBounds()
		debugText := fmt.Sprintf("Main: %dx%d Container: (%d,%d)->(%d,%d)",
			width, height, bounds.Min.X, bounds.Min.Y, bounds.Max.X, bounds.Max.Y)
		frame.PrintStyled(0, height-1, debugText, gooey.NewStyle().WithForeground(gooey.ColorYellow))

		mainContainer.Draw(frame)
		terminal.EndFrame(frame)
	}

	// Initial draw
	drawUI()

	// Set up signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Event loop with input handling
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
			// Graceful shutdown
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
					if mouseAware, ok := interface{}(mainContainer).(gooey.MouseAware); ok {
						mouseAware.HandleMouse(*event)
						drawUI()
					}
				}
			}
		case <-ticker.C:
			// Periodic redraw (for animations, etc.)
			if mainContainer.NeedsRedraw() {
				drawUI()
			}
		}
	}
}
