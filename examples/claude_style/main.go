package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

type Message struct {
	Role    string // "user" or "assistant"
	Content string
	Time    time.Time
}

type ClaudeStyleDemo struct {
	terminal *gooey.Terminal
	decoder  *gooey.KeyDecoder
	messages []Message
	input    string

	// UI state
	scrollOffset int
	maxScroll    int

	// Synchronization
	mu sync.RWMutex
}

func NewClaudeStyleDemo() (*ClaudeStyleDemo, error) {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		return nil, err
	}

	demo := &ClaudeStyleDemo{
		terminal: terminal,
		decoder:  gooey.NewKeyDecoder(os.Stdin),
		messages: []Message{
			{
				Role:    "assistant",
				Content: "Hello! I'm Claude Code. I can help you with software engineering tasks.\n\nTry typing a message below and press Enter.",
				Time:    time.Now(),
			},
		},
	}

	return demo, nil
}

func (d *ClaudeStyleDemo) Run() error {
	defer d.terminal.Close()

	// Open debug log file
	debugLog, err := os.OpenFile("/tmp/claude_style_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		defer debugLog.Close()
		log.SetOutput(debugLog)
		log.Println("=== Starting Claude Style Demo ===")
	}

	// Enable raw mode for keyboard input
	d.terminal.EnableRawMode()
	defer d.terminal.DisableRawMode()

	// Use alternate screen buffer
	d.terminal.EnableAlternateScreen()
	defer d.terminal.DisableAlternateScreen()

	// Clear screen initially
	d.terminal.Clear()

	// Hide cursor for cleaner UI
	d.terminal.HideCursor()
	defer d.terminal.ShowCursor()

	// Main event loop - simple blocking approach
	for {
		// Render current state
		if err := d.render(); err != nil {
			return err
		}

		// Read and process one key event (blocking)
		event, err := d.decoder.ReadKeyEvent()
		if err != nil {
			return err
		}

		log.Printf("KeyEvent: Key=%d, Rune=%q, Alt=%v, Ctrl=%v, Shift=%v",
			event.Key, event.Rune, event.Alt, event.Ctrl, event.Shift)

		// Process the event
		if d.processKeyEvent(event) {
			return nil // Exit requested
		}

		// Render again immediately after processing to show updated state
		if err := d.render(); err != nil {
			return err
		}
	}
}

func (d *ClaudeStyleDemo) render() error {
	frame, err := d.terminal.BeginFrame()
	if err != nil {
		return err
	}
	defer d.terminal.EndFrame(frame)

	width, height := frame.Size()

	// Lock for reading state
	d.mu.RLock()

	// Calculate input area height (number of lines + 2 for prompt and help)
	inputLines := strings.Count(d.input, "\n") + 1
	inputAreaHeight := inputLines + 2 // +2 for prompt line and help line

	log.Printf("RENDER: input=%q, newlines=%d, inputLines=%d, inputAreaHeight=%d",
		d.input, strings.Count(d.input, "\n"), inputLines, inputAreaHeight)

	// Ensure input area doesn't take more than half the screen
	maxInputHeight := height / 2
	if inputAreaHeight > maxInputHeight {
		inputAreaHeight = maxInputHeight
	}

	// Calculate areas
	contentHeight := height - inputAreaHeight - 1 // -1 for separator
	inputY := contentHeight

	// Draw content area
	d.renderContentLocked(frame, width, contentHeight)

	// Draw separator line
	separatorStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan)
	frame.FillStyled(0, inputY, width, 1, '─', separatorStyle)

	// Draw input area
	d.renderInputLocked(frame, inputY+1, width, inputAreaHeight-1)

	d.mu.RUnlock()

	return nil
}

func (d *ClaudeStyleDemo) renderContentLocked(frame gooey.RenderFrame, width, height int) {
	// IMPORTANT: Assumes d.mu.RLock() is held by caller
	// Render messages from bottom to top
	currentY := height - 1

	for i := len(d.messages) - 1 - d.scrollOffset; i >= 0 && currentY >= 0; i-- {
		msg := d.messages[i]

		// Render message
		lines := d.formatMessage(msg, width-4)
		msgHeight := len(lines)

		// Calculate starting Y position for this message
		startY := currentY - msgHeight + 1

		if startY < 0 {
			// Message partially visible, only show bottom part
			lines = lines[len(lines)-(currentY+1):]
			startY = 0
		}

		// Draw message
		for j, line := range lines {
			y := startY + j
			if y >= 0 && y < height {
				if msg.Role == "user" {
					// User messages in cyan
					style := gooey.NewStyle().WithForeground(gooey.ColorCyan)
					frame.PrintStyled(2, y, line, style)
				} else {
					// Assistant messages in default color
					defaultStyle := gooey.NewStyle()
					frame.PrintStyled(2, y, line, defaultStyle)
				}
			}
		}

		currentY = startY - 2 // Add spacing between messages
	}

	// Update max scroll
	totalLines := 0
	for _, msg := range d.messages {
		lines := d.formatMessage(msg, width-4)
		totalLines += len(lines) + 2 // +2 for spacing
	}
	d.maxScroll = max(0, (totalLines-height)/2)
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

func (d *ClaudeStyleDemo) renderInputLocked(frame gooey.RenderFrame, y, width, availableHeight int) {
	// IMPORTANT: Assumes d.mu.RLock() is held by caller

	// Clear the entire input area first to remove any old text
	clearStyle := gooey.NewStyle()
	for clearY := y; clearY < y+availableHeight; clearY++ {
		frame.FillStyled(0, clearY, width, 1, ' ', clearStyle)
	}

	// Split input into lines
	inputLines := strings.Split(d.input, "\n")

	// Calculate how many lines we can display
	maxDisplayLines := availableHeight - 1 // -1 for help text
	startLine := 0
	if len(inputLines) > maxDisplayLines {
		startLine = len(inputLines) - maxDisplayLines
	}

	// Draw prompt and input lines
	promptStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()
	inputStyle := gooey.NewStyle()

	currentY := y
	for i := startLine; i < len(inputLines) && currentY < y+availableHeight-1; i++ {
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
	helpText := "Ctrl+C: exit | Enter: send | Shift+Enter: new line"
	if len(helpText) < width {
		helpX := width - len(helpText)
		frame.PrintStyled(helpX, y+availableHeight-1, helpText, helpStyle)
	}
}

func (d *ClaudeStyleDemo) processKeyEvent(event gooey.KeyEvent) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	switch {
	// Handle regular characters first (Rune != 0)
	case event.Rune != 0:
		d.input += string(event.Rune)
		log.Printf("Added char to input. Input now: %q (len=%d)", d.input, len(d.input))

	case event.Key == gooey.KeyCtrlC:
		log.Println("Ctrl+C pressed, exiting")
		return true // Exit

	case event.Key == gooey.KeyEnter:
		log.Printf("=== ENTER KEY PRESSED === Alt=%v, Shift=%v, input=%q", event.Alt, event.Shift, d.input)
		// Alt+Enter or Shift+Enter adds a new line, plain Enter sends the message
		if event.Alt || event.Shift {
			log.Printf("*** ADDING NEWLINE (Alt=%v, Shift=%v) ***", event.Alt, event.Shift)
			d.input += "\n"
			log.Printf("After adding newline: input=%q (len=%d, newlines=%d)",
				d.input, len(d.input), strings.Count(d.input, "\n"))
		} else {
			// Send message
			trimmed := strings.TrimSpace(d.input)
			log.Printf("Trimmed input: %q (len=%d, empty=%v)", trimmed, len(trimmed), trimmed == "")
			if trimmed != "" {
				log.Println("Sending message and clearing input")
				d.sendMessageLocked(d.input)
				d.input = "" // Clear input after sending
				d.scrollOffset = 0 // Reset scroll to show new message
				log.Printf("Input cleared. Input now: %q", d.input)
			} else {
				log.Println("Input is empty after trim, not sending")
			}
		}

	case event.Key == gooey.KeyBackspace:
		// Delete character
		if len(d.input) > 0 {
			d.input = d.input[:len(d.input)-1]
		}

	case event.Key == gooey.KeyArrowUp:
		// Scroll up
		if d.scrollOffset < d.maxScroll {
			d.scrollOffset++
		}

	case event.Key == gooey.KeyArrowDown:
		// Scroll down
		if d.scrollOffset > 0 {
			d.scrollOffset--
		}

	case event.Key == gooey.KeyPageUp:
		// Page up
		d.scrollOffset = min(d.scrollOffset+10, d.maxScroll)

	case event.Key == gooey.KeyPageDown:
		// Page down
		d.scrollOffset = max(d.scrollOffset-10, 0)
	}

	return false
}

func (d *ClaudeStyleDemo) sendMessageLocked(text string) {
	// IMPORTANT: Assumes d.mu.Lock() is held by caller
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
	demo, err := NewClaudeStyleDemo()
	if err != nil {
		fmt.Printf("Error creating demo: %v\n", err)
		return
	}

	if err := demo.Run(); err != nil {
		fmt.Printf("Error running demo: %v\n", err)
	}
}
