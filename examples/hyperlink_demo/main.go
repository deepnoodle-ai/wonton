package main

import (
	"log"
	"time"

	"github.com/deepnoodle-ai/gooey/tui"
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
func (app *HyperlinkApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch event.(type) {
	case tui.KeyEvent:
		// Exit on any key press or Ctrl+C
		return []tui.Cmd{tui.Quit()}

	case tui.TickEvent:
		// Auto-quit after 30 seconds
		if time.Now().After(app.autoQuitAt) {
			return []tui.Cmd{tui.Quit()}
		}
	}

	return nil
}

// View returns the declarative view hierarchy.
func (app *HyperlinkApp) View() tui.View {
	// Calculate remaining time for footer
	elapsed := time.Since(app.startTime)
	remaining := 30*time.Second - elapsed
	if remaining < 0 {
		remaining = 0
	}

	return tui.VStack(
		tui.Spacer().MinHeight(1),
		// Title and description
		tui.Text("OSC 8 Hyperlink Support Demo").Bold().Fg(tui.ColorCyan),
		tui.Text("Click the links below (if your terminal supports OSC 8)").Fg(tui.ColorWhite),
		tui.Spacer().MinHeight(1),
		// Separator
		tui.Divider().Fg(tui.ColorBlue),
		tui.Spacer().MinHeight(1),

		// Example 1: Default styled hyperlink
		tui.HStack(
			tui.Text("1. Default styled link: "),
			tui.Link("https://github.com/myzie/gooey", "Gooey on GitHub"),
		).Gap(0),
		tui.Spacer().MinHeight(1),

		// Example 2: Custom styled hyperlink
		tui.HStack(
			tui.Text("2. Custom styled link: "),
			tui.Link("https://example.com", "Example.com").Fg(tui.ColorMagenta).Bold(),
		).Gap(0),
		tui.Spacer().MinHeight(1),

		// Example 3: Multiple links on same line
		tui.HStack(
			tui.Text("3. Multiple links: "),
			tui.InlineLinks(" | ",
				tui.NewHyperlink("https://go.dev", "Go"),
				tui.NewHyperlink("https://github.com", "GitHub"),
				tui.NewHyperlink("https://stackoverflow.com", "Stack Overflow"),
			),
		).Gap(0),
		tui.Spacer().MinHeight(1),

		// Example 4: Link with emoji
		tui.HStack(
			tui.Text("4. Link with emoji: "),
			tui.Link("https://www.anthropic.com", "🤖 Anthropic"),
		).Gap(0),
		tui.Spacer().MinHeight(1),

		// Example 5: Fallback format (showing URL)
		tui.HStack(
			tui.Text("5. Fallback format: "),
			tui.Link("https://golang.org", "Go Programming").ShowURL(),
		).Gap(0),
		tui.Spacer().MinHeight(1),

		// Example 6: Long URL
		tui.HStack(
			tui.Text("6. Long URL: "),
			tui.Link(
				"https://example.com/very/long/path/to/resource?with=many&query=params&and=more#section",
				"Complex URL",
			),
		).Gap(0),
		tui.Spacer().MinHeight(1),

		// Example 7: Different link styles
		tui.HStack(
			tui.Text("7. Different styles: "),
			tui.InlineLinks("  ",
				tui.NewHyperlink("https://example.com/red", "Red").
					WithStyle(tui.NewStyle().WithForeground(tui.ColorRed).WithUnderline()),
				tui.NewHyperlink("https://example.com/green", "Green").
					WithStyle(tui.NewStyle().WithForeground(tui.ColorGreen).WithUnderline()),
				tui.NewHyperlink("https://example.com/yellow", "Yellow").
					WithStyle(tui.NewStyle().WithForeground(tui.ColorYellow).WithUnderline()),
			),
		).Gap(0),
		tui.Spacer().MinHeight(1),

		// Example 8: Links in a table-like layout
		tui.HStack(
			tui.Spacer(),
			tui.VStack(
				tui.Text("8. Table of links:").Bold(),
				tui.LinkRow("Documentation", "https://pkg.go.dev", "https://pkg.go.dev").LabelFg(tui.ColorWhite),
				tui.LinkRow("Source Code", "https://github.com", "https://github.com").LabelFg(tui.ColorWhite),
				tui.LinkRow("Issue Tracker", "https://github.com/issues", "https://github.com/issues").LabelFg(tui.ColorWhite),
				tui.LinkRow("Community", "https://www.reddit.com/r/golang", "https://www.reddit.com/r/golang").LabelFg(tui.ColorWhite),
			),
			tui.Spacer(),
		),
		tui.Spacer().MinHeight(1),

		// Information box at the bottom
		tui.HStack(
			tui.Spacer(),
			tui.VStack(
				tui.Text("Terminal Support Information:").Bold(),
				tui.Text("✓ Supported: iTerm2, WezTerm, kitty, foot, Rio, and others").Fg(tui.ColorGreen),
				tui.Text("✗ Not supported: Most terminals will show the text without clickable links").Fg(tui.ColorYellow),
				tui.Text("  (OSC 8 escape codes are ignored, text displays normally)").Fg(tui.ColorWhite).Dim(),
			),
			tui.Spacer(),
		),
		tui.Spacer(),

		// Footer with countdown
		tui.Text("Press any key to exit (auto-exit in %.0fs)", remaining.Seconds()).Fg(tui.ColorCyan),
		tui.Spacer().MinHeight(1),
	).Align(tui.AlignCenter)
}

func main() {
	if err := tui.Run(&HyperlinkApp{}); err != nil {
		log.Fatal(err)
	}
}
