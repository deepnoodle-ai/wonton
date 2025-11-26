# Declarative Rendering API Plan

## Overview

This plan introduces an **opt-in declarative rendering API** for Gooey that coexists with the existing imperative `Render(frame RenderFrame)` approach. Applications can choose either style, or mix them freely.

## Design Goals

1. **Elegant API**: Natural Go patterns, minimal boilerplate, intuitive composition
2. **Type-safe**: Leverage Go's type system for compile-time checks
3. **Zero-cost opt-in**: Apps using imperative rendering pay nothing
4. **Escape hatches**: Seamlessly mix declarative and imperative code
5. **Familiar patterns**: Draw inspiration from SwiftUI/Flutter/React, adapted for Go idioms

## Core API Design

### The `View` Interface

Applications implementing declarative rendering return a `View` from a `View()` method:

```go
// DeclarativeApp is the interface for declarative applications
type DeclarativeApp interface {
    View() View
    HandleEvent(event Event) []Cmd  // Same as before
}

// View is the core abstraction - anything that can render
type View interface {
    // Render draws this view into the given frame at the given bounds
    render(frame RenderFrame, bounds image.Rectangle)

    // Size returns the preferred size of this view
    size(maxWidth, maxHeight int) (width, height int)
}
```

Note: The `View` interface methods are unexported. Users compose views using builder functions, not by implementing the interface.

### Basic Views

```go
// Text displays styled text
Text("Hello, World!")
Text("Count: %d", count)  // Printf-style formatting

// Styled text using method chaining
Text("Error!").Fg(ColorRed).Bold()

// Spacer fills available space
Spacer()

// Empty renders nothing (useful for conditionals)
Empty()
```

### Layout Containers

```go
// VStack arranges children vertically
VStack(
    Text("Line 1"),
    Text("Line 2"),
    Text("Line 3"),
)

// HStack arranges children horizontally
HStack(
    Text("Left"),
    Spacer(),
    Text("Right"),
)

// ZStack layers children (back to front)
ZStack(
    Fill(' ').Bg(ColorBlue),  // Background
    Text("Foreground"),
)
```

### Layout Modifiers

```go
// Padding adds space around content
VStack(...).Padding(1)           // All sides
VStack(...).PaddingH(2)          // Horizontal only
VStack(...).PaddingV(1)          // Vertical only
VStack(...).PaddingLTRB(1,2,1,2) // Left, Top, Right, Bottom

// Gap adds space between children
VStack(...).Gap(1)
HStack(...).Gap(2)

// Alignment
VStack(...).Align(AlignCenter)
HStack(...).Align(AlignStart, AlignEnd)  // Main axis, cross axis

// Size constraints
Text("Fixed").Width(20)
Text("Bounded").MaxWidth(40)
VStack(...).Height(10)
Container(...).Size(40, 20)

// Frame creates a box with optional border
VStack(...).Frame().Border(BorderSingle)
VStack(...).Frame().Border(BorderDouble).Title("Settings")
```

### Styling

```go
// Foreground/Background colors
Text("Colored").Fg(ColorGreen).Bg(ColorBlack)

// RGB colors
Text("Orange").FgRGB(255, 128, 0)

// Text attributes
Text("Styled").Bold().Italic().Underline()

// Apply a complete style
style := NewStyle().WithForeground(ColorCyan).WithBold()
Text("Styled").Style(style)
```

### Conditional Rendering

```go
// Show/hide based on condition
If(isLoggedIn,
    Text("Welcome back!"),
)

// If/Else
IfElse(isLoading,
    Text("Loading..."),
    Text("Ready"),
)

// Switch-like pattern
Switch(status,
    Case("loading", Spinner()),
    Case("error", Text("Error!").Fg(ColorRed)),
    Case("ready", Text("Ready").Fg(ColorGreen)),
    Default(Empty()),
)
```

### Collections

```go
// ForEach renders a view for each item
ForEach(items, func(item Item, i int) View {
    return Text("%d. %s", i+1, item.Name)
})

// With separator
ForEach(items, func(item Item, i int) View {
    return Text(item.Name)
}).Separator(Text("---"))
```

### Interactive Elements

```go
// Button with callback
Button("Click Me", func() {
    app.count++
})

// Styled button
Button("+", app.increment).Fg(ColorGreen).Bold()

// Text input with two-way binding
Input(&app.name)
Input(&app.name).Placeholder("Enter name...")
Input(&app.password).Mask('*')

// With callbacks
Input(&app.query).
    OnChange(func(s string) { app.search(s) }).
    OnSubmit(func(s string) { app.executeSearch() })

// Checkbox
Checkbox(&app.agree, "I agree to terms")

// Selection
Select(&app.choice, []string{"Option A", "Option B", "Option C"})
```

### Focus Management

```go
// Focusable wraps content to make it focusable
Focusable("input-name",
    Input(&app.name),
)

// Focus ring styling
Focusable("btn-submit",
    Button("Submit", app.submit),
).FocusStyle(NewStyle().WithReverse())

// Programmatic focus (in HandleEvent)
func (app *App) HandleEvent(e Event) []Cmd {
    if key, ok := e.(KeyEvent); ok && key.Key == KeyTab {
        return []Cmd{Focus("next")}  // or Focus("input-name")
    }
    return nil
}
```

### Imperative Escape Hatch

```go
// Canvas allows imperative drawing within declarative tree
Canvas(func(frame RenderFrame, bounds image.Rectangle) {
    // Full access to imperative API
    drawSparkline(frame, app.data)
    drawCustomChart(frame, app.metrics)
})

// With size hints
Canvas(func(frame RenderFrame, bounds image.Rectangle) {
    drawGraph(frame, app.points)
}).Size(40, 10)
```

## Complete Example

```go
package main

import "gooey"

type App struct {
    name    string
    count   int
    items   []string
    focused string
}

func (app *App) View() gooey.View {
    return VStack(
        // Header
        Text("Counter App").Bold().Fg(ColorCyan),

        // Name input
        HStack(
            Text("Name: "),
            Input(&app.name).Placeholder("Enter name..."),
        ).Gap(1),

        // Counter controls
        HStack(
            Button("-", app.decrement).Width(3),
            Text(" %d ", app.count).Bold(),
            Button("+", app.increment).Width(3),
        ).Gap(1).Align(AlignCenter),

        // Conditional message
        If(app.count > 10,
            Text("That's a lot!").Fg(ColorYellow),
        ),

        // Item list
        IfElse(len(app.items) > 0,
            VStack(
                Text("Items:").Underline(),
                ForEach(app.items, func(item string, i int) View {
                    return Text("  • %s", item)
                }),
            ),
            Text("No items yet").Fg(ColorGray),
        ),

        Spacer(),

        // Footer with custom drawing
        Canvas(func(frame RenderFrame, bounds image.Rectangle) {
            drawStatusBar(frame, bounds, app)
        }).Height(1),
    ).Padding(1)
}

func (app *App) HandleEvent(e gooey.Event) []gooey.Cmd {
    switch ev := e.(type) {
    case gooey.KeyEvent:
        if ev.Rune == 'q' {
            return []gooey.Cmd{gooey.Quit()}
        }
    }
    return nil
}

func (app *App) increment() { app.count++ }
func (app *App) decrement() { app.count-- }

func main() {
    gooey.Run(&App{name: "World"})
}
```

## Implementation Plan

### Phase 1: Core Infrastructure

**Files to create:**
- `view.go` - Core `View` interface and base implementations
- `text_view.go` - Text view with styling
- `layout_views.go` - VStack, HStack, ZStack, Spacer
- `modifiers.go` - Padding, Gap, Frame, Size modifiers

**Key tasks:**

1. **Define the View interface**
   - `render(frame RenderFrame, bounds image.Rectangle)` - Draw to frame
   - `size(maxWidth, maxHeight int) (width, height int)` - Calculate size

2. **Implement modifier pattern**
   - Views return new wrapped views when modifiers are applied
   - Use struct embedding for composition
   - Chain modifiers fluently

3. **Implement layout algorithm**
   - VStack: Measure children heights, distribute space, render top-to-bottom
   - HStack: Measure children widths, distribute space, render left-to-right
   - Handle flexible items (Spacer) vs fixed-size items

4. **Update Runtime to detect DeclarativeApp**
   - Check if app implements `View() View`
   - Call View() and render the tree instead of Render(frame)

### Phase 2: Styling and Conditionals

**Files to modify/create:**
- `text_view.go` - Add all style modifiers
- `conditional_views.go` - If, IfElse, Switch
- `collection_views.go` - ForEach

**Key tasks:**

1. **Style propagation**
   - Modifiers like `.Fg()`, `.Bold()` wrap views with style overlays
   - Styles compose/override parent styles

2. **Conditional views**
   - `If(cond, view)` returns view or Empty based on condition
   - `IfElse(cond, trueView, falseView)` for branching
   - Evaluated fresh each render (no state)

3. **Collection rendering**
   - `ForEach` calls mapper function for each item
   - Returns VStack/HStack of results
   - Optional separator support

### Phase 3: Interactive Elements

**Files to create:**
- `button_view.go` - Button with callbacks
- `input_view.go` - Declarative wrapper for TextInput
- `focus.go` - Focus management system

**Key tasks:**

1. **Button implementation**
   - Render as styled text
   - Track mouse hover/click regions
   - Call callback on click

2. **Input binding**
   - `Input(&app.field)` creates two-way binding via pointer
   - Wraps existing TextInput widget
   - Syncs value on each render

3. **Focus management**
   - Global focus registry
   - Tab navigation
   - Focus ring styling

### Phase 4: Canvas and Advanced Features

**Files to create:**
- `canvas_view.go` - Imperative escape hatch
- `scroll_view.go` - Scrollable container (future)

**Key tasks:**

1. **Canvas view**
   - Takes render function
   - Provides SubFrame for custom drawing
   - Size hints for layout

2. **Integration testing**
   - Verify imperative and declarative work together
   - Performance benchmarks
   - Example applications

## Architecture Details

### View Wrapping Pattern

Views are immutable value types. Modifiers return new wrapped views:

```go
type textView struct {
    format string
    args   []any
    style  Style
}

func (t textView) Fg(c Color) View {
    t.style = t.style.WithForeground(c)
    return t
}

func (t textView) Bold() View {
    t.style = t.style.WithBold()
    return t
}
```

### Layout Algorithm

For VStack with children c1, c2, c3:

1. **Measure phase**: Ask each child for preferred size given max constraints
2. **Allocate phase**: Distribute available height among children
   - Fixed-size children get their requested size
   - Flexible children (Spacer) split remaining space
3. **Render phase**: Draw each child at calculated bounds

```
Available: 100x50
c1 wants: 100x3 (fixed)
c2 wants: flexible (Spacer)
c3 wants: 100x5 (fixed)

Allocation:
c1: y=0, height=3
c2: y=3, height=42 (fills remaining)
c3: y=45, height=5
```

### Event Routing for Interactive Elements

Interactive views (Button, Input) register themselves during render:

```go
type interactiveRegistry struct {
    buttons []buttonRegion
    inputs  []*inputState
    focused string
}

func (r *Runtime) renderDeclarative(app DeclarativeApp) {
    r.registry.clear()
    view := app.View()
    view.render(frame, fullBounds)
    // Registry now contains all interactive regions
}

func (r *Runtime) handleMouseClick(x, y int) {
    for _, btn := range r.registry.buttons {
        if btn.bounds.Contains(x, y) {
            btn.callback()
            return
        }
    }
}
```

### Two-Way Binding

Input uses pointer binding for simplicity:

```go
func Input(binding *string) View {
    return &inputView{
        binding: binding,
        // On render, creates/updates TextInput with current *binding value
        // On change, updates *binding directly
    }
}
```

## Migration Path

Existing apps continue to work unchanged. To adopt declarative:

1. **Keep HandleEvent** - Event handling stays the same
2. **Replace Render with View** - Return view tree instead of imperative calls
3. **Gradual adoption** - Use Canvas for complex parts initially

## File Structure

```
gooey/
├── view.go              # View interface, Empty, Spacer
├── text_view.go         # Text view
├── layout_views.go      # VStack, HStack, ZStack
├── modifiers.go         # Padding, Gap, Frame, Size, Align
├── conditional_views.go # If, IfElse, Switch
├── collection_views.go  # ForEach
├── button_view.go       # Button
├── input_view.go        # Input (wraps TextInput)
├── canvas_view.go       # Canvas escape hatch
├── focus.go             # Focus management
├── declarative.go       # DeclarativeApp interface, rendering integration
└── examples/
    └── declarative/
        └── main.go      # Example declarative app
```

## Open Questions

1. **State management**: Should we add any reactive primitives (signals/atoms) or keep it simple with struct fields?

2. **Animation**: How should animations work in declarative mode? Perhaps `Animated(view, duration)` wrapper?

3. **Diffing**: Should we diff view trees to minimize redraws, or rely on dirty region tracking?

4. **Keyboard shortcuts**: How to bind keys to actions declaratively? Perhaps `Button(...).Key('+')`?

## Summary

This plan provides a clean, idiomatic Go API for declarative UI that:

- Feels natural to Go developers
- Integrates seamlessly with existing imperative code
- Supports all current Gooey features
- Enables rapid UI prototyping
- Maintains type safety and performance
