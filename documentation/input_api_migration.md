# Input API Migration Guide

## Overview

The Gooey input API has been simplified and rationalized to provide a clearer, more consistent interface. This guide helps you migrate from the old input methods (now removed) to the new recommended API.

**BREAKING CHANGE**: The following methods have been removed in favor of three clear primary methods:
- ❌ `ReadLine()` - Removed
- ❌ `ReadLineEnhanced()` - Removed
- ❌ `ReadLineSimple()` - Removed
- ❌ `ReadInteractive()` - Removed
- ❌ `ReadSecure()` - Removed
- ❌ `ReadWithSuggestions()` - Removed
- ❌ `ReadBasic()` - Removed

## New API (Recommended)

The input API now consists of three primary methods:

### 1. `Read()` - Full-Featured Input
**Use for:** Interactive input with advanced features
- Arrow key navigation
- Command history (up/down arrows)
- Autocomplete suggestions (Tab key)
- Cursor editing (Home, End, Left, Right)
- Multiline support
- Custom hotkeys

**Example:**
```go
input := gooey.NewInput(terminal)
input.WithPrompt("Enter command: ", gooey.NewStyle().WithForeground(gooey.ColorCyan))
input.SetSuggestions([]string{"start", "stop", "restart", "status"})

text, err := input.Read()
if err != nil {
    // Handle error
}
```

### 2. `ReadPassword()` - Secure Password Input
**Use for:** Password entry or any sensitive input
- No echo to terminal
- Secure using `golang.org/x/term.ReadPassword()`
- Does not add to history

**Example:**
```go
input := gooey.NewInput(terminal)
input.WithPrompt("Password: ", gooey.NewStyle())

password, err := input.ReadPassword()
if err != nil {
    // Handle error
}
```

### 3. `ReadSimple()` - Basic Line Reading
**Use for:** Simple line input without special features
- Fast and lightweight
- Uses `bufio.Scanner`
- No raw mode, no key decoding
- Suitable for non-interactive scripts

**Example:**
```go
input := gooey.NewInput(terminal)
input.WithPrompt("Name: ", gooey.NewStyle())

name, err := input.ReadSimple()
if err != nil {
    // Handle error
}
```

## Migration Path

### From `ReadLine()` → `ReadSimple()`
```go
// Old (removed)
input := gooey.NewInput(terminal)
text, err := input.ReadLine()

// New (recommended)
input := gooey.NewInput(terminal)
text, err := input.ReadSimple()
```

### From `ReadLineEnhanced()` → `Read()`
```go
// Old (removed)
input := gooey.NewInput(terminal)
input.SetSuggestions(suggestions)
text, err := input.ReadLineEnhanced()

// New (recommended)
input := gooey.NewInput(terminal)
input.SetSuggestions(suggestions)
text, err := input.Read()
```

### From `ReadLineSimple()` → `Read()`
```go
// Old (removed)
input := gooey.NewInput(terminal)
input.SetSuggestions(suggestions)
text, err := input.ReadLineSimple()

// New (recommended)
input := gooey.NewInput(terminal)
input.SetSuggestions(suggestions)
text, err := input.Read()
```

### From `ReadInteractive()` → `Read()`
```go
// Old (removed)
input := gooey.NewInput(terminal)
input.WithMask('*')
input.SetSuggestions(suggestions)
text, err := input.ReadInteractive()

// New (recommended)
input := gooey.NewInput(terminal)
input.WithMask('*')
input.SetSuggestions(suggestions)
text, err := input.Read()
```

### From `ReadSecure()` → `ReadPassword()`
```go
// Old (removed)
input := gooey.NewInput(terminal)
input.WithMask('*')
password, err := input.ReadSecure()

// New (recommended)
input := gooey.NewInput(terminal)
password, err := input.ReadPassword()
```

**Note:** `ReadPassword()` doesn't use the mask character - it provides true secure input with no echo.

### From `ReadWithSuggestions()` → `Read()` with `SetSuggestions()`
```go
// Old (removed)
input := gooey.NewInput(terminal)
input.SetSuggestions(suggestions)
text, err := input.ReadWithSuggestions()

// New (recommended)
input := gooey.NewInput(terminal)
input.SetSuggestions(suggestions)
text, err := input.Read()
```

### From `ReadBasic()` → `ReadSimple()`
```go
// Old (removed)
input := gooey.NewInput(terminal)
text, err := input.ReadBasic()

// New (recommended)
input := gooey.NewInput(terminal)
text, err := input.ReadSimple()
```

## Why These Changes?

### Problem: API Proliferation
The previous API had 8 different read methods with unclear differences:
- `Read()`, `ReadLine()`, `ReadLineEnhanced()`, `ReadLineSimple()`
- `ReadInteractive()`, `ReadSecure()`, `ReadWithSuggestions()`, `ReadBasic()`

### Solution: Rationalized API
The new API provides:
1. **Clarity** - Each method has a distinct, clear purpose
2. **Consistency** - All methods use unified `KeyDecoder` for key handling
3. **Simplicity** - Three methods cover all use cases
4. **Better naming** - `ReadPassword()` is clearer than `ReadSecure()`

## Implementation Details

### Unified KeyDecoder
All input methods now use the same `KeyDecoder` for consistent key handling:
- Proper ANSI escape sequence parsing
- UTF-8 multi-byte character support
- Modifier key detection (Ctrl, Alt)
- Comprehensive special key support (arrows, function keys, etc.)

### Raw Mode Handling
- `Read()` - Uses terminal raw mode for key-by-key input
- `ReadPassword()` - Uses `golang.org/x/term.ReadPassword()` for secure input
- `ReadSimple()` - No raw mode, uses standard `bufio.Scanner`

### Feature Comparison

| Feature | `Read()` | `ReadPassword()` | `ReadSimple()` |
|---------|----------|------------------|----------------|
| Arrow keys | ✓ | ✗ | ✗ |
| History | ✓ | ✗ | ✓ |
| Suggestions | ✓ | ✗ | ✗ |
| Autocomplete | ✓ | ✗ | ✗ |
| Cursor editing | ✓ | ✗ | ✗ |
| Multiline | ✓ | ✗ | ✗ |
| Hotkeys | ✓ | ✗ | ✗ |
| Masking | ✓ | N/A | ✗ |
| No echo | ✗ | ✓ | ✗ |
| Raw mode | ✓ | ✓ | ✗ |

## Breaking Change Notice

**This is a breaking change.** The old input methods have been completely removed from the codebase:

- Files removed: `input_enhanced.go`, `input_fixed.go`, `input_interactive.go`, `input_simple.go`
- Methods removed: `ReadLine()`, `ReadLineEnhanced()`, `ReadLineSimple()`, `ReadInteractive()`, `ReadSecure()`, `ReadWithSuggestions()`, `ReadBasic()`

If your code uses any of these methods, you must update to the new API before upgrading. Use the migration examples above to update your code.

## Questions?

If you have questions or encounter issues during migration, please:
1. Check the examples in `examples/` directory
2. Review the godoc comments for each method
3. See `documentation/INPUT_GUIDE.md` for comprehensive input handling documentation
4. File an issue on GitHub
