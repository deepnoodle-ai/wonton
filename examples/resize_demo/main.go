package main

import (
	"fmt"
	"image"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		panic(err)
	}
	defer terminal.Close()

	// Enable raw mode and alternate screen
	terminal.EnableRawMode()
	terminal.EnableAlternateScreen()
	terminal.HideCursor()

	// Enable automatic resize watching
	terminal.WatchResize()
	defer terminal.StopWatchResize()

	// Create a layout with header and footer
	layout := gooey.NewLayout(terminal)

	// Configure header
	headerStyle := gooey.NewStyle().
		WithBold().
		WithForeground(gooey.ColorCyan)
	layout.SetHeader(&gooey.Header{
		Center:      "Terminal Resize Demo with Containers",
		Style:       headerStyle,
		Border:      true,
		BorderStyle: gooey.RoundedBorder,
		Height:      3,
	})

	// Configure footer with status bar
	layout.SetFooter(&gooey.Footer{
		StatusBar: true,
		StatusItems: []gooey.StatusItem{
			{Key: "Status", Value: "Ready", Icon: "✓"},
			{Key: "Controls", Value: "Press Ctrl+C to exit", Icon: "⌨"},
		},
		Border:      true,
		BorderStyle: gooey.RoundedBorder,
		Height:      3,
	})

	// Create main container
	mainContainer := gooey.NewContainer(gooey.NewVBoxLayout(1))

	// Add some labels
	titleLabel := gooey.NewComposableLabel("Container with Auto-Resize").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()).
		WithAlign(gooey.AlignCenter)
	mainContainer.AddChild(titleLabel)

	sizeLabel := gooey.NewComposableLabel("Size: ...").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow)).
		WithAlign(gooey.AlignCenter)
	mainContainer.AddChild(sizeLabel)

	instructionLabel := gooey.NewComposableLabel("Try resizing your terminal window!").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen)).
		WithAlign(gooey.AlignCenter)
	mainContainer.AddChild(instructionLabel)

	// Create button container
	buttonContainer := gooey.NewContainer(gooey.NewHBoxLayout(2))

	clickCount := 0
	button1 := gooey.NewComposableButton("Click Me!", func() {
		clickCount++
		titleLabel.SetText(fmt.Sprintf("Clicked %d times!", clickCount))
	})
	button1.Style = gooey.NewStyle().
		WithForeground(gooey.ColorBlack).
		WithBackground(gooey.ColorCyan)
	buttonContainer.AddChild(button1)

	button2 := gooey.NewComposableButton("Reset", func() {
		clickCount = 0
		titleLabel.SetText("Container with Auto-Resize")
	})
	button2.Style = gooey.NewStyle().
		WithForeground(gooey.ColorBlack).
		WithBackground(gooey.ColorYellow)
	buttonContainer.AddChild(button2)

	mainContainer.AddChild(buttonContainer)

	// Initialize container
	width, _ := terminal.Size()
	contentY, contentHeight := layout.ContentArea()
	mainContainer.SetBounds(image.Rect(0, contentY, width, contentY+contentHeight))
	mainContainer.Init()

	// Enable resize watching for the container
	mainContainer.WatchResize(terminal)

	// Track resize count
	resizeCount := 0

	// Update display
	updateDisplay := func() {
		w, h := terminal.Size()
		cy, ch := layout.ContentArea()

		sizeLabel.SetText(fmt.Sprintf("Terminal: %dx%d | Content: Y=%d H=%d", w, h, cy, ch))

		// Update container bounds
		mainContainer.SetBounds(image.Rect(0, cy, w, cy+ch))

		// Draw everything
		frame, err := terminal.BeginFrame()
		if err != nil {
			return
		}

		layout.DrawTo(frame)
		mainContainer.Draw(frame)

		terminal.EndFrame(frame)
	}

	// Register resize callback
	terminal.OnResize(func(width, height int) {
		resizeCount++
		layout.SetFooter(&gooey.Footer{
			StatusBar: true,
			StatusItems: []gooey.StatusItem{
				{Key: "Resizes", Value: fmt.Sprintf("%d", resizeCount), Icon: "📏"},
				{Key: "Size", Value: fmt.Sprintf("%dx%d", width, height), Icon: "📐"},
				{Key: "Controls", Value: "Press Ctrl+C to exit", Icon: "⌨"},
			},
			Border:      true,
			BorderStyle: gooey.RoundedBorder,
			Height:      3,
		})
		updateDisplay()
	})

	// Initial render
	updateDisplay()

	// Main loop - just wait for Ctrl+C
	input := gooey.NewInput(terminal)
	for {
		event := input.ReadKeyEvent()
		if event.Key == gooey.KeyCtrlC {
			return
		}
		// Could handle button clicks here with mouse events if needed
	}
}
