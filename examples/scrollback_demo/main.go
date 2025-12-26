// Package main demonstrates the scrollback + live region pattern.
//
// This example shows how to build a CLI app with:
//   - No alternate screen (stays in normal terminal mode)
//   - Raw input (captures individual keypresses)
//   - Static scrollback history (above) - printed with tui.Print
//   - Dynamic live region (below) - updated with tui.LivePrinter
//
// This pattern is ideal for chat interfaces, REPL-style tools, and CLIs
// where you want persistent history with a live status/input area.
//
// Run with: go run ./examples/scrollback_demo
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/terminal"
	"github.com/deepnoodle-ai/wonton/tui"
	"golang.org/x/term"
)

// AppState holds the runtime state of the demo application.
type AppState struct {
	// Messages in the scrollback history
	messages []Message

	// Current input buffer
	input []rune

	// Cursor position in input
	cursorPos int

	// Status text shown in the live region
	status string

	// Counter for async activity simulation
	activityCounter int

	// Whether the app is running
	running bool
}

// Message represents a message in the scrollback history.
type Message struct {
	Text   string
	IsUser bool
	Time   time.Time
}

func main() {
	// Print welcome header (becomes scrollback history)
	printHeader()

	// Check if stdin is a terminal
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		fmt.Println("This demo requires an interactive terminal.")
		return
	}

	// Save terminal state and enable raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Printf("Failed to enable raw mode: %v\n", err)
		return
	}
	defer term.Restore(fd, oldState)

	// Enable bracketed paste mode for better paste handling
	fmt.Print("\033[?2004h")
	defer fmt.Print("\033[?2004l")

	// Initialize app state
	state := &AppState{
		messages: []Message{
			{Text: "Welcome! Type a message and press Enter.", IsUser: false, Time: time.Now()},
		},
		status:  "Ready",
		running: true,
	}

	// Create the live printer for the dynamic bottom region
	live := tui.NewLivePrinter(tui.WithWidth(80))
	defer live.Stop()

	// Render initial state
	live.Update(state.buildLiveView())

	// Start async activity simulation in background
	activityChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				activityChan <- struct{}{}
			case <-activityChan:
				return
			}
		}
	}()

	// Main input loop
	decoder := terminal.NewKeyDecoder(os.Stdin)

	for state.running {
		// Use a timeout-based approach to handle both input and async updates
		// In a real app, you'd use a select with channels

		// Check for keyboard input (non-blocking would be ideal, but we'll use ReadKeyEvent)
		// For demo purposes, we'll just do blocking reads

		event, err := decoder.ReadKeyEvent()
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		// Handle the event
		state.handleKeyEvent(event, live)

		// Update the live region
		if state.running {
			live.Update(state.buildLiveView())
		}
	}

	// Print final message to scrollback
	fmt.Println("\nGoodbye!")
}

// printHeader prints the initial header that becomes part of scrollback history.
func printHeader() {
	tui.Print(
		tui.Bordered(
			tui.Stack(
				tui.Text("Scrollback + Live Region Demo").Bold().Fg(tui.ColorCyan),
				tui.Text(""),
				tui.Text("This demonstrates:"),
				tui.Text("  - Static scrollback history (above)"),
				tui.Text("  - Dynamic live region (below)"),
				tui.Text("  - Raw keyboard input"),
				tui.Text(""),
				tui.Text("Commands:"),
				tui.Text("  Enter    - Submit message (adds to scrollback)"),
				tui.Text("  Ctrl+C   - Exit"),
				tui.Text("  Ctrl+L   - Clear scrollback"),
			).Padding(1),
		).Border(&tui.RoundedBorder).BorderFg(tui.ColorBrightBlack),
		tui.WithWidth(60),
	)
	fmt.Println()
}

// handleKeyEvent processes a keyboard event.
func (s *AppState) handleKeyEvent(event terminal.KeyEvent, live *tui.LivePrinter) {
	// Handle paste events
	if event.Paste != "" {
		for _, r := range event.Paste {
			if r != '\n' && r != '\r' {
				s.insertRune(r)
			}
		}
		return
	}

	switch event.Key {
	case terminal.KeyEnter:
		if len(s.input) > 0 {
			// Clear the live region before printing to scrollback
			live.Clear()

			// Add message to scrollback
			s.printMessageToScrollback(string(s.input), true)

			// Simulate a response after a short delay
			response := s.generateResponse(string(s.input))
			s.printMessageToScrollback(response, false)

			// Clear input
			s.input = nil
			s.cursorPos = 0
			s.status = "Ready"

			// Re-initialize live printer (it was cleared)
			live.Update(s.buildLiveView())
		}

	case terminal.KeyCtrlC:
		live.Clear()
		s.running = false

	case terminal.KeyCtrlL:
		// Clear scrollback (print escape sequence)
		live.Clear()
		fmt.Print("\033[2J\033[H") // Clear screen and move cursor to home
		printHeader()
		live.Update(s.buildLiveView())

	case terminal.KeyBackspace:
		if s.cursorPos > 0 {
			s.input = append(s.input[:s.cursorPos-1], s.input[s.cursorPos:]...)
			s.cursorPos--
			s.updateStatus()
		}

	case terminal.KeyDelete:
		if s.cursorPos < len(s.input) {
			s.input = append(s.input[:s.cursorPos], s.input[s.cursorPos+1:]...)
			s.updateStatus()
		}

	case terminal.KeyArrowLeft:
		if s.cursorPos > 0 {
			s.cursorPos--
		}

	case terminal.KeyArrowRight:
		if s.cursorPos < len(s.input) {
			s.cursorPos++
		}

	case terminal.KeyHome, terminal.KeyCtrlA:
		s.cursorPos = 0

	case terminal.KeyEnd, terminal.KeyCtrlE:
		s.cursorPos = len(s.input)

	case terminal.KeyCtrlU:
		s.input = nil
		s.cursorPos = 0
		s.updateStatus()

	case terminal.KeyCtrlW:
		// Delete word backward
		if s.cursorPos > 0 {
			pos := s.cursorPos - 1
			for pos > 0 && s.input[pos] == ' ' {
				pos--
			}
			for pos > 0 && s.input[pos-1] != ' ' {
				pos--
			}
			s.input = append(s.input[:pos], s.input[s.cursorPos:]...)
			s.cursorPos = pos
			s.updateStatus()
		}

	default:
		if event.Rune != 0 {
			s.insertRune(event.Rune)
		}
	}
}

// insertRune inserts a character at the cursor position.
func (s *AppState) insertRune(r rune) {
	s.input = append(s.input[:s.cursorPos], append([]rune{r}, s.input[s.cursorPos:]...)...)
	s.cursorPos++
	s.updateStatus()
}

// updateStatus updates the status text based on input length.
func (s *AppState) updateStatus() {
	if len(s.input) == 0 {
		s.status = "Ready"
	} else {
		s.status = fmt.Sprintf("Typing... (%d chars)", len(s.input))
	}
}

// printMessageToScrollback prints a message to the scrollback history.
// This uses tui.Print which outputs to the terminal and becomes permanent history.
func (s *AppState) printMessageToScrollback(text string, isUser bool) {
	timestamp := time.Now().Format("15:04:05")

	var view tui.View
	if isUser {
		view = tui.Group(
			tui.Text("[%s] ", timestamp).Dim(),
			tui.Text("You: ").Bold().Fg(tui.ColorGreen),
			tui.Text("%s", text),
		)
	} else {
		view = tui.Group(
			tui.Text("[%s] ", timestamp).Dim(),
			tui.Text("Bot: ").Bold().Fg(tui.ColorCyan),
			tui.Text("%s", text),
		)
	}

	tui.Print(view, tui.WithWidth(80))
	fmt.Println() // Newline after each message
}

// generateResponse creates a simple response based on the input.
func (s *AppState) generateResponse(input string) string {
	input = strings.ToLower(input)

	switch {
	case strings.Contains(input, "hello") || strings.Contains(input, "hi"):
		return "Hello! How can I help you today?"
	case strings.Contains(input, "help"):
		return "I'm a simple demo bot. Try saying hello or asking about the weather!"
	case strings.Contains(input, "weather"):
		return "I don't have real weather data, but it's always sunny in the terminal!"
	case strings.Contains(input, "bye") || strings.Contains(input, "goodbye"):
		return "Goodbye! Press Ctrl+C to exit."
	default:
		return fmt.Sprintf("You said: %q. Try 'help' for more options.", input)
	}
}

// buildLiveView creates the dynamic view for the bottom live region.
// This view updates in place without scrolling the terminal.
func (s *AppState) buildLiveView() tui.View {
	// Build input line with cursor
	var inputView tui.View
	if len(s.input) == 0 {
		inputView = tui.Group(
			tui.Text("> ").Fg(tui.ColorGreen).Bold(),
			tui.Text("Type a message...").Dim(),
			tui.Text(" ").Reverse(), // Cursor
		)
	} else {
		// Split input at cursor position
		beforeCursor := string(s.input[:s.cursorPos])
		var cursorChar string
		var afterCursor string
		if s.cursorPos < len(s.input) {
			cursorChar = string(s.input[s.cursorPos])
			afterCursor = string(s.input[s.cursorPos+1:])
		} else {
			cursorChar = " "
			afterCursor = ""
		}

		inputView = tui.Group(
			tui.Text("> ").Fg(tui.ColorGreen).Bold(),
			tui.Text("%s", beforeCursor),
			tui.Text("%s", cursorChar).Reverse(),
			tui.Text("%s", afterCursor),
		)
	}

	// Build status bar
	statusView := tui.Group(
		tui.Text(" Status: ").Dim(),
		tui.Text("%s", s.status).Fg(tui.ColorYellow),
		tui.Spacer(),
		tui.Text("Ctrl+C to exit ").Dim(),
	)

	// Combine into the live region
	return tui.Stack(
		tui.Divider().Fg(tui.ColorBrightBlack),
		inputView,
		tui.Divider().Fg(tui.ColorBrightBlack),
		statusView,
	)
}
