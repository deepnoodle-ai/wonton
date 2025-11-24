package main

import (
	"fmt"
	"image"

	"github.com/deepnoodle-ai/gooey"
	"github.com/mattn/go-runewidth"
)

// SimpleBox is a basic widget that draws a colored box with text.
type SimpleBox struct {
	Text   string
	Style  gooey.Style
	bounds image.Rectangle
}

// Draw implements the Widget interface for SimpleBox.
func (sb *SimpleBox) Draw(frame gooey.RenderFrame) {
	sb.bounds = frame.GetBounds()
	width, height := sb.bounds.Dx(), sb.bounds.Dy()

	// IMPORTANT: When drawing to a frame (especially a SubFrame), always use
	// coordinates relative to (0, 0). The frame automatically handles translation
	// to the correct screen position.
	//
	// Using Fill() is preferred over FillStyled(0, 0, width, height, ...)
	frame.Fill(' ', sb.Style)

	// Print text centered
	if sb.Text != "" {
		textWidth := runewidth.StringWidth(sb.Text)
		if textWidth > width {
			textWidth = width // Clip text if too long
		}
		textX := (width - textWidth) / 2
		textY := (height - 1) / 2 // Center vertically
		frame.PrintStyled(textX, textY, sb.Text, sb.Style)
	}
}

// HandleKey implements the Widget interface for SimpleBox.
func (sb *SimpleBox) HandleKey(event gooey.KeyEvent) bool {
	// SimpleBox doesn't handle any keys
	return false
}

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error creating terminal: %v\n", err)
		return
	}
	defer terminal.Reset()

	// Enable raw mode and alternate screen for interactive demo
	terminal.EnableRawMode()
	defer terminal.DisableRawMode()
	terminal.EnableAlternateScreen()
	defer terminal.DisableAlternateScreen()
	terminal.HideCursor()
	defer terminal.ShowCursor()

	// Enable resize watching for full terminal window resizing support
	terminal.WatchResize()
	defer terminal.StopWatchResize()

	// Terminal size will be determined dynamically in draw function

	// Create the Grid layout with 3 columns of equal weight
	grid := gooey.NewGrid(terminal).
		AddCol(0, 1). // Flexible column, takes 1/3 of width
		AddCol(0, 1). // Flexible column, takes 1/3 of width
		AddCol(0, 1)  // Flexible column, takes 1/3 of width

	// Add 3 rows: header, content, footer with proportional heights
	grid.AddRow(0, 1). // Flexible row for header (takes 1/6 of height)
				AddRow(0, 4). // Flexible row for main content area (takes 4/6 of height)
				AddRow(0, 1)  // Flexible row for footer (takes 1/6 of height)

	// Row 0: Header spanning all 3 columns
	_ = grid.AddWidgetSpan(&SimpleBox{
		Text:  "HEADER - Grid Layout Demo (Spans 3 Columns)",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite).WithBold(),
	}, 0, 0, 1, 3) // row=0, col=0, rowspan=1, colspan=3

	// Row 1: Content Area spanning 2 columns, and Sidebar taking 1 column
	_ = grid.AddWidgetSpan(&SimpleBox{
		Text:  "MAIN CONTENT AREA (Spans 2 Columns)",
		Style: gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorWhite).WithBold(),
	}, 1, 0, 1, 2) // row=1, col=0, rowspan=1, colspan=2

	_ = grid.AddWidget(&SimpleBox{
		Text:  "SIDEBAR - Navigation & Tools",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBrightMagenta).WithForeground(gooey.ColorWhite).WithBold(),
	}, 1, 2) // row=1, col=2

	// Row 2: Footer with 3 sections
	_ = grid.AddWidget(&SimpleBox{
		Text:  "Footer: Status",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBrightBlue).WithForeground(gooey.ColorWhite),
	}, 2, 0)

	_ = grid.AddWidget(&SimpleBox{
		Text:  "Footer: Info & Help",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBrightCyan).WithForeground(gooey.ColorBlack),
	}, 2, 1)

	_ = grid.AddWidget(&SimpleBox{
		Text:  "Footer: Version 1.0",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBrightYellow).WithForeground(gooey.ColorBlack),
	}, 2, 2)

	// Draw function that handles the current terminal size
	drawGrid := func() {
		// Get current terminal size
		currentWidth, currentHeight := terminal.Size()

		// Create frame and draw the grid to fill the entire terminal
		frame, err := terminal.BeginFrame()
		if err != nil {
			return
		}

		// Set grid bounds to fill entire terminal
		grid.Draw(frame)

		// Draw resize indicator in bottom-right corner
		sizeText := fmt.Sprintf("%dx%d", currentWidth, currentHeight)
		textWidth := runewidth.StringWidth(sizeText)
		frame.PrintStyled(currentWidth-textWidth-1, currentHeight-1, sizeText,
			gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

		terminal.EndFrame(frame)
	}

	// Register resize callback to redraw when terminal is resized
	resizeCount := 0
	unregisterResize := terminal.OnResize(func(width, height int) {
		resizeCount++
		// Redraw immediately when resized
		drawGrid()
	})
	defer unregisterResize()

	// Initial draw
	drawGrid()

	// Main event loop
	input := gooey.NewInput(terminal)
	for {
		event := input.ReadKeyEvent()

		// Exit on Ctrl+C or Escape
		if event.Key == gooey.KeyCtrlC || event.Key == gooey.KeyEscape {
			break
		}

		// Handle any other keys if needed (grid widgets don't handle keys in this demo)

		// Redraw after each key event (in case widgets need updating)
		drawGrid()
	}

	fmt.Println("Demo finished.")
}
