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
container := gooey.NewContainer(layout)
container.AddChild(widget1)
container.AddChild(widget2)
container.AddChild(widget3)

// With border
container := gooey.NewContainerWithBorder(layout, &gooey.SingleBorder)
```

### BaseWidget

Provides default implementation of ComposableWidget for embedding:

```go
type MyWidget struct {
    gooey.BaseWidget
    // your fields
}

func NewMyWidget() *MyWidget {
    return &MyWidget{
        BaseWidget: gooey.NewBaseWidget(),
    }
}
```

## Layout Managers

### VBoxLayout - Vertical Stack

Arranges children top to bottom:

```go
layout := gooey.NewVBoxLayout(spacing)
layout.WithAlignment(gooey.LayoutAlignCenter)
layout.WithDistribute(true) // Distribute extra space
```

### HBoxLayout - Horizontal Stack

Arranges children left to right:

```go
layout := gooey.NewHBoxLayout(spacing)
layout.WithAlignment(gooey.LayoutAlignCenter)
layout.WithDistribute(true)
```

### FlexLayout - Flexible Box

Flexbox-style layout with advanced options:

```go
layout := gooey.NewFlexLayout().
    WithDirection(gooey.FlexRow).           // or FlexColumn
    WithJustify(gooey.FlexJustifySpaceBetween). // Start, End, Center, SpaceBetween, etc.
    WithAlignItems(gooey.FlexAlignItemsCenter). // Start, End, Center, Stretch
    WithSpacing(2).
    WithWrap(gooey.FlexWrapOn)              // Enable wrapping
```

## Layout Parameters

Control how a widget behaves within its container:

```go
params := gooey.DefaultLayoutParams()
params.Grow = 1              // Take extra space (flex-grow)
params.Shrink = 1            // Shrink when constrained
params.Align = gooey.LayoutAlignCenter
params.MarginTop = 1         // Margin around widget
params.MarginBottom = 1
params.PaddingLeft = 2       // Padding inside widget
params.PaddingRight = 2

widget.SetLayoutParams(params)
```

### Size Constraints

```go
params.Constraints = gooey.SizeConstraints{
    MinWidth:  10,
    MinHeight: 3,
    MaxWidth:  50,  // 0 = no maximum
    MaxHeight: 0,
}
```

## Built-in Composable Widgets

### ComposableButton

```go
button := gooey.NewComposableButton("Click Me", func() {
    // Handle click
})

button.Style = gooey.NewStyle().WithBackground(gooey.ColorBlue)
button.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorCyan)
```

### ComposableLabel

```go
label := gooey.NewComposableLabel("Hello World")
label.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan))
label.WithAlign(gooey.AlignCenter)

// Update text
label.SetText("New text")
```

### ComposableMultiLineLabel

```go
label := gooey.NewComposableMultiLineLabel([]string{
    "Line 1",
    "Line 2",
    "Line 3",
})
label.WithStyle(style)
label.WithAlign(gooey.AlignCenter)
```

## Example: Complex Nested Layout

```go
// Main container with vertical layout
main := gooey.NewContainer(gooey.NewVBoxLayout(2))

// Header
header := gooey.NewComposableLabel("My Application")
header.WithStyle(gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan))
main.AddChild(header)

// Button bar with horizontal layout
buttonBar := gooey.NewContainer(gooey.NewHBoxLayout(2))
buttonBar.AddChild(gooey.NewComposableButton("Save", onSave))
buttonBar.AddChild(gooey.NewComposableButton("Cancel", onCancel))
main.AddChild(buttonBar)

// Content area with border
content := gooey.NewContainerWithBorder(
    gooey.NewVBoxLayout(1),
    &gooey.RoundedBorder,
)
contentParams := gooey.DefaultLayoutParams()
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
button := &gooey.Button{
    X: 10,
    Y: 5,
    Label: "Click",
    OnClick: handler,
}
screen.AddWidget(button)
```

### New Approach (Composition)

```go
container := gooey.NewContainer(gooey.NewVBoxLayout(2))
button := gooey.NewComposableButton("Click", handler)
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
    gooey.BaseWidget
    text  string
    style gooey.Style
}

func NewMyCustomWidget(text string) *MyCustomWidget {
    w := &MyCustomWidget{
        BaseWidget: gooey.NewBaseWidget(),
        text:       text,
        style:      gooey.NewStyle(),
    }
    w.SetMinSize(image.Point{X: len(text), Y: 1})
    return w
}

func (w *MyCustomWidget) Draw(frame gooey.RenderFrame) {
    if !w.IsVisible() {
        return
    }

    bounds := w.GetBounds()
    frame.PrintStyled(bounds.Min.X, bounds.Min.Y, w.text, w.style)
    w.ClearDirty()
}

func (w *MyCustomWidget) HandleKey(event gooey.KeyEvent) bool {
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
