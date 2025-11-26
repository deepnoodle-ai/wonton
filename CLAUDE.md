# CLAUDE.md

This file provides guidance to Claude Code when working with this codebase.

## Project Overview

Gooey is a terminal GUI library for Go that provides flicker-free rendering,
smooth animations (30-60 FPS), and a race-free message-driven architecture. The
library supports double-buffered rendering with dirty region tracking, full
keyboard/mouse input handling, and a widget composition system.

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
go run examples/runtime_counter/main.go    # Basic keyboard input
go run examples/runtime_http/main.go       # Async HTTP requests
go run examples/runtime_animation/main.go  # Smooth animations
go run examples/all/main.go                # Comprehensive demo
```

## Architecture

### Core Components

- **Terminal** (`terminal.go`): Low-level terminal control with double-buffered rendering. Manages raw mode, cursor, colors, and screen operations via `BeginFrame()`/`EndFrame()`.

- **Runtime** (`runtime.go`): The event loop that coordinates everything. Runs three goroutines:

  1. Main event loop (processes events, calls HandleEvent/Render)
  2. Input reader (reads from stdin, forwards KeyEvent/MouseEvent)
  3. Command executor (runs async commands, returns results as events)

- **Application Interface** (`runtime.go`): User code implements this to define app behavior:

  - `HandleEvent(event Event) []Cmd` - Process events, return commands for async work
  - `Render(frame RenderFrame)` - Draw current state to the frame

- **Events** (`events.go`, `input.go`, `mouse.go`): Event types including `KeyEvent`, `MouseEvent`, `TickEvent`, `ResizeEvent`, `QuitEvent`, `ErrorEvent`.

- **Commands** (`commands.go`): Functions that perform async work and return events. Built-ins: `Quit()`, `Tick()`, `After()`, `Batch()`, `Sequence()`.

- **Composition System** (`composition.go`, `container.go`): Widget system with `ComposableWidget` interface, `BaseWidget` base class, and layout managers (`VerticalLayout`, `HorizontalLayout`, `FlexLayout`, `GridLayout`).

### Rendering Flow

```
BeginFrame() -> get RenderFrame -> app.Render(frame) -> EndFrame() -> diff & flush
```

The frame is a back buffer. `EndFrame()` diffs against the front buffer and only sends changes to the terminal.

## Common Patterns

### Basic Application Structure

```go
type MyApp struct {
    // State fields - no mutex needed!
    count int
}

func (app *MyApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    switch e := event.(type) {
    case gooey.KeyEvent:
        if e.Rune == 'q' {
            return []gooey.Cmd{gooey.Quit()}
        }
    case gooey.TickEvent:
        // Called every frame (30-60 FPS based on runtime config)
    case gooey.ResizeEvent:
        // Terminal resized to e.Width x e.Height
    }
    return nil
}

func (app *MyApp) Render(frame gooey.RenderFrame) {
    width, height := frame.Size()
    frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())
    frame.PrintStyled(0, 0, "Hello", gooey.NewStyle().WithForeground(gooey.ColorGreen))
}

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    runtime := gooey.NewRuntime(terminal, &MyApp{}, 30) // 30 FPS
    runtime.Run()
}
```

### Async Operations (HTTP, etc.)

```go
// Custom event for results
type DataReceived struct {
    Data string
}
func (d DataReceived) Timestamp() time.Time { return time.Now() }

// Command that performs async work
func FetchData(url string) gooey.Cmd {
    return func() gooey.Event {
        resp, err := http.Get(url)
        if err != nil {
            return gooey.ErrorEvent{Time: time.Now(), Err: err}
        }
        // ... process response
        return DataReceived{Data: result}
    }
}

// In HandleEvent:
case gooey.KeyEvent:
    if e.Rune == 'f' {
        return []gooey.Cmd{FetchData("https://api.example.com")}
    }
case DataReceived:
    app.data = e.Data
```

### Styles

```go
style := gooey.NewStyle().
    WithForeground(gooey.ColorGreen).
    WithBackground(gooey.ColorBlack).
    WithBold().
    WithUnderline()

// RGB colors
style = gooey.NewStyle().WithFgRGB(gooey.RGB{R: 255, G: 128, B: 0})

frame.PrintStyled(x, y, "text", style)
```

### Mouse Handling

```go
func (app *MyApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    switch e := event.(type) {
    case gooey.MouseEvent:
        switch e.Type {
        case gooey.MouseClick:
            // Synthesized when press+release at same position
            app.handleClick(e.X, e.Y, e.Button)
        case gooey.MousePress, gooey.MouseRelease:
            // Raw press/release events
        case gooey.MouseMove:
            app.hoverX, app.hoverY = e.X, e.Y
        }
    }
    return nil
}
```

### TextInput Widget

```go
input := gooey.NewTextInput().
    WithPlaceholder("Enter text...").
    WithMaxLength(100).
    WithMask('*')  // For passwords

input.OnSubmit = func(value string) {
    // Handle enter key
}

// In HandleEvent, forward key events:
if input.HandleKey(keyEvent) {
    return nil // Input consumed the event
}
```

## Important Constraints

### Coordinate System

- **(0,0) is top-left** of the terminal
- X increases rightward, Y increases downward
- All positions are in character cells, not pixels
- Wide characters (CJK, emoji) occupy 2 cells - the library handles this via `runewidth`

### Thread Safety

- **HandleEvent and Render are NEVER called concurrently** - the Runtime guarantees this
- **No locks needed** in application state - mutate freely in HandleEvent
- **Commands run in separate goroutines** - don't access app state directly from commands, return data via events

### Frame Lifecycle

- **BeginFrame/EndFrame must be paired** - frame is invalid after EndFrame
- **Only one frame active at a time** - BeginFrame blocks if another frame is active
- **Don't hold frames across async operations** - complete rendering in a single call

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
case gooey.KeyEvent:
    if e.Key == gooey.KeyEnter {
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

Tests use the standard Go testing package with testify/require for assertions:

```go
func TestSomething(t *testing.T) {
    require := require.New(t)

    terminal, err := gooey.NewTerminal()
    require.NoError(err)
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

| File             | Purpose                                               |
| ---------------- | ----------------------------------------------------- |
| `runtime.go`     | Event loop, Application interface, command system     |
| `terminal.go`    | Low-level terminal ops, double buffering, RenderFrame |
| `events.go`      | Event types (Tick, Resize, Quit, Error, Batch)        |
| `input.go`       | KeyEvent, Key constants                               |
| `mouse.go`       | MouseEvent, mouse tracking, MouseRegion               |
| `commands.go`    | Built-in commands (Quit, Tick, After, Batch)          |
| `composition.go` | Widget interfaces, BaseWidget, layout params          |
| `container.go`   | Container widget implementation                       |
| `style.go`       | Style, Color constants, RGB support                   |
| `text_input.go`  | TextInput widget                                      |
| `key_decoder.go` | Keyboard/paste input decoding                         |
