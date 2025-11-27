# Gooey CLI Framework Brainstorm

A vision for combining Gooey's TUI capabilities with Cobra-like CLI command infrastructure, specifically targeting AI agent CLIs.

## Core Insight

The key differentiator: **Progressive Interactivity**. Most CLI tools are either:
- Fully non-interactive (traditional Unix tools)
- Fully interactive (TUI apps like lazygit)

Gooey CLI could seamlessly blend both, with the same command working as a quick one-liner OR a rich interactive experience.

## Design Sketch

### 1. Simple API, Zero Boilerplate

```go
package main

import "github.com/myzie/gooey/cli"

func main() {
    app := cli.New("agent", "An AI agent CLI")

    app.Command("chat", "Start a chat session", func(c *cli.Context) error {
        // Auto-detects: TTY? -> rich TUI. Pipe? -> streaming text
        return c.Stream(func(yield func(string)) error {
            for chunk := range llm.Chat(c.Arg(0)) {
                yield(chunk)
            }
            return nil
        })
    })

    app.Run()
}
```

### 2. Declarative Command Definition with Generics

```go
// Type-safe flags using struct tags
type RunFlags struct {
    Model    string  `flag:"model,m" default:"claude-sonnet" help:"Model to use"`
    Temp     float64 `flag:"temperature,t" default:"0.7"`
    Stream   bool    `flag:"stream,s" help:"Stream output"`
    Format   string  `flag:"format,f" enum:"text,json,markdown" default:"text"`
}

app.Command("run", "Run a prompt",
    cli.Flags[RunFlags](),
    cli.Args("prompt"),
).Run(func(c *cli.Context, flags RunFlags) error {
    // flags is fully typed, validated, with defaults applied
})
```

### 3. Hybrid Interactive/Non-Interactive

```go
app.Command("select-model", "Choose a model",
    cli.Interactive(func() cli.View {
        // Rich TUI with fuzzy search, previews
        return cli.SelectList(models, &selected).
            Preview(func(m Model) cli.View {
                return cli.VStack(
                    cli.Text(m.Name).Bold(),
                    cli.Text(m.Description).Dim(),
                    cli.KeyValue("Context", m.ContextWindow),
                    cli.KeyValue("Cost", m.PricePerToken),
                )
            })
    }),
    cli.NonInteractive(func(c *cli.Context) error {
        // Falls back to: just use --model flag or error
        return fmt.Errorf("interactive terminal required, or use --model")
    }),
)
```

### 4. AI Agent Native: Tool Definitions

This is where it gets interesting for AI agents:

```go
// Commands automatically expose as MCP-compatible tools
app.Command("create-file", "Create a file",
    cli.Param("path", "File path").Required(),
    cli.Param("content", "File content").Required(),
    cli.Tool(), // Marks as AI-callable tool
).Run(func(c *cli.Context) error {
    return os.WriteFile(c.String("path"), []byte(c.String("content")), 0644)
})

// Auto-generates tool schema
// $ agent --tools-json
// [{"name": "create-file", "params": [...], "description": "..."}]
```

### 5. Streaming & Progress (AI Agent Friendly)

```go
app.Command("generate", "Generate code", func(c *cli.Context) error {
    // Three output modes, same code:
    // 1. TTY: animated spinner + streaming text
    // 2. Pipe: raw streaming chunks
    // 3. --json: newline-delimited JSON events

    return c.WithProgress("Generating...", func(p *cli.Progress) error {
        for event := range llm.Stream(prompt) {
            switch e := event.(type) {
            case llm.Chunk:
                p.Append(e.Text)  // Streams to output
            case llm.Done:
                p.Complete()
            }
        }
        return nil
    })
})
```

### 6. Middleware Chain (Go-Idiomatic)

```go
// Global middleware
app.Use(
    cli.Logger(),                    // Structured logging
    cli.Recover(),                   // Panic recovery
    cli.Telemetry("posthog-key"),    // Usage analytics
)

// Auth middleware with auto-prompt
app.Use(cli.Auth(func(c *cli.Context) (string, error) {
    // If interactive: show login TUI
    // If not: check env var or error
    if c.Interactive() {
        return showLoginFlow(c)
    }
    return os.Getenv("API_KEY"), nil
}))

// Per-command middleware
app.Command("deploy", "Deploy to production",
    cli.Confirm("This will deploy to PRODUCTION. Continue?"),
    cli.RequireFlag("environment"),
)
```

### 7. Config Resolution (No Viper, Pure Go)

```go
type Config struct {
    APIKey   string `env:"AGENT_API_KEY" flag:"api-key" yaml:"api_key"`
    Model    string `env:"AGENT_MODEL" flag:"model" yaml:"model" default:"claude-sonnet"`
    Endpoint string `env:"AGENT_ENDPOINT" yaml:"endpoint"`
}

app := cli.New("agent", "AI Agent",
    cli.Config[Config](
        "~/.config/agent/config.yaml",  // User config
        ".agent.yaml",                   // Project config
    ),
)

// Resolution order: flag > env > project config > user config > default
```

### 8. Conversational Commands (AI Agent Pattern)

```go
// Multi-turn conversation state
app.Command("chat", "Interactive chat",
    cli.Conversation(func(conv *cli.Conversation) error {
        // Renders as: rich chat TUI in terminal
        //            or: stdin/stdout in pipe mode

        conv.System("You are a helpful assistant")

        for {
            input, err := conv.Input("You: ")
            if err == io.EOF {
                break
            }

            response := llm.Chat(conv.History(), input)
            conv.Reply("Assistant: ", response) // Streams
        }
        return nil
    }),
)
```

### 9. Rich Error Handling

```go
// Errors render beautifully
return cli.Error("Authentication failed").
    Hint("Run 'agent login' to authenticate").
    Code("AUTH_FAILED").
    Detail("Token expired at %s", expiry)

// In TTY: colored, formatted error box
// In pipe/JSON: structured error object
```

### 10. Testing Support

```go
func TestChatCommand(t *testing.T) {
    app := NewApp()

    result := app.Test(t,
        cli.Args("chat", "Hello"),
        cli.Stdin("How are you?\n"),
        cli.Env("API_KEY", "test-key"),
    )

    require.Equal(t, 0, result.ExitCode)
    require.Contains(t, result.Stdout, "Hello!")
    require.JSONEq(t, expectedEvents, result.Events) // Structured output
}
```

## Unique Gooey Advantages

Since we already have the TUI layer, we get things Cobra can't do:

| Feature | Cobra | Gooey CLI |
|---------|-------|-----------|
| Help display | Static text | Interactive browser with search |
| Prompts | Requires survey/promptui | Native, pretty, animated |
| Progress | Requires extra lib | Built-in, beautiful |
| Errors | Plain text | Rich formatting, hints |
| Selection | Requires extra lib | Native fuzzy select |
| Forms | Requires extra lib | Native input views |

## The "Agent Loop" Pattern

For AI agents specifically, a first-class pattern:

```go
app.Agent("assistant", "An AI assistant",
    cli.SystemPrompt("You are helpful..."),
    cli.Tools(
        createFileTool,
        readFileTool,
        runCommandTool,
    ),
    cli.OnToolCall(func(call ToolCall) cli.View {
        // Show what the agent is doing
        return cli.VStack(
            cli.Text("🔧 " + call.Name).Bold(),
            cli.Text(call.Args).Dim().Padding(1),
        )
    }),
)
```

## Cobra Feature Mapping

Key features from Cobra and how they'd map to Gooey CLI:

### Subcommand Architecture
Cobra supports hierarchical commands with `rootCmd.AddCommand()`. Gooey CLI would use:

```go
app := cli.New("myapp", "My application")

// Top-level commands
app.Command("version", "Show version", ...)

// Subcommands via groups
users := app.Group("users", "User management")
users.Command("list", "List users", ...)
users.Command("create", "Create a user", ...)
// Results in: myapp users list, myapp users create
```

### Persistent vs Local Flags
- **Persistent flags**: Available to command and all subcommands
- **Local flags**: Only available to specific command

```go
// Persistent (global) flags
app := cli.New("agent", "AI Agent",
    cli.GlobalFlags[GlobalConfig](), // Available everywhere
)

// Local flags per command
app.Command("run", "Run prompt",
    cli.Flags[RunFlags](), // Only for this command
)
```

### Pre and Post Run Hooks
Cobra's `PersistentPreRun` and `PersistentPostRun` become middleware:

```go
app.Use(
    cli.Before(func(c *cli.Context) error {
        // Runs before any command (like PersistentPreRun)
        return loadConfig(c)
    }),
    cli.After(func(c *cli.Context) error {
        // Runs after any command (like PersistentPostRun)
        return saveState(c)
    }),
)
```

### Argument Validation
Cobra's `cobra.ExactArgs()` and custom validation:

```go
app.Command("add", "Add item",
    cli.Args("name"),                           // Exactly 1 arg named "name"
    cli.ArgsRange(1, 3),                        // 1-3 args
    cli.Validate(func(c *cli.Context) error {   // Custom validation
        if c.Int("priority") > 10 {
            return fmt.Errorf("priority must be <= 10")
        }
        return nil
    }),
)
```

## Open Questions

1. **How deep to integrate?** Should this be `gooey/cli` subpackage or a separate module?

2. **Backward compatibility with Cobra?** Could we support Cobra command structs as an adapter?

3. **Config format?** YAML only? Or support TOML, JSON, env files?

4. **Shell completions?** Auto-generate for bash/zsh/fish?

5. **Man page generation?** Auto-generate from command definitions?

## Implementation Phases

### Phase 1: Core CLI
- Command registration and parsing
- Flag parsing with struct tags
- Help generation
- Basic TTY detection

### Phase 2: Interactive Features
- Integrate with Gooey views for prompts
- Progress indicators
- Fuzzy select
- Rich error display

### Phase 3: AI Agent Features
- Tool schema generation
- Streaming output modes
- Conversation state management
- MCP compatibility

### Phase 4: Polish
- Shell completions
- Man page generation
- Config file support
- Telemetry hooks
