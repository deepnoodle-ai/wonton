// Example: terminal demonstrates terminal-style GIF creation with typed text.
//
// Run with: go run ./examples/gif/terminal
package main

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/gif"
)

func main() {
	// Create a small terminal screen (40 columns x 10 rows)
	screen := gif.NewTerminalScreen(40, 10)

	// Create renderer with 8px padding
	renderer := gif.NewTerminalRenderer(screen, 8)
	renderer.SetLoopCount(0)

	// Text to type out
	lines := []string{
		"$ echo 'Hello, World!'",
		"Hello, World!",
		"$ ls -la",
		"total 16",
		"drwxr-xr-x  4 user  staff  128 Dec 16 10:00 .",
		"-rw-r--r--  1 user  staff  256 Dec 16 10:00 README.md",
		"$ _",
	}

	// Type out each line character by character
	for _, line := range lines {
		for _, char := range line {
			screen.WriteString(string(char), gif.White, gif.Black)
			renderer.RenderFrame(3) // 30ms per character
		}
		// Add newline after each line
		screen.WriteString("\n", gif.White, gif.Black)
		renderer.RenderFrame(20) // 200ms pause between lines
	}

	// Add a few frames showing the final state with blinking cursor
	for i := 0; i < 10; i++ {
		renderer.RenderFrame(30) // 300ms per frame
	}

	// Save the GIF
	if err := renderer.Save("terminal.gif"); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Created terminal.gif (%d frames)\n", renderer.GIF().FrameCount())
}
