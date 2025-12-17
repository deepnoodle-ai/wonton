package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

// Message represents a chat message
type Message struct {
	Role    string // "user" or "assistant"
	Content string
	Time    time.Time
}

// ClaudeStyleDemo implements a Claude Code-style interface with fixed input at the bottom
// using the Runtime message-driven architecture.
//
// This example shows:
// - Fixed input area at bottom with multi-line support
// - Message history above
// - Clean, modern design similar to Claude Code
// - Proper keyboard input handling (Shift+Enter for newlines)
type ClaudeStyleDemo struct {
	messages []Message
	input    string

	// Command history
	history      []string // Past commands
	historyIndex int      // Current position in history (-1 = not browsing)
	savedInput   string   // Input saved when starting to browse history

	// Scroll position for message area
	// Large initial value gets clamped to maxScroll (bottom)
	scrollY int
}

// Init implements the Initializable interface
func (d *ClaudeStyleDemo) Init() error {
	d.historyIndex = -1 // Not browsing history
	d.scrollY = 999999  // Start at bottom (gets clamped to maxScroll)
	d.messages = []Message{
		{
			Role:    "assistant",
			Content: "Hello! I'm Claude Code. I can help you with software engineering tasks.\n\nTry typing a message below and press Enter.",
			Time:    time.Now(),
		},
	}
	return nil
}

// View returns the declarative UI for this app
func (d *ClaudeStyleDemo) View() tui.View {
	// Calculate footer height: divider (1) + input lines + help line (1)
	inputLines := strings.Count(d.input, "\n") + 1
	footerHeight := 1 + inputLines + 1

	return tui.Stack(
		// Scrollable message area, anchored to bottom
		tui.Scroll(d.renderMessages(), &d.scrollY).Bottom(),

		// Fixed footer: separator + input area
		tui.Height(footerHeight, tui.Stack(
			tui.Divider().Fg(tui.ColorCyan),
			d.renderInputArea(),
		)),
	)
}

// HandleEvent processes events from the runtime
func (d *ClaudeStyleDemo) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		return d.handleKeyEvent(e)
	case tui.MouseEvent:
		return d.handleMouseEvent(e)
	}

	return nil
}

func (d *ClaudeStyleDemo) handleMouseEvent(event tui.MouseEvent) []tui.Cmd {
	if event.Type == tui.MouseScroll {
		switch event.Button {
		case tui.MouseButtonWheelUp:
			// Scroll up to see older messages
			d.scrollY -= 3
			if d.scrollY < 0 {
				d.scrollY = 0
			}
		case tui.MouseButtonWheelDown:
			// Scroll down to see newer messages
			d.scrollY += 3
		}
	}
	return nil
}

func (d *ClaudeStyleDemo) handleKeyEvent(event tui.KeyEvent) []tui.Cmd {
	switch {
	case event.Key == tui.KeyCtrlC:
		return []tui.Cmd{tui.Quit()}

	case event.Key == tui.KeyEnter:
		if event.Shift {
			// Shift+Enter adds a new line
			d.input += "\n"
		} else {
			// Plain Enter sends the message
			trimmed := strings.TrimSpace(d.input)
			if trimmed != "" {
				// Add to history
				d.history = append(d.history, d.input)
				d.historyIndex = -1
				d.savedInput = ""

				d.sendMessage(d.input)
				d.input = ""       // Clear input after sending
				d.scrollY = 999999 // Scroll to bottom to see new message
			}
		}

	case event.Key == tui.KeyBackspace:
		// Delete character
		if len(d.input) > 0 {
			d.input = d.input[:len(d.input)-1]
		}
		// Reset history browsing when editing
		d.historyIndex = -1

	case event.Key == tui.KeyArrowUp:
		// Navigate to older history
		if len(d.history) > 0 {
			if d.historyIndex == -1 {
				// Starting to browse history, save current input
				d.savedInput = d.input
				d.historyIndex = len(d.history) - 1
			} else if d.historyIndex > 0 {
				d.historyIndex--
			}
			d.input = d.history[d.historyIndex]
		}

	case event.Key == tui.KeyArrowDown:
		// Navigate to newer history
		if d.historyIndex != -1 {
			if d.historyIndex < len(d.history)-1 {
				d.historyIndex++
				d.input = d.history[d.historyIndex]
			} else {
				// Back to current input
				d.historyIndex = -1
				d.input = d.savedInput
			}
		}

	case event.Key == tui.KeyPageUp:
		// Scroll up to see older messages (decrease offset toward top)
		d.scrollY -= 10
		if d.scrollY < 0 {
			d.scrollY = 0
		}

	case event.Key == tui.KeyPageDown:
		// Scroll down to see newer messages (increase offset toward bottom)
		// The scroll view will clamp to maxScroll
		d.scrollY += 10

	case event.Rune != 0:
		// Regular character - reset history browsing
		d.input += string(event.Rune)
		d.historyIndex = -1
	}

	return nil
}

// renderMessages returns a view for all messages
func (d *ClaudeStyleDemo) renderMessages() tui.View {
	// Build all message views
	var messageViews []tui.View

	for i, msg := range d.messages {
		// Add spacing between messages (except before first)
		if i > 0 {
			messageViews = append(messageViews, tui.Spacer().MinHeight(1))
		}

		messageViews = append(messageViews, d.renderMessage(msg))
	}

	// Wrap in a Stack with left padding
	return tui.PaddingHV(2, 0, tui.Stack(messageViews...))
}

// renderMessage returns a view for a single message
func (d *ClaudeStyleDemo) renderMessage(msg Message) tui.View {
	// Determine header and style based on role
	var header string
	var headerColor tui.Color

	if msg.Role == "user" {
		header = "You:"
		headerColor = tui.ColorCyan
	} else {
		header = "Claude Code:"
		headerColor = tui.ColorWhite
	}

	// Split content into lines for WrappedText
	contentLines := strings.Split(msg.Content, "\n")
	var contentViews []tui.View

	for _, line := range contentLines {
		if line == "" {
			contentViews = append(contentViews, tui.Text(""))
		} else {
			contentViews = append(contentViews, tui.WrappedText(line).Fg(headerColor))
		}
	}

	return tui.Stack(
		tui.Text("%s", header).Bold().Fg(headerColor),
		tui.PaddingHV(2, 0, tui.Stack(contentViews...)),
	)
}

// renderInputArea returns a view for the input area
func (d *ClaudeStyleDemo) renderInputArea() tui.View {
	// Split input into lines
	inputLines := strings.Split(d.input, "\n")

	// Build input line views
	var inputViews []tui.View

	for i, line := range inputLines {
		var prefix string
		if i == 0 {
			prefix = "> "
		} else {
			prefix = "  "
		}

		// Add cursor to the last line
		displayLine := line
		if i == len(inputLines)-1 {
			displayLine = line + "█"
		}

		inputViews = append(inputViews,
			tui.Group(
				tui.Text("%s", prefix).Bold().Fg(tui.ColorGreen),
				tui.Text("%s", displayLine),
			),
		)
	}

	// Help text at the bottom, right-aligned
	helpText := "Ctrl+C: exit | Enter: send | \\+Enter: newline | ↑↓: history | PgUp/PgDn: scroll"

	return tui.Stack(
		tui.Stack(inputViews...),
		tui.Spacer(),
		tui.Group(
			tui.Spacer(),
			tui.Text("%s", helpText).Dim(),
		),
	)
}

func (d *ClaudeStyleDemo) sendMessage(text string) {
	// Add user message
	d.messages = append(d.messages, Message{
		Role:    "user",
		Content: text,
		Time:    time.Now(),
	})

	// Generate assistant response
	response := d.generateResponse(text)
	d.messages = append(d.messages, Message{
		Role:    "assistant",
		Content: response,
		Time:    time.Now(),
	})
}

func (d *ClaudeStyleDemo) generateResponse(input string) string {
	// Simple response generation for demo purposes
	input = strings.ToLower(strings.TrimSpace(input))

	if strings.Contains(input, "hello") || strings.Contains(input, "hi") {
		return "Hello! How can I help you today?"
	}

	if strings.Contains(input, "help") {
		return "I can help you with:\n• Building TUI applications\n• Code examples and patterns\n• Terminal rendering techniques\n\nWhat would you like to know more about?"
	}

	if strings.Contains(input, "feature") {
		return "The Wonton library includes:\n\n• Flicker-free rendering with double buffering\n• 30+ FPS animations\n• Composable widget system\n• Mouse and keyboard input handling\n• Flexible layout managers (VBox, HBox, Flex)\n• Rich text styling and colors\n\nWould you like to learn more about any specific feature?"
	}

	if strings.Contains(input, "example") {
		return "Here are some examples you can run:\n\ngo run examples/all/main.go\ngo run examples/interactive/main.go\ngo run examples/composition_demo/main.go\n\nEach demonstrates different library features!"
	}

	// Default response
	return fmt.Sprintf("You said: %s\n\nThis is a demo showing a Claude Code-style interface with:\n• Fixed input at the bottom\n• Scrollable message history\n• Clean, modern design\n\nTry asking about 'features', 'examples', or 'help'!", input)
}

func main() {
	if err := tui.Run(&ClaudeStyleDemo{}, tui.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
