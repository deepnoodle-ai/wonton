package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	fmt.Println("Welcome to the Inline Input Demo!")
	fmt.Println("This demonstrates tui.Prompt() with various features:")
	fmt.Println("  - History navigation (up/down arrows)")
	fmt.Println("  - File autocomplete (type @filename, auto-triggers)")
	fmt.Println("  - Multi-line input (Shift+Enter or Ctrl+J)")
	fmt.Println("  - Ctrl+C to exit")
	fmt.Println()

	// Simulate a chat interface
	var history []string

	for {
		// Show a simulated assistant response with live updates
		if len(history) > 0 {
			simulateResponse(history[len(history)-1])
		}

		// Get input from user
		input, err := tui.Prompt(" > ",
			tui.WithHistory(&history),
			tui.WithAutocomplete(fileAutocomplete),
			tui.WithMultiLine(true),
			tui.WithPlaceholder("Type a message... (@filename for autocomplete)"),
			tui.WithPromptStyle(tui.NewStyle().WithForeground(tui.ColorCyan)),
		)

		if err != nil {
			if err == tui.ErrInterrupted {
				fmt.Println("\nGoodbye!")
				return
			}
			if err == tui.ErrEOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}

		if input == "" {
			continue
		}

		// Print user message to scrollback
		userView := tui.Stack(
			tui.Text("You:").Bold().Fg(tui.ColorGreen),
			tui.Padding(1, tui.Text("%s", input)),
		)
		tui.Print(userView)
		fmt.Println()
	}
}

// simulateResponse simulates a streaming assistant response
func simulateResponse(userInput string) {
	live := tui.NewLivePrinter()

	// Simulate thinking
	for i := 0; i < 3; i++ {
		live.Update(tui.Group(
			tui.Text("🤔").Fg(tui.ColorYellow),
			tui.Text(" Thinking..."),
		))
		time.Sleep(200 * time.Millisecond)
	}

	// Simulate streaming response
	response := fmt.Sprintf("I received your message: \"%s\". This is a simulated response!", userInput)
	var partial string
	for _, char := range response {
		partial += string(char)
		live.Update(tui.Stack(
			tui.Text("Assistant:").Bold().Fg(tui.ColorBlue),
			tui.Padding(1, tui.Text("%s", partial)),
		))
		time.Sleep(20 * time.Millisecond)
	}

	// Finalize (print to scrollback)
	live.Stop()
	fmt.Println()
}

// fileAutocomplete provides file completion for @filename syntax
func fileAutocomplete(input string, cursorPos int) ([]string, int) {
	// Find @ before cursor
	atIdx := strings.LastIndex(input[:cursorPos], "@")
	if atIdx == -1 {
		return nil, 0
	}

	// Get the prefix after @
	prefix := input[atIdx+1 : cursorPos]

	// Get files in current directory
	files, err := os.ReadDir(".")
	if err != nil {
		return nil, 0
	}

	// Filter matches
	var matches []string
	for _, f := range files {
		name := f.Name()
		if strings.HasPrefix(name, prefix) {
			// Add trailing / for directories
			if f.IsDir() {
				name += "/"
			}
			matches = append(matches, name)
		}
	}

	// If there are Go files, also check parent directory
	if len(matches) == 0 && filepath.Ext(prefix) == ".go" {
		parentFiles, err := os.ReadDir("..")
		if err == nil {
			for _, f := range parentFiles {
				name := f.Name()
				if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".go") {
					matches = append(matches, name)
				}
			}
		}
	}

	// Return position after @ so the @ is preserved
	return matches, atIdx + 1
}
