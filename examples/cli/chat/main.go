// Example: Conversation Pattern
//
// Demonstrates the conversation/chat pattern for multi-turn interactions:
// - Building conversation history
// - Interactive input prompts
// - Streaming responses
// - System messages
//
// Run interactively:
//
//	go run examples/cli_chat/main.go chat
//	go run examples/cli_chat/main.go repl
//
// Run non-interactively (pipe):
//
//	echo "Hello" | go run examples/cli_chat/main.go chat
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	app := cli.New("chatdemo").
		Description("Demonstrates conversation patterns").
		Version("1.0.0")

	// Simple chat command using conversation pattern
	app.Command("chat").
		Description("Start a chat session").
		Run(func(ctx *cli.Context) error {
			conv := cli.NewConversation(ctx)
			conv.System("You are a helpful assistant. This is a demo - responses are simulated.")

			ctx.Println("Chat Demo (type 'quit' to exit)")
			ctx.Println("================================")
			ctx.Println("")

			for {
				input, err := conv.Input("You: ")
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				input = strings.TrimSpace(input)
				if input == "" {
					continue
				}
				if input == "quit" || input == "exit" {
					break
				}

				// Simulate a response (in real app, call LLM here)
				response := simulateResponse(input, conv.History())
				conv.Reply("Assistant: ", response)
				ctx.Println("")
			}

			ctx.Println("\nGoodbye!")
			return nil
		})

	// REPL-style command
	app.Command("repl").
		Description("Start a REPL session").
		Run(func(ctx *cli.Context) error {
			ctx.Println("REPL Demo")
			ctx.Println("Commands: help, history, clear, quit")
			ctx.Println("")

			conv := cli.NewConversation(ctx)
			conv.SetPrompt(">>> ")

			for {
				input, err := conv.Input("")
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				input = strings.TrimSpace(input)
				if input == "" {
					continue
				}

				switch input {
				case "quit", "exit":
					ctx.Println("Bye!")
					return nil
				case "help":
					ctx.Println("Available commands:")
					ctx.Println("  help    - Show this help")
					ctx.Println("  history - Show conversation history")
					ctx.Println("  clear   - Clear history")
					ctx.Println("  quit    - Exit")
				case "history":
					ctx.Println("History:")
					for i, msg := range conv.History() {
						ctx.Printf("  [%d] %s: %s\n", i+1, msg.Role, msg.Content)
					}
				case "clear":
					// Create new conversation to clear history
					conv = cli.NewConversation(ctx)
					conv.SetPrompt(">>> ")
					ctx.Println("History cleared")
				default:
					// Echo back the input
					response := fmt.Sprintf("You said: %s", input)
					conv.Reply("", response)
				}
			}

			return nil
		})

	// Full TUI chat using Wonton
	app.Command("tui").
		Description("Full TUI chat interface").
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("TUI mode requires an interactive terminal")
			}

			chatApp := &ChatApp{
				messages: []cli.Message{
					{Role: "system", Content: "Welcome! Type a message and press Enter to send."},
				},
			}

			return cli.RunInteractive(ctx, chatApp,
				tui.WithMouseTracking(true),
				tui.WithFPS(30),
			)
		})

	// Streaming demo
	app.Command("stream").
		Description("Demonstrate streaming output").
		Args("prompt?").
		Run(func(ctx *cli.Context) error {
			prompt := ctx.Arg(0)
			if prompt == "" {
				prompt = "Tell me a story"
			}

			ctx.Printf("Prompt: %s\n\n", prompt)
			ctx.Print("Response: ")

			// Simulate streaming response
			response := simulateStreamingResponse(prompt)
			for _, char := range response {
				ctx.Print(string(char))
				time.Sleep(30 * time.Millisecond)
			}
			ctx.Println("")

			return nil
		})

	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

// simulateResponse generates a simulated response based on input
func simulateResponse(input string, history []cli.Message) string {
	input = strings.ToLower(input)

	if strings.Contains(input, "hello") || strings.Contains(input, "hi") {
		return "Hello! How can I help you today?"
	}
	if strings.Contains(input, "how are you") {
		return "I'm doing well, thank you for asking! I'm a demo assistant, so I don't have real feelings, but I appreciate the thought."
	}
	if strings.Contains(input, "name") {
		return "I'm ChatDemo, a demonstration assistant built with the Wonton CLI framework."
	}
	if strings.Contains(input, "help") {
		return "I can help you explore the conversation pattern in the CLI framework. Try asking me questions or just chat!"
	}
	if strings.Contains(input, "history") {
		return fmt.Sprintf("We have %d messages in our conversation so far.", len(history))
	}

	return fmt.Sprintf("I received your message: \"%s\". This is a simulated response - in a real application, this would call an LLM.", input)
}

// simulateStreamingResponse generates a streaming response
func simulateStreamingResponse(prompt string) string {
	return "Once upon a time, in a land of code and creativity, there lived a developer who discovered the power of terminal user interfaces. They built amazing applications that combined the simplicity of CLI with the beauty of interactive design..."
}

// ChatApp is a full TUI chat application
type ChatApp struct {
	messages []cli.Message
	input    string
}

func (app *ChatApp) View() tui.View {
	views := []tui.View{
		tui.Text("Chat Demo").Bold().Fg(tui.ColorCyan),
		tui.Divider(),
	}

	// Show messages
	for _, msg := range app.messages {
		var style tui.Style
		prefix := ""
		switch msg.Role {
		case "system":
			style = tui.NewStyle().WithForeground(tui.ColorYellow).WithItalic()
			prefix = "[System] "
		case "user":
			style = tui.NewStyle().WithForeground(tui.ColorGreen)
			prefix = "You: "
		case "assistant":
			style = tui.NewStyle().WithForeground(tui.ColorCyan)
			prefix = "Bot: "
		}
		views = append(views, tui.Text("%s%s", prefix, msg.Content).Style(style))
	}

	views = append(views, tui.Spacer())
	views = append(views, tui.Divider())
	views = append(views, tui.Group(
		tui.Text("> "),
		tui.Input(&app.input).
			Placeholder("Type a message...").
			Width(60).
			OnSubmit(app.onSubmit),
	))
	views = append(views, tui.Text("Press Enter to send, Ctrl+C to quit").Dim())

	return tui.Stack(views...).Padding(1)
}

func (app *ChatApp) onSubmit(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	// Add user message
	app.messages = append(app.messages, cli.Message{Role: "user", Content: text})

	// Generate response
	response := simulateResponse(text, app.messages)
	app.messages = append(app.messages, cli.Message{Role: "assistant", Content: response})

	app.input = ""
}

func (app *ChatApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}
