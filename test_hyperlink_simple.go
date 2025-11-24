//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	gooey "github.com/deepnoodle-ai/gooey"
)

func main() {
	term, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer term.Close()

	term.EnableRawMode()
	term.EnableAlternateScreen()

	width, height := term.Size()

	frame, _ := term.BeginFrame()
	frame.Fill(' ', gooey.NewStyle())

	// Title
	title := "Hyperlink Quick Test"
	frame.PrintStyled((width-len(title))/2, 2, title, gooey.NewStyle().WithBold())

	// Simple link
	link1 := gooey.NewHyperlink("https://github.com", "GitHub")
	frame.PrintHyperlink(5, 5, link1)

	// Custom styled link
	link2 := gooey.NewHyperlink("https://example.com", "Example.com")
	link2 = link2.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorMagenta).WithUnderline())
	frame.PrintHyperlink(5, 7, link2)

	// Fallback format
	link3 := gooey.NewHyperlink("https://golang.org", "Go")
	frame.PrintHyperlinkFallback(5, 9, link3)

	// Footer
	footer := "Links should be clickable (if your terminal supports OSC 8)"
	frame.PrintStyled((width-len(footer))/2, height-3, footer, gooey.NewStyle().WithForeground(gooey.ColorCyan))

	term.EndFrame(frame)

	// Wait for keypress
	fmt.Println("\nPress Enter to exit...")
	fmt.Scanln()
}
