package main

import (
	"fmt"
	"log"
	"os"

	"github.com/deepnoodle-ai/wonton/terminal"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: playback <recording.cast>")
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

	// Raw mode delivers each key press without waiting for Enter.
	if err := terminal.EnableRawMode(); err != nil {
		log.Fatalf("Failed to enable raw mode: %v\n", err)
	}

	// Play has no input handling of its own, so read keys here and drive
	// the controller while Play runs.
	go handleKeys(controller)

	// Play the recording (blocking)
	fmt.Print("Starting playback...\r\n")
	err = controller.Play(terminal)
	terminal.DisableRawMode()
	if err != nil {
		log.Fatalf("\nPlayback error: %v\n", err)
	}

	fmt.Println("\n\nPlayback complete!")
}

// handleKeys maps the controls listed in the usage text to the controller.
func handleKeys(controller *tui.PlaybackController) {
	decoder := terminal.NewKeyDecoder(os.Stdin)
	loop := false
	for {
		event, err := decoder.ReadKeyEvent()
		if err != nil {
			return
		}
		switch {
		case event.Key == terminal.KeyEscape, event.Key == terminal.KeyCtrlC, event.Rune == 'q':
			controller.Stop()
			return
		case event.Rune == ' ':
			controller.TogglePause()
		case event.Rune == '+', event.Rune == ']':
			controller.SetSpeed(controller.Speed() * 2)
		case event.Rune == '-', event.Rune == '[':
			controller.SetSpeed(controller.Speed() / 2)
		case event.Rune == '1':
			controller.SetSpeed(1.0)
		case event.Rune == '2':
			controller.SetSpeed(2.0)
		case event.Rune == 'l':
			loop = !loop
			controller.SetLoop(loop)
		}
	}
}
