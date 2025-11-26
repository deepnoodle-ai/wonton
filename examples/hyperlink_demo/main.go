package main

import (
	"fmt"
	"image"
	"log"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// HyperlinkApp demonstrates OSC 8 hyperlink support using the declarative View API.
type HyperlinkApp struct {
	startTime  time.Time
	autoQuitAt time.Time
}

// Init initializes the application.
func (app *HyperlinkApp) Init() error {
	app.startTime = time.Now()
	app.autoQuitAt = app.startTime.Add(30 * time.Second)
	return nil
}

// HandleEvent processes events from the runtime.
func (app *HyperlinkApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch event.(type) {
	case gooey.KeyEvent:
		// Exit on any key press or Ctrl+C
		return []gooey.Cmd{gooey.Quit()}

	case gooey.TickEvent:
		// Auto-quit after 30 seconds
		if time.Now().After(app.autoQuitAt) {
			return []gooey.Cmd{gooey.Quit()}
		}
	}

	return nil
}

// View returns the declarative view hierarchy.
func (app *HyperlinkApp) View() gooey.View {
	// Calculate remaining time for footer
	elapsed := time.Since(app.startTime)
	remaining := 30*time.Second - elapsed
	if remaining < 0 {
		remaining = 0
	}

	return gooey.VStack(
		gooey.Spacer().MinHeight(1),
		// Title and description
		gooey.Text("OSC 8 Hyperlink Support Demo").Bold().Fg(gooey.ColorCyan),
		gooey.Text("Click the links below (if your terminal supports OSC 8)").Fg(gooey.ColorWhite),
		gooey.Spacer().MinHeight(1),
		// Separator
		gooey.Text("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━").Fg(gooey.ColorBlue).Dim(),
		gooey.Spacer().MinHeight(1),

		// Example 1: Default styled hyperlink
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			label := "1. Default styled link: "
			frame.PrintStyled(4, 0, label, gooey.NewStyle())
			link := gooey.NewHyperlink("https://github.com/myzie/gooey", "Gooey on GitHub")
			frame.PrintHyperlink(4+len(label), 0, link)
		}),
		gooey.Spacer().MinHeight(1),

		// Example 2: Custom styled hyperlink
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			label := "2. Custom styled link: "
			frame.PrintStyled(4, 0, label, gooey.NewStyle())
			link := gooey.NewHyperlink("https://example.com", "Example.com")
			customStyle := gooey.NewStyle().WithForeground(gooey.ColorMagenta).WithBold().WithUnderline()
			link = link.WithStyle(customStyle)
			frame.PrintHyperlink(4+len(label), 0, link)
		}),
		gooey.Spacer().MinHeight(1),

		// Example 3: Multiple links on same line
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			label := "3. Multiple links: "
			frame.PrintStyled(4, 0, label, gooey.NewStyle())
			x := 4 + len(label)

			linkA := gooey.NewHyperlink("https://go.dev", "Go")
			frame.PrintHyperlink(x, 0, linkA)
			x += len("Go")

			frame.PrintStyled(x, 0, " | ", gooey.NewStyle())
			x += 3

			linkB := gooey.NewHyperlink("https://github.com", "GitHub")
			frame.PrintHyperlink(x, 0, linkB)
			x += len("GitHub")

			frame.PrintStyled(x, 0, " | ", gooey.NewStyle())
			x += 3

			linkC := gooey.NewHyperlink("https://stackoverflow.com", "Stack Overflow")
			frame.PrintHyperlink(x, 0, linkC)
		}),
		gooey.Spacer().MinHeight(1),

		// Example 4: Link with emoji
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			label := "4. Link with emoji: "
			frame.PrintStyled(4, 0, label, gooey.NewStyle())
			link := gooey.NewHyperlink("https://www.anthropic.com", "🤖 Anthropic")
			frame.PrintHyperlink(4+len(label), 0, link)
		}),
		gooey.Spacer().MinHeight(1),

		// Example 5: Fallback format (showing URL)
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			label := "5. Fallback format: "
			frame.PrintStyled(4, 0, label, gooey.NewStyle())
			link := gooey.NewHyperlink("https://golang.org", "Go Programming")
			frame.PrintHyperlinkFallback(4+len(label), 0, link)
		}),
		gooey.Spacer().MinHeight(1),

		// Example 6: Long URL
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			label := "6. Long URL: "
			frame.PrintStyled(4, 0, label, gooey.NewStyle())
			link := gooey.NewHyperlink(
				"https://example.com/very/long/path/to/resource?with=many&query=params&and=more#section",
				"Complex URL",
			)
			frame.PrintHyperlink(4+len(label), 0, link)
		}),
		gooey.Spacer().MinHeight(1),

		// Example 7: Different link styles
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			label := "7. Different styles: "
			frame.PrintStyled(4, 0, label, gooey.NewStyle())
			x := 4 + len(label)

			// Red link
			redLink := gooey.NewHyperlink("https://example.com/red", "Red").
				WithStyle(gooey.NewStyle().WithForeground(gooey.ColorRed).WithUnderline())
			frame.PrintHyperlink(x, 0, redLink)
			x += len("Red") + 2

			// Green link
			greenLink := gooey.NewHyperlink("https://example.com/green", "Green").
				WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen).WithUnderline())
			frame.PrintHyperlink(x, 0, greenLink)
			x += len("Green") + 2

			// Yellow link
			yellowLink := gooey.NewHyperlink("https://example.com/yellow", "Yellow").
				WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow).WithUnderline())
			frame.PrintHyperlink(x, 0, yellowLink)
		}),
		gooey.Spacer().MinHeight(1),

		// Example 8: Links in a table-like layout
		gooey.HStack(
			gooey.Spacer(),
			gooey.VStack(
				gooey.Text("8. Table of links:").Bold(),
				gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
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

					y := 0
					for _, res := range resources {
						// Print resource name in a fixed width column
						nameStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
						frame.PrintStyled(3, y, fmt.Sprintf("%-20s", res.name), nameStyle)

						// Print link
						link := gooey.NewHyperlink(res.url, res.url)
						frame.PrintHyperlink(24, y, link)
						y++
					}
				}),
			),
			gooey.Spacer(),
		),
		gooey.Spacer().MinHeight(1),

		// Information box at the bottom
		gooey.HStack(
			gooey.Spacer(),
			gooey.VStack(
				gooey.Text("Terminal Support Information:").Bold(),
				gooey.Text("✓ Supported: iTerm2, WezTerm, kitty, foot, Rio, and others").Fg(gooey.ColorGreen),
				gooey.Text("✗ Not supported: Most terminals will show the text without clickable links").Fg(gooey.ColorYellow),
				gooey.Text("  (OSC 8 escape codes are ignored, text displays normally)").Fg(gooey.ColorWhite).Dim(),
			),
			gooey.Spacer(),
		),
		gooey.Spacer(),

		// Footer with countdown
		gooey.Text(fmt.Sprintf("Press any key to exit (auto-exit in %.0fs)", remaining.Seconds())).Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),
	).Align(gooey.AlignCenter)
}

func main() {
	if err := gooey.Run(&HyperlinkApp{}); err != nil {
		log.Fatal(err)
	}
}
