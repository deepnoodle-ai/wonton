package main

import (
	"fmt"
	"log"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// RecordingApp demonstrates terminal session recording with the Runtime.
// This uses a different pattern than typical Runtime apps - it drives itself
// forward using tick events and state machine pattern.
type RecordingApp struct {
	terminal     *gooey.Terminal
	recordFile   string
	state        int
	frame        uint64
	lastPrintAt  uint64
	delays       []uint64 // Delays in frames (at 30 FPS)
	messages     []string  // Messages to display
}

// Init initializes the recording session.
func (app *RecordingApp) Init() error {
	// Start recording with default options
	opts := gooey.DefaultRecordingOptions()
	opts.Title = "Gooey Recording Demo"
	opts.Env = map[string]string{
		"TERM": os.Getenv("TERM"),
	}

	app.recordFile = "demo.cast"
	if err := app.terminal.StartRecording(app.recordFile, opts); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	// Define delays between prints (in frames at 30 FPS)
	// 30 frames = 1 second, so 9 frames ≈ 300ms
	app.delays = []uint64{
		9,  // 300ms
		6,  // 200ms
		9,  // 300ms
		3,  // 100ms
		15, // 500ms
		6,  // 200ms
		5,  // 150ms
		5,  // 150ms
		5,  // 150ms
		9,  // 300ms
	}

	app.state = 0
	app.lastPrintAt = 0
	app.messages = []string{} // Start with empty messages

	return nil
}

// Destroy cleans up the recording.
func (app *RecordingApp) Destroy() {
	app.terminal.StopRecording()
}

// HandleEvent processes events from the runtime.
func (app *RecordingApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle Ctrl+C to quit early
		if e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.TickEvent:
		app.frame = e.Frame

		// State machine to add messages with delays
		if app.state < 10 {
			// Wait for the delay
			delayFrames := app.delays[app.state]
			if app.frame >= app.lastPrintAt+delayFrames {
				app.addNextMessage()
				app.lastPrintAt = app.frame
				app.state++
			}
		} else if app.state >= 10 && app.state < 13 {
			// Input phase - we need to return a command to read input
			// For this demo, we'll skip interactive input in Runtime mode
			// since Runtime's input reader is already consuming stdin
			app.state = 13 // Skip to end
		}

		// Check if we're done
		if app.state >= 13 {
			// Add completion message
			if app.state == 13 {
				app.messages = append(app.messages, "") // Empty line
				app.messages = append(app.messages, "Recording complete! Saved to "+app.recordFile)
				app.messages = append(app.messages, "")
				app.messages = append(app.messages, "Play it back with:")
				app.messages = append(app.messages, "  go run examples/playback_demo/main.go "+app.recordFile)
				app.messages = append(app.messages, "")
				app.state++ // Move to next state to avoid reprinting
			}

			// Wait 1 second then quit
			if app.frame >= app.lastPrintAt+30 {
				return []gooey.Cmd{gooey.Quit()}
			}
		}
	}

	return nil
}

// addNextMessage adds the next message to the display.
func (app *RecordingApp) addNextMessage() {
	switch app.state {
	case 0:
		app.messages = append(app.messages, "=== Session Recording Demo ===")
	case 1:
		app.messages = append(app.messages, "Recording to: "+app.recordFile)
	case 2:
		app.messages = append(app.messages, "This demo captures the timing of each Print() call!")
	case 3:
		app.messages = append(app.messages, "")
	case 4:
		app.messages = append(app.messages, "Watch how the text appears with the original timing...")
	case 5:
		app.messages = append(app.messages, "")
		app.messages = append(app.messages, "Features demonstrated:")
	case 6:
		app.messages = append(app.messages, "  • ANSI color output")
	case 7:
		app.messages = append(app.messages, "  • Timing preservation")
	case 8:
		app.messages = append(app.messages, "  • User input capture")
	case 9:
		app.messages = append(app.messages, "  • gzip compression")
	}
}

// View returns the declarative view of the recording demo.
func (app *RecordingApp) View() gooey.View {
	// Build views for each message with appropriate styling
	var views []gooey.View

	for i, msg := range app.messages {
		if msg == "" {
			// Empty line - use a spacer
			views = append(views, gooey.Spacer().MinHeight(1))
			continue
		}

		// Apply styling based on message content and position
		var view gooey.View
		switch {
		case i == 0 && msg == "=== Session Recording Demo ===":
			// Title - bold cyan
			view = gooey.Text(msg).Bold().Fg(gooey.ColorCyan)
		case i == 1 && len(app.messages) > 1:
			// Recording file - green
			view = gooey.Text(msg).Fg(gooey.ColorGreen)
		case i == 2 && len(app.messages) > 2:
			// Demo description - yellow
			view = gooey.Text(msg).Fg(gooey.ColorYellow)
		case msg == "Recording complete! Saved to "+app.recordFile:
			// Completion message - bold green
			view = gooey.Text(msg).Bold().Fg(gooey.ColorGreen)
		default:
			// Default styling
			view = gooey.Text(msg)
		}

		views = append(views, view)
	}

	// If no views yet, return empty VStack
	if len(views) == 0 {
		views = append(views, gooey.Spacer())
	}

	return gooey.VStack(views...)
}

func main() {
	// Note: This example uses the old pattern (NewTerminal + NewRuntime) instead of
	// gooey.Run() because it needs to call terminal.StartRecording() which requires
	// access to the terminal instance.

	terminal, err := gooey.NewTerminal()
	if err != nil {
		log.Fatalf("Failed to create terminal: %v\n", err)
	}
	defer terminal.Close()

	app := &RecordingApp{
		terminal: terminal,
	}

	runtime := gooey.NewRuntime(terminal, app, 30)

	if err := runtime.Run(); err != nil {
		log.Fatalf("Runtime error: %v\n", err)
	}
}
