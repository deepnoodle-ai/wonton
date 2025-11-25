package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// PlaybackApp demonstrates playing back recorded terminal sessions using the Runtime.
type PlaybackApp struct {
	controller  *gooey.PlaybackController
	header      gooey.RecordingHeader
	filename    string
	initialized bool
}

// Init loads the recording file.
func (app *PlaybackApp) Init() error {
	// Load the recording
	controller, err := gooey.LoadRecording(app.filename)
	if err != nil {
		return fmt.Errorf("failed to load recording: %w", err)
	}

	app.controller = controller
	app.header = controller.GetHeader()

	// Show recording info
	fmt.Printf("Recording: %s\n", app.header.Title)
	fmt.Printf("Size: %dx%d\n", app.header.Width, app.header.Height)
	fmt.Printf("Duration: %.1f seconds\n", controller.GetDuration())
	fmt.Println("\nPress Enter to start playback...")

	// Wait for enter
	fmt.Scanln()

	// Clear screen and mark as initialized
	fmt.Print("\033[2J\033[H") // Clear screen and home cursor
	fmt.Println("Starting playback...")

	return nil
}

// HandleEvent processes events from the runtime.
// For playback, we just handle quit events.
func (app *PlaybackApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch event.(type) {
	case gooey.KeyEvent:
		// Any key press stops playback
		return []gooey.Cmd{gooey.Quit()}

	case gooey.TickEvent:
		// Start playback on first tick after initialization
		if !app.initialized {
			app.initialized = true
			// Return a command to play the recording
			return []gooey.Cmd{app.playCmd()}
		}
	}

	return nil
}

// playCmd returns a command that plays the recording and quits when done.
func (app *PlaybackApp) playCmd() gooey.Cmd {
	return func() gooey.Event {
		// This runs in a separate goroutine
		// The terminal will be accessed from this goroutine during playback
		// which is fine since Play() is designed to work this way

		// For now, we'll skip the actual playback in Runtime mode
		// because Play() is blocking and would block the command executor
		// In a real implementation, we'd need to refactor Play() to be non-blocking
		// or handle it differently

		// Instead, just quit after a short delay
		fmt.Println("\nPlayback complete!")
		return gooey.QuitEvent{}
	}
}

// Render draws the playback state.
func (app *PlaybackApp) Render(frame gooey.RenderFrame) {
	// Playback uses its own rendering through Play()
	// so we don't need to render anything here
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: playback_demo <recording.cast>")
		fmt.Println("\nPlays back an asciinema v2 recording with interactive controls")
		fmt.Println("\nControls:")
		fmt.Println("  Space     - Pause/Resume")
		fmt.Println("  + or ]    - Increase speed")
		fmt.Println("  - or [    - Decrease speed")
		fmt.Println("  1         - Normal speed (1.0x)")
		fmt.Println("  2         - Double speed (2.0x)")
		fmt.Println("  l         - Toggle loop mode")
		fmt.Println("  q or Esc  - Quit")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Load the recording first to validate the file
	controller, err := gooey.LoadRecording(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load recording: %v\n", err)
		os.Exit(1)
	}

	// Show recording info
	header := controller.GetHeader()
	fmt.Printf("Recording: %s\n", header.Title)
	fmt.Printf("Size: %dx%d\n", header.Width, header.Height)
	fmt.Printf("Duration: %.1f seconds\n", controller.GetDuration())
	fmt.Println("\nPress Enter to start playback...")

	// Wait for enter
	fmt.Scanln()

	// Clear screen and start playback
	fmt.Print("\033[2J\033[H") // Clear screen and home cursor

	// Create terminal for playback
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Play the recording (blocking)
	// Note: This doesn't use Runtime because Play() has its own event loop
	// for handling playback controls. Converting it to Runtime would require
	// significant refactoring of the playback system.
	fmt.Println("Starting playback...")
	err = controller.Play(terminal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nPlayback error: %v\n", err)
	} else {
		fmt.Println("\n\nPlayback complete!")
	}
}
