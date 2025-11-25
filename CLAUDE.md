# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gooey is a sophisticated Terminal GUI library for Go that provides flicker-free rendering, advanced animations (30+ FPS), and interactive components. It abstracts low-level terminal control to enable building complex TUIs with animated text, layouts, and user input handling.

## Development Commands

### Build and Test
```bash
go build ./...
go test ./...
go fmt ./...
go vet ./...
```

**⚠️ IMPORTANT: Building Binaries**

When building example programs or testing builds, **DO NOT create binaries in this repository directory**. They may be accidentally committed to version control.

**Recommended approach:**
- Use `go run` instead of `go build` whenever possible (see examples below)
- If you need to create a binary, build it in `/tmp`:
  ```bash
  go build -o /tmp/my_example examples/my_example/main.go
  /tmp/my_example
  ```

### Running Examples

**Runtime Examples (Recommended - Start Here):**
```bash
go run examples/runtime_counter/main.go      # Simple counter with keyboard input
go run examples/runtime_http/main.go         # Async HTTP requests without blocking UI
go run examples/runtime_animation/main.go    # Smooth 60 FPS animation
go run examples/runtime_composition/main.go  # Composition system with Runtime
```

**Other Examples (All Updated to Use Runtime):**
```bash
go run examples/all/main.go              # Comprehensive demo showcasing all features
go run examples/simple_animation_demo.go # Animation styles and text effects
go run examples/progress_spinners/main.go # Progress bars and spinners
go run examples/composition_demo/main.go  # Advanced composition system
go run examples/metrics_demo/main.go      # Performance metrics visualization
go run examples/claude_style/main.go     # Claude Code-style interface
```

### Testing Individual Components
```bash
go test -run TestSpecificFunction ./...
```

## Architecture

### Message-Driven Runtime (Recommended)

**The primary way to build Gooey applications is through the Runtime's message-driven architecture.** This eliminates race conditions, removes the need for manual synchronization, and provides a clean event-driven pattern.

**Core Components:**

1. **Runtime** (`runtime.go`) - Event loop orchestrator:
   - Single-threaded event processing eliminates race conditions
   - Automatic rendering after each event
   - Three-goroutine architecture (event loop, input reader, command executor)
   - Configurable FPS (30-60 recommended)
   - Implements the Application interface pattern

2. **Application Interface** - Your application implements:
   - `HandleEvent(event Event) []Cmd` - Process events, return async commands
   - `Render(frame RenderFrame)` - Draw current state
   - Optional: `Init() error` and `Destroy()` for lifecycle management

3. **Event System** (`events.go`) - Unified event handling:
   - `TickEvent` - Animation frames at configured FPS
   - `KeyEvent` - Keyboard input
   - `MouseEvent` - Mouse interactions
   - `ResizeEvent` - Terminal size changes
   - `QuitEvent` - Application shutdown
   - Custom events for async operation results

4. **Command System** (`commands.go`) - Async operations:
   - `Cmd` type for non-blocking operations (HTTP, timers, I/O)
   - Built-in commands: `Quit()`, `Tick()`, `After()`, `Batch()`
   - Commands execute in separate goroutines
   - Results return as events to the main loop

**Benefits:**
- ✅ **Race-Free** - HandleEvent and Render never called concurrently
- ✅ **No Locks Needed** - Single-threaded event loop guarantees safety
- ✅ **Simple Code** - No manual goroutine or channel management
- ✅ **Async Support** - Commands handle I/O without blocking UI
- ✅ **Automatic Rendering** - Screen updates after every event

See `MESSAGE_DRIVEN_ARCHITECTURE.md` for complete details and `examples/runtime_*` for examples.

### Rendering Pipeline

The library's rendering foundation (used by Runtime):

1. **Terminal** (`terminal.go`) - Low-level foundation:
   - Raw mode and alternate screen buffer
   - Double-buffered rendering (front/back buffers)
   - ANSI escape sequence generation
   - Wide character (Unicode) handling via `github.com/mattn/go-runewidth`

2. **Frame-based Rendering** (`frame.go`) - Atomic operations:
   - `BeginFrame()` / `EndFrame()` provide transactional rendering
   - All operations between BeginFrame/EndFrame are batched
   - Only dirty regions are flushed to minimize output
   - Prevents interleaved writes from concurrent renderers

3. **Composition System** (`composition.go`, `container.go`, layout managers):
   - `ComposableWidget` interface with bounds-based positioning
   - `Container` component for managing child widgets
   - Layout managers (VBox, HBox, FlexLayout) for automatic positioning
   - Parent-child relationships for event propagation and lifecycle management
   - Works seamlessly with Runtime

### Key Abstractions

**Cell and Buffer System:**
- `Cell` represents a single terminal cell with char, style, width, and continuation flag
- Double-buffering: front buffer (displayed) vs back buffer (being rendered)
- `DirtyRegion` tracks modified areas for efficient partial updates

**Styles:**
- `Style` struct supports ANSI colors, RGB colors, and text attributes (bold, italic, underline, etc.)
- RGB support via `FgRGB` and `BgRGB` fields
- Predefined border styles: `SingleBorder`, `DoubleBorder`, `RoundedBorder`, `ThickBorder`

**Input Handling:**
- Three primary input methods:
  - `Read()` - Full-featured input with history, suggestions, cursor editing, multiline support
  - `ReadPassword()` - Secure password input with no echo
  - `ReadSimple()` - Basic line reading using bufio.Scanner
- `KeyEvent` encapsulates keyboard events with modifiers
- `KeyDecoder` - Unified key decoding for ANSI escape sequences and UTF-8
- Legacy methods (ReadLine, ReadLineEnhanced, ReadInteractive, etc.) are removed
- Mouse support via `MouseHandler` and `MouseRegion` (`mouse.go`)

**Layouts:**
- `Layout` (`layout.go`) - Basic header/footer/content organization

**Animations:**
- `TextAnimation` interface: `GetStyle(frame, charIndex, totalChars) Style`
- Built-in: `RainbowAnimation`, `PulseAnimation`, `WaveAnimation`
- Helper functions: `CreateRainbowText()`, `CreatePulseText()`, `CreateReverseRainbowText()`
- Animations update per-character styles based on frame counter

**Components:**
- `Button` - Clickable buttons with hover states
- `TabCompleter` - Tab completion dropdown
- `Spinner` - Loading spinners with multiple predefined styles
- `AnimatedStatusBar` - Status bars with animated values

**Composition System:**
- `ComposableWidget` interface - Enhanced widget interface with bounds management, lifecycle (Init/Destroy), parent-child relationships, visibility, and dirty tracking
- `BaseWidget` - Helper struct providing default implementations; widgets can embed this for automatic composition support
- `Container` - General-purpose widget container with pluggable layout managers, border support, and automatic event delegation
- `LayoutManager` interface - Pluggable layout system for automatic positioning
  - `VBoxLayout` - Vertical stacking with configurable spacing and alignment
  - `HBoxLayout` - Horizontal arrangement with configurable spacing and alignment
  - `FlexLayout` - CSS Flexbox-style layout with justify-content, align-items, flex-grow/shrink, and optional wrapping
- `LayoutParams` - Per-widget layout configuration (flex grow/shrink, margins, padding, alignment, size constraints)
- Composable widgets: `ComposableButton`, `ComposableLabel`, `ComposableMultiLineLabel`

### Thread Safety

The library uses extensive locking to support concurrent operations:

- Terminal operations: `mu sync.RWMutex` protects terminal state
- Runtime ensures single-threaded access to application state in `HandleEvent` and `Render`
- Always use provided synchronization; avoid direct terminal writes

### Flicker Prevention Strategy

1. **Double Buffering:** Changes written to back buffer, swapped on flush
2. **Dirty Regions:** Only modified cells are updated (`DirtyRegion.Mark()`)
3. **Batched Output:** Frame-based rendering batches all ANSI codes
4. **Rate Limiting:** Runtime enforces FPS limits
5. **Atomic Frames:** BeginFrame/EndFrame prevents partial renders

### Performance Metrics

The metrics system (`metrics.go`) provides optional performance profiling:

- **Tracked Statistics:**
  - Frames rendered and skipped
  - Cells updated per frame
  - ANSI escape codes emitted
  - Bytes written to terminal
  - Frame render times (min, max, average, last)
  - Dirty region sizes

- **API:**
  ```go
  terminal.EnableMetrics()        // Turn on tracking
  metrics := terminal.GetMetrics() // Get snapshot
  terminal.ResetMetrics()         // Clear stats
  terminal.DisableMetrics()       // Turn off tracking
  ```

- **Features:**
  - Disabled by default (zero overhead)
  - Thread-safe with RWMutex
  - Minimal overhead when enabled (<5%)
  - Formatted output via `String()` and `Compact()`
  - Helper methods: `FPS()`, `Efficiency()`

- **Use Cases:**
  - Performance profiling during development
  - Regression testing in benchmarks
  - Live monitoring in debug mode
  - Optimization validation

See `documentation/metrics.md` and `examples/metrics_demo/` for details.

### Coordinate Systems and Positioning

**CRITICAL:** Understanding coordinate systems is essential for working with nested components and avoiding rendering bugs.

#### Two Coordinate Systems

1. **Absolute (Screen) Coordinates** - Used by old-style widgets:
   - Origin (0, 0) is top-left of the terminal
   - Widget stores X, Y fields for position
   - Widget draws directly at these absolute positions
   - Example: `Button{X: 10, Y: 5}` always draws at column 10, row 5

2. **Bounds-Based (Relative) Coordinates** - Used by composition system:
   - Widget stores an `image.Rectangle` defining its area
   - Rectangle coordinates are relative to the parent's coordinate space
   - Parent uses `SubFrame()` to create a clipped drawing area for each child
   - Child draws at (0, 0) within its SubFrame, which is automatically translated

#### How SubFrame Works

`SubFrame(rect)` creates a new RenderFrame for a child widget:

```go
// Parent creates SubFrame for child
childBounds := child.GetBounds()  // e.g., Rect(5, 2, 25, 8)
childFrame := frame.SubFrame(childBounds)

// Inside SubFrame implementation:
// 1. Translates rect to absolute coordinates: rect.Add(parentFrame.bounds.Min)
// 2. Clips to parent's bounds: parentFrame.bounds.Intersect(translated)
// 3. Returns new frame with clipped bounds
```

**Key Insight:** When a child draws at (0, 0) in its SubFrame, it's actually drawing at the child's bounds origin in the parent's coordinate space.

#### Common Mistakes and Solutions

❌ **WRONG: Using absolute coordinates in composable widgets**
```go
func (w *MyWidget) Draw(frame RenderFrame) {
    // BUG: Hardcoded position ignores widget's bounds!
    frame.PrintStyled(10, 5, "text", style)
}
```

✅ **CORRECT: Using bounds-relative coordinates**
```go
func (w *MyWidget) Draw(frame RenderFrame) {
    bounds := w.GetBounds()
    // Draw at bounds origin (relative to parent)
    frame.PrintStyled(bounds.Min.X, bounds.Min.Y, "text", style)
}
```

❌ **WRONG: Manual coordinate translation in child**
```go
func (w *MyWidget) Draw(frame RenderFrame) {
    bounds := w.GetBounds()
    // BUG: Don't manually add parent offsets!
    frame.PrintStyled(bounds.Min.X + parentX, bounds.Min.Y + parentY, "text", style)
}
```

✅ **CORRECT: Let SubFrame handle translation**
```go
// In Container.Draw():
for _, child := range c.children {
    childBounds := child.GetBounds()
    // SubFrame automatically handles coordinate translation
    childFrame := frame.SubFrame(childBounds)
    child.Draw(childFrame)  // Child draws at (0,0) in its own space
}
```

❌ **WRONG: Forgetting bounds are relative**
```go
// Setting child bounds with absolute screen coordinates
child.SetBounds(image.Rect(10, 5, 30, 10))  // Absolute position on screen
container.AddChild(child)
// BUG: When container is at (20, 10), child will be at (30, 15), not (10, 5)!
```

✅ **CORRECT: Bounds are relative to parent**
```go
// Layout manager sets bounds relative to container's content area
child.SetBounds(image.Rect(0, 0, 20, 5))  // Position 0,0 within parent
container.AddChild(child)
// Child renders at container's position + (0, 0)
```

#### Best Practices

1. **In Composable Widgets:**
   - Always use `GetBounds()` to determine where to draw
   - Draw relative to `bounds.Min.X` and `bounds.Min.Y`
   - Never hardcode absolute positions

2. **When Creating SubFrames:**
   - Use `frame.SubFrame(childBounds)` for each child
   - The bounds passed to SubFrame should be relative to the current frame
   - SubFrame handles all coordinate translation automatically

3. **When Using Containers:**
   - Let layout managers set child bounds
   - Bounds are relative to the container's content area (inside borders)
   - Don't manually adjust child bounds for container position

4. **Debugging Coordinate Issues:**
   - Check if bounds are being set correctly by layout manager
   - Verify SubFrame is being used for nested rendering
   - Ensure widget isn't mixing absolute and relative coordinates
   - Use bounds.Min.X/Y, not hardcoded values

#### Example: Correct Nested Rendering

```go
// Container at screen position (10, 5) with size 50x20
container.SetBounds(image.Rect(10, 5, 60, 25))

// Layout manager sets child bounds RELATIVE to container
// Child at (2, 2) within container means screen position (12, 7)
child.SetBounds(image.Rect(2, 2, 22, 7))

// In Container.Draw(frame):
childFrame := frame.SubFrame(child.GetBounds())  // Handles translation
child.Draw(childFrame)

// In Child.Draw(frame):
bounds := child.GetBounds()
frameWidth, frameHeight := frame.Size()

// Detect if we're in a SubFrame created by parent
inSubFrame := (frameWidth == bounds.Dx() && frameHeight == bounds.Dy())

if inSubFrame {
    // Parent created SubFrame for us - draw at (0, 0)
    frame.PrintStyled(0, 0, "text", style)
} else {
    // Drawing directly to terminal frame - use absolute bounds
    frame.PrintStyled(bounds.Min.X, bounds.Min.Y, "text", style)
}
```

**Why the detection?** Composable widgets can be used in two ways:
1. **In a Container** - Parent calls `frame.SubFrame(childBounds)` and passes the clipped frame to child. Child should draw at (0, 0).
2. **Standalone** - Widget is drawn directly to terminal frame without a parent container. Widget should draw at its absolute bounds position.

The SubFrame detection pattern (used in `ComposableLabel`, `ComposableButton`, etc.) allows widgets to work correctly in both scenarios.

**Alternative approach:** If your widget will ONLY be used inside containers, you can simplify by always drawing at (0, 0):

```go
func (w *MyWidget) Draw(frame RenderFrame) {
    // Simpler: assume parent always creates SubFrame
    frame.PrintStyled(0, 0, "text", style)
}
```

This works because containers always call `frame.SubFrame(childBounds)` before passing the frame to children.

## Common Patterns

### Creating a Runtime Application (Recommended)

```go
type MyApp struct {
    count int
    // ... your state here (no locks needed!)
}

func (app *MyApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    switch e := event.(type) {
    case gooey.KeyEvent:
        if e.Rune == 'q' {
            return []gooey.Cmd{gooey.Quit()}
        }
        // Handle other keys...
    case gooey.TickEvent:
        // Update animations
        app.count++
    case gooey.ResizeEvent:
        // Handle terminal resize
    }
    return nil
}

func (app *MyApp) Render(frame gooey.RenderFrame) {
    style := gooey.NewStyle().WithForeground(gooey.ColorGreen)
    frame.PrintStyled(0, 0, fmt.Sprintf("Count: %d", app.count), style)
}

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    runtime := gooey.NewRuntime(terminal, &MyApp{}, 30)
    runtime.Run()  // Blocks until quit
}
```

### Async Operations with Commands

```go
func (app *MyApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    switch e := event.(type) {
    case gooey.KeyEvent:
        if e.Rune == 'f' {
            app.loading = true
            return []gooey.Cmd{FetchData()}  // Start async operation
        }
    case DataResultEvent:
        app.loading = false
        app.data = e.Data
    }
    return nil
}

// Command runs in separate goroutine, doesn't block UI
func FetchData() gooey.Cmd {
    return func() gooey.Event {
        resp, err := http.Get("https://api.example.com/data")
        if err != nil {
            return gooey.ErrorEvent{Time: time.Now(), Err: err}
        }
        defer resp.Body.Close()
        data, _ := io.ReadAll(resp.Body)
        return DataResultEvent{Data: string(data)}
    }
}
```

### Direct Frame Rendering

```go
frame, err := terminal.BeginFrame()
if err != nil {
    return err
}

frame.PrintStyled(x, y, text, style)
frame.FillStyled(x, y, width, height, ' ', style)
frame.SetCell(x, y, rune, style)

terminal.EndFrame(frame)
```

### Custom Animation

```go
type CustomAnimation struct {
    Speed int
}

func (a *CustomAnimation) GetStyle(frame uint64, charIndex, totalChars int) gooey.Style {
    // Return style based on frame counter and character position
    offset := int(frame/uint64(a.Speed)) + charIndex
    color := calculateColor(offset)
    return gooey.NewStyle().WithFgRGB(color)
}
```

### Building Layouts with Composition System and Runtime

```go
type CompositionApp struct {
    container *gooey.Container
    label     *gooey.ComposableLabel
    count     int
}

func (app *CompositionApp) Init() error {
    // Create main container with vertical layout
    app.container = gooey.NewContainer(gooey.NewVBoxLayout(2))

    // Add header
    header := gooey.NewComposableLabel("My Application")
    header.WithStyle(gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan))
    app.container.AddChild(header)

    // Add counter label
    app.label = gooey.NewComposableLabel("Count: 0")
    app.container.AddChild(app.label)

    // Create button bar with horizontal layout
    buttonBar := gooey.NewContainer(gooey.NewHBoxLayout(2))
    buttonBar.AddChild(gooey.NewComposableButton("Increment", func() {}))
    buttonBar.AddChild(gooey.NewComposableButton("Quit", func() {}))
    app.container.AddChild(buttonBar)

    // Initialize and set bounds (will be resized in first Render)
    app.container.Init()
    return nil
}

func (app *CompositionApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    switch e := event.(type) {
    case gooey.KeyEvent:
        if e.Key == gooey.KeyEnter {
            app.count++
            app.label.SetText(fmt.Sprintf("Count: %d", app.count))
        } else if e.Rune == 'q' {
            return []gooey.Cmd{gooey.Quit()}
        }
    case gooey.ResizeEvent:
        app.container.SetBounds(image.Rect(0, 0, e.Width, e.Height))
    }
    return nil
}

func (app *CompositionApp) Render(frame gooey.RenderFrame) {
    app.container.Draw(frame)
}

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    runtime := gooey.NewRuntime(terminal, &CompositionApp{}, 30)
    runtime.Run()
}
```

### Handling Terminal Resize with Runtime

With Runtime, resize handling is automatic through ResizeEvent:

```go
type MyApp struct {
    container *gooey.Container
    width     int
    height    int
}

func (app *MyApp) HandleEvent(event gooey.Event) []gooey.Cmd {
    switch e := event.(type) {
    case gooey.ResizeEvent:
        // Runtime automatically sends ResizeEvent when terminal is resized
        app.width = e.Width
        app.height = e.Height

        // Update container bounds if using composition system
        if app.container != nil {
            app.container.SetBounds(image.Rect(0, 0, e.Width, e.Height))
        }
    }
    return nil
}

// Runtime automatically handles terminal.WatchResize() - no manual setup needed!
```

### Using FlexLayout for Advanced Layouts

```go
// Create flexbox container
flex := gooey.NewContainer(
    gooey.NewFlexLayout().
        WithDirection(gooey.FlexRow).
        WithJustify(gooey.FlexJustifySpaceBetween).
        WithAlignItems(gooey.FlexAlignItemsCenter).
        WithSpacing(2),
)

// Add items with different flex behaviors
label1 := gooey.NewComposableLabel("Fixed")
flex.AddChild(label1)

label2 := gooey.NewComposableLabel("Grows")
params := gooey.DefaultLayoutParams()
params.Grow = 1  // This label will grow to fill space
label2.SetLayoutParams(params)
flex.AddChild(label2)

label3 := gooey.NewComposableLabel("Fixed")
flex.AddChild(label3)
```

## Important Constraints

1. **Always call `terminal.Restore()`** - Use `defer terminal.Restore()` immediately after creating terminal to restore normal mode on exit

2. **Coordinate Validation** - Most methods silently ignore out-of-bounds coordinates; validate before calling or expect silent failures

3. **Wide Character Handling** - The library handles wide characters (CJK) via continuation cells; when a wide char is written, the next cell is marked as continuation

4. **Animation Lifecycle** - Always stop animations before closing terminal: call `StopAnimations()` or `animator.Stop()` before cleanup

5. **Input Methods** - Use `Read()` for full-featured input (arrow keys, history, suggestions), `ReadPassword()` for secure password input, or `ReadSimple()` for basic line reading. All legacy input methods are deprecated.

6. **Terminal Size Changes** - The library now provides comprehensive automatic resize handling:
   - Call `terminal.WatchResize()` to enable automatic resize detection via SIGWINCH signals
   - All major components (Layout, AnimatedLayout, ScreenManager, Container) automatically handle resize events
   - For custom resize handling, use `terminal.OnResize(callback)` to register a callback
   - Manual resize detection is still available via `terminal.RefreshSize()`

7. **Composition Lifecycle** - When using the composition system:
   - Always call `Init()` on the root container after building the layout
   - Set bounds on the root container before drawing
   - Call `Destroy()` on containers to clean up child widgets
   - Use `MarkDirty()` when widget state changes to trigger redraws

8. **Coordinate Systems** - Understanding coordinate systems is CRITICAL (see detailed section above):
   - Composable widgets use bounds-based (relative) coordinates, not absolute screen positions
   - Child bounds are relative to parent's content area
   - Use `SubFrame()` for nested rendering - it handles all coordinate translation
   - In composable widgets, detect if in SubFrame and draw at (0,0), or use absolute bounds.Min
   - Never manually add parent offsets - SubFrame does this automatically

## Known Issues and Workarounds

See `API_RECOMMENDATIONS.md` for detailed discussion of:
- Rendering atomicity guarantees
- Coordinate validation
- Terminal size change handling
- Component lifecycle management

## Testing

Test files:
- `stability_test.go` - Concurrent rendering stress tests
- `double_buffer_test.go` - Buffer switching and flicker tests
- `reproduce_issues_test.go` - Regression tests for specific bugs
- `components_test.go` - Component functionality tests

Run with race detector:
```bash
go test -race ./...
```

## Documentation

Additional documentation in `documentation/`:
- `animations.md` - Comprehensive animation guide with examples
- `double_buffering.md` - Buffer implementation details and flicker prevention
- `composition_guide.md` - Complete guide to using the composition system
- `composition_implementation.md` - Implementation details of the composition system
- `composition.md` - Original research and design rationale for composition
- `INPUT_GUIDE.md` - Input handling guide
- `metrics.md` - Performance metrics system guide
- `improvements.md` - Planned enhancements
- `library_review.md` - Architecture review
- `world_class_features_analysis.md` - Feature comparison with modern UI frameworks

## Dependencies

- `golang.org/x/term` - Terminal raw mode and control
- `github.com/mattn/go-runewidth` - Unicode width calculations for proper alignment
- `github.com/stretchr/testify` - Testing assertions
