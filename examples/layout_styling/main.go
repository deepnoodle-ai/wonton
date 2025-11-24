package main

import (
	"fmt"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error initializing terminal: %v\n", err)
		return
	}
	defer terminal.Close()

	// Enable automatic resize handling
	terminal.WatchResize()
	defer terminal.StopWatchResize()

	layout := gooey.NewLayout(terminal)

	// 1. Simple Header & Footer
	terminal.Clear()
	layout.SetHeader(gooey.SimpleHeader(" Simple Layout ", gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)))
	layout.SetFooter(gooey.SimpleFooter("Left", "Center", "Right", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)))

	layout.Draw()
	layout.PrintInContent("\n  This is the Simple Header & Footer layout.\n  Notice the single line header and footer.\n\n  Switching in 3 seconds...")
	terminal.Flush()
	time.Sleep(3 * time.Second)

	// 2. Bordered Header & Footer
	headerStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	layout.SetHeader(gooey.BorderedHeader(" Bordered Layout ", headerStyle))

	footerStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	footer := gooey.SimpleFooter("Status: Active", "Page 1/3", "Press Ctrl+C to quit", footerStyle)
	footer.Border = true
	footer.BorderStyle = gooey.DoubleBorder
	layout.SetFooter(footer)

	layout.Draw()
	layout.PrintInContent("\n  Now using Bordered Header and Footer.\n  Headers and footers can have borders for better separation.\n\n  Switching in 3 seconds...")
	terminal.Flush()
	time.Sleep(3 * time.Second)

	// 3. Status Bar Footer
	layout.SetHeader(gooey.SimpleHeader(" Status Bar Demo ", gooey.NewStyle().WithBold().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)))

	statusItems := []gooey.StatusItem{
		{Key: "Branch", Value: "main", Icon: "ᛘ", Style: gooey.NewStyle().WithForeground(gooey.ColorMagenta)},
		{Key: "File", Value: "layout.go", Icon: "📄", Style: gooey.NewStyle().WithForeground(gooey.ColorCyan)},
		{Key: "Ln", Value: "42", Style: gooey.NewStyle().WithForeground(gooey.ColorYellow)},
		{Key: "Col", Value: "12", Style: gooey.NewStyle().WithForeground(gooey.ColorYellow)},
		{Value: "UTF-8", Style: gooey.NewStyle().WithDim()},
	}

	footer = gooey.StatusBarFooter(statusItems)
	footer.Background = gooey.NewStyle().WithBackground(gooey.ColorBrightBlack) // Dark gray background
	layout.SetFooter(footer)

	layout.Draw()
	layout.PrintInContent("\n  This layout features a Status Bar Footer.\n  Common in editors like Vim or VS Code.\n  Note the background color and icons.\n\n  Demo complete. Exiting in 3 seconds...")
	terminal.Flush()
	time.Sleep(3 * time.Second)
}
