package main

import (
	"fmt"
	"image"
	"log"

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

// GridApp demonstrates the Grid layout system using the Runtime.
type GridApp struct {
	grid        *gooey.Grid
	resizeCount int
	width       int
	height      int
}

// Init initializes the grid layout.
func (app *GridApp) Init() error {
	// Create the Grid layout with 3 columns of equal weight
	app.grid = gooey.NewGrid(nil). // Pass nil since we'll use the frame directly
					AddCol(0, 1). // Flexible column, takes 1/3 of width
					AddCol(0, 1). // Flexible column, takes 1/3 of width
					AddCol(0, 1)  // Flexible column, takes 1/3 of width

	// Add 3 rows: header, content, footer with proportional heights
	app.grid.AddRow(0, 1). // Flexible row for header (takes 1/6 of height)
				AddRow(0, 4). // Flexible row for main content area (takes 4/6 of height)
				AddRow(0, 1)  // Flexible row for footer (takes 1/6 of height)

	// Row 0: Header spanning all 3 columns
	_ = app.grid.AddWidgetSpan(&SimpleBox{
		Text:  "HEADER - Grid Layout Demo (Spans 3 Columns)",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite).WithBold(),
	}, 0, 0, 1, 3) // row=0, col=0, rowspan=1, colspan=3

	// Row 1: Content Area spanning 2 columns, and Sidebar taking 1 column
	_ = app.grid.AddWidgetSpan(&SimpleBox{
		Text:  "MAIN CONTENT AREA (Spans 2 Columns)",
		Style: gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorWhite).WithBold(),
	}, 1, 0, 1, 2) // row=1, col=0, rowspan=1, colspan=2

	_ = app.grid.AddWidget(&SimpleBox{
		Text:  "SIDEBAR - Navigation & Tools",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBrightMagenta).WithForeground(gooey.ColorWhite).WithBold(),
	}, 1, 2) // row=1, col=2

	// Row 2: Footer with 3 sections
	_ = app.grid.AddWidget(&SimpleBox{
		Text:  "Footer: Status",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBrightBlue).WithForeground(gooey.ColorWhite),
	}, 2, 0)

	_ = app.grid.AddWidget(&SimpleBox{
		Text:  "Footer: Info & Help",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBrightCyan).WithForeground(gooey.ColorBlack),
	}, 2, 1)

	_ = app.grid.AddWidget(&SimpleBox{
		Text:  "Footer: Version 1.0",
		Style: gooey.NewStyle().WithBackground(gooey.ColorBrightYellow).WithForeground(gooey.ColorBlack),
	}, 2, 2)

	return nil
}

// HandleEvent processes events from the runtime.
func (app *GridApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Exit on Ctrl+C or Escape
		if e.Key == gooey.KeyCtrlC || e.Key == gooey.KeyEscape {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		// Update stored dimensions
		app.width = e.Width
		app.height = e.Height
		app.resizeCount++
	}

	return nil
}

// Render draws the grid layout.
func (app *GridApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Update dimensions if not set
	if app.width == 0 || app.height == 0 {
		app.width = width
		app.height = height
	}

	// Draw the grid to fill the entire terminal
	app.grid.Draw(frame)

	// Draw resize indicator in bottom-right corner
	sizeText := fmt.Sprintf("%dx%d", width, height)
	textWidth := runewidth.StringWidth(sizeText)
	frame.PrintStyled(width-textWidth-1, height-1, sizeText,
		gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
}

func main() {
	if err := gooey.Run(&GridApp{}); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Demo finished.")
}
