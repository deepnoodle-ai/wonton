# Focus Management Design Document

## Overview

The focus management system provides keyboard navigation and focus state tracking for interactive TUI elements. It enables:

- **Tab/Shift+Tab navigation** between focusable elements
- **Click-to-focus** for mouse-enabled applications
- **Programmatic focus control** via commands
- **Per-instance isolation** - each Runtime/InlineApp has its own focus manager

This system is essential for building forms, multi-input interfaces, and any application where users need to navigate between interactive elements.

## Design Goals

1. **Instance isolation** - No global state; multiple TUI apps can run independently
2. **Transparent registration** - Views register automatically during render
3. **Consistent API** - Same focus behavior in Runtime and InlineApp
4. **Event-driven** - Focus commands flow through the event system
5. **Extensible** - Custom views can participate in focus navigation

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Runtime / InlineApp                             │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ FocusManager                                                │    │
│  │  - focusables map[string]Focusable                          │    │
│  │  - focusedID string                                         │    │
│  │  - order []string                                           │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                      │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ RenderContext                                               │    │
│  │  - focusMgr *FocusManager  ←─── propagated to all views     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                              │                                      │
│        ┌─────────────────────┼─────────────────────┐                │
│        ▼                     ▼                     ▼                │
│  ┌──────────┐         ┌──────────┐         ┌──────────┐             │
│  │ Button   │         │ Input    │         │ Custom   │             │
│  │ (impl    │         │ (impl    │         │ Focusable│             │
│  │ Focusable│         │ Focusable│         │ View)    │             │
│  └──────────┘         └──────────┘         └──────────┘             │
└─────────────────────────────────────────────────────────────────────┘
```

### FocusManager

`FocusManager` is the central coordinator for focus state. Each Runtime or InlineApp creates its own instance.

```go
type FocusManager struct {
    mu         sync.Mutex
    focusables map[string]Focusable  // registered elements by ID
    focusedID  string                // currently focused element ID
    order      []string              // registration order for Tab navigation
}
```

**Key behaviors:**

- **Auto-focus first element**: When the first focusable registers, it receives focus automatically
- **Focus persistence**: The `focusedID` survives Clear() calls, allowing focus to be restored across renders
- **Registration order**: Tab navigation follows the order elements are registered (render order)

### Focusable Interface

Any view can participate in focus navigation by implementing the `Focusable` interface:

```go
type Focusable interface {
    // FocusID returns a unique identifier for this element.
    FocusID() string

    // IsFocused returns whether this element currently has focus.
    IsFocused() bool

    // SetFocused is called when focus changes to/from this element.
    SetFocused(focused bool)

    // HandleKeyEvent processes keys while focused. Return true to consume.
    HandleKeyEvent(event KeyEvent) bool

    // FocusBounds returns screen bounds for click-to-focus detection.
    FocusBounds() image.Rectangle
}
```

### RenderContext Integration

The `FocusManager` flows through the view tree via `RenderContext`:

```go
type RenderContext struct {
    frame      RenderFrame
    frameCount uint64
    bounds     image.Rectangle
    focusMgr   *FocusManager  // focus manager for this render tree
}

// FocusManager returns the focus manager, or nil if none.
func (c *RenderContext) FocusManager() *FocusManager

// WithFocusManager returns a new context with the given focus manager.
func (c *RenderContext) WithFocusManager(fm *FocusManager) *RenderContext
```

The focus manager propagates automatically through `SubContext()` and `WithFrame()`.

## Render Lifecycle

Each render cycle follows this pattern:

```
1. FocusManager.Clear()           Clear registration state
        ↓
2. View.render(ctx)               Views render and register with fm
        ↓
3. Events processed               Tab/click/commands handled
        ↓
4. (repeat from 1)
```

### Registration During Render

Focusable views register themselves during render:

```go
func (b *buttonView) render(ctx *RenderContext) {
    // ... rendering logic ...

    // Register with focus manager (if available)
    bounds := ctx.AbsoluteBounds()
    if fm := ctx.FocusManager(); fm != nil {
        fm.Register(state)
    }
}
```

The `nil` check allows views to render without focus management (e.g., in tests or Print() output).

## Focus Commands and Events

Focus changes use an event-driven model to integrate properly with the event loop.

### Focus Events

```go
// FocusSetEvent sets focus to a specific element.
type FocusSetEvent struct {
    ID   string    // Element ID to focus
    Time time.Time
}

// FocusNextEvent moves focus to the next element.
type FocusNextEvent struct {
    Time time.Time
}

// FocusPrevEvent moves focus to the previous element.
type FocusPrevEvent struct {
    Time time.Time
}
```

### Focus Commands

Commands produce events that the runtime processes:

```go
// Focus returns a command that sets focus to the specified element.
func Focus(id string) Cmd {
    return func() Event {
        return FocusSetEvent{ID: id, Time: time.Now()}
    }
}

// FocusNext returns a command that moves to the next element.
func FocusNext() Cmd

// FocusPrev returns a command that moves to the previous element.
func FocusPrev() Cmd
```

### Event Processing

The Runtime/InlineApp processes focus events:

```go
func (r *Runtime) processEvent(event Event) {
    // Handle focus events from commands
    switch e := event.(type) {
    case FocusSetEvent:
        r.focusMgr.SetFocus(e.ID)
        return
    case FocusNextEvent:
        r.focusMgr.FocusNext()
        return
    case FocusPrevEvent:
        r.focusMgr.FocusPrev()
        return
    }
    // ... other event processing
}
```

## Tab Navigation

Tab and Shift+Tab are handled automatically by `FocusManager.HandleKey()`:

```go
func (fm *FocusManager) HandleKey(event KeyEvent) bool {
    if event.Key == KeyTab {
        if event.Shift {
            fm.FocusPrev()
        } else {
            fm.FocusNext()
        }
        return true
    }
    // Delegate to focused element
    // ...
}
```

Navigation wraps around:

- Tab at last element → first element
- Shift+Tab at first element → last element

## Built-in Focusable Views

These views implement `Focusable` automatically:

| View                      | Focus ID       | Key Handling                  |
| ------------------------- | -------------- | ----------------------------- |
| `Button(id, ...)`         | `id` parameter | Enter/Space triggers callback |
| `Input(id)`               | `id` parameter | Text input, cursor movement   |
| `InputField(id)`          | `id` parameter | Text input with label/border  |
| `TextArea(...).ID(id)`    | `id` parameter | Scroll navigation             |
| `Table(...).ID(id)`       | `id` parameter | Row selection                 |
| `Toggle(id, ...)`         | `id` parameter | Space toggles value           |
| `TreeView(...).ID(id)`    | `id` parameter | Expand/collapse, navigation   |
| `FilterableList(id, ...)` | `id` parameter | Selection, filtering          |
| `CheckboxList(id, ...)`   | `id` parameter | Multi-selection               |
| `RadioList(id, ...)`      | `id` parameter | Single selection              |
| `SelectList(id, ...)`     | `id` parameter | Dropdown selection            |

## Usage Examples

### Basic Form

```go
type FormApp struct {
    name    string
    email   string
    message string
}

func (app *FormApp) View() tui.View {
    return tui.Stack(
        tui.Text("Contact Form").Bold(),
        tui.Text(""),
        tui.InputField("name", &app.name).
            Label("Name").
            Placeholder("Enter your name"),
        tui.Text(""),
        tui.InputField("email", &app.email).
            Label("Email").
            Placeholder("you@example.com"),
        tui.Text(""),
        tui.TextArea(&app.message).
            ID("message").
            Size(60, 5).
            Bordered(true).
            Title("Message"),
        tui.Text(""),
        tui.Group(
            tui.Button("submit", "Submit", app.submit),
            tui.Text(" "),
            tui.Button("cancel", "Cancel", app.cancel),
        ),
        tui.Text(""),
        tui.Text("Tab to navigate, Enter to activate").Dim(),
    )
}
```

### Programmatic Focus

```go
func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
    switch e := event.(type) {
    case tui.KeyEvent:
        // Press 'n' to jump to name field
        if e.Rune == 'n' {
            return []tui.Cmd{tui.Focus("name-input")}
        }
        // Press 'e' to jump to email field
        if e.Rune == 'e' {
            return []tui.Cmd{tui.Focus("email-input")}
        }
    }
    return nil
}
```

### Custom Focusable View

Use `FocusableView` to make any view participate in focus navigation:

```go
func (app *MyApp) View() tui.View {
    return tui.Stack(
        // Wrap a scroll view to make it focusable
        tui.FocusableView(
            app.buildContentView(),
            &tui.FocusHandler{
                ID: "content-viewer",
                OnKey: func(e tui.KeyEvent) bool {
                    switch e.Key {
                    case tui.KeyArrowDown:
                        app.scrollY++
                        return true
                    case tui.KeyArrowUp:
                        if app.scrollY > 0 {
                            app.scrollY--
                        }
                        return true
                    }
                    return false
                },
            },
        ),
        tui.Button("done", "Done", app.finish),
    )
}
```

### Focus-Aware Styling

Use focus state for conditional styling:

```go
func (app *MyApp) View() tui.View {
    return tui.Stack(
        // InputField with focus-aware border
        tui.InputField("search", &app.query).
            Bordered(true).
            Border(&tui.RoundedBorder).
            FocusBorderFg(tui.ColorCyan),

        // Bordered content with focus indicator
        tui.Bordered(
            tui.Text("Content here"),
        ).
            FocusID("content-panel").
            FocusBorderFg(tui.ColorGreen),
    )
}
```

## Click-to-Focus

When mouse tracking is enabled, clicking on a focusable element focuses it:

```go
app := tui.NewInlineApp(tui.InlineAppConfig{
    MouseTracking: true,
})

// Or for Runtime:
tui.Run(app, tui.WithMouse(true))
```

The FocusManager checks click coordinates against each element's `FocusBounds()`:

```go
func (fm *FocusManager) HandleClick(x, y int) bool {
    pt := image.Pt(x, y)
    for id, f := range fm.focusables {
        if pt.In(f.FocusBounds()) {
            fm.SetFocus(id)
            return true
        }
    }
    return false
}
```

## API Reference

### FocusManager Methods

| Method                       | Description                                     |
| ---------------------------- | ----------------------------------------------- |
| `NewFocusManager()`          | Create a new focus manager instance             |
| `Clear()`                    | Clear registrations (called before each render) |
| `Register(f Focusable)`      | Register a focusable element                    |
| `GetFocused() Focusable`     | Get currently focused element                   |
| `GetFocusedID() string`      | Get ID of focused element                       |
| `SetFocus(id string)`        | Set focus to specific element                   |
| `FocusNext()`                | Move focus to next element                      |
| `FocusPrev()`                | Move focus to previous element                  |
| `HandleKey(KeyEvent) bool`   | Handle Tab navigation and delegate keys         |
| `HandleClick(x, y int) bool` | Handle click-to-focus                           |

### Focus Commands

| Command                | Description                        |
| ---------------------- | ---------------------------------- |
| `Focus(id string) Cmd` | Set focus to element with given ID |
| `FocusNext() Cmd`      | Move focus to next element         |
| `FocusPrev() Cmd`      | Move focus to previous element     |

### FocusHandler

```go
type FocusHandler struct {
    // ID is the unique identifier for this focusable element.
    ID string

    // OnKey is called when a key event occurs while focused.
    // Return true to consume the event.
    OnKey func(event KeyEvent) bool
}
```

### FocusableView

```go
// FocusableView wraps any view to make it participate in focus navigation.
func FocusableView(inner View, handler *FocusHandler) View
```

## Testing

### Unit Testing FocusManager

```go
func TestMyFocusLogic(t *testing.T) {
    fm := tui.NewFocusManager()

    // Create mock focusables
    btn1 := &mockFocusable{id: "btn1"}
    btn2 := &mockFocusable{id: "btn2"}

    fm.Register(btn1)
    fm.Register(btn2)

    // First element auto-focused
    assert.Equal(t, "btn1", fm.GetFocusedID())

    // Tab to next
    fm.FocusNext()
    assert.Equal(t, "btn2", fm.GetFocusedID())
}
```

### Testing Views with Focus

```go
func TestInputFieldFocus(t *testing.T) {
    var buf bytes.Buffer
    terminal := tui.NewTestTerminal(80, 24, &buf)
    frame, _ := terminal.BeginFrame()
    defer terminal.EndFrame(frame)

    fm := tui.NewFocusManager()
    ctx := tui.NewRenderContext(frame, 0).WithFocusManager(fm)

    value := ""
    input := tui.InputField("test-input", &value)
    input.render(ctx.SubContext(image.Rect(0, 0, 40, 3)))

    // Verify registration
    assert.Equal(t, "test-input", fm.GetFocusedID())
}
```

## Implementation Notes

### Why Instance-Based?

Global focus state caused issues:

- Multiple TUI instances (e.g., tests running in parallel) would conflict
- Embedded TUI components couldn't have independent focus
- Testing required careful state cleanup

The instance-based approach:

- Each Runtime/InlineApp owns its FocusManager
- Focus state flows through RenderContext
- No global variables to manage

### Focus Persistence Across Renders

The `focusedID` is preserved when `Clear()` is called:

```go
func (fm *FocusManager) Clear() {
    fm.order = fm.order[:0]
    for k := range fm.focusables {
        delete(fm.focusables, k)
    }
    // Note: focusedID is NOT cleared
}
```

This allows focus to persist across renders. When the previously-focused element re-registers, `SetFocused(true)` is called on it:

```go
func (fm *FocusManager) Register(f Focusable) {
    id := f.FocusID()
    fm.focusables[id] = f
    fm.order = append(fm.order, id)

    if fm.focusedID == "" {
        // Auto-focus first element
        fm.focusedID = id
        f.SetFocused(true)
    } else {
        // Restore focus if this is the focused element
        f.SetFocused(fm.focusedID == id)
    }
}
```

### Nil Focus Manager Handling

Views check for nil before using the focus manager:

```go
if fm := ctx.FocusManager(); fm != nil {
    fm.Register(state)
}
```

This allows views to render in contexts without focus management:

- `Print()` output (no focus needed)
- Tests without full runtime
- Preview rendering

### Thread Safety

FocusManager uses a mutex for all operations. However, in practice:

- All focus operations happen on the main event loop goroutine
- The mutex protects against edge cases and future-proofs the API

## Migration from Global Focus

If upgrading code that used the previous global focus manager:

**Before (global):**

```go
focusManager.SetFocus("input")
id := GetFocusedID()
```

**After (instance-based):**

```go
// In HandleEvent, use Focus command:
return []tui.Cmd{tui.Focus("input")}

// In render, use context:
if fm := ctx.FocusManager(); fm != nil {
    id := fm.GetFocusedID()
}
```
