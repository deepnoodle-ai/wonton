package main

import (
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
		gooey.Divider().Fg(gooey.ColorBlue),
		gooey.Spacer().MinHeight(1),

		// Example 1: Default styled hyperlink
		gooey.HStack(
			gooey.Text("1. Default styled link: "),
			gooey.Link("https://github.com/myzie/gooey", "Gooey on GitHub"),
		).Gap(0),
		gooey.Spacer().MinHeight(1),

		// Example 2: Custom styled hyperlink
		gooey.HStack(
			gooey.Text("2. Custom styled link: "),
			gooey.Link("https://example.com", "Example.com").Fg(gooey.ColorMagenta).Bold(),
		).Gap(0),
		gooey.Spacer().MinHeight(1),

		// Example 3: Multiple links on same line
		gooey.HStack(
			gooey.Text("3. Multiple links: "),
			gooey.InlineLinks(" | ",
				gooey.NewHyperlink("https://go.dev", "Go"),
				gooey.NewHyperlink("https://github.com", "GitHub"),
				gooey.NewHyperlink("https://stackoverflow.com", "Stack Overflow"),
			),
		).Gap(0),
		gooey.Spacer().MinHeight(1),

		// Example 4: Link with emoji
		gooey.HStack(
			gooey.Text("4. Link with emoji: "),
			gooey.Link("https://www.anthropic.com", "🤖 Anthropic"),
		).Gap(0),
		gooey.Spacer().MinHeight(1),

		// Example 5: Fallback format (showing URL)
		gooey.HStack(
			gooey.Text("5. Fallback format: "),
			gooey.Link("https://golang.org", "Go Programming").ShowURL(),
		).Gap(0),
		gooey.Spacer().MinHeight(1),

		// Example 6: Long URL
		gooey.HStack(
			gooey.Text("6. Long URL: "),
			gooey.Link(
				"https://example.com/very/long/path/to/resource?with=many&query=params&and=more#section",
				"Complex URL",
			),
		).Gap(0),
		gooey.Spacer().MinHeight(1),

		// Example 7: Different link styles
		gooey.HStack(
			gooey.Text("7. Different styles: "),
			gooey.InlineLinks("  ",
				gooey.NewHyperlink("https://example.com/red", "Red").
					WithStyle(gooey.NewStyle().WithForeground(gooey.ColorRed).WithUnderline()),
				gooey.NewHyperlink("https://example.com/green", "Green").
					WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen).WithUnderline()),
				gooey.NewHyperlink("https://example.com/yellow", "Yellow").
					WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow).WithUnderline()),
			),
		).Gap(0),
		gooey.Spacer().MinHeight(1),

		// Example 8: Links in a table-like layout
		gooey.HStack(
			gooey.Spacer(),
			gooey.VStack(
				gooey.Text("8. Table of links:").Bold(),
				gooey.LinkRow("Documentation", "https://pkg.go.dev", "https://pkg.go.dev").LabelFg(gooey.ColorWhite),
				gooey.LinkRow("Source Code", "https://github.com", "https://github.com").LabelFg(gooey.ColorWhite),
				gooey.LinkRow("Issue Tracker", "https://github.com/issues", "https://github.com/issues").LabelFg(gooey.ColorWhite),
				gooey.LinkRow("Community", "https://www.reddit.com/r/golang", "https://www.reddit.com/r/golang").LabelFg(gooey.ColorWhite),
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
		gooey.Text("Press any key to exit (auto-exit in %.0fs)", remaining.Seconds()).Fg(gooey.ColorCyan),
		gooey.Spacer().MinHeight(1),
	).Align(gooey.AlignCenter)
}

func main() {
	if err := gooey.Run(&HyperlinkApp{}); err != nil {
		log.Fatal(err)
	}
}
