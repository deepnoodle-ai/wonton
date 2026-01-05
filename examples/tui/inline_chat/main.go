// Example: inline_chat
//
// This example demonstrates advanced InlineApp features:
// - Rich styled views in scrollback
// - Async operations with the Cmd pattern
// - Live status updates during processing
// - Keyboard input handling
//
// Run with: go run ./examples/inline_chat
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/deepnoodle-ai/wonton/tui"
)

// ResponseEvent is a custom event for async responses
type ResponseEvent struct {
	Text string
	Time time.Time
}

func (e ResponseEvent) Timestamp() time.Time { return e.Time }

type ChatApp struct {
	runner   *tui.InlineApp
	input    []rune
	thinking bool
}

func (app *ChatApp) LiveView() tui.View {
	var statusView tui.View
	if app.thinking {
		statusView = tui.Text(" Thinking...").Fg(tui.ColorYellow)
	} else {
		statusView = tui.Text(" Ready").Fg(tui.ColorGreen)
	}

	// Show input with cursor
	inputText := string(app.input) + "\u2588" // Block cursor

	return tui.Stack(
		tui.Divider(),
		tui.Group(
			tui.Text("> ").Fg(tui.ColorCyan).Bold(),
			tui.Text("%s", inputText),
		),
		tui.Divider(),
		statusView,
	)
}

func (app *ChatApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyEnter:
			if len(app.input) > 0 && !app.thinking {
				msg := string(app.input)
				app.input = nil
				app.thinking = true

				// Print user message to scrollback
				app.runner.Print(app.formatMessage(msg, true))

				// Start async response generation
				return []tui.Cmd{app.generateResponse(msg)}
			}

		case tui.KeyBackspace:
			if len(app.input) > 0 {
				app.input = app.input[:len(app.input)-1]
			}

		case tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}

		default:
			if e.Rune != 0 && !app.thinking {
				app.input = append(app.input, e.Rune)
			}
		}

	case ResponseEvent:
		// Async response arrived
		app.thinking = false
		app.runner.Print(app.formatMessage(e.Text, false))
	}
	return nil
}

func (app *ChatApp) formatMessage(text string, isUser bool) tui.View {
	timestamp := time.Now().Format("15:04:05")

	var sender tui.View
	if isUser {
		sender = tui.Text("You: ").Bold().Fg(tui.ColorGreen)
	} else {
		sender = tui.Text("Bot: ").Bold().Fg(tui.ColorCyan)
	}

	return tui.Group(
		tui.Text("[%s] ", timestamp).Dim(),
		sender,
		tui.Text("%s", text),
	)
}

func (app *ChatApp) generateResponse(input string) tui.Cmd {
	return func() tui.Event {
		// Simulate async API call
		time.Sleep(500 * time.Millisecond)
		return ResponseEvent{
			Text: fmt.Sprintf("You said: %q", input),
			Time: time.Now(),
		}
	}
}

func main() {
	fmt.Println("Chat Demo - Type messages and press Enter")
	fmt.Println("Press Ctrl+C to quit")
	fmt.Println()

	app := &ChatApp{}
	app.runner = tui.NewInlineApp(tui.InlineAppConfig{
		Width:          80,
		BracketedPaste: true,
	})

	if err := app.runner.Run(app); err != nil {
		log.Fatal(err)
	}
}
