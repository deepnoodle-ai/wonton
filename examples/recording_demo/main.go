package main

import (
	"fmt"
	"os"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	// Note: Terminal cleanup is handled by OS

	// Start recording with default options
	opts := gooey.DefaultRecordingOptions()
	opts.Title = "Gooey Recording Demo"
	opts.Env = map[string]string{
		"TERM": os.Getenv("TERM"),
	}

	recordingFile := "demo.cast"
	if err := terminal.StartRecording(recordingFile, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start recording: %v\n", err)
		os.Exit(1)
	}
	defer terminal.StopRecording()

	// Print welcome message with delays - each Print() is recorded with its timing
	terminal.SetStyle(gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan))
	terminal.Println("=== Session Recording Demo ===")
	terminal.Reset()
	time.Sleep(300 * time.Millisecond)

	terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen))
	terminal.Println("Recording to: " + recordingFile)
	terminal.Reset()
	time.Sleep(200 * time.Millisecond)

	terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))
	terminal.Println("This demo captures the timing of each Print() call!")
	terminal.Reset()
	time.Sleep(300 * time.Millisecond)

	terminal.Println("")
	time.Sleep(100 * time.Millisecond)

	terminal.Println("Watch how the text appears with the original timing...")
	time.Sleep(500 * time.Millisecond)

	terminal.Println("")
	terminal.Println("Features demonstrated:")
	time.Sleep(200 * time.Millisecond)

	terminal.Println("  • ANSI color output")
	time.Sleep(150 * time.Millisecond)

	terminal.Println("  • Timing preservation")
	time.Sleep(150 * time.Millisecond)

	terminal.Println("  • User input capture")
	time.Sleep(150 * time.Millisecond)

	terminal.Println("  • gzip compression")
	time.Sleep(300 * time.Millisecond)

	terminal.Println("")
	time.Sleep(200 * time.Millisecond)

	// Create input handler
	input := gooey.NewInput(terminal)

	// Simple line-based input
	for lineNum := 0; lineNum < 3; lineNum++ {
		terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorWhite))
		terminal.Print(fmt.Sprintf("Enter text (line %d/3): ", lineNum+1))
		terminal.Reset()
		terminal.Flush()

		line, err := input.ReadSimple()
		if err != nil {
			break
		}

		terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan))
		terminal.Println("You entered: " + line)
		terminal.Reset()
		time.Sleep(200 * time.Millisecond)
	}

	terminal.Println("")
	terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold())
	terminal.Println("Recording complete! Saved to " + recordingFile)
	terminal.Reset()
	terminal.Println("")
	terminal.Println("Play it back with:")
	terminal.Println("  go run examples/playback_demo/main.go " + recordingFile)
	terminal.Println("")

	time.Sleep(1 * time.Second)
}
