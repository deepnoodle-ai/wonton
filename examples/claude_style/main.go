package main

import (
	"fmt"
	"image"
	"log"
	"strings"
	"time"

	"github.com/deepnoodle-ai/gooey"
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
// - Scrollable message history
// - Clean, modern design similar to Claude Code
// - Proper keyboard input handling (Shift+Enter for newlines)
type ClaudeStyleDemo struct {
	messages []Message
	input    string

	// Command history
	history      []string // Past commands
	historyIndex int      // Current position in history (-1 = not browsing)
	savedInput   string   // Input saved when starting to browse history

	// UI state
	scrollOffset int
	maxScroll    int
	width        int
	height       int
}

// Init implements the Initializable interface
func (d *ClaudeStyleDemo) Init() error {
	d.historyIndex = -1 // Not browsing history
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
func (d *ClaudeStyleDemo) View() gooey.View {
	// Calculate input area height (number of lines + 2 for prompt and help)
	inputLines := strings.Count(d.input, "\n") + 1
	inputAreaHeight := inputLines + 2 // +2 for prompt line and help line

	// Ensure input area doesn't take more than half the screen (assume reasonable terminal)
	maxInputHeight := 20 // fallback if we don't have height info
	if d.height > 0 {
		maxInputHeight = d.height / 2
	}
	if inputAreaHeight > maxInputHeight {
		inputAreaHeight = maxInputHeight
	}

	return gooey.VStack(
		// Message content area (flexible - takes remaining space)
		d.renderContentView(),

		// Separator line
		gooey.Height(1, gooey.Fill('─').Fg(gooey.ColorCyan)),

		// Input area (fixed height based on input content)
		gooey.Height(inputAreaHeight, d.renderInputView()),
	)
}

// HandleEvent processes events from the runtime
func (d *ClaudeStyleDemo) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		return d.handleKeyEvent(e)

	case gooey.MouseEvent:
		return d.handleMouseEvent(e)

	case gooey.ResizeEvent:
		d.width = e.Width
		d.height = e.Height
		return nil
	}

	return nil
}

func (d *ClaudeStyleDemo) handleKeyEvent(event gooey.KeyEvent) []gooey.Cmd {
	switch {
	case event.Key == gooey.KeyCtrlC:
		return []gooey.Cmd{gooey.Quit()}

	case event.Key == gooey.KeyEnter:
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
				d.scrollOffset = 0 // Reset scroll to show new message
			}
		}

	case event.Key == gooey.KeyBackspace:
		// Delete character
		if len(d.input) > 0 {
			d.input = d.input[:len(d.input)-1]
		}
		// Reset history browsing when editing
		d.historyIndex = -1

	case event.Key == gooey.KeyArrowUp:
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

	case event.Key == gooey.KeyArrowDown:
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

	case event.Key == gooey.KeyPageUp:
		// Page up - scroll content
		d.scrollOffset = min(d.scrollOffset+10, d.maxScroll)

	case event.Key == gooey.KeyPageDown:
		// Page down - scroll content
		d.scrollOffset = max(d.scrollOffset-10, 0)

	case event.Rune != 0:
		// Regular character - reset history browsing
		d.input += string(event.Rune)
		d.historyIndex = -1
	}

	return nil
}

func (d *ClaudeStyleDemo) handleMouseEvent(event gooey.MouseEvent) []gooey.Cmd {
	// Handle scroll wheel events
	if event.Type == gooey.MouseScroll {
		switch event.Button {
		case gooey.MouseButtonWheelUp:
			// Scroll up (show older messages)
			if d.scrollOffset < d.maxScroll {
				d.scrollOffset++
			}
		case gooey.MouseButtonWheelDown:
			// Scroll down (show newer messages)
			if d.scrollOffset > 0 {
				d.scrollOffset--
			}
		}
	}
	return nil
}

// renderContentView returns a declarative view for the message content area
func (d *ClaudeStyleDemo) renderContentView() gooey.View {
	return gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
		if bounds.Empty() {
			return
		}

		width := bounds.Dx()
		height := bounds.Dy()

		// Clear the content area first to prevent ghost text when scrolling
		clearStyle := gooey.NewStyle()
		for y := 0; y < height; y++ {
			frame.FillStyled(0, y, width, 1, ' ', clearStyle)
		}

		// Build all content lines from messages
		type contentLine struct {
			text  string
			style gooey.Style
		}
		var allLines []contentLine
		userStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
		assistantStyle := gooey.NewStyle()

		for i, msg := range d.messages {
			// Add spacing between messages (except before first)
			if i > 0 {
				allLines = append(allLines, contentLine{"", clearStyle})
			}

			// Get the style for this message
			style := assistantStyle
			if msg.Role == "user" {
				style = userStyle
			}

			// Format and add message lines
			lines := d.formatMessage(msg, width-4)
			for _, line := range lines {
				allLines = append(allLines, contentLine{line, style})
			}
		}

		// Calculate max scroll (in lines)
		totalLines := len(allLines)
		d.maxScroll = max(0, totalLines-height)

		// Calculate which lines to display
		// scrollOffset=0 means show the bottom (newest), higher values scroll up (older)
		endLine := totalLines - d.scrollOffset
		startLine := endLine - height

		// Clamp to valid range
		if startLine < 0 {
			startLine = 0
		}
		if endLine > totalLines {
			endLine = totalLines
		}

		// Render visible lines
		screenY := 0
		for i := startLine; i < endLine && screenY < height; i++ {
			frame.PrintStyled(2, screenY, allLines[i].text, allLines[i].style)
			screenY++
		}
	})
}

func (d *ClaudeStyleDemo) formatMessage(msg Message, maxWidth int) []string {
	var lines []string

	// Add header
	var header string
	if msg.Role == "user" {
		header = "You:"
	} else {
		header = "Claude Code:"
	}
	lines = append(lines, header)

	// Wrap content
	contentLines := wrapText(msg.Content, maxWidth-2)
	for _, line := range contentLines {
		lines = append(lines, "  "+line)
	}

	return lines
}

func wrapText(text string, maxWidth int) []string {
	var lines []string
	paragraphs := strings.Split(text, "\n")

	for _, para := range paragraphs {
		if para == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) <= maxWidth {
				currentLine += " " + word
			} else {
				lines = append(lines, currentLine)
				currentLine = word
			}
		}
		if currentLine != "" {
			lines = append(lines, currentLine)
		}
	}

	return lines
}

// renderInputView returns a declarative view for the input area
func (d *ClaudeStyleDemo) renderInputView() gooey.View {
	return gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
		if bounds.Empty() {
			return
		}

		width := bounds.Dx()
		height := bounds.Dy()

		// Clear the entire input area first to remove any old text
		clearStyle := gooey.NewStyle()
		for clearY := 0; clearY < height; clearY++ {
			frame.FillStyled(0, clearY, width, 1, ' ', clearStyle)
		}

		// Split input into lines
		inputLines := strings.Split(d.input, "\n")

		// Calculate how many lines we can display
		maxDisplayLines := height - 1 // -1 for help text
		if maxDisplayLines < 1 {
			maxDisplayLines = 1
		}
		startLine := 0
		if len(inputLines) > maxDisplayLines {
			startLine = len(inputLines) - maxDisplayLines
		}

		// Draw prompt and input lines
		promptStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()
		inputStyle := gooey.NewStyle()

		currentY := 0
		for i := startLine; i < len(inputLines) && currentY < height-1; i++ {
			// Draw prompt only on first line
			if i == startLine {
				frame.PrintStyled(0, currentY, "> ", promptStyle)
			} else {
				frame.PrintStyled(0, currentY, "  ", promptStyle)
			}

			// Draw line content
			line := inputLines[i]
			if len(line) > width-3 {
				line = line[:width-3]
			}
			frame.PrintStyled(2, currentY, line, inputStyle)

			// Draw cursor if this is the last line
			if i == len(inputLines)-1 {
				cursorX := 2 + len(line)
				if cursorX < width {
					cursorStyle := gooey.NewStyle().WithReverse()
					frame.PrintStyled(cursorX, currentY, " ", cursorStyle)
				}
			}

			currentY++
		}

		// Draw help text
		helpStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite).WithDim()
		helpText := "Ctrl+C: exit | Enter: send | Shift+Enter: newline | ↑↓: history"
		if len(helpText) < width {
			helpX := width - len(helpText)
			frame.PrintStyled(helpX, height-1, helpText, helpStyle)
		}
	})
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
		return "The Gooey library includes:\n\n• Flicker-free rendering with double buffering\n• 30+ FPS animations\n• Composable widget system\n• Mouse and keyboard input handling\n• Flexible layout managers (VBox, HBox, Flex)\n• Rich text styling and colors\n\nWould you like to learn more about any specific feature?"
	}

	if strings.Contains(input, "example") {
		return "Here are some examples you can run:\n\n```bash\ngo run examples/all/main.go\ngo run examples/interactive/main.go\ngo run examples/composition_demo/main.go\n```\n\nEach demonstrates different library features!"
	}

	// Default response
	return fmt.Sprintf("You said: %s\n\nThis is a demo showing a Claude Code-style interface with:\n• Fixed input at the bottom\n• Scrollable message history\n• Clean, modern design\n\nTry asking about 'features', 'examples', or 'help'!", input)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	if err := gooey.Run(&ClaudeStyleDemo{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
