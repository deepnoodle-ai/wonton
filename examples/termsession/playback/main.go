// Example: Session Playback
//
// Plays back an asciinema v2 recording (.cast file, optionally gzipped)
// using the tui package's PlaybackController, which preserves the original
// timing of the recorded output.
//
// Run with:
//
//	go run examples/termsession/playback/main.go <recording.cast>
package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: playback <recording.cast>")
		fmt.Fprintln(os.Stderr, "\nPlays back an asciinema v2 recording with its original timing.")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Load the recording first to validate the file
	controller, err := tui.LoadRecording(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load recording: %v\n", err)
		os.Exit(1)
	}

	// Show recording info
	header := controller.GetHeader()
	fmt.Printf("Recording: %s\n", filename)
	if header.Title != "" {
		fmt.Printf("Title: %s\n", header.Title)
	}
	fmt.Printf("Size: %dx%d\n", header.Width, header.Height)
	fmt.Printf("Duration: %.1f seconds\n", controller.GetDuration())
	fmt.Println("\nPress Enter to start playback...")

	// Wait for enter
	fmt.Scanln()

	// Clear screen and start playback
	fmt.Print("\033[2J\033[H") // Clear screen and home cursor

	// Create terminal for playback
	terminal, err := tui.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Play the recording. This blocks, writing the recorded output to the
	// terminal with the original timing. The controller also supports
	// programmatic Pause/Resume, SetSpeed, Seek, and Stop from another
	// goroutine; this demo just plays straight through.
	if err := controller.Play(terminal); err != nil {
		// Close the terminal before exiting: os.Exit skips deferred calls.
		terminal.Close()
		fmt.Fprintf(os.Stderr, "\nPlayback error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n\nPlayback complete!")
}
