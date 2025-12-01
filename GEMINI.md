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

### Running Examples
The comprehensive demo showcases all features:
```bash
go run examples/all/main.go
```

Other examples:
```bash
go run examples/simple_animation_demo.go
go run examples/interactive/main.go
go run examples/progress_spinners/main.go
go run examples/layout_styling/main.go
go run examples/composition_demo/main.go  # Demonstrates the composition system
```

### Testing Individual Components
```bash
go test -run TestSpecificFunction ./...
```

## Architecture

### Rendering Pipeline

The library uses a multi-layered rendering architecture designed to eliminate flicker and enable concurrent updates:

1. **Terminal** (`terminal.go`) - Low-level foundation that manages:
   - Raw mode and alternate screen buffer
   - Double-buffered rendering (front/back buffers)
   - ANSI escape sequence generation
   - Wide character (Unicode) handling via `github.com/mattn/go-runewidth`

2. **Frame-based Rendering** (`frame.go`) - Atomic rendering operations:
   - `BeginFrame()` / `EndFrame()` provide transactional rendering
   - All operations between BeginFrame/EndFrame are batched
   - Only dirty regions are flushed to minimize output
   - Prevents interleaved writes from concurrent renderers

3. **ScreenManager** (`screen_manager.go`) - Virtual screen coordination:
   - Manages named regions (header, body, footer, etc.)
   - Handles animations via `TextAnimation` interface
   - Runs draw loop at configurable FPS (default 30)
   - Batches rapid updates to prevent excessive redraws

4. **Animator** (`animator.go`) - Animation engine:
   - Dedicated goroutine for animation updates
   - Manages `AnimatedElement` instances
   - Calls `Update(frame)` and `Draw(frame)` on each element
   - Thread-safe element addition/removal

5. **Composition System** (`composition.go`, `container.go`, layout managers) - Modern component hierarchy:
   - `ComposableWidget` interface with bounds-based positioning
   - `Container` component for managing child widgets
   - Layout managers (VBox, HBox, FlexLayout) for automatic positioning
   - Parent-child relationships for event propagation and lifecycle management
   - Enables building complex nested UIs similar to web frameworks

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
- Multiple input variants: `input.go`, `input_enhanced.go`, `input_fixed.go`, `input_interactive.go`
- `KeyEvent` encapsulates keyboard events with modifiers
- Mouse support via `MouseHandler` and `MouseRegion` (`mouse.go`)

**Layouts:**
- `Layout` (`layout.go`) - Basic header/footer/content organization
- `AnimatedLayout` (`animated_layout.go`) - Adds animation support to layouts
- `AnimatedInputLayout` - Combines animated regions with input handling

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
- ScreenManager: `mu sync.RWMutex` for regions, `drawMutex sync.Mutex` ensures single draw
- Animator: `mu sync.RWMutex` protects element list
- Always use provided synchronization; avoid direct terminal writes

### Flicker Prevention Strategy

1. **Double Buffering:** Changes written to back buffer, swapped on flush
2. **Dirty Regions:** Only modified cells are updated (`DirtyRegion.Mark()`)
3. **Batched Output:** Frame-based rendering batches all ANSI codes
4. **Rate Limiting:** ScreenManager enforces minimum draw intervals (50ms)
5. **Atomic Frames:** BeginFrame/EndFrame prevents partial renders

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

**Update (Nov 2025):** Standard widgets (`Button`, `RadioGroup`, `CheckboxGroup`) have been updated to automatically detect if they are rendering within a SubFrame (by checking if frame dimensions match content size) and adjust their coordinates to `(0,0)` if so. This mitigates the common mistake of double translation when using these widgets inside containers, but explicit relative positioning support (via `ComposableWidget` interface) remains the recommended approach for new components.

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

### Creating an Animated TUI

```go
// 1. Initialize terminal
terminal, _ := tui.NewTerminal()
defer terminal.Restore()

// 2. Create animated layout (30 FPS)
layout := tui.NewAnimatedInputLayout(terminal, 30)

// 3. Set up animated regions
layout.SetAnimatedHeader(1)
layout.SetHeaderLine(0, "My App", tui.CreateRainbowText("My App", 20))

// 4. Start animations
layout.StartAnimations()
defer layout.StopAnimations()
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

func (a *CustomAnimation) GetStyle(frame uint64, charIndex, totalChars int) tui.Style {
    // Return style based on frame counter and character position
    offset := int(frame/uint64(a.Speed)) + charIndex
    color := calculateColor(offset)
    return tui.NewStyle().WithFgRGB(color)
}
```

### Building Layouts with Composition System

```go
// Create main container with vertical layout
main := tui.NewContainer(tui.NewVBoxLayout(2))

// Add header
header := tui.NewComposableLabel("My Application")
header.WithStyle(tui.NewStyle().WithBold().WithForeground(tui.ColorCyan))
main.AddChild(header)

// Create button bar with horizontal layout
buttonBar := tui.NewContainer(tui.NewHBoxLayout(2))
buttonBar.AddChild(tui.NewComposableButton("Save", onSave))
buttonBar.AddChild(tui.NewComposableButton("Cancel", onCancel))
main.AddChild(buttonBar)

// Create content area with border and flex-grow
content := tui.NewContainerWithBorder(
    tui.NewVBoxLayout(1),
    &tui.RoundedBorder,
)
contentParams := tui.DefaultLayoutParams()
contentParams.Grow = 1  // Take all remaining space
content.SetLayoutParams(contentParams)
main.AddChild(content)

// Set bounds and initialize
main.SetBounds(image.Rect(0, 0, width, height))
main.Init()

// Draw
frame, _ := terminal.BeginFrame()
main.Draw(frame)
terminal.EndFrame(frame)
```

### Using FlexLayout for Advanced Layouts

```go
// Create flexbox container
flex := tui.NewContainer(
    tui.NewFlexLayout().
        WithDirection(tui.FlexRow).
        WithJustify(tui.FlexJustifySpaceBetween).
        WithAlignItems(tui.FlexAlignItemsCenter).
        WithSpacing(2),
)

// Add items with different flex behaviors
label1 := tui.NewComposableLabel("Fixed")
flex.AddChild(label1)

label2 := tui.NewComposableLabel("Grows")
params := tui.DefaultLayoutParams()
params.Grow = 1  // This label will grow to fill space
label2.SetLayoutParams(params)
flex.AddChild(label2)

label3 := tui.NewComposableLabel("Fixed")
flex.AddChild(label3)
```

## Important Constraints

1. **Always call `terminal.Restore()`** - Use `defer terminal.Restore()` immediately after creating terminal to restore normal mode on exit

2. **Coordinate Validation** - Most methods silently ignore out-of-bounds coordinates; validate before calling or expect silent failures

3. **Wide Character Handling** - The library handles wide characters (CJK) via continuation cells; when a wide char is written, the next cell is marked as continuation

4. **Animation Lifecycle** - Always stop animations before closing terminal: call `StopAnimations()` or `animator.Stop()` before cleanup

5. **Input Modes** - Different input implementations have different capabilities; use `input_enhanced.go` variants for arrow keys and advanced features

6. **Terminal Size Changes** - Call `terminal.RefreshSize()` to detect terminal resizes; no automatic resize detection

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
- `improvements.md` - Planned enhancements
- `library_review.md` - Architecture review
- `world_class_features_analysis.md` - Feature comparison with modern UI frameworks

## Dependencies

- `golang.org/x/term` - Terminal raw mode and control
- `github.com/mattn/go-runewidth` - Unicode width calculations for proper alignment
- Internal `assert` and `require` packages for testing assertions
