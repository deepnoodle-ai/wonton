package main

import (
	"fmt"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

type TextWidget struct {
	Text     string
	Wrap     bool // If true, use WrapText() to insert newlines at word boundaries
	Truncate bool // If true, use PrintTruncated() to clip at edge (no wrapping)
	Align    gooey.Alignment
	Style    gooey.Style
}

func (tw *TextWidget) Draw(frame gooey.RenderFrame) {
	bounds := frame.GetBounds()
	width := bounds.Dx()

	displayText := tw.Text
	if tw.Wrap {
		displayText = gooey.WrapText(displayText, width)
	}

	// If wrapping resulted in multiple lines, AlignText will align each line.
	// However, AlignText also pads to full width, which effectively fills the background.
	displayText = gooey.AlignText(displayText, width, tw.Align)

	// IMPORTANT: Always use local coordinates (0, 0) when drawing to a frame.
	// Fill() is a convenience method that fills the entire frame.
	frame.Fill(' ', tw.Style)

	// Choose between PrintStyled (wraps at edge) and PrintTruncated (clips at edge)
	if tw.Truncate {
		// PrintTruncated clips text at frame edge without wrapping
		frame.PrintTruncated(0, 0, displayText, tw.Style)
	} else {
		// PrintStyled auto-wraps text at frame edge (default terminal behavior)
		frame.PrintStyled(0, 0, displayText, tw.Style)
	}
}

func (tw *TextWidget) HandleKey(event gooey.KeyEvent) bool {
	return false
}

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error creating terminal: %v\n", err)
		return
	}
	defer terminal.Close()

	screen := gooey.NewScreen(terminal)

	// Layout
	grid := gooey.NewGrid(terminal).
		AddCol(0, 1). // Left column
		AddCol(0, 1)  // Right column

	grid.AddRow(0, 1). // Top row
				AddRow(0, 1) // Bottom row

	longText := "This is a very long sentence that should automatically wrap when it reaches the boundary of the container. It serves as a demonstration of the text wrapping capability."

	// Top Left: Wrapped, Left Aligned
	grid.AddWidget(&TextWidget{
		Text:  "WRAPPED LEFT:\n" + longText,
		Wrap:  true,
		Align: gooey.AlignLeft,
		Style: gooey.NewStyle().WithForeground(gooey.ColorWhite).WithBackground(gooey.ColorBlue),
	}, 0, 0)

	// Top Right: Wrapped, Center Aligned
	grid.AddWidget(&TextWidget{
		Text:  "WRAPPED CENTER:\n" + longText,
		Wrap:  true,
		Align: gooey.AlignCenter,
		Style: gooey.NewStyle().WithForeground(gooey.ColorBlack).WithBackground(gooey.ColorGreen),
	}, 0, 1)

	// Bottom Left: Wrapped, Right Aligned
	grid.AddWidget(&TextWidget{
		Text:  "WRAPPED RIGHT:\n" + longText,
		Wrap:  true,
		Align: gooey.AlignRight,
		Style: gooey.NewStyle().WithForeground(gooey.ColorWhite).WithBackground(gooey.ColorRed),
	}, 1, 0)

	// Bottom Right: Truncated (clipped at edge, no wrapping), Center Aligned
	grid.AddWidget(&TextWidget{
		Text:     "TRUNCATED (Clipped at edge):\n" + longText,
		Wrap:     false,
		Truncate: true,
		Align:    gooey.AlignCenter,
		Style:    gooey.NewStyle().WithForeground(gooey.ColorBlack).WithBackground(gooey.ColorYellow),
	}, 1, 1)

	screen.AddWidget(grid)
	screen.SetLayout(gooey.NewLayout(terminal).SetHeader(gooey.SimpleHeader("Text Wrapping & Alignment Demo", gooey.NewStyle().WithBold())))

	terminal.WatchResize()
	defer terminal.StopWatchResize()

	go func() {
		time.Sleep(10 * time.Second)
		screen.Stop()
	}()

	screen.Run()
}
