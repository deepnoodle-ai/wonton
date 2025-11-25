package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// LayoutStylingDemo demonstrates different layout and styling options
// using the Runtime message-driven architecture.
//
// This example shows:
// - Simple header and footer layouts
// - Bordered layouts with different border styles
// - Status bar footers with icons and colors
// - Automatic layout updates based on terminal size
type LayoutStylingDemo struct {
	terminal *gooey.Terminal
	layout   *gooey.Layout
	stage    int
}

func NewLayoutStylingDemo(terminal *gooey.Terminal) *LayoutStylingDemo {
	return &LayoutStylingDemo{
		terminal: terminal,
		layout:   gooey.NewLayout(terminal),
		stage:    0,
	}
}

// Init implements the Initializable interface
func (d *LayoutStylingDemo) Init() error {
	// Clear screen and start with stage 1
	d.terminal.Clear()
	d.showStage1()

	// Schedule transition to next stage after 3 seconds
	return nil
}

// Destroy implements the Destroyable interface
func (d *LayoutStylingDemo) Destroy() {
	// Nothing to clean up
}

// HandleEvent processes events from the runtime
func (d *LayoutStylingDemo) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Key == gooey.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.TickEvent:
		// Every 90 frames (3 seconds at 30 FPS), advance to next stage
		if e.Frame%90 == 0 && e.Frame > 0 {
			d.stage++
			switch d.stage {
			case 1:
				d.showStage2()
			case 2:
				d.showStage3()
			case 3:
				// Demo complete, exit after brief pause
				return []gooey.Cmd{gooey.After(500*time.Millisecond, func() {
					// Quit will be handled by the next event
				}), gooey.Quit()}
			}
		}

	case gooey.ResizeEvent:
		// Layout automatically handles resize
		return nil
	}

	return nil
}

// Render draws the current application state
func (d *LayoutStylingDemo) Render(frame gooey.RenderFrame) {
	// Layout handles all rendering, we just need to make sure it's drawn
	d.layout.DrawTo(frame)
}

func (d *LayoutStylingDemo) showStage1() {
	// 1. Simple Header & Footer
	d.terminal.Clear()
	d.layout.SetHeader(gooey.SimpleHeader(" Simple Layout ", gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)))
	d.layout.SetFooter(gooey.SimpleFooter("Left", "Center", "Right", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)))

	d.layout.Draw()
	d.layout.PrintInContent("\n  This is the Simple Header & Footer layout.\n  Notice the single line header and footer.\n\n  Switching in 3 seconds...")
	d.terminal.Flush()
}

func (d *LayoutStylingDemo) showStage2() {
	// 2. Bordered Header & Footer
	headerStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	d.layout.SetHeader(gooey.BorderedHeader(" Bordered Layout ", headerStyle))

	footerStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	footer := gooey.SimpleFooter("Status: Active", "Page 1/3", "Press Ctrl+C to quit", footerStyle)
	footer.Border = true
	footer.BorderStyle = gooey.DoubleBorder
	d.layout.SetFooter(footer)

	d.layout.Draw()
	d.layout.PrintInContent("\n  Now using Bordered Header and Footer.\n  Headers and footers can have borders for better separation.\n\n  Switching in 3 seconds...")
	d.terminal.Flush()
}

func (d *LayoutStylingDemo) showStage3() {
	// 3. Status Bar Footer
	d.layout.SetHeader(gooey.SimpleHeader(" Status Bar Demo ", gooey.NewStyle().WithBold().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)))

	statusItems := []gooey.StatusItem{
		{Key: "Branch", Value: "main", Icon: "ᛘ", Style: gooey.NewStyle().WithForeground(gooey.ColorMagenta)},
		{Key: "File", Value: "layout.go", Icon: "📄", Style: gooey.NewStyle().WithForeground(gooey.ColorCyan)},
		{Key: "Ln", Value: "42", Style: gooey.NewStyle().WithForeground(gooey.ColorYellow)},
		{Key: "Col", Value: "12", Style: gooey.NewStyle().WithForeground(gooey.ColorYellow)},
		{Value: "UTF-8", Style: gooey.NewStyle().WithDim()},
	}

	footer := gooey.StatusBarFooter(statusItems)
	footer.Background = gooey.NewStyle().WithBackground(gooey.ColorBrightBlack) // Dark gray background
	d.layout.SetFooter(footer)

	d.layout.Draw()
	d.layout.PrintInContent("\n  This layout features a Status Bar Footer.\n  Common in editors like Vim or VS Code.\n  Note the background color and icons.\n\n  Demo complete. Exiting shortly...")
	d.terminal.Flush()
}

func main() {
	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create application
	app := NewLayoutStylingDemo(terminal)

	// Create and run runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)

	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
