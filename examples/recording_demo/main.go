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
	lineCount    int
	currentInput string
	frame        uint64
	lastPrintAt  uint64
	delays       []uint64 // Delays in frames (at 30 FPS)
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

		// State machine to print messages with delays
		if app.state < 10 {
			// Wait for the delay
			delayFrames := app.delays[app.state]
			if app.frame >= app.lastPrintAt+delayFrames {
				app.printNextMessage()
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
			// Print completion message
			if app.state == 13 {
				app.terminal.Println("")
				app.terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold())
				app.terminal.Println("Recording complete! Saved to " + app.recordFile)
				app.terminal.Reset()
				app.terminal.Println("")
				app.terminal.Println("Play it back with:")
				app.terminal.Println("  go run examples/playback_demo/main.go " + app.recordFile)
				app.terminal.Println("")
				app.terminal.Flush()
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

// printNextMessage prints the next message in the sequence.
func (app *RecordingApp) printNextMessage() {
	switch app.state {
	case 0:
		app.terminal.SetStyle(gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan))
		app.terminal.Println("=== Session Recording Demo ===")
		app.terminal.Reset()
		app.terminal.Flush()

	case 1:
		app.terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen))
		app.terminal.Println("Recording to: " + app.recordFile)
		app.terminal.Reset()
		app.terminal.Flush()

	case 2:
		app.terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))
		app.terminal.Println("This demo captures the timing of each Print() call!")
		app.terminal.Reset()
		app.terminal.Flush()

	case 3:
		app.terminal.Println("")
		app.terminal.Flush()

	case 4:
		app.terminal.Println("Watch how the text appears with the original timing...")
		app.terminal.Flush()

	case 5:
		app.terminal.Println("")
		app.terminal.Println("Features demonstrated:")
		app.terminal.Flush()

	case 6:
		app.terminal.Println("  • ANSI color output")
		app.terminal.Flush()

	case 7:
		app.terminal.Println("  • Timing preservation")
		app.terminal.Flush()

	case 8:
		app.terminal.Println("  • User input capture")
		app.terminal.Flush()

	case 9:
		app.terminal.Println("  • gzip compression")
		app.terminal.Flush()
	}
}

// Render is called after each event, but we don't need to render anything
// since we're using the terminal's Print API directly.
func (app *RecordingApp) Render(frame gooey.RenderFrame) {
	// No rendering needed - we're using Terminal.Print API for recording
}

func main() {
	// Note: This example uses the old pattern (NewTerminal + NewRuntime) instead of
	// gooey.Run() because it needs to call terminal.StartRecording() and use terminal.Print*()
	// methods directly, which requires access to the terminal instance.

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
