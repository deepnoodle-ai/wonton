# Gooey Input Handling Guide

This guide explains the different input handling approaches in Gooey and when to use each.

## Overview

Gooey provides several input handling implementations, each suited for different use cases:

1. **`input.go`** - Basic input handling with hotkeys
2. **`input_enhanced.go`** - Enhanced with better arrow key support
3. **`input_fixed.go`** - Fixed version with improved key parsing
4. **`input_interactive.go`** - Interactive input with visual feedback

## Input Variants

### Basic Input (`input.go`)

The foundation input handler with essential functionality.

**Features:**
- Basic keyboard input reading
- Hotkey registration and handling
- Simple key event processing
- Special key detection (Enter, Tab, Backspace, Escape, etc.)

**Use when:**
- You need simple keyboard input
- Building basic interactive applications
- You want minimal overhead

**Example:**
```go
input := NewInput(term)
input.SetHotkey(KeyCtrlC, func() {
    // Handle Ctrl+C
})

event, err := input.Read()
if err != nil {
    // Handle error
}
```

### Enhanced Input (`input_enhanced.go`)

Improved input handling with better special key support.

**Features:**
- All basic input features
- Improved arrow key detection
- Better escape sequence parsing
- Function key support (F1-F12)

**Use when:**
- You need arrow key navigation
- Building TUI applications with cursor movement
- You need function key support

### Fixed Input (`input_fixed.go`)

Stabilized version with bug fixes and improvements.

**Features:**
- Refined key parsing
- Better error handling
- Consistent behavior across platforms

**Use when:**
- You've encountered issues with basic input
- You need reliable, tested input handling
- Building production applications

### Interactive Input (`input_interactive.go`)

Full-featured input with visual feedback and editing.

**Features:**
- All enhanced features
- Line editing capabilities
- Visual cursor management
- Input history support
- Better integration with terminal state

**Use when:**
- Building interactive forms
- You need line editing (backspace, cursor movement in text)
- Creating CLI applications with user prompts

## Key Types

Gooey defines these special keys:

```go
const (
    KeyEnter
    KeyTab
    KeyBackspace
    KeyEscape
    KeyArrowUp
    KeyArrowDown
    KeyArrowLeft
    KeyArrowRight
    KeyHome
    KeyEnd
    KeyPageUp
    KeyPageDown
    KeyDelete
    KeyInsert
    KeyF1 - KeyF12
    KeyCtrlC
    KeyCtrlD
    // ... and more
)
```

## KeyEvent Structure

Input events are represented by:

```go
type KeyEvent struct {
    Key  Key   // Special key type (if applicable)
    Rune rune  // Character input (if printable)
    // Additional fields may vary by input implementation
}
```

## Reading Input

All input variants follow a similar pattern:

```go
// Create input handler
input := NewInput(term) // or NewEnhancedInput, etc.

// Read events
for {
    event, err := input.Read()
    if err != nil {
        // Handle error
        break
    }

    // Check for special keys
    if event.Key == KeyEnter {
        // Handle Enter
    }

    // Check for character input
    if event.Rune != 0 {
        // Handle character: event.Rune
    }
}
```

## Hotkeys

Register callbacks for specific key combinations:

```go
input.SetHotkey(KeyCtrlC, func() {
    fmt.Println("Ctrl+C pressed")
    os.Exit(0)
})

input.SetHotkey(KeyCtrlD, func() {
    fmt.Println("Ctrl+D pressed")
})
```

## Best Practices

### 1. Choose the Right Variant

- **Simple CLI tools**: Use `input.go`
- **TUI with navigation**: Use `input_enhanced.go`
- **Production apps**: Use `input_fixed.go`
- **Interactive forms**: Use `input_interactive.go`

### 2. Handle Errors

Always check for errors when reading input:

```go
event, err := input.Read()
if err != nil {
    if err == io.EOF {
        // Normal termination
        return
    }
    // Handle other errors
    log.Printf("Input error: %v", err)
    return
}
```

### 3. Register Hotkeys Early

Set up hotkeys before starting your main loop:

```go
input := NewInput(term)

// Register all hotkeys first
input.SetHotkey(KeyCtrlC, handleExit)
input.SetHotkey(KeyF1, showHelp)

// Then start event loop
for {
    event, err := input.Read()
    // ...
}
```

### 4. Use Context for Cancellation

For long-running input loops, use context for clean shutdown:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

input.SetHotkey(KeyCtrlC, cancel)

for {
    select {
    case <-ctx.Done():
        return
    default:
        event, err := input.Read()
        // Process event
    }
}
```

## Common Patterns

### Menu Navigation

```go
selectedIndex := 0
maxItems := 10

for {
    event, _ := input.Read()

    switch event.Key {
    case KeyArrowUp:
        if selectedIndex > 0 {
            selectedIndex--
        }
    case KeyArrowDown:
        if selectedIndex < maxItems-1 {
            selectedIndex++
        }
    case KeyEnter:
        // Select item
        handleSelection(selectedIndex)
        return
    case KeyEscape:
        // Cancel
        return
    }

    // Redraw menu with new selection
    drawMenu(selectedIndex)
}
```

### Text Input

```go
var buffer []rune

for {
    event, _ := input.Read()

    switch event.Key {
    case KeyEnter:
        // Submit input
        return string(buffer)
    case KeyBackspace:
        if len(buffer) > 0 {
            buffer = buffer[:len(buffer)-1]
        }
    case KeyEscape:
        // Cancel
        return ""
    default:
        if event.Rune != 0 {
            buffer = append(buffer, event.Rune)
        }
    }

    // Redraw input field
    term.PrintAt(x, y, string(buffer))
}
```

## Migration Guide

### From Basic to Enhanced

```go
// Before
input := NewInput(term)

// After
input := NewEnhancedInput(term)
// API remains the same, just better arrow key support
```

### From Enhanced to Interactive

```go
// Before
input := NewEnhancedInput(term)

// After
input := NewInteractiveInput(term)
// Gains line editing features
```

## Unified Input Architecture (v2.0+)

**Latest Update:** Gooey now features a unified input architecture with centralized key decoding!

### KeyDecoder - Unified Key Event Parsing

All input variants now use the same underlying `KeyDecoder` for consistent key handling:

- **Comprehensive escape sequence support** - Handles all ANSI CSI and SS3 sequences
- **Multi-byte UTF-8** - Proper support for emojis, Chinese, Arabic, and other Unicode characters
- **Alt/Meta modifiers** - ESC+key combinations properly decoded
- **Ctrl combinations** - All Ctrl+Letter keys supported
- **Testable** - Accepts any `io.Reader` for easy unit testing

### Testing Your Input Code

The `Input` struct now supports injecting a custom reader for testing:

```go
// In your tests
input := NewInput(term)

// Inject test data
testData := bytes.NewReader([]byte{'a', 0x1B, '[', 'A', 0x0D}) // 'a', Arrow Up, Enter
input.SetReader(testData)

// Now reads come from your test data instead of stdin
event := input.readKeyEvent()
// event will be 'a'
```

This makes it easy to write unit tests for input handling without requiring actual terminal interaction.

### Migration Notes

The unified KeyDecoder automatically handles:
- Buffered reading (only consumes bytes for one key event at a time)
- Proper EOF and error handling
- Consistent behavior across all input methods

No changes are required to existing code - all input methods now use the unified decoder internally.

## Troubleshooting

### Arrow Keys Not Working

Try using `input_enhanced.go` or `input_interactive.go` which have improved escape sequence parsing.

### Ctrl+C Not Detected

Some terminals send signals directly to the process. Register a signal handler:

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

go func() {
    <-sigChan
    // Handle interrupt
    os.Exit(0)
}()
```

### Input Blocking Forever

Ensure the terminal is in raw mode and input is available:

```go
term, err := NewTerminal()
if err != nil {
    log.Fatal(err)
}
defer term.Restore()

// Now input will work correctly
input := NewInput(term)
```

## See Also

- [CLAUDE.md](CLAUDE.md) - Project overview and architecture
- [API_RECOMMENDATIONS.md](API_RECOMMENDATIONS.md) - API design guidelines
- [examples/interactive/](examples/interactive/) - Interactive input examples
