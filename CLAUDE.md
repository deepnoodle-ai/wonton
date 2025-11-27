# CLAUDE.md

This file provides guidance to Claude Code when working with this codebase.

## Project Overview

Gooey is a terminal GUI library for Go that provides flicker-free rendering,
smooth animations (30-60 FPS), and a race-free message-driven architecture. The
library supports double-buffered rendering with dirty region tracking, full
keyboard/mouse input handling, and a declarative view system.

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

### Core Components

- **Terminal** (`terminal.go`): Low-level terminal control with double-buffered rendering. Manages raw mode, cursor, colors, and screen operations via `BeginFrame()`/`EndFrame()`.

- **Runtime** (`runtime.go`): The event loop that coordinates everything. Runs three goroutines:

  1. Main event loop (processes events, calls HandleEvent/View)
  2. Input reader (reads from stdin, forwards KeyEvent/MouseEvent)
  3. Command executor (runs async commands, returns results as events)

- **Application Interface** (`runtime.go`): User code implements this to define app behavior:

  - `View() View` - Return a declarative view tree representing the UI (required)
  - `HandleEvent(event Event) []Cmd` - Process events, return commands (optional, via EventHandler interface)

- **Declarative Views** (`view.go`, `layout_views.go`, `text_view.go`, `modifiers.go`): Composable view components:
  - `VStack()`, `HStack()`, `ZStack()` - Layout containers
  - `Text()` - Styled text display
  - `Clickable()` - Interactive buttons
  - `Input()` - Text input fields
  - `Spacer()`, `Bordered()`, `Canvas()` - Layout helpers

- **Events** (`events.go`, `input.go`, `mouse.go`): Event types including `KeyEvent`, `MouseEvent`, `TickEvent`, `ResizeEvent`, `QuitEvent`, `ErrorEvent`.

- **Commands** (`commands.go`): Functions that perform async work and return events. Built-ins: `Quit()`, `Tick()`, `After()`, `Batch()`, `Sequence()`.

### Rendering Flow

```
View() -> measure views -> render views -> diff & flush
```

The declarative View() method returns a view tree. The runtime measures, renders to a back buffer, then diffs against the front buffer and sends only changes to the terminal.

## Common Patterns

### Simplest Application (Recommended - Declarative)

```go
type MyApp struct {
    count int
}

func (app *MyApp) View() gooey.View {
    return gooey.VStack(
        gooey.Text("Count: %d", app.count).Bold().Fg(gooey.ColorGreen),
        gooey.Spacer().MinHeight(1),
        gooey.HStack(
            gooey.Clickable("[ + ]", func() { app.count++ }).Fg(gooey.ColorGreen),
            gooey.Clickable("[ - ]", func() { app.count-- }).Fg(gooey.ColorRed),
        ).Gap(2),
        gooey.Spacer(),
        gooey.Text("Press 'q' to quit").Dim(),
    ).Align(gooey.AlignCenter).Padding(2)
}

func (app *MyApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    switch e := event.(type) {
    case gooey.KeyEvent:
        if e.Rune == 'q' {
            return []gooey.Cmd{gooey.Quit()}
        }
    }
    return nil
}

func main() {
    if err := gooey.Run(&MyApp{}, gooey.WithMouseTracking(true)); err != nil {
        log.Fatal(err)
    }
}
```

Options can be passed to customize behavior:

```go
gooey.Run(&MyApp{},
    gooey.WithFPS(60),              // Default: 30
    gooey.WithMouseTracking(true),  // Default: false
    gooey.WithAlternateScreen(false), // Default: true
    gooey.WithHideCursor(false),    // Default: true
)
```

### Declarative View Components

```go
// Layout containers
gooey.VStack(children...)     // Vertical stack
gooey.HStack(children...)     // Horizontal stack
gooey.ZStack(children...)     // Layered stack

// Text with styling
gooey.Text("Hello %s", name).Bold().Fg(gooey.ColorCyan)
gooey.Text("Count: %d", count).Italic().Bg(gooey.ColorBlue)

// Interactive elements
gooey.Clickable("Click me", func() { /* handler */ })
gooey.Input(&app.textField).Placeholder("Enter text...").Width(30)
gooey.Input(&app.password).Mask('*')

// Layout helpers
gooey.Spacer()                 // Flexible space
gooey.Spacer().MinHeight(2)    // Minimum height spacer
gooey.Bordered(content)        // Add border around content
gooey.Canvas(drawFunc)         // Custom imperative drawing

// Progress and Loading indicators
gooey.Progress(current, total).Width(30).Fg(gooey.ColorGreen)  // Progress bar
gooey.Loading(app.frame).Label("Loading...")                    // Animated spinner

// Dividers and Bars
gooey.Divider()                            // Horizontal separator line
gooey.Divider().Title("Section")           // Divider with centered title
gooey.HeaderBar("My App")                  // Full-width header with centered text
gooey.StatusBar("Status message")          // Full-width footer bar

// List Views
gooey.SelectList(items, &selected)         // Selectable list
gooey.SelectListStrings(labels, &selected) // String list shorthand
gooey.CheckboxList(items, checked, &cursor) // Checkbox list
gooey.RadioList(items, &selected)          // Radio button list
gooey.Meter("CPU", 75, 100).Width(20)      // Labeled gauge/meter

// Animated Text
gooey.AnimatedTextView("Hello", animation, app.frame)  // Per-character animation
// Use with: CreateRainbowText, CreatePulseText, CreateReverseRainbowText

// Panels and Boxes
gooey.Panel(content).Width(30).Height(10).Border(gooey.BorderSingle)  // Bordered panel
gooey.Panel(nil).Bg(gooey.ColorBlue).Title("Section")                 // Empty panel with title

// Styled Buttons
gooey.StyledButton("Submit", callback).Width(20).Height(3).Bg(gooey.ColorBlue)  // Button with dimensions

// Data Display
gooey.KeyValue("Name", "John").LabelFg(gooey.ColorYellow)  // Key: value pairs
gooey.Toggle(&app.enabled).OnChange(func(v bool) {...})    // On/off toggle switch

// Hyperlinks
gooey.Link("https://example.com", "Click here")            // Clickable hyperlink (OSC 8)
gooey.Link("https://example.com", "").Fg(gooey.ColorCyan)  // URL as text when second arg empty
gooey.LinkRow("Docs", "https://docs.example.com", "docs")  // Label + link pair
gooey.InlineLinks(" | ", link1, link2, link3)              // Multiple links on one line

// Wrapped Text
gooey.WrappedText("Long text...").Center()                 // Auto-wrapping text
gooey.WrappedText("Text").Bg(gooey.ColorBlue).FillBg()     // Fill background

// Grids
gooey.ColorGrid(5, 5, state, colors).CellSize(6, 3)        // Clickable color cycling grid
gooey.CellGrid(5, 5).OnClick(func(c, r int) {...})         // Generic clickable grid
gooey.CharGrid([][]rune{{...}})                            // Display character grid

// Data Tables
gooey.Table(columns, &selected).Rows(rows).Height(20)      // Scrollable data table
gooey.Table(columns, &selected).OnSelect(func(row int) {...})

// File Picker
gooey.FilePicker(items, &filter, &selected).CurrentPath(dir).Height(20)  // File browser
gooey.FilePicker(items, &filter, &selected).OnSelect(func(item ListItem) {...})

// Markdown and Diff Viewers
gooey.Markdown(content, &scrollY).Height(30).MaxWidth(80)  // Rendered markdown
gooey.Markdown(content, &scrollY).Theme(customTheme)
gooey.DiffView(diff, "go", &scrollY).Height(30)            // Syntax-highlighted diff
gooey.DiffView(diff, "go", &scrollY).ShowLineNumbers(true)

// Modifiers (chain on views)
.Fg(color)      // Foreground color
.Bg(color)      // Background color
.Bold()         // Bold text
.Dim()          // Dimmed text
.Style(s)       // Apply a Style
.Padding(n)     // Add padding
.Gap(n)         // Gap between children (stacks)
.Align(align)   // Alignment (stacks)
```

### Conditional Rendering

```go
// Show view conditionally
gooey.If(app.showDetails,
    gooey.Text("Details here"),
)

// If-else pattern
gooey.IfElse(app.isLoading,
    gooey.Text("Loading..."),
    gooey.Text("Content loaded"),
)

// Switch on value
gooey.Switch(app.state,
    gooey.Case("idle", gooey.Text("Idle")),
    gooey.Case("loading", gooey.Text("Loading...")),
    gooey.Default(gooey.Text("Unknown")),
)
```

### Collection Rendering

```go
// Render list of items vertically
gooey.ForEach(app.items, func(item Item, i int) gooey.View {
    return gooey.Text("%d. %s", i+1, item.Name)
})

// Render horizontally
gooey.HForEach(app.tabs, func(tab Tab, i int) gooey.View {
    return gooey.Clickable(tab.Name, func() { app.selectTab(i) })
})
```

### Custom Drawing with Canvas

```go
// Canvas provides escape hatch for imperative drawing
gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
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
style = gooey.NewStyle().WithFgRGB(gooey.NewRGB(255, 128, 0))

// Apply to text
gooey.Text("styled").Style(style)
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

## Important Constraints

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

| File                 | Purpose                                               |
| -------------------- | ----------------------------------------------------- |
| `run.go`             | Simplified Run() API for quick application startup    |
| `runtime.go`         | Event loop, Application interface, command system     |
| `view.go`            | View interface, Empty, Spacer, Fill views             |
| `layout_views.go`    | VStack, HStack, ZStack layout containers              |
| `text_view.go`       | Text view with styling                                |
| `button_view.go`     | Clickable interactive view                            |
| `input_view.go`      | Input text field view                                 |
| `modifiers.go`       | Padding, Bordered, Size modifiers                     |
| `canvas_view.go`     | Canvas for custom imperative drawing                  |
| `conditional_views.go` | If, IfElse, Switch conditional views                |
| `collection_views.go` | ForEach, HForEach collection views                   |
| `progress_view.go`   | Progress bar and Loading spinner views                |
| `divider_view.go`    | Divider, HeaderBar, StatusBar views                   |
| `list_view.go`       | SelectList, CheckboxList, RadioList, Meter views      |
| `animated_text_view.go` | AnimatedTextView, Panel, StyledButton, KeyValue, Toggle |
| `link_text_views.go` | Link, LinkRow, InlineLinks, WrappedText views         |
| `grid_view.go`       | CellGrid, ColorGrid, CharGrid views                   |
| `table_view.go`      | Table view for tabular data display                   |
| `file_picker_view.go` | FilePicker view for file browsing                    |
| `markdown_view.go`   | Markdown view with syntax highlighting                |
| `diff_view.go`       | DiffView for displaying file diffs                    |
| `terminal.go`        | Low-level terminal ops, double buffering, RenderFrame |
| `events.go`          | Event types (Tick, Resize, Quit, Error, Batch)        |
| `input.go`           | KeyEvent, Key constants                               |
| `mouse.go`           | MouseEvent, mouse tracking, MouseRegion               |
| `commands.go`        | Built-in commands (Quit, Tick, After, Batch)          |
| `style.go`           | Style, Color constants, RGB support                   |
| `text_input.go`      | TextInput widget (imperative)                         |
| `key_decoder.go`     | Keyboard/paste input decoding                         |
