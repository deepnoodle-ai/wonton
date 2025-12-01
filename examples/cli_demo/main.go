// Package main demonstrates the Wonton CLI framework.
// Run with: go run examples/cli_demo/main.go --help
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/wonton/cli"
	"github.com/deepnoodle-ai/wonton/tui"
)

func main() {
	app := cli.New("agent", "An AI agent CLI demo")
	app.Version("0.1.0")

	// Simple command with arguments
	app.Command("greet", "Greet someone", cli.WithArgs("name")).
		Run(func(ctx *cli.Context) error {
			name := ctx.Arg(0)
			if name == "" {
				name = "World"
			}
			ctx.Printf("Hello, %s!\n", name)
			return nil
		})

	// Command with flags
	runCmd := app.Command("run", "Run a prompt")
	runCmd.AddFlag(&cli.Flag{
		Name:        "model",
		Short:       "m",
		Description: "Model to use",
		Default:     "claude-sonnet",
	})
	runCmd.AddFlag(&cli.Flag{
		Name:        "temperature",
		Short:       "t",
		Description: "Temperature for generation",
		Default:     0.7,
	})
	runCmd.AddFlag(&cli.Flag{
		Name:        "format",
		Short:       "f",
		Description: "Output format",
		Default:     "text",
		Enum:        []string{"text", "json", "markdown"},
	})
	runCmd.AddFlag(&cli.Flag{
		Name:        "stream",
		Short:       "s",
		Description: "Stream output",
		Default:     false,
	})
	runCmd.AddArg(&cli.Arg{
		Name:        "prompt",
		Description: "The prompt to run",
		Required:    true,
	})
	runCmd.Run(func(ctx *cli.Context) error {
		model := ctx.String("model")
		temp := ctx.Float64("temperature")
		format := ctx.String("format")
		stream := ctx.Bool("stream")
		prompt := ctx.Arg(0)

		ctx.Printf("Model: %s\n", model)
		ctx.Printf("Temperature: %.1f\n", temp)
		ctx.Printf("Format: %s\n", format)
		ctx.Printf("Stream: %v\n", stream)
		ctx.Printf("Prompt: %s\n", prompt)

		// Simulate response
		ctx.Printf("\nResponse: This is a simulated response to '%s'\n", prompt)
		return nil
	})

	// Interactive command that uses Wonton views
	app.Command("select", "Select a model interactively").
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("interactive terminal required").
					Hint("Run in a terminal with TTY support")
			}

			models := []string{
				"claude-sonnet-4",
				"claude-opus-4",
				"gpt-4o",
				"gpt-4o-mini",
				"gemini-pro",
			}

			selected, err := cli.Select(ctx, "Select a model:", models...)
			if err != nil {
				return err
			}

			ctx.Printf("You selected: %s\n", selected)
			return nil
		})

	// Command group for users
	users := app.Group("users", "User management")
	users.Command("list", "List users").
		Run(func(ctx *cli.Context) error {
			ctx.Println("Users:")
			ctx.Println("  - alice")
			ctx.Println("  - bob")
			ctx.Println("  - charlie")
			return nil
		})

	users.Command("create", "Create a user", cli.WithArgs("username")).
		Run(func(ctx *cli.Context) error {
			username := ctx.Arg(0)
			ctx.Printf("Created user: %s\n", username)
			return nil
		})

	// AI Tool command (generates schema)
	createFile := app.Command("create-file", "Create a file", cli.WithTool())
	createFile.AddArg(&cli.Arg{Name: "path", Description: "File path", Required: true})
	createFile.AddArg(&cli.Arg{Name: "content", Description: "File content", Required: true})
	createFile.Run(func(ctx *cli.Context) error {
		path := ctx.Arg(0)
		content := ctx.Arg(1)
		ctx.Printf("Would create file at %s with content:\n%s\n", path, content)
		return nil
	})

	// Tools schema output
	app.Command("tools", "Output tool schemas as JSON").
		Run(cli.PrintToolsJSON)

	// Full interactive TUI command
	app.Command("chat", "Interactive chat session").
		Run(func(ctx *cli.Context) error {
			if !ctx.Interactive() {
				return cli.Error("chat requires an interactive terminal")
			}

			chatApp := &ChatApp{
				messages: []Message{
					{Role: "system", Content: "Welcome to the chat! Type a message and press Enter."},
				},
			}
			return cli.RunInteractive(ctx, chatApp, tui.WithMouseTracking(true))
		})

	// Run the app
	if err := app.Run(); err != nil {
		if cli.IsHelpRequested(err) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(cli.GetExitCode(err))
	}
}

// ChatApp is a simple chat TUI
type ChatApp struct {
	messages []Message
	input    string
}

type Message struct {
	Role    string
	Content string
}

func (app *ChatApp) View() tui.View {
	views := make([]tui.View, 0)

	// Header
	views = append(views, tui.Text("Chat Demo").Bold().Fg(tui.ColorCyan))
	views = append(views, tui.Divider())

	// Messages
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
			prefix = "Assistant: "
		}
		views = append(views, tui.Text("%s%s", prefix, msg.Content).Style(style))
	}

	views = append(views, tui.Spacer())

	// Input
	views = append(views, tui.Divider())
	views = append(views, tui.HStack(
		tui.Text("> "),
		tui.Input(&app.input).Placeholder("Type a message...").Width(60).OnSubmit(app.onSubmit),
	))
	views = append(views, tui.Text("Press Enter to send, Ctrl+C to quit").Dim())

	return tui.VStack(views...).Padding(1)
}

func (app *ChatApp) onSubmit(text string) {
	text = strings.TrimSpace(text)
	if text != "" {
		app.messages = append(app.messages, Message{Role: "user", Content: text})
		// Simulate response
		app.messages = append(app.messages, Message{
			Role:    "assistant",
			Content: fmt.Sprintf("You said: %s", text),
		})
		app.input = ""
	}
}

func (app *ChatApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyCtrlC:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}
