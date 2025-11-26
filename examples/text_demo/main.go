package main

import (
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// TextWidget demonstrates text wrapping, alignment, and truncation.
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

// TextDemoApp demonstrates text wrapping and alignment capabilities.
type TextDemoApp struct {
	grid   *gooey.Grid
	layout *gooey.Layout
	header *gooey.Header
}

// Init initializes the application.
func (app *TextDemoApp) Init() error {
	return nil
}

// HandleEvent processes events from the runtime.
func (app *TextDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle keyboard input
		if e.Key == gooey.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		// Grid will automatically adapt to new size in Render
		return nil
	}

	return nil
}

// Render draws the current application state.
func (app *TextDemoApp) Render(frame gooey.RenderFrame) {
	// Lazy initialization on first render
	if app.grid == nil {
		app.initializeComponents()
	}

	// Draw header and grid
	if app.layout != nil && app.header != nil {
		// Draw header manually since we're not using the full Layout system
		headerStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
		frame.PrintStyled(0, 0, "Text Wrapping & Alignment Demo", headerStyle)
		frame.PrintStyled(0, 1, "Press Q or Ctrl+C to exit", gooey.NewStyle().WithForeground(gooey.ColorWhite))

		// Draw separator
		width, _ := frame.Size()
		separator := ""
		for i := 0; i < width; i++ {
			separator += "─"
		}
		frame.PrintStyled(0, 2, separator, gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

		// Create a sub-frame for the grid (below header)
		gridFrame := frame.SubFrame(frame.GetBounds().Add(frame.GetBounds().Min).Inset(0))
		// Adjust the grid frame to start below the header (3 lines)
		bounds := gridFrame.GetBounds()
		gridBounds := bounds
		gridBounds.Min.Y += 3
		gridFrame = frame.SubFrame(gridBounds)

		app.grid.Draw(gridFrame)
	} else {
		app.grid.Draw(frame)
	}
}

// initializeComponents creates the grid and widgets.
func (app *TextDemoApp) initializeComponents() {
	app.layout = &gooey.Layout{}
	app.header = &gooey.Header{
		Center: "Text Wrapping & Alignment Demo",
		Style:  gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan),
	}

	// Create grid with 2x2 layout
	app.grid = &gooey.Grid{}
	app.grid.AddCol(0, 1).AddCol(0, 1) // Two equal columns
	app.grid.AddRow(0, 1).AddRow(0, 1) // Two equal rows

	longText := "This is a very long sentence that should automatically wrap when it reaches the boundary of the container. It serves as a demonstration of the text wrapping capability."

	// Top Left: Wrapped, Left Aligned
	app.grid.AddWidget(&TextWidget{
		Text:  "WRAPPED LEFT:\n" + longText,
		Wrap:  true,
		Align: gooey.AlignLeft,
		Style: gooey.NewStyle().WithForeground(gooey.ColorWhite).WithBackground(gooey.ColorBlue),
	}, 0, 0)

	// Top Right: Wrapped, Center Aligned
	app.grid.AddWidget(&TextWidget{
		Text:  "WRAPPED CENTER:\n" + longText,
		Wrap:  true,
		Align: gooey.AlignCenter,
		Style: gooey.NewStyle().WithForeground(gooey.ColorBlack).WithBackground(gooey.ColorGreen),
	}, 0, 1)

	// Bottom Left: Wrapped, Right Aligned
	app.grid.AddWidget(&TextWidget{
		Text:  "WRAPPED RIGHT:\n" + longText,
		Wrap:  true,
		Align: gooey.AlignRight,
		Style: gooey.NewStyle().WithForeground(gooey.ColorWhite).WithBackground(gooey.ColorRed),
	}, 1, 0)

	// Bottom Right: Truncated (clipped at edge, no wrapping), Center Aligned
	app.grid.AddWidget(&TextWidget{
		Text:     "TRUNCATED (Clipped at edge):\n" + longText,
		Wrap:     false,
		Truncate: true,
		Align:    gooey.AlignCenter,
		Style:    gooey.NewStyle().WithForeground(gooey.ColorBlack).WithBackground(gooey.ColorYellow),
	}, 1, 1)
}

func main() {
	// Create application
	app := &TextDemoApp{}

	// Run the application (30 FPS is default)
	if err := gooey.Run(app); err != nil {
		log.Fatal(err)
	}
}
