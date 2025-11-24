package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	// Create terminal
	term, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer term.Close()

	// Enable raw mode and alternate screen
	if err := term.EnableRawMode(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to enable raw mode: %v\n", err)
		os.Exit(1)
	}
	term.EnableAlternateScreen()

	// Get terminal size
	width, height := term.Size()

	// Create frame
	frame, err := term.BeginFrame()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to begin frame: %v\n", err)
		os.Exit(1)
	}

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Title
	title := "OSC 8 Hyperlink Support Demo"
	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	frame.PrintStyled((width-len(title))/2, 2, title, titleStyle)

	// Description
	desc := "Click the links below (if your terminal supports OSC 8)"
	descStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	frame.PrintStyled((width-len(desc))/2, 3, desc, descStyle)

	// Draw a separator
	separator := "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	sepStyle := gooey.NewStyle().WithForeground(gooey.ColorBlue).WithDim()
	frame.PrintStyled((width-len(separator))/2, 5, separator, sepStyle)

	y := 7

	// Example 1: Default styled hyperlink
	label1 := "1. Default styled link: "
	frame.PrintStyled(4, y, label1, gooey.NewStyle())
	link1 := gooey.NewHyperlink("https://github.com/myzie/gooey", "Gooey on GitHub")
	frame.PrintHyperlink(4+len(label1), y, link1)
	y += 2

	// Example 2: Custom styled hyperlink
	label2 := "2. Custom styled link: "
	frame.PrintStyled(4, y, label2, gooey.NewStyle())
	link2 := gooey.NewHyperlink("https://example.com", "Example.com")
	customStyle := gooey.NewStyle().WithForeground(gooey.ColorMagenta).WithBold().WithUnderline()
	link2 = link2.WithStyle(customStyle)
	frame.PrintHyperlink(4+len(label2), y, link2)
	y += 2

	// Example 3: Multiple links on same line
	label3 := "3. Multiple links: "
	frame.PrintStyled(4, y, label3, gooey.NewStyle())
	x := 4 + len(label3)

	linkA := gooey.NewHyperlink("https://go.dev", "Go")
	frame.PrintHyperlink(x, y, linkA)
	x += len("Go") + 3

	frame.PrintStyled(x, y, " | ", gooey.NewStyle())
	x += 3

	linkB := gooey.NewHyperlink("https://github.com", "GitHub")
	frame.PrintHyperlink(x, y, linkB)
	x += len("GitHub") + 3

	frame.PrintStyled(x, y, " | ", gooey.NewStyle())
	x += 3

	linkC := gooey.NewHyperlink("https://stackoverflow.com", "Stack Overflow")
	frame.PrintHyperlink(x, y, linkC)
	y += 2

	// Example 4: Link with emoji
	label4 := "4. Link with emoji: "
	frame.PrintStyled(4, y, label4, gooey.NewStyle())
	link4 := gooey.NewHyperlink("https://www.anthropic.com", "🤖 Anthropic")
	frame.PrintHyperlink(4+len(label4), y, link4)
	y += 2

	// Example 5: Fallback format (showing URL)
	label5 := "5. Fallback format: "
	frame.PrintStyled(4, y, label5, gooey.NewStyle())
	link5 := gooey.NewHyperlink("https://golang.org", "Go Programming")
	frame.PrintHyperlinkFallback(4+len(label5), y, link5)
	y += 2

	// Example 6: Long URL
	label6 := "6. Long URL: "
	frame.PrintStyled(4, y, label6, gooey.NewStyle())
	link6 := gooey.NewHyperlink(
		"https://example.com/very/long/path/to/resource?with=many&query=params&and=more#section",
		"Complex URL",
	)
	frame.PrintHyperlink(4+len(label6), y, link6)
	y += 2

	// Example 7: Different link styles
	label7 := "7. Different styles: "
	frame.PrintStyled(4, y, label7, gooey.NewStyle())

	x = 4 + len(label7)

	// Red link
	redLink := gooey.NewHyperlink("https://example.com/red", "Red").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorRed).WithUnderline())
	frame.PrintHyperlink(x, y, redLink)
	x += len("Red") + 2

	// Green link
	greenLink := gooey.NewHyperlink("https://example.com/green", "Green").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen).WithUnderline())
	frame.PrintHyperlink(x, y, greenLink)
	x += len("Green") + 2

	// Yellow link
	yellowLink := gooey.NewHyperlink("https://example.com/yellow", "Yellow").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow).WithUnderline())
	frame.PrintHyperlink(x, y, yellowLink)
	y += 2

	// Example 8: Links in a table-like layout
	frame.PrintStyled(4, y, "8. Table of links:", gooey.NewStyle().WithBold())
	y += 1

	type Resource struct {
		name string
		url  string
	}

	resources := []Resource{
		{"Documentation", "https://pkg.go.dev"},
		{"Source Code", "https://github.com"},
		{"Issue Tracker", "https://github.com/issues"},
		{"Community", "https://www.reddit.com/r/golang"},
	}

	for _, res := range resources {
		// Print resource name in a fixed width column
		nameStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
		frame.PrintStyled(7, y, fmt.Sprintf("%-20s", res.name), nameStyle)

		// Print link
		link := gooey.NewHyperlink(res.url, res.url)
		frame.PrintHyperlink(28, y, link)
		y += 1
	}

	y += 1

	// Information box at the bottom
	infoY := height - 6
	frame.PrintStyled(4, infoY, "Terminal Support Information:", gooey.NewStyle().WithBold())
	infoY++

	supportedTerms := "✓ Supported: iTerm2, WezTerm, kitty, foot, Rio, and others"
	frame.PrintStyled(4, infoY, supportedTerms, gooey.NewStyle().WithForeground(gooey.ColorGreen))
	infoY++

	unsupportedTerms := "✗ Not supported: Most terminals will show the text without clickable links"
	frame.PrintStyled(4, infoY, unsupportedTerms, gooey.NewStyle().WithForeground(gooey.ColorYellow))
	infoY++

	fallbackNote := "  (OSC 8 escape codes are ignored, text displays normally)"
	frame.PrintStyled(4, infoY, fallbackNote, gooey.NewStyle().WithForeground(gooey.ColorWhite).WithDim())

	// Footer
	footer := "Press Ctrl+C to exit"
	footerStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
	frame.PrintStyled((width-len(footer))/2, height-1, footer, footerStyle)

	// End frame
	if err := term.EndFrame(frame); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to end frame: %v\n", err)
		os.Exit(1)
	}

	// Wait for a while to allow viewing
	time.Sleep(30 * time.Second)
}
