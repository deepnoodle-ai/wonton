package main

import (
	"log"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

// ProgressEnhancedApp demonstrates the enhanced progress bar features
type ProgressEnhancedApp struct {
	progress1 int
	progress2 int
	progress3 int
	progress4 int
	frame     uint64
	startTime time.Time
}

// Init initializes the app
func (app *ProgressEnhancedApp) Init() error {
	app.startTime = time.Now()
	return nil
}

// HandleEvent processes events
func (app *ProgressEnhancedApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.TickEvent:
		app.frame = e.Frame

		// Increment progress bars at different rates
		if app.progress1 < 100 {
			app.progress1++
		}
		if app.frame%2 == 0 && app.progress2 < 100 {
			app.progress2++
		}
		if app.frame%3 == 0 && app.progress3 < 100 {
			app.progress3++
		}
		if app.frame%4 == 0 && app.progress4 < 100 {
			app.progress4++
		}

		// Reset when all complete
		if app.progress1 >= 100 && app.progress2 >= 100 &&
			app.progress3 >= 100 && app.progress4 >= 100 {
			app.progress1 = 0
			app.progress2 = 0
			app.progress3 = 0
			app.progress4 = 0
		}

	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}

	return nil
}

// View returns the app view
func (app *ProgressEnhancedApp) View() tui.View {
	elapsed := time.Since(app.startTime).Truncate(time.Millisecond)

	return tui.Stack(
		// Header
		tui.HeaderBar("Enhanced Progress Bar Demo").Bg(tui.ColorBlue).Fg(tui.ColorWhite),
		tui.Spacer().MinHeight(1),

		// Example 1: Standard progress with percentage (default)
		tui.Text("Standard with percentage:").Bold(),
		tui.Progress(app.progress1, 100).
			Width(40).
			Fg(tui.ColorGreen),
		tui.Spacer().MinHeight(1),

		// Example 2: Progress with custom empty pattern "·-"
		tui.Text("Custom empty pattern '·-':").Bold(),
		tui.Progress(app.progress2, 100).
			Width(40).
			Fg(tui.ColorCyan).
			EmptyPattern("·-").
			EmptyFg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),

		// Example 3: Progress with different empty pattern "░▒"
		tui.Text("Custom empty pattern '░▒':").Bold(),
		tui.Progress(app.progress3, 100).
			Width(40).
			Fg(tui.ColorYellow).
			EmptyPattern("░▒").
			EmptyFg(tui.ColorBrightBlack),
		tui.Spacer().MinHeight(1),

		// Example 4: Progress with custom percentage style
		tui.Text("Custom percentage style (magenta):").Bold(),
		tui.Progress(app.progress4, 100).
			Width(40).
			Fg(tui.ColorGreen).
			EmptyChar('·').
			EmptyFg(tui.ColorBrightBlack).
			PercentFg(tui.ColorMagenta),
		tui.Spacer().MinHeight(1),

		// Example 5: Progress without percentage
		tui.Text("Without percentage:").Bold(),
		tui.Progress(app.progress1, 100).
			Width(40).
			Fg(tui.ColorBlue).
			HidePercent(),
		tui.Spacer().MinHeight(1),

		// Example 6: Progress with fraction instead of percentage
		tui.Text("With fraction:").Bold(),
		tui.Progress(app.progress2, 100).
			Width(40).
			Fg(tui.ColorRed).
			ShowFraction(),
		tui.Spacer().MinHeight(1),

		// Example 7: Multiple patterns demonstration
		tui.Text("More pattern examples:").Bold(),
		tui.Group(
			tui.Stack(
				tui.Text("Pattern: '━┅'").Dim(),
				tui.Progress(app.progress3, 100).Width(20).EmptyPattern("━┅").Fg(tui.ColorGreen),
				tui.Spacer().MinHeight(1),
				tui.Text("Pattern: '◼◻'").Dim(),
				tui.Progress(app.progress4, 100).Width(20).EmptyPattern("◼◻").Fg(tui.ColorCyan),
			),
			tui.Spacer().MinWidth(2),
			tui.Stack(
				tui.Text("Pattern: '▓▒░'").Dim(),
				tui.Progress(app.progress1, 100).Width(20).EmptyPattern("▓▒░").Fg(tui.ColorYellow),
				tui.Spacer().MinHeight(1),
				tui.Text("Pattern: '●○'").Dim(),
				tui.Progress(app.progress2, 100).Width(20).EmptyPattern("●○").Fg(tui.ColorMagenta),
			),
		),

		tui.Spacer(),

		// Divider before footer
		tui.Divider(),
		tui.Spacer().MinHeight(1),

		// Bottom controls
		tui.Group(
			tui.Text("Press 'q' to quit").Fg(tui.ColorWhite),
			tui.Spacer(),
			tui.Text("Elapsed: %s", elapsed).Dim(),
		),
	).Padding(2)
}

func main() {
	app := &ProgressEnhancedApp{}

	if err := tui.Run(app); err != nil {
		log.Fatal(err)
	}

	log.Println("\n✨ Demo complete!")
}
