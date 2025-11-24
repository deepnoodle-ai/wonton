package main

import (
	"fmt"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error initializing terminal: %v\n", err)
		return
	}
	// No defer terminal.Restore() here because MultiProgress manages its own cursor/output mostly,
	// but good practice to restore if we mess with it.
	// However, MultiProgress prints to stdout/newlines.
	defer terminal.Close()

	terminal.Clear()
	terminal.HideCursor()

	// Use terminal methods for output so the buffer tracks position
	headerStyle := gooey.NewStyle().WithBold()
	terminal.Println("") // Newline
	terminal.PrintStyled("🚀 Multi-Progress Demo", headerStyle)
	terminal.Println("")
	terminal.PrintStyled("======================", headerStyle)
	terminal.Println("")

	mp := gooey.NewMultiProgress(terminal)

	// Add progress items
	dl := mp.Add("download", 100, false) // ID, Total, SpinnerOnly
	dl.Message = "Downloading assets..."
	dl.Style = gooey.NewStyle().WithForeground(gooey.ColorCyan)

	db := mp.Add("db", 1, true) // Spinner only
	db.Message = "Connecting to database..."
	db.Style = gooey.NewStyle().WithForeground(gooey.ColorYellow)

	proc := mp.Add("process", 50, false)
	proc.Message = "Waiting to process..."
	proc.Style = gooey.NewStyle().WithForeground(gooey.ColorGreen)

	// Start rendering
	mp.Start()

	// Simulate work
	go func() {
		for i := 0; i <= 100; i++ {
			time.Sleep(50 * time.Millisecond)
			mp.Update("download", i, "Downloading assets...")
		}
		mp.Update("download", 100, "Download complete!")
	}()

	go func() {
		time.Sleep(2 * time.Second)
		mp.Update("db", 0, "Connected to DB!")
		// Convert spinner to done? MultiProgress simple update doesn't change type,
		// but we can update message.
		// In a real app you might remove it or mark it done.

		// Start processing after DB
		for i := 0; i <= 50; i++ {
			time.Sleep(100 * time.Millisecond)
			mp.Update("process", i, fmt.Sprintf("Processing records... %d/50", i))
		}
		mp.Update("process", 50, "Processing complete!")
	}()

	// Wait for all to finish
	time.Sleep(8 * time.Second)
	mp.Stop()

	fmt.Println("\n✨ All tasks finished!")
}
