// Example: inline_test
//
// This example tests InlineApp implementation:
// - Live view that dynamically changes size
// - Scrollback history with various content
// - Verifies no extra newlines are printed
// - Tests different view types and compositions
//
// Run with: go run ./examples/inline_test
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

type TestApp struct {
	runner     *tui.InlineApp
	phase      int
	itemCount  int
	showStatus bool
	messages   []string
}

func (app *TestApp) Init() error {
	app.messages = []string{
		"System initialized",
		"Loading configuration",
		"Connecting to services",
	}
	return nil
}

func (app *TestApp) LiveView() tui.View {
	var views []tui.View

	// Header - always present
	views = append(views, tui.Divider())
	views = append(views, tui.Text("═ Live View Test ═").Bold().Center())
	views = append(views, tui.Divider())

	// Dynamic content based on phase
	switch app.phase {
	case 0:
		// Small view
		views = append(views, tui.Text("Phase 1: Minimal view"))
		views = append(views, tui.Text("Press 's' to start test").Dim())

	case 1:
		// Medium view with some details
		views = append(views, tui.Text("Phase 2: Expanded view"))
		views = append(views, tui.Text(""))
		views = append(views, tui.Text("Items: %d", app.itemCount).Fg(tui.ColorBlue))
		views = append(views, tui.Text("Status: Active").Fg(tui.ColorGreen))
		views = append(views, tui.Text(""))
		views = append(views, tui.Text("Press 'n' for next phase").Dim())

	case 2:
		// Large view with multiple sections
		views = append(views, tui.Text("Phase 3: Large view with multiple sections"))
		views = append(views, tui.Text(""))

		// Progress section
		views = append(views, tui.Text("▸ Progress Section").Bold())
		for i := 0; i < 3; i++ {
			if i < app.itemCount {
				views = append(views, tui.Text("  ✓ Task %d completed", i+1).Fg(tui.ColorGreen))
			} else {
				views = append(views, tui.Text("  ○ Task %d pending", i+1).Dim())
			}
		}
		views = append(views, tui.Text(""))

		// Status section (conditional)
		if app.showStatus {
			views = append(views, tui.Text("▸ Status Details").Bold())
			views = append(views, tui.Text("  Memory: 42 MB").Dim())
			views = append(views, tui.Text("  Connections: 5").Dim())
			views = append(views, tui.Text(""))
		}

		views = append(views, tui.Text("Press 'n' for next phase, 't' to toggle status").Dim())

	case 3:
		// Back to small view
		views = append(views, tui.Text("Phase 4: Minimal view (testing shrink)"))
		views = append(views, tui.Text("Press 'n' for next phase").Dim())

	case 4:
		// Final phase with styled content
		views = append(views, tui.Text("Phase 5: Styled content").Bold().Fg(tui.ColorMagenta))
		views = append(views, tui.Text(""))
		views = append(views, tui.Group(
			tui.Text("Success: ").Fg(tui.ColorGreen).Bold(),
			tui.Text("All tests passed"),
		))
		views = append(views, tui.Text(""))
		views = append(views, tui.Text("Press 'r' to restart, 'q' to quit").Dim())
	}

	// Footer
	views = append(views, tui.Divider())
	views = append(views, tui.Text("q: quit | Commands shown above").Dim().Right())

	return tui.Stack(views...)
}

func (app *TestApp) HandleEvent(event tui.Event) []tui.Cmd {
	// Handle custom events first
	switch e := event.(type) {
	case ItemIncrementEvent:
		if app.phase == 1 {
			app.itemCount++
			app.runner.Printf("Item added: #%d", app.itemCount)

			if app.itemCount < 5 {
				return []tui.Cmd{app.autoProgressCmd()}
			} else {
				app.runner.Print(tui.Stack(
					tui.Text(""),
					tui.Text("✓ Auto-progress complete!").Fg(tui.ColorGreen).Bold(),
					tui.Text("Press 'n' to continue to next phase").Dim(),
				))
			}
		}
		return nil

	case BatchPrintEvent:
		app.runner.Print(tui.Group(
			tui.Text("[%d] ", e.Index+1).Dim(),
			tui.Text("%s", e.Message),
		))

		// Continue batch if more messages
		if e.Index+1 < len(app.messages) {
			return []tui.Cmd{func() tui.Event {
				time.Sleep(300 * time.Millisecond)
				return BatchPrintEvent{
					Index:   e.Index + 1,
					Message: app.messages[e.Index+1],
					Time:    time.Now(),
				}
			}}
		} else {
			app.runner.Print(tui.Stack(
				tui.Text(""),
				tui.Text("✓ Batch printing complete").Fg(tui.ColorGreen),
			))
		}
		return nil
	}

	// Handle keyboard events
	if e, ok := event.(tui.KeyEvent); ok {
		switch e.Rune {
		case 's':
			if app.phase == 0 {
				app.runner.Print(tui.Stack(
					tui.Divider(),
					tui.Text("Test started at %s", time.Now().Format("15:04:05")),
					tui.Divider(),
				))
				app.phase = 1
				return []tui.Cmd{app.autoProgressCmd()}
			}

		case 'n':
			if app.phase < 4 {
				app.phase++
				app.runner.Printf("→ Advanced to phase %d", app.phase+1)

				if app.phase == 2 {
					// Print multiple messages to scrollback during phase 3
					return []tui.Cmd{app.multiPrintCmd()}
				}
			}

		case 't':
			if app.phase == 2 {
				app.showStatus = !app.showStatus
				status := "hidden"
				if app.showStatus {
					status = "visible"
				}
				app.runner.Printf("Status section is now %s", status)
			}

		case 'r':
			app.phase = 0
			app.itemCount = 0
			app.showStatus = false
			app.runner.Print(tui.Stack(
				tui.Text(""),
				tui.Divider(),
				tui.Text("Test reset at %s", time.Now().Format("15:04:05")).Dim(),
				tui.Divider(),
				tui.Text(""),
			))

		case 'q':
			app.runner.Printf("Exiting test program...")
			return []tui.Cmd{tui.Quit()}

		case 'h':
			app.runner.Print(tui.Stack(
				tui.Text("This is a test of scrollback printing."),
				tui.Text("Multiple lines can be added."),
				tui.Text("Each Print() call adds content above the live view."),
			))

		case 'c':
			app.runner.ClearScrollback()
			app.runner.Printf("Scrollback cleared!")
		}

		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}

	return nil
}

// autoProgressCmd automatically increments itemCount for phase 2
func (app *TestApp) autoProgressCmd() tui.Cmd {
	return func() tui.Event {
		time.Sleep(500 * time.Millisecond)
		return ItemIncrementEvent{Time: time.Now()}
	}
}

// multiPrintCmd prints multiple items to scrollback in sequence
func (app *TestApp) multiPrintCmd() tui.Cmd {
	return func() tui.Event {
		time.Sleep(300 * time.Millisecond)
		if len(app.messages) > 0 {
			return BatchPrintEvent{
				Index:   0,
				Message: app.messages[0],
				Time:    time.Now(),
			}
		}
		return nil
	}
}

type ItemIncrementEvent struct {
	Time time.Time
}

func (e ItemIncrementEvent) Timestamp() time.Time { return e.Time }

type BatchPrintEvent struct {
	Index   int
	Message string
	Time    time.Time
}

func (e BatchPrintEvent) Timestamp() time.Time { return e.Time }

func main() {
	app := &TestApp{}
	app.runner = tui.NewInlineApp(
		tui.WithInlineWidth(70),
	)

	fmt.Println("=== InlineApp Test Example ===")
	fmt.Println("This example tests:")
	fmt.Println("  • Dynamic live view sizing")
	fmt.Println("  • Scrollback history printing")
	fmt.Println("  • No extra newlines")
	fmt.Println("  • Multiple view compositions")
	fmt.Println()

	if err := app.runner.Run(app); err != nil {
		log.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Test complete. Check output above for extra newlines.")
}
