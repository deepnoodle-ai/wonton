# InlineApp Design Document

## Overview

`InlineApp` provides a way to build terminal applications that coexist with normal terminal output rather than taking over the entire screen. This enables a pattern where:

- **Scrollback history** (above): Rich `View` content that becomes part of terminal history
- **Live region** (below): Dynamic `View` content that updates in place

Both regions render full `tui.View` hierarchies with styling, layout, borders, and all other view capabilities. The difference is persistence: scrollback is permanent, live region is ephemeral.

This pattern is ideal for:
- Chat interfaces and REPLs
- CLI tools with progress indicators
- Interactive prompts with live feedback
- Agent-style applications (like Claude Code) that mix output with status

## Design Goals

1. **API consistency with tui.App** - Reuse existing interfaces (`EventHandler`, `Initializable`, `Destroyable`), option names, and event types
2. **Correct raw mode handling** - Properly manage terminal state and line endings
3. **Flexible live region** - Support variable height, async updates, and focus
4. **Composable** - Build on existing tui primitives (View, LivePrinter, Print)
5. **Clean lifecycle** - Clear Run/Stop semantics matching tui.App

## Terminal Layout

```
┌─────────────────────────────────────────────┐
│ $ previous command                          │  ← Normal terminal
│ output from previous command                │    scrollback
├─────────────────────────────────────────────┤
│ ╭─────────────────────────────────────────╮ │  ← Scrollback history:
│ │ Welcome to the application!             │ │    Rich Views printed
│ ╰─────────────────────────────────────────╯ │    via Print(view)
│                                             │
│ [12:34:56] User: Hello                      │    Each Print() call
│ [12:34:57] Bot: Hi there!                   │    renders a View with
│                                             │    full styling, borders,
│ ╭── Error ────────────────────────────────╮ │    layout, etc.
│ │ Connection failed: timeout              │ │
│ ╰─────────────────────────────────────────╯ │
│                                             │
│ [12:34:58] User: retry                      │
│ [12:34:59] Bot: Reconnecting...             │
├─────────────────────────────────────────────┤
│ ──────────────────────────────────          │  ← Live region:
│ Status: Processing request...               │    Single View that
│ [████████████░░░░░░░░] 60%                  │    updates in place
│ ──────────────────────────────────          │    via LiveView()
│ > Type a message...█                        │
│ ──────────────────────────────────          │
└─────────────────────────────────────────────┘
```

## Core Concepts

### Scrollback vs Live Region

| Aspect        | Scrollback                   | Live Region                              |
| ------------- | ---------------------------- | ---------------------------------------- |
| Content       | Full `View` hierarchy        | Full `View` hierarchy                    |
| Persistence   | Permanent (terminal history) | Ephemeral (replaced on update)           |
| Output method | `Print(view)` from handler   | `LiveView()` method                      |
| Height        | Determined by View content   | Variable, tracked for cursor positioning |
| Focus/Input   | N/A (already rendered)       | Full focus and input support             |

### Scrollback Composition

Each `Print()` call renders a complete `View` to the scrollback. Views can be simple or complex:

```go
// Simple styled text
app.Print(tui.Text("Operation complete").Fg(tui.ColorGreen))

// Grouped inline content
app.Print(tui.Group(
    tui.Text("[%s] ", timestamp).Dim(),
    tui.Text("User: ").Bold().Fg(tui.ColorCyan),
    tui.Text("%s", message),
))

// Complex bordered view with layout
app.Print(tui.Bordered(
    tui.Stack(
        tui.Text("Error").Bold().Fg(tui.ColorRed),
        tui.Text(""),
        tui.Text("Connection refused: %s", host),
        tui.Text("Retrying in %d seconds...", delay).Dim(),
    ).Padding(1),
).Border(&tui.RoundedBorder).BorderFg(tui.ColorRed))

// Multi-column layout
app.Print(tui.Group(
    tui.Text("%-20s", "Status:"),
    tui.Text("Online").Fg(tui.ColorGreen),
    tui.Spacer(),
    tui.Text("Uptime: %s", uptime).Dim(),
))
```

Each printed view becomes permanent terminal history. The full range of `tui.View` types works: `Text`, `Stack`, `Group`, `Bordered`, `Padding`, tables, and custom views.

### Live Region Composition

The live region is rendered as a single `View` via the `LiveView()` method, but can contain multiple independently-updating sections:

```go
func (app *MyApp) LiveView() tui.View {
    return tui.Stack(
        app.buildHeader(),           // Static within live region
        app.buildAsyncStatus1(),     // Updates from async Cmd
        app.buildAsyncStatus2(),     // Updates from another async Cmd
        app.buildInputArea(),        // Focusable input field
        app.buildFooter(),           // Static within live region
    )
}
```

When any state changes (via `HandleEvent` or async `Cmd` results), the entire live region is re-rendered. This is efficient because:
- The live region is typically small (5-20 lines)
- ANSI cursor movement is fast
- The `LivePrinter` clears and redraws efficiently

### Variable Height

The live region height can change between renders:

```
Render 1 (3 lines):        Render 2 (5 lines):        Render 3 (2 lines):
┌──────────────────┐       ┌──────────────────┐       ┌──────────────────┐
│ Status: Ready    │       │ Status: Loading  │       │ Status: Done     │
│ > input█         │       │ [████░░░░] 40%   │       │ > input█         │
│ ─────────────────│       │ File: data.json  │       └──────────────────┘
└──────────────────┘       │ > input█         │       (extra lines cleared)
                           │ ─────────────────│
                           └──────────────────┘
```

The implementation tracks the previous height and:
- Moves cursor up to the start of the live region
- Renders new content
- Clears any extra lines if new height < old height

## Threading Model

InlineApp uses the **same threading model as tui.App**: a single-threaded event loop with async operations via the `Cmd` pattern.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        InlineApp.Run()                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Goroutine 1: Event Loop (main)                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  for {                                                  │    │
│  │      event := <-events                                  │    │
│  │      cmds := app.HandleEvent(event)  // single-threaded │    │
│  │      queue(cmds)                                        │    │
│  │      view := app.LiveView()          // single-threaded │    │
│  │      render(view)                                       │    │
│  │  }                                                      │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              ↑                                  │
│              ┌───────────────┼───────────────┐                  │
│              │               │               │                  │
│  Goroutine 2: Input      Goroutine 3: Cmd Executor              │
│  ┌─────────────────┐     ┌─────────────────────────┐            │
│  │  for {          │     │  for cmd := range cmds { │            │
│  │    event := read│     │    go func() {          │            │
│  │    events <- evt│     │      result := cmd()    │            │
│  │  }              │     │      events <- result   │            │
│  └─────────────────┘     │    }()                  │            │
│                          │  }                      │            │
│                          └─────────────────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

### Thread Safety Guarantees

1. **HandleEvent runs on a single goroutine** - No locks needed when mutating app state
2. **LiveView runs on the same goroutine** - Safe to read state without locks
3. **Async operations use Cmd** - Results come back as Events on the main goroutine
4. **Print() is called from HandleEvent** - Safe, no coordination needed

### Async Updates via Cmd

To update state from async operations (HTTP requests, timers, background tasks), use the `Cmd` pattern:

```go
// Define a custom event for your async result
type DownloadProgressEvent struct {
    ID       string
    Progress int
    Done     bool
}

func (e DownloadProgressEvent) Timestamp() time.Time { return time.Now() }

// Return a Cmd from HandleEvent to start async work
func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
    switch e := event.(type) {
    case tui.KeyEvent:
        if e.Rune == 'd' {
            // Start a download - returns immediately, result comes as event
            return []tui.Cmd{app.startDownload("file.zip")}
        }

    case DownloadProgressEvent:
        // Async result arrived - update state (single-threaded, no locks)
        app.downloads[e.ID] = e.Progress
        if e.Done {
            // Print completion to scrollback
            app.Print(tui.Text("Download complete: %s", e.ID).Fg(tui.ColorGreen))
        }
    }
    return nil
}

// Cmd that performs async work and returns an event
func (app *MyApp) startDownload(filename string) tui.Cmd {
    return func() tui.Event {
        // This runs in a separate goroutine
        for progress := 0; progress <= 100; progress += 10 {
            time.Sleep(100 * time.Millisecond)
            // Send progress update (will trigger re-render)
            app.SendEvent(DownloadProgressEvent{
                ID:       filename,
                Progress: progress,
            })
        }
        return DownloadProgressEvent{ID: filename, Progress: 100, Done: true}
    }
}
```

### Multiple Async Processes

Multiple `Cmd` instances can run concurrently. Each sends events back to the main loop:

```go
func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
    switch e := event.(type) {
    case StartAllEvent:
        // Start multiple async operations simultaneously
        return []tui.Cmd{
            app.runDownloader(),   // Updates app.downloadProgress
            app.runCompiler(),     // Updates app.buildStatus
            app.runTestRunner(),   // Updates app.testResults
        }

    case DownloadUpdate:
        app.downloadProgress = e.Progress  // Safe: single-threaded

    case CompileUpdate:
        app.buildStatus = e.Status         // Safe: single-threaded

    case TestUpdate:
        app.testResults = e.Results        // Safe: single-threaded
    }
    return nil
}

func (app *MyApp) LiveView() tui.View {
    // All state reads are safe - same goroutine as HandleEvent
    return tui.Stack(
        app.renderDownloadProgress(),  // Reads app.downloadProgress
        app.renderBuildStatus(),       // Reads app.buildStatus
        app.renderTestResults(),       // Reads app.testResults
    )
}
```

## Event Types

InlineApp uses the same event types as `tui.App`:

| Event         | Description             | Fields                                             |
| ------------- | ----------------------- | -------------------------------------------------- |
| `KeyEvent`    | Keyboard input          | `Rune rune, Key Key, Modifiers KeyModifier`        |
| `MouseEvent`  | Mouse input             | `X, Y int, Button MouseButton, Action MouseAction` |
| `TickEvent`   | Frame tick (if FPS > 0) | `Frame uint64`                                     |
| `ResizeEvent` | Terminal window resized | `Width, Height int`                                |
| `ErrorEvent`  | Error occurred          | `Err error`                                        |
| `QuitEvent`   | Quit requested          | none                                               |

**ResizeEvent Note**: When the terminal is resized, InlineApp automatically re-renders the live region. Handle `ResizeEvent` if you need to adjust your layout or store dimensions.

## Input and Focus

The live region supports full keyboard and mouse input, including focus management for interactive elements.

### Focus Behaviors

Focus works the same as in tui.App:

```go
func (app *MyApp) LiveView() tui.View {
    return tui.Stack(
        tui.Divider(),
        // Focusable input field - Tab/Shift+Tab navigation works
        tui.InputField(&app.inputValue).
            Placeholder("Type here...").
            OnSubmit(func(value string) {
                app.Print(tui.Text("Submitted: %s", value))
            }),
        tui.Divider(),
        // Focusable buttons
        tui.Group(
            tui.Button("Submit", func() { app.submit() }),
            tui.Text(" "),
            tui.Button("Cancel", func() { app.cancel() }),
        ),
    )
}
```

### Mouse Support

Mouse events work within the live region:

```go
// Enable mouse tracking in options
app := tui.NewInlineApp(tui.WithMouseTracking(true))

// Clickable elements in LiveView work automatically
func (app *MyApp) LiveView() tui.View {
    return tui.Stack(
        tui.Clickable(
            tui.Text("[Click me]").Fg(tui.ColorCyan),
            func() { app.handleClick() },
        ),
    )
}
```

### Key Events

Key events are delivered to `HandleEvent` and also routed to focused elements:

```go
func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
    switch e := event.(type) {
    case tui.KeyEvent:
        // Global key handling
        if e.Key == tui.KeyCtrlC {
            return []tui.Cmd{tui.Quit()}
        }
        if e.Key == tui.KeyEscape {
            app.clearInput()
        }
        // Other keys may be consumed by focused elements (InputField, etc.)
    }
    return nil
}
```

## API Design

### Interfaces

InlineApp reuses the existing `tui` interfaces with one new interface for the live region:

```go
// InlineApplication is the interface for inline applications.
// It mirrors tui.Application but uses LiveView() for the live region.
type InlineApplication interface {
    // LiveView returns the view for the live region.
    // Called after each event is processed to re-render the live region.
    LiveView() View
}
```

The following existing interfaces are supported (same as `tui.App`):

| Interface       | Method                     | Description                           |
| --------------- | -------------------------- | ------------------------------------- |
| `EventHandler`  | `HandleEvent(Event) []Cmd` | Process events, return async commands |
| `Initializable` | `Init() error`             | One-time initialization before Run    |
| `Destroyable`   | `Destroy()`                | Cleanup after Run completes           |

**Note**: Call `app.Print(view)` from `HandleEvent` to add content to scrollback.

### InlineApp Type

```go
// InlineApp manages the runtime for inline applications.
type InlineApp struct {
    // contains filtered or unexported fields
}
```

### Constructor and Options

```go
// NewInlineApp creates a new inline application runner.
func NewInlineApp(opts ...InlineOption) *InlineApp

// InlineOption configures an InlineApp.
type InlineOption func(*inlineConfig)
```

**Supported Options** (same names as `tui.Run` where applicable):

| Option                         | Description                                    | Default        |
| ------------------------------ | ---------------------------------------------- | -------------- |
| `WithWidth(w int)`             | Rendering width                                | terminal or 80 |
| `WithOutput(w io.Writer)`      | Output writer                                  | `os.Stdout`    |
| `WithInput(r io.Reader)`       | Input reader (useful for testing)              | `os.Stdin`     |
| `WithFPS(fps int)`             | Frame rate for TickEvents                      | 0 (no ticks)   |
| `WithMouseTracking(enabled)`   | Enable mouse event tracking in live region     | false          |
| `WithBracketedPaste(enabled)`  | Enable bracketed paste mode                    | false          |
| `WithPasteTabWidth(width int)` | Convert tabs to spaces in paste (0 = preserve) | 0              |
| `WithKittyKeyboard(enabled)`   | Enable Kitty keyboard protocol (Shift+Enter)   | false          |

**Options NOT applicable to InlineApp** (these are for alternate screen mode):

| Option                | Why Not Applicable                         |
| --------------------- | ------------------------------------------ |
| `WithAlternateScreen` | InlineApp always uses normal terminal mode |
| `WithHideCursor`      | Cursor visibility managed by live region   |

### Run Method

```go
// Run starts the inline application and blocks until it exits.
// This is the main entry point, similar to tui.Run().
//
// The app parameter must implement InlineApplication (LiveView method).
// Optionally implement InlineEventHandler, InlineInitializable, InlineDestroyable.
//
// Lifecycle:
//  1. Call Init() if implemented
//  2. Enter raw mode, enable configured terminal features
//  3. Start event loop, input reader, and command executor goroutines
//  4. Render initial LiveView()
//  5. Process events until Quit command
//  6. Restore terminal state
//  7. Call Destroy() if implemented
//
// Example:
//
//     type MyApp struct {
//         messages []string
//         input    string
//     }
//
//     func (app *MyApp) LiveView() tui.View {
//         return tui.Stack(
//             tui.Divider(),
//             tui.Text("> %s█", app.input),
//         )
//     }
//
//     func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
//         // Handle events, call app.Print() for scrollback
//         return nil
//     }
//
//     func main() {
//         runner := tui.NewInlineApp()
//         if err := runner.Run(&MyApp{}); err != nil {
//             log.Fatal(err)
//         }
//     }
func (r *InlineApp) Run(app any) error
```

### Print Method

```go
// Print renders a View to the scrollback history.
// The view can be any tui.View: Text, Stack, Group, Bordered, etc.
//
// This method is available on the InlineApp and should be called from
// HandleEvent. It temporarily clears the live region, prints the view,
// and restores the live region below.
//
// Thread-safe: Can be called from HandleEvent (recommended) or from
// a Cmd goroutine via the app reference.
func (r *InlineApp) Print(view View)

// Printf is a convenience method that prints formatted text.
// Equivalent to: r.Print(tui.Text(format, args...))
func (r *InlineApp) Printf(format string, args ...any)
```

### Convenience Function

```go
// RunInline is a convenience function that creates and runs an InlineApp.
// Equivalent to: NewInlineApp(opts...).Run(app)
func RunInline(app any, opts ...InlineOption) error
```

### InlineApp Methods

| Method                    | Thread Safety                | Description                         |
| ------------------------- | ---------------------------- | ----------------------------------- |
| `Run(app any) error`      | Main goroutine               | Start event loop, blocks until quit |
| `Print(view View)`        | Safe from HandleEvent or Cmd | Add view to scrollback              |
| `Printf(format, args...)` | Safe from HandleEvent or Cmd | Add formatted text to scrollback    |
| `Stop()`                  | Any goroutine                | Graceful shutdown via QuitEvent     |
| `SendEvent(event Event)`  | Any goroutine                | Inject custom event into event loop |
| `ClearScrollback()`       | Safe from HandleEvent        | Clear terminal scrollback history   |

## Usage Examples

### Basic Usage

```go
type BasicApp struct {
    runner *tui.InlineApp
    count  int
}

func (app *BasicApp) LiveView() tui.View {
    return tui.Stack(
        tui.Divider(),
        tui.Text(" Count: %d", app.count).Bold(),
        tui.Text(" Press +/- to change, q to quit").Dim(),
        tui.Divider(),
    )
}

func (app *BasicApp) HandleEvent(event tui.Event) []tui.Cmd {
    if key, ok := event.(tui.KeyEvent); ok {
        switch key.Rune {
        case '+':
            app.count++
            app.runner.Print(tui.Text("Incremented to %d", app.count))
        case '-':
            app.count--
            app.runner.Print(tui.Text("Decremented to %d", app.count))
        case 'q':
            return []tui.Cmd{tui.Quit()}
        }
    }
    return nil
}

func main() {
    app := &BasicApp{}
    app.runner = tui.NewInlineApp(tui.WithWidth(60))

    if err := app.runner.Run(app); err != nil {
        log.Fatal(err)
    }
}
```

### Chat Interface with Async Responses

```go
type ChatApp struct {
    runner   *tui.InlineApp
    input    string
    status   string
    thinking bool
}

// Custom event for async response
type ResponseEvent struct {
    Text string
    Time time.Time
}

func (e ResponseEvent) Timestamp() time.Time { return e.Time }

func (app *ChatApp) LiveView() tui.View {
    var statusView tui.View
    if app.thinking {
        statusView = tui.Text(" Thinking...").Fg(tui.ColorYellow)
    } else {
        statusView = tui.Text(" Ready").Fg(tui.ColorGreen)
    }

    return tui.Stack(
        tui.Divider(),
        tui.Group(
            tui.Text("> ").Fg(tui.ColorCyan),
            tui.Text("%s█", app.input),
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
            if app.input != "" && !app.thinking {
                msg := app.input
                app.input = ""
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
            if e.Rune != 0 {
                app.input += string(e.Rune)
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
    app := &ChatApp{status: "Ready"}
    app.runner = tui.NewInlineApp(
        tui.WithWidth(80),
        tui.WithBracketedPaste(true),
    )

    // Print welcome message before starting
    fmt.Println("Chat Demo - Type messages and press Enter")
    fmt.Println()

    if err := app.runner.Run(app); err != nil {
        log.Fatal(err)
    }
}
```

### Dashboard with Multiple Async Updates

```go
type DashboardApp struct {
    runner    *tui.InlineApp
    downloads map[string]int  // filename -> progress
    builds    map[string]string // target -> status
}

type DownloadProgress struct {
    File     string
    Progress int
    Done     bool
    Time     time.Time
}

func (e DownloadProgress) Timestamp() time.Time { return e.Time }

type BuildStatus struct {
    Target string
    Status string
    Time   time.Time
}

func (e BuildStatus) Timestamp() time.Time { return e.Time }

func (app *DashboardApp) Init() error {
    app.downloads = make(map[string]int)
    app.builds = make(map[string]string)
    return nil
}

func (app *DashboardApp) LiveView() tui.View {
    var downloadViews []tui.View
    for file, progress := range app.downloads {
        downloadViews = append(downloadViews,
            tui.Group(
                tui.Text("  %s: ", file),
                tui.Text("[%s] %d%%", progressBar(progress, 20), progress),
            ),
        )
    }

    var buildViews []tui.View
    for target, status := range app.builds {
        color := tui.ColorYellow
        if status == "done" {
            color = tui.ColorGreen
        }
        buildViews = append(buildViews,
            tui.Group(
                tui.Text("  %s: ", target),
                tui.Text("%s", status).Fg(color),
            ),
        )
    }

    return tui.Stack(
        tui.Divider().Fg(tui.ColorBrightBlack),
        tui.Text(" Dashboard").Bold(),
        tui.Divider().Fg(tui.ColorBrightBlack),
        tui.Text(""),
        tui.Text(" Downloads:").Fg(tui.ColorCyan),
        tui.Stack(downloadViews...),
        tui.Text(""),
        tui.Text(" Builds:").Fg(tui.ColorMagenta),
        tui.Stack(buildViews...),
        tui.Text(""),
        tui.Divider().Fg(tui.ColorBrightBlack),
        tui.Text(" Press 's' to start, 'q' to quit").Dim(),
    )
}

func (app *DashboardApp) HandleEvent(event tui.Event) []tui.Cmd {
    switch e := event.(type) {
    case tui.KeyEvent:
        if e.Rune == 's' {
            // Start multiple async operations
            return []tui.Cmd{
                app.startDownload("data.zip"),
                app.startDownload("assets.tar"),
                app.startBuild("linux"),
                app.startBuild("darwin"),
            }
        }
        if e.Rune == 'q' {
            return []tui.Cmd{tui.Quit()}
        }

    case DownloadProgress:
        app.downloads[e.File] = e.Progress
        if e.Done {
            delete(app.downloads, e.File)
            app.runner.Print(tui.Text("Download complete: %s", e.File).Fg(tui.ColorGreen))
        }

    case BuildStatus:
        app.builds[e.Target] = e.Status
        if e.Status == "done" {
            delete(app.builds, e.Target)
            app.runner.Print(tui.Text("Build complete: %s", e.Target).Fg(tui.ColorGreen))
        }
    }
    return nil
}

func (app *DashboardApp) startDownload(file string) tui.Cmd {
    return func() tui.Event {
        for p := 0; p <= 100; p += 10 {
            app.runner.SendEvent(DownloadProgress{File: file, Progress: p, Time: time.Now()})
            time.Sleep(100 * time.Millisecond)
        }
        return DownloadProgress{File: file, Progress: 100, Done: true, Time: time.Now()}
    }
}

func (app *DashboardApp) startBuild(target string) tui.Cmd {
    return func() tui.Event {
        stages := []string{"compiling", "linking", "optimizing", "done"}
        for _, stage := range stages {
            app.runner.SendEvent(BuildStatus{Target: target, Status: stage, Time: time.Now()})
            time.Sleep(200 * time.Millisecond)
        }
        return BuildStatus{Target: target, Status: "done", Time: time.Now()}
    }
}
```

## Implementation Notes

### Raw Mode Line Endings

In raw terminal mode, `\n` only moves down without returning to column 0. The implementation must:

1. Use `\r\n` instead of `\n` when printing to scrollback (via `WithRawMode()` option on Print)
2. Use `\r` at the start of each line in live region updates (already done in `renderToANSILive`)

### Print During Active Live Region

When `Print()` is called while a live region is active:

1. Clear the live region content (move cursor up, clear lines)
2. Print the scrollback content with proper line endings
3. Re-render the live region below

This ensures scrollback content appears above the live region.

### Terminal State Management

`Run()` must:
1. Check stdin is a terminal (`term.IsTerminal`)
2. Call `Init()` if implemented
3. Save current terminal state and enter raw mode
4. Enable configured features (bracketed paste, Kitty keyboard, mouse)
5. Start three goroutines: event loop, input reader, command executor
6. Process events until `QuitEvent`
7. Stop goroutines
8. Disable features and restore terminal state
9. Call `Destroy()` if implemented

### Focus Management

The focus system (used by InputField, Button, etc.) works within the live region:
- Focus state is cleared before each render
- Elements register during render
- Tab/Shift+Tab navigation works
- Click-to-focus works (if mouse enabled)

### Thread Safety Summary

| Operation       | Thread Safety                                          |
| --------------- | ------------------------------------------------------ |
| `HandleEvent()` | Single-threaded, safe to mutate state                  |
| `LiveView()`    | Single-threaded, safe to read state                    |
| `Print()`       | Safe from HandleEvent or Cmd (internally synchronized) |
| `SendEvent()`   | Thread-safe, can call from any goroutine               |
| `Stop()`        | Thread-safe, can call from any goroutine               |

## Relationship to Existing Types

### Comparison: tui.Run() vs RunInline()

| Aspect           | `tui.Run()`                                    | `RunInline()`                              |
| ---------------- | ---------------------------------------------- | ------------------------------------------ |
| Screen mode      | Alternate screen (full takeover)               | Normal terminal (coexists with scrollback) |
| Main interface   | `Application.View()`                           | `InlineApplication.LiveView()`             |
| Other interfaces | `EventHandler`, `Initializable`, `Destroyable` | Same (reused)                              |
| Event types      | `KeyEvent`, `MouseEvent`, `TickEvent`, etc.    | Same (reused)                              |
| Cmd pattern      | `type Cmd func() Event`                        | Same (reused)                              |
| Scrollback       | N/A                                            | `Print(view)` method                       |
| Rendering        | Full screen each frame                         | Live region only                           |
| Use case         | Full TUI apps, dashboards                      | CLI tools, chat, REPL                      |

### Shared Components

InlineApp reuses these existing types and functions:

| Component           | Usage                                                      |
| ------------------- | ---------------------------------------------------------- |
| `Cmd`               | Async operations, same pattern as tui.App                  |
| `Quit()`            | Returns quit command to exit                               |
| `Batch()`           | Execute multiple commands                                  |
| `Tick()`, `After()` | Timer commands                                             |
| All `Event` types   | `KeyEvent`, `MouseEvent`, `TickEvent`, `ResizeEvent`, etc. |
| All `View` types    | All views work in both scrollback and live region          |

### Print() and PrintOption

The standalone `Print()` function gains a new option for raw mode:

```go
// WithRawMode configures Print to use \r\n line endings for raw mode terminals.
func WithRawMode() PrintOption
```

`InlineApp.Print()` uses this internally.

### LivePrinter

`InlineApp` uses `LivePrinter` internally for the live region. Users don't interact with it directly.

## Testing

InlineApp supports testability through the `WithInput()` and `WithOutput()` options:

```go
func TestMyInlineApp(t *testing.T) {
    // Create simulated input
    input := strings.NewReader("hello\n")

    // Capture output
    var output bytes.Buffer

    app := &MyApp{}
    runner := tui.NewInlineApp(
        tui.WithInput(input),
        tui.WithOutput(&output),
        tui.WithWidth(80),
    )

    // Run with timeout
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()

    go func() {
        <-ctx.Done()
        runner.Stop()
    }()

    runner.Run(app)

    // Assert on captured output
    assert.Contains(t, output.String(), "expected content")
}
```

For testing the live region view without a full run loop:

```go
func TestLiveView(t *testing.T) {
    app := &MyApp{status: "Ready"}

    // Render live view directly
    view := app.LiveView()
    screen := tui.SprintScreen(view, tui.WithWidth(80), tui.WithHeight(10))

    // Use termtest for snapshot testing
    termtest.AssertScreen(t, screen)
}
```

## Future Considerations

### Scroll Region

For very long live regions, could support a scrollable area within the live region using terminal scroll regions (`\033[r`).

### Nested InlineApps

Could support running one InlineApp within another for complex hierarchical UIs.

## Migration Path

Existing code using manual `LivePrinter` + raw mode can migrate to `InlineApp`:

**Before:**
```go
oldState, _ := term.MakeRaw(fd)
defer term.Restore(fd, oldState)

live := tui.NewLivePrinter(tui.WithWidth(80))
defer live.Stop()

decoder := terminal.NewKeyDecoder(os.Stdin)
for {
    event, _ := decoder.ReadKeyEvent()
    // Handle event...
    live.Update(buildView())
}
```

**After (using convenience function):**
```go
type MyApp struct {
    // state...
}

func (app *MyApp) LiveView() tui.View {
    return buildView()
}

func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
    // Handle event...
    return nil
}

func main() {
    tui.RunInline(&MyApp{})
}
```

**After (with explicit runner for Print access):**
```go
type MyApp struct {
    runner *tui.InlineApp
    // state...
}

func (app *MyApp) LiveView() tui.View {
    return buildView()
}

func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
    // Handle event, with access to Print:
    app.runner.Print(tui.Text("Message logged").Dim())
    return nil
}

func main() {
    app := &MyApp{}
    app.runner = tui.NewInlineApp()
    app.runner.Run(app)
}
```
