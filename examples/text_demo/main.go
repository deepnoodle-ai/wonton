package main

import (
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// newWrappedText creates a canvas view that demonstrates text wrapping, alignment, and truncation.
func newWrappedText(text string, wrap, truncate bool, align gooey.Alignment, style gooey.Style) gooey.View {
	return gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
		if bounds.Empty() {
			return
		}

		width := bounds.Dx()

		displayText := text
		if wrap {
			displayText = gooey.WrapText(displayText, width)
		}

		// Align text within the width
		displayText = gooey.AlignText(displayText, width, align)

		// Fill background
		frame.Fill(' ', style)

		// Render text
		if truncate {
			frame.PrintTruncated(0, 0, displayText, style)
		} else {
			frame.PrintStyled(0, 0, displayText, style)
		}
	})
}

// TextDemoApp demonstrates text wrapping and alignment capabilities.
type TextDemoApp struct{}

// HandleEvent processes events from the runtime.
func (app *TextDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Key == gooey.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}
	}
	return nil
}

// View returns the declarative UI for this app.
func (app *TextDemoApp) View() gooey.View {
	longText := "This is a very long sentence that should automatically wrap when it reaches the boundary of the container. It serves as a demonstration of the text wrapping capability."

	return gooey.VStack(
		// Header
		gooey.Text("Text Wrapping & Alignment Demo").Bold().Fg(gooey.ColorCyan),
		gooey.Text("Press Q or Ctrl+C to exit").Fg(gooey.ColorWhite),
		gooey.Height(1, gooey.Fill('─').Fg(gooey.ColorBrightBlack)),

		// 2x2 Grid using nested HStack and VStack
		gooey.VStack(
			// Top row
			gooey.HStack(
				// Top Left: Wrapped, Left Aligned
				newWrappedText(
					"WRAPPED LEFT:\n"+longText,
					true,
					false,
					gooey.AlignLeft,
					gooey.NewStyle().WithForeground(gooey.ColorWhite).WithBackground(gooey.ColorBlue),
				),

				// Top Right: Wrapped, Center Aligned
				newWrappedText(
					"WRAPPED CENTER:\n"+longText,
					true,
					false,
					gooey.AlignCenter,
					gooey.NewStyle().WithForeground(gooey.ColorBlack).WithBackground(gooey.ColorGreen),
				),
			),

			// Bottom row
			gooey.HStack(
				// Bottom Left: Wrapped, Right Aligned
				newWrappedText(
					"WRAPPED RIGHT:\n"+longText,
					true,
					false,
					gooey.AlignRight,
					gooey.NewStyle().WithForeground(gooey.ColorWhite).WithBackground(gooey.ColorRed),
				),

				// Bottom Right: Truncated (clipped at edge, no wrapping), Center Aligned
				newWrappedText(
					"TRUNCATED (Clipped at edge):\n"+longText,
					false,
					true,
					gooey.AlignCenter,
					gooey.NewStyle().WithForeground(gooey.ColorBlack).WithBackground(gooey.ColorYellow),
				),
			),
		),
	)
}

func main() {
	// Run the application (30 FPS is default)
	if err := gooey.Run(&TextDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
