# Constraints and Reflow System

This guide explains the "True Dynamic Reflow" system in Gooey, modeled after Flutter and modern web layout engines. This system enables widgets to adapt their size based on constraints passed down from their parents, allowing for responsive features like text wrapping and flexible layouts.

## The Problem: Constraint-Ignorant Layouts

In the classic 2-pass layout system (Measure -> Layout):
1.  **Measure Pass:** Parents ask children "How big do you want to be?" (`GetPreferredSize`). Children answer without knowing how much space is available.
2.  **Layout Pass:** Parents set the final size (`SetBounds`).

**Limitation:** A text label cannot say "I want to be 1 line tall if I have infinite width, but 3 lines tall if I'm only 100px wide." It has to guess.

## The Solution: Constraint-Based Layout

The new system introduces a `Measure` method that takes `SizeConstraints`.

1.  **Parent:** "You have between 0 and 100px of width. How big do you want to be?"
2.  **Child:** "At 100px width, I need to wrap my text, so I need 3 lines of height."
3.  **Parent:** "Okay, I'll position you with that size."

### Key Interfaces

#### 1. SizeConstraints

Defines the permissible range for width and height.

```go
type SizeConstraints struct {
    MinWidth  int
    MinHeight int
    MaxWidth  int // 0 means unconstrained (infinite)
    MaxHeight int // 0 means unconstrained
}
```

#### 2. Measurable Interface

Widgets that want to participate in dynamic reflow should implement this interface:

```go
type Measurable interface {
    ComposableWidget
    // Measure calculates the desired size of the widget given the constraints.
    Measure(constraints SizeConstraints) image.Point
}
```

#### 3. ConstraintLayoutManager

Layout managers (like VBox, HBox) now implement this interface to propagate constraints:

```go
type ConstraintLayoutManager interface {
    LayoutManager
    Measure(children []ComposableWidget, constraints SizeConstraints) image.Point
    LayoutWithConstraints(children []ComposableWidget, constraints SizeConstraints, containerBounds image.Rectangle)
}
```

## Using the System

### 1. Using WrappingLabel

The `WrappingLabel` is a built-in widget that implements `Measurable`. It automatically wraps text to fit the available width.

```go
// Create a wrapping label
label := gooey.NewWrappingLabel("This text will automatically wrap if the container is too narrow.")

// Add to a container (VBox supports constraints automatically)
container := gooey.NewContainer(gooey.NewVBoxLayout(1))
container.AddChild(label)
```

### 2. Implementing Custom Measurable Widgets

To create a custom widget that adapts to constraints:

```go
type MyAdaptiveWidget struct {
    gooey.BaseWidget
}

func (w *MyAdaptiveWidget) Measure(c gooey.SizeConstraints) image.Point {
    // 1. Check available width
    targetWidth := 100 // Default preferred width
    if c.HasMaxWidth() && c.MaxWidth < targetWidth {
        targetWidth = c.MaxWidth
    }

    // 2. Calculate height based on width
    height := 10
    if targetWidth < 50 {
        height = 20 // Become taller if narrow
    }

    // 3. Return size respecting constraints
    return c.Constrain(image.Point{X: targetWidth, Y: height})
}
```

### 3. Manually Triggering Measure

If you are manually managing a layout loop (rare), you must call `Measure` before `SetBounds`:

```go
// 1. Measure
size := container.Measure(gooey.SizeConstraints{
    MaxWidth: terminalWidth,
    MaxHeight: terminalHeight,
})

// 2. Layout
container.SetBounds(image.Rect(0, 0, size.X, size.Y))
```

## Supported Layouts

Currently, the following layout managers support constraint propagation:

*   **VBoxLayout:** Constrains child width to its own width (minus margins). Allows unconstrained height.
*   **HBoxLayout:** Constrains child height to its own height (minus margins). Allows unconstrained width.

## Best Practices

*   **Always use `MeasureWidget` helper:** When writing custom layouts, use `gooey.MeasureWidget(child, constraints)` instead of checking for the interface manually. It handles legacy fallback automatically.
*   **Respect Constraints:** Your `Measure` implementation MUST return a size that falls within the `MinWidth`/`MaxWidth` and `MinHeight`/`MaxHeight`. Use `constraints.Constrain(size)` to ensure this.
*   **Cache Results:** Measuring can be expensive (e.g., text wrapping). Consider caching the result if the constraints haven't changed (like `WrappingLabel` does).

## Example: Reflow Demo

See `examples/reflow_demo/main.go` for a working example of a UI that adapts to changing terminal width.
