# Component Composition Guide

This guide explains the new composition system in Gooey that enables building complex, nested layouts similar to web design patterns.

## Overview

The composition system introduces:
- **Bounds-based positioning** instead of absolute X, Y coordinates
- **Container components** that manage child widgets
- **Layout managers** for automatic positioning (VBox, HBox, FlexLayout)
- **Parent-child relationships** for event propagation
- **Lifecycle management** (Init/Destroy)

## Key Interfaces

### ComposableWidget

All composable widgets implement this interface:

```go
type ComposableWidget interface {
    Widget // Draw(frame), HandleKey(event)

    // Lifecycle
    Init()
    Destroy()

    // Bounds management
    SetBounds(bounds image.Rectangle)
    GetBounds() image.Rectangle
    GetMinSize() image.Point
    GetPreferredSize() image.Point

    // Parent-child tree
    GetParent() ComposableWidget
    SetParent(ComposableWidget)

    // Visibility
    IsVisible() bool
    SetVisible(bool)
    NeedsRedraw() bool
    MarkDirty()
    ClearDirty()
}
```

### LayoutManager

Layout managers position children within a container:

```go
type LayoutManager interface {
    Layout(containerBounds image.Rectangle, children []ComposableWidget)
    CalculateMinSize(children []ComposableWidget) image.Point
    CalculatePreferredSize(children []ComposableWidget) image.Point
}
```

## Core Components

### Container

A general-purpose widget that holds children:

```go
container := tui.NewContainer(layout)
container.AddChild(widget1)
container.AddChild(widget2)
container.AddChild(widget3)

// With border
container := tui.NewContainerWithBorder(layout, &tui.SingleBorder)
```

### BaseWidget

Provides default implementation of ComposableWidget for embedding:

```go
type MyWidget struct {
    tui.BaseWidget
    // your fields
}

func NewMyWidget() *MyWidget {
    return &MyWidget{
        BaseWidget: tui.NewBaseWidget(),
    }
}
```

## Layout Managers

### VBoxLayout - Vertical Stack

Arranges children top to bottom:

```go
layout := tui.NewVBoxLayout(spacing)
layout.WithAlignment(tui.LayoutAlignCenter)
layout.WithDistribute(true) // Distribute extra space
```

### HBoxLayout - Horizontal Stack

Arranges children left to right:

```go
layout := tui.NewHBoxLayout(spacing)
layout.WithAlignment(tui.LayoutAlignCenter)
layout.WithDistribute(true)
```

### FlexLayout - Flexible Box

Flexbox-style layout with advanced options:

```go
layout := tui.NewFlexLayout().
    WithDirection(tui.FlexRow).           // or FlexColumn
    WithJustify(tui.FlexJustifySpaceBetween). // Start, End, Center, SpaceBetween, etc.
    WithAlignItems(tui.FlexAlignItemsCenter). // Start, End, Center, Stretch
    WithSpacing(2).
    WithWrap(tui.FlexWrapOn)              // Enable wrapping
```

## Layout Parameters

Control how a widget behaves within its container:

```go
params := tui.DefaultLayoutParams()
params.Grow = 1              // Take extra space (flex-grow)
params.Shrink = 1            // Shrink when constrained
params.Align = tui.LayoutAlignCenter
params.MarginTop = 1         // Margin around widget
params.MarginBottom = 1
params.PaddingLeft = 2       // Padding inside widget
params.PaddingRight = 2

widget.SetLayoutParams(params)
```

### Size Constraints

```go
params.Constraints = tui.SizeConstraints{
    MinWidth:  10,
    MinHeight: 3,
    MaxWidth:  50,  // 0 = no maximum
    MaxHeight: 0,
}
```

## Built-in Composable Widgets

### ComposableButton

```go
button := tui.NewComposableButton("Click Me", func() {
    // Handle click
})

button.Style = tui.NewStyle().WithBackground(tui.ColorBlue)
button.HoverStyle = tui.NewStyle().WithBackground(tui.ColorCyan)
```

### ComposableLabel

```go
label := tui.NewComposableLabel("Hello World")
label.WithStyle(tui.NewStyle().WithForeground(tui.ColorCyan))
label.WithAlign(tui.AlignCenter)

// Update text
label.SetText("New text")
```

### ComposableMultiLineLabel

```go
label := tui.NewComposableMultiLineLabel([]string{
    "Line 1",
    "Line 2",
    "Line 3",
})
label.WithStyle(style)
label.WithAlign(tui.AlignCenter)
```

## Example: Complex Nested Layout

```go
// Main container with vertical layout
main := tui.NewContainer(tui.NewVBoxLayout(2))

// Header
header := tui.NewComposableLabel("My Application")
header.WithStyle(tui.NewStyle().WithBold().WithForeground(tui.ColorCyan))
main.AddChild(header)

// Button bar with horizontal layout
buttonBar := tui.NewContainer(tui.NewHBoxLayout(2))
buttonBar.AddChild(tui.NewComposableButton("Save", onSave))
buttonBar.AddChild(tui.NewComposableButton("Cancel", onCancel))
main.AddChild(buttonBar)

// Content area with border
content := tui.NewContainerWithBorder(
    tui.NewVBoxLayout(1),
    &tui.RoundedBorder,
)
contentParams := tui.DefaultLayoutParams()
contentParams.Grow = 1 // Take remaining space
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

## Event Handling

Events automatically propagate through the component tree:

```go
// Container delegates to children in reverse order (top to bottom)
func (c *Container) HandleKey(event KeyEvent) bool {
    for i := len(c.children) - 1; i >= 0; i-- {
        if c.children[i].HandleKey(event) {
            return true // Event was handled
        }
    }
    return false
}
```

### Mouse Events

Implement the `MouseAware` interface:

```go
type MouseAware interface {
    HandleMouse(event MouseEvent) bool
}
```

Container checks if children implement this interface and delegates appropriately.

## Lifecycle

```go
// Called when widget is added to container
widget.Init()

// Called when widget is removed
widget.Destroy()

// Clean up all children
container.Destroy() // Calls Destroy() on all children
```

## Comparison: Old vs New

### Old Approach (Absolute Positioning)

```go
button := &tui.Button{
    X: 10,
    Y: 5,
    Label: "Click",
    OnClick: handler,
}
screen.AddWidget(button)
```

### New Approach (Composition)

```go
container := tui.NewContainer(tui.NewVBoxLayout(2))
button := tui.NewComposableButton("Click", handler)
container.AddChild(button)

container.SetBounds(image.Rect(0, 0, width, height))
```

## Migration Strategy

The new system is designed to coexist with the old:

1. **Keep existing code working** - Old widgets still work
2. **Gradual migration** - Convert widgets to composable as needed
3. **Adapters** - Wrap old widgets for use in containers if needed

## Advanced: Creating Custom Composable Widgets

```go
type MyCustomWidget struct {
    tui.BaseWidget
    text  string
    style tui.Style
}

func NewMyCustomWidget(text string) *MyCustomWidget {
    w := &MyCustomWidget{
        BaseWidget: tui.NewBaseWidget(),
        text:       text,
        style:      tui.NewStyle(),
    }
    w.SetMinSize(image.Point{X: len(text), Y: 1})
    return w
}

func (w *MyCustomWidget) Draw(frame tui.RenderFrame) {
    if !w.IsVisible() {
        return
    }

    bounds := w.GetBounds()
    frame.PrintStyled(bounds.Min.X, bounds.Min.Y, w.text, w.style)
    w.ClearDirty()
}

func (w *MyCustomWidget) HandleKey(event tui.KeyEvent) bool {
    // Handle keyboard input
    return false
}
```

## Best Practices

1. **Always call Init()** on root container after building layout
2. **Set bounds on root container** to match available space
3. **Use MarkDirty()** when widget state changes to trigger redraw
4. **Prefer composition** over absolute positioning for complex layouts
5. **Use layout params** to control sizing and spacing
6. **Clean up with Destroy()** when removing widgets

## Performance Considerations

- **Dirty tracking** prevents unnecessary redraws
- **Visibility checks** skip hidden widgets
- **Layout caching** - layouts only recalculate when bounds change
- **Event delegation** stops at first handler

## See Also

- `examples/composition_demo/main.go` - Full working example
- `composition.go` - Core interfaces and types
- `container.go` - Container implementation
- `layout_managers.go` - VBox and HBox layouts
- `flex_layout.go` - Flexbox-style layout
