# CLAUDE.md

This file provides guidance to Claude Code when working with this codebase.

## Project Overview

Gooey is a collection of Go packages for building command-line applications. It provides everything needed to create professional CLIs: from simple argument parsing to rich terminal UIs with animations.

### Packages

| Package      | Import Path                               | Purpose                                                    |
| ------------ | ----------------------------------------- | ---------------------------------------------------------- |
| **cli**      | `github.com/deepnoodle-ai/gooey/cli`      | CLI framework with commands, flags, config, and middleware |
| **color**    | `github.com/deepnoodle-ai/gooey/color`    | ANSI color types, RGB, HSL, and gradient utilities         |
| **env**      | `github.com/deepnoodle-ai/gooey/env`      | Config loading from env vars, .env files, and JSON         |
| **slog**     | `github.com/deepnoodle-ai/gooey/slog`     | Colorized slog.Handler for terminal output                 |
| **terminal** | `github.com/deepnoodle-ai/gooey/terminal` | Low-level terminal control, input decoding, styles         |
| **tui**      | `github.com/deepnoodle-ai/gooey/tui`      | Full TUI library with declarative views and runtime        |
| **unidiff**  | `github.com/deepnoodle-ai/gooey/unidiff`  | Unified diff parsing for display and analysis              |

## Development Commands

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a specific test
go test -v -run TestName ./...

# Run examples
go run examples/declarative_counter/main.go  # Declarative counter with buttons
go run examples/declarative/main.go          # Declarative input form
go run examples/runtime_animation/main.go    # Smooth animations
go run examples/all/main.go                  # Comprehensive demo
```

## Architecture

### cli Package

CLI framework for building command-line applications with subcommands, flags, and configuration.

```go
app := cli.New("myapp", "My CLI application")
app.Version("1.0.0")

app.Command("serve", "Start the server", func(ctx *cli.Context, flags *ServeFlags) error {
    return runServer(flags.Port)
})

app.Run(os.Args)
```

Key features:

- Hierarchical command structure with groups
- Flag parsing with struct tags
- Config file support (YAML, JSON)
- Middleware chain for cross-cutting concerns
- Shell completions generation

### color Package

Color types for terminal output including ANSI colors, RGB, HSL, and gradients.

```go
c := color.Green
rgb := color.NewRGB(255, 128, 0)
hsl := color.NewHSL(180, 0.5, 0.5)
gradient := color.NewGradient(color.Red, color.Blue, 10)
```

### env Package

Configuration loading from environment variables with struct binding.

```go
type Config struct {
    Host string `env:"HOST" default:"localhost"`
    Port int    `env:"PORT" default:"8080"`
}

cfg, err := env.Parse[Config](
    env.WithPrefix("MYAPP"),
    env.WithEnvFile(".env"),
)
```

### slog Package

Colorized slog.Handler for terminal output with auto-detection.

```go
logger := slog.New(gooeyslog.NewHandler(os.Stderr, &gooeyslog.Options{
    Level:     slog.LevelDebug,
    AddSource: true,
    NoColor:   !gooeyslog.IsTerminal(os.Stderr),
}))
```

### terminal Package

Low-level terminal control with double-buffered rendering, input decoding, and styles.

- Raw mode management
- Keyboard and mouse event decoding
- Style system (colors, bold, italic, etc.)
- Double-buffered rendering with dirty region tracking
- Hyperlink support (OSC 8)

### tui Package

Full TUI library with declarative views, runtime event loop, and animations.

**Runtime**: Event loop coordinating three goroutines:

1. Main event loop (processes events, calls HandleEvent/View)
2. Input reader (reads from stdin, forwards KeyEvent/MouseEvent)
3. Command executor (runs async commands, returns results as events)

**Rendering Flow**:

```
View() -> measure views -> render views -> diff & flush
```

**Application Interface**: User code implements:

- `View() View` - Return declarative view tree (required)
- `HandleEvent(event Event) []Cmd` - Process events (optional)

### unidiff Package

Unified diff parsing for display and analysis.

```go
diff, err := unidiff.Parse(diffText)
for _, file := range diff.Files {
    for _, hunk := range file.Hunks {
        for _, line := range hunk.Lines {
            fmt.Printf("%s: %s\n", line.Type, line.Content)
        }
    }
}
```

## TUI Patterns

### Simplest Application (Recommended - Declarative)

```go
type MyApp struct {
    count int
}

func (app *MyApp) View() tui.View {
    return tui.VStack(
        tui.Text("Count: %d", app.count).Bold().Fg(tui.ColorGreen),
        tui.Spacer().MinHeight(1),
        tui.HStack(
            tui.Clickable("[ + ]", func() { app.count++ }).Fg(tui.ColorGreen),
            tui.Clickable("[ - ]", func() { app.count-- }).Fg(tui.ColorRed),
        ).Gap(2),
        tui.Spacer(),
        tui.Text("Press 'q' to quit").Dim(),
    ).Align(tui.AlignCenter).Padding(2)
}

func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
    switch e := event.(type) {
    case tui.KeyEvent:
        if e.Rune == 'q' {
            return []tui.Cmd{tui.Quit()}
        }
    }
    return nil
}

func main() {
    if err := tui.Run(&MyApp{}, tui.WithMouseTracking(true)); err != nil {
        log.Fatal(err)
    }
}
```

Options can be passed to customize behavior:

```go
tui.Run(&MyApp{},
    tui.WithFPS(60),              // Default: 30
    tui.WithMouseTracking(true),  // Default: false
    tui.WithAlternateScreen(false), // Default: true
    tui.WithHideCursor(false),    // Default: true
)
```

### Custom Drawing with Canvas

```go
// Canvas provides escape hatch for imperative drawing
tui.Canvas(func(frame tui.RenderFrame, bounds image.Rectangle) {
    width, height := bounds.Dx(), bounds.Dy()
    // Use frame.SetCell(), frame.PrintStyled(), etc.
    frame.SetCell(bounds.Min.X, bounds.Min.Y, '█', style)
})
```

### Async Operations (HTTP, etc.)

```go
// Custom event for results
type DataReceived struct {
    Data string
}
func (d DataReceived) Timestamp() time.Time { return time.Now() }

// Command that performs async work
func FetchData(url string) tui.Cmd {
    return func() tui.Event {
        resp, err := http.Get(url)
        if err != nil {
            return tui.ErrorEvent{Time: time.Now(), Err: err}
        }
        // ... process response
        return DataReceived{Data: result}
    }
}

// In HandleEvent:
case tui.KeyEvent:
    if e.Rune == 'f' {
        return []tui.Cmd{FetchData("https://api.example.com")}
    }
case DataReceived:
    app.data = e.Data
```

### Styles

```go
style := tui.NewStyle().
    WithForeground(tui.ColorGreen).
    WithBackground(tui.ColorBlack).
    WithBold().
    WithUnderline()

// RGB colors
style = tui.NewStyle().WithFgRGB(tui.NewRGB(255, 128, 0))

// Apply to text
tui.Text("styled").Style(style)
```

### Mouse Handling

```go
func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
    switch e := event.(type) {
    case tui.MouseEvent:
        switch e.Type {
        case tui.MouseClick:
            // Synthesized when press+release at same position
            app.handleClick(e.X, e.Y, e.Button)
        case tui.MousePress, tui.MouseRelease:
            // Raw press/release events
        case tui.MouseMove:
            app.hoverX, app.hoverY = e.X, e.Y
        }
    }
    return nil
}
```

## TUI Constraints

### Coordinate System

- **(0,0) is top-left** of the terminal
- X increases rightward, Y increases downward
- All positions are in character cells, not pixels
- Wide characters (CJK, emoji) occupy 2 cells - the library handles this via `runewidth`

### Thread Safety

- **View and HandleEvent are NEVER called concurrently** - the Runtime guarantees this
- **No locks needed** in application state - mutate freely in HandleEvent or Clickable callbacks
- **Commands run in separate goroutines** - don't access app state directly from commands, return data via events

### Event Ordering

For mouse clicks, events arrive in this order:

1. `MousePress` - button went down
2. `MouseClick` - synthesized click (only if release at same position as press)
3. `MouseRelease` - button came up

### Shift+Enter Detection

Gooey supports Shift+Enter for multi-line text input through two mechanisms:

1. **Kitty Keyboard Protocol**: For terminals that support it (Kitty, some versions of iTerm2 with CSI-u enabled), Shift+Enter is detected natively via modifier bits in CSI sequences.

2. **Backslash+Enter Fallback**: For ALL terminals, typing `\` followed by Enter is automatically converted to Shift+Enter. This provides universal support without terminal configuration.

The Runtime's input reader handles this transparently. Applications receive `KeyEvent{Key: KeyEnter, Shift: true}` regardless of which method was used.

```go
case tui.KeyEvent:
    if e.Key == tui.KeyEnter {
        if e.Shift {
            // Shift+Enter: insert newline in multi-line input
            app.insertNewline()
        } else {
            // Plain Enter: submit the form
            app.submit()
        }
    }
```

### Terminal Cleanup

Always use `defer terminal.Close()` to restore terminal state on exit.

## Testing

Tests use the standard Go testing package with the internal `require` and `assert` packages for assertions:

```go
import (
    "testing"
    "github.com/deepnoodle-ai/gooey/require"
)

func TestSomething(t *testing.T) {
    terminal, err := tui.NewTerminal()
    require.NoError(t, err)
    defer terminal.Close()

    // Test code...
}
```

Run tests with:

```bash
go test ./...           # All tests
go test -v -run TestFoo # Specific test
```

## Key Files

### cli/

| File             | Purpose                                             |
| ---------------- | --------------------------------------------------- |
| `cli.go`         | App struct, command registration, Run() entry point |
| `command.go`     | Command and Group definitions                       |
| `flags.go`       | Flag parsing with struct tags                       |
| `config.go`      | Configuration file loading                          |
| `middleware.go`  | Middleware chain for commands                       |
| `context.go`     | Execution context passed to handlers                |
| `completions.go` | Shell completion generation                         |

### color/

| File          | Purpose                       |
| ------------- | ----------------------------- |
| `color.go`    | ANSI Color type and constants |
| `gradient.go` | Color gradient generation     |
| `hsl.go`      | HSL color space support       |

### env/

| File        | Purpose                                |
| ----------- | -------------------------------------- |
| `config.go` | Parse[T]() and options for env loading |
| `dotenv.go` | .env file parsing                      |
| `json.go`   | JSON config file support               |
| `errors.go` | Aggregate error handling               |

### slog/

| File         | Purpose                               |
| ------------ | ------------------------------------- |
| `handler.go` | Colorized slog.Handler implementation |
| `options.go` | Handler configuration options         |

### terminal/

| File           | Purpose                                      |
| -------------- | -------------------------------------------- |
| `terminal.go`  | Terminal struct, raw mode, screen operations |
| `decoder.go`   | Input sequence decoding                      |
| `key.go`       | Key types and constants                      |
| `mouse.go`     | Mouse event types                            |
| `style.go`     | Style struct for colors and attributes       |
| `frame.go`     | RenderFrame for drawing                      |
| `hyperlink.go` | OSC 8 hyperlink support                      |

### tui/

| File                   | Purpose                             |
| ---------------------- | ----------------------------------- |
| `run.go`               | Simplified Run() API                |
| `runtime.go`           | Event loop, Application interface   |
| `view.go`              | View interface, Empty, Spacer views |
| `layout_views.go`      | VStack, HStack, ZStack              |
| `text_view.go`         | Text view with styling              |
| `button_view.go`       | Clickable interactive view          |
| `input_view.go`        | Input text field                    |
| `modifiers.go`         | Padding, Bordered, Size modifiers   |
| `canvas_view.go`       | Canvas for imperative drawing       |
| `conditional_views.go` | If, IfElse, Switch                  |
| `collection_views.go`  | ForEach, HForEach                   |
| `events.go`            | Event types                         |
| `commands.go`          | Built-in commands                   |

### unidiff/

| File      | Purpose             |
| --------- | ------------------- |
| `diff.go` | Unified diff parser |
