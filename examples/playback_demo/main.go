package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

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

	// Load the recording
	controller, err := gooey.LoadRecording(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load recording: %v\n", err)
		os.Exit(1)
	}

	// Create terminal for playback
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	// Note: Terminal cleanup handled by OS

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

	// Play the recording (blocking)
	fmt.Println("Starting playback...")
	time.Sleep(500 * time.Millisecond)

	err = controller.Play(terminal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nPlayback error: %v\n", err)
	} else {
		fmt.Println("\n\nPlayback complete!")
	}
}
