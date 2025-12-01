# Full Mouse Support Documentation

Gooey provides comprehensive mouse support including clicks, drag-and-drop, hover effects, mouse wheel scrolling, and keyboard modifiers.

## Features

### Event Types

Gooey supports all major mouse event types:

- **MousePress** - Button pressed down
- **MouseRelease** - Button released
- **MouseClick** - Complete click (press + release)
- **MouseDoubleClick** - Two clicks in quick succession
- **MouseTripleClick** - Three clicks in quick succession
- **MouseDrag** - Mouse moved while button held
- **MouseDragStart** - Drag operation started (after threshold)
- **MouseDragEnd** - Drag operation completed
- **MouseDragCancel** - Drag operation cancelled (e.g., via Escape key)
- **MouseMove** - Mouse moved without button pressed
- **MouseEnter** - Mouse entered a region
- **MouseLeave** - Mouse left a region
- **MouseScroll** - Mouse wheel scrolled

### Mouse Buttons

- **MouseButtonLeft** - Primary button
- **MouseButtonMiddle** - Middle/wheel button
- **MouseButtonRight** - Secondary button
- **MouseButtonNone** - No button (for move/release events)
- **MouseButtonWheelUp** - Scroll wheel up
- **MouseButtonWheelDown** - Scroll wheel down
- **MouseButtonWheelLeft** - Horizontal scroll left
- **MouseButtonWheelRight** - Horizontal scroll right

### Keyboard Modifiers

Mouse events can include keyboard modifiers:

- **ModShift** - Shift key held
- **ModCtrl** - Control key held
- **ModAlt** - Alt key held
- **ModMeta** - Meta/Command key held

## Basic Usage

### 1. Enable Mouse Tracking

```go
terminal, _ := tui.NewTerminal()
terminal.EnableMouseTracking()
defer terminal.DisableMouseTracking()
```

### 2. Create Mouse Handler

```go
mouse := tui.NewMouseHandler()
```

### 3. Define Mouse Regions

```go
button := &tui.MouseRegion{
    X:      10,
    Y:      5,
    Width:  20,
    Height: 3,
    ZIndex: 1,
    OnClick: func(event *tui.MouseEvent) {
        fmt.Println("Button clicked!")
    },
    OnEnter: func(event *tui.MouseEvent) {
        // Show hover state
    },
    OnLeave: func(event *tui.MouseEvent) {
        // Remove hover state
    },
}

mouse.AddRegion(button)
```

### 4. Parse and Handle Mouse Events

```go
buf := make([]byte, 128)
for {
    n, _ := os.Stdin.Read(buf)

    // Check for mouse event (ESC [ <)
    if buf[0] == 27 && n > 2 && buf[1] == '[' && buf[2] == '<' {
        event, err := tui.ParseMouseEvent(buf[2:n])
        if err == nil {
            mouse.HandleEvent(event)
        }
    }
}
```

## Advanced Features

### Double and Triple Click Detection

```go
region := &tui.MouseRegion{
    OnDoubleClick: func(event *tui.MouseEvent) {
        fmt.Println("Double-clicked!")
    },
    OnTripleClick: func(event *tui.MouseEvent) {
        fmt.Println("Triple-clicked!")
    },
}

// Configure thresholds (optional)
mouse.DoubleClickThreshold = 500 * time.Millisecond
mouse.TripleClickThreshold = 500 * time.Millisecond
```

### Drag and Drop

```go
draggable := &tui.MouseRegion{
    OnDragStart: func(event *tui.MouseEvent) {
        fmt.Println("Drag started")
    },
    OnDrag: func(event *tui.MouseEvent) {
        // Update position based on event.X, event.Y
        fmt.Printf("Dragging to (%d, %d)\n", event.X, event.Y)
    },
    OnDragEnd: func(event *tui.MouseEvent) {
        fmt.Println("Drag ended")
    },
}

// Configure drag threshold (optional)
mouse.DragStartThreshold = 5 // pixels
```

### Mouse Wheel Scrolling

```go
scrollArea := &tui.MouseRegion{
    OnScroll: func(event *tui.MouseEvent) {
        if event.DeltaY > 0 {
            fmt.Println("Scrolled down")
        } else if event.DeltaY < 0 {
            fmt.Println("Scrolled up")
        }

        // Horizontal scrolling (if supported by terminal)
        if event.DeltaX != 0 {
            fmt.Printf("Horizontal scroll: %d\n", event.DeltaX)
        }
    },
}
```

### Keyboard Modifiers

```go
region := &tui.MouseRegion{
    OnClick: func(event *tui.MouseEvent) {
        if event.Modifiers&tui.ModShift != 0 {
            fmt.Println("Shift+Click")
        }
        if event.Modifiers&tui.ModCtrl != 0 {
            fmt.Println("Ctrl+Click")
        }
        if event.Modifiers&tui.ModAlt != 0 {
            fmt.Println("Alt+Click")
        }
    },
}
```

### Z-Index Layering

Overlapping regions are hit-tested in Z-index order (highest first):

```go
background := &tui.MouseRegion{
    X:      0,
    Y:      0,
    Width:  100,
    Height: 30,
    ZIndex: 1,
    OnClick: func(event *tui.MouseEvent) {
        fmt.Println("Background clicked")
    },
}

foreground := &tui.MouseRegion{
    X:      10,
    Y:      10,
    Width:  50,
    Height: 10,
    ZIndex: 2, // Higher z-index = clicked first
    OnClick: func(event *tui.MouseEvent) {
        fmt.Println("Foreground clicked")
    },
}
```

### Pointer Capture

When a mouse button is pressed in a region, that region "captures" the pointer until release, ensuring drag operations work correctly even if the cursor moves outside the region:

```go
// Capture is automatic - no configuration needed
// All drag events go to the region where the press occurred
```

### Cancelling Drag Operations

```go
// In your event loop:
if escapeKeyPressed {
    mouse.CancelDrag()
}
```

### Debug Mode

Enable debug logging to see mouse event details (output goes to stderr, not stdout):

```go
mouse.EnableDebug()
// Debug output: [Mouse] Event: Type=2 Button=0 X=15 Y=10 Mods=0
```

This is useful during development but should be disabled in production. Debug messages are written to stderr so they won't interfere with your TUI output on stdout.

## MouseEvent Structure

```go
type MouseEvent struct {
    X, Y       int            // Cursor position
    Button     MouseButton    // Which button
    Type       MouseEventType // Event type
    Modifiers  MouseModifiers // Keyboard modifiers
    DeltaX     int            // Wheel horizontal delta
    DeltaY     int            // Wheel vertical delta
    Timestamp  time.Time      // When event occurred
    ClickCount int            // 1=single, 2=double, 3=triple
}
```

## MouseRegion Structure

```go
type MouseRegion struct {
    X, Y          int         // Position
    Width, Height int         // Size
    ZIndex        int         // Layering order
    Label         string      // Optional label for debugging
    CursorStyle   CursorStyle // Cursor hint (may not be supported)

    // Event handlers
    OnPress       func(event *MouseEvent)
    OnRelease     func(event *MouseEvent)
    OnClick       func(event *MouseEvent)
    OnDoubleClick func(event *MouseEvent)
    OnTripleClick func(event *MouseEvent)
    OnEnter       func(event *MouseEvent)
    OnLeave       func(event *MouseEvent)
    OnMove        func(event *MouseEvent)
    OnDragStart   func(event *MouseEvent)
    OnDrag        func(event *MouseEvent)
    OnDragEnd     func(event *MouseEvent)
    OnScroll      func(event *MouseEvent)

    // Legacy handlers (for backward compatibility)
    Handler      func(event *MouseEvent) // Maps to OnClick
    HoverHandler func(hovering bool)     // Maps to OnEnter/OnLeave
}
```

## Configuration Options

```go
handler := tui.NewMouseHandler()

// Double-click detection threshold
handler.DoubleClickThreshold = 500 * time.Millisecond

// Triple-click detection threshold
handler.TripleClickThreshold = 500 * time.Millisecond

// Maximum movement to still count as a click
handler.ClickMoveThreshold = 2 // pixels

// Minimum movement to start a drag
handler.DragStartThreshold = 5 // pixels
```

## Complete Example

See `examples/mouse_demo/main.go` for a comprehensive demonstration of all mouse features including:

- Click, double-click, and triple-click detection
- Drag-and-drop operations
- Hover effects (enter/leave)
- Mouse wheel scrolling
- Keyboard modifier detection
- Z-index layering

## Terminal Compatibility

Mouse support requires:

- **SGR 1006** extended mouse mode (coordinates beyond 255)
- **All mouse events** tracking mode (not just button press/release)

These are enabled automatically via:
```go
terminal.EnableMouseTracking()
```

Most modern terminals support these features:
- iTerm2
- WezTerm
- Windows Terminal
- xterm
- gnome-terminal
- etc.

## Backward Compatibility

The old `Handler` and `HoverHandler` fields are still supported for backward compatibility. They map to the new event handlers:

- `Handler(event)` → `OnClick(event)`
- `HoverHandler(true)` → `OnEnter(event)`
- `HoverHandler(false)` → `OnLeave(event)`

Legacy button constants are also supported:
- `MouseLeft` → `MouseButtonLeft`
- `MouseMiddle` → `MouseButtonMiddle`
- `MouseRight` → `MouseButtonRight`
- `MouseWheelUp` → `MouseButtonWheelUp`
- `MouseWheelDown` → `MouseButtonWheelDown`

## Best Practices

1. **Always disable mouse tracking on exit:**
   ```go
   defer terminal.DisableMouseTracking()
   ```

2. **Use Z-index for overlapping regions** to control which region receives events

3. **Configure thresholds** based on your application's needs

4. **Test on multiple terminals** as mouse support can vary

5. **Provide keyboard alternatives** for all mouse operations (accessibility)

6. **Use debug mode** during development to understand event flow (outputs to stderr):
   ```go
   mouse.EnableDebug()  // Only during development
   defer mouse.DisableDebug()
   ```

7. **Handle drag cancellation** (e.g., via Escape key) for better UX

8. **Check modifiers** to provide power-user features (Shift+Click, Ctrl+Click, etc.)

## Troubleshooting

**Mouse events not firing:**
- Ensure `EnableMouseTracking()` was called
- Check that you're in raw terminal mode
- Verify mouse event parsing code is correct

**Drag not working:**
- Check that `DragStartThreshold` isn't too high
- Ensure pointer capture is working (automatic)
- Verify drag handlers are attached to the region

**Double-click not detected:**
- Check `DoubleClickThreshold` value
- Ensure clicks are on the same region and same button
- Verify minimal movement between clicks

**Wrong region receiving events:**
- Check Z-index values (higher = front)
- Verify region bounds are correct
- Use debug mode to see event routing
