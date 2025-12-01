package main

import (
	"fmt"
	"log"
	"os"

	"github.com/deepnoodle-ai/gooey/tui"
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
		return
	}

	filename := os.Args[1]

	// Load the recording first to validate the file
	controller, err := tui.LoadRecording(filename)
	if err != nil {
		log.Fatalf("Failed to load recording: %v\n", err)
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
	terminal, err := tui.NewTerminal()
	if err != nil {
		log.Fatalf("Failed to create terminal: %v\n", err)
	}
	defer terminal.Close()

	// Play the recording (blocking)
	// Note: This doesn't use Runtime because Play() has its own event loop
	// for handling playback controls. Converting it to Runtime would require
	// significant refactoring of the playback system.
	fmt.Println("Starting playback...")
	err = controller.Play(terminal)
	if err != nil {
		log.Fatalf("\nPlayback error: %v\n", err)
	}

	fmt.Println("\n\nPlayback complete!")
}
