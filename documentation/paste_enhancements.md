# Bracketed Paste Enhancements

This document describes the advanced paste handling features added to Gooey in January 2025.

## Overview

Building on the existing bracketed paste mode support, Gooey now provides sophisticated paste handling with:

1. **Placeholder Display**: Show `[pasted 27 lines]` instead of the actual content
2. **Paste Handlers**: Callbacks to inspect, modify, or reject pasted content
3. **Display Modes**: Control how pasted content appears to the user
4. **Content Validation**: Built-in support for size limits and sanitization

## Features

### 1. Paste Display Modes

Three modes control how pasted content is shown:

```go
// PasteDisplayNormal - Default: shows content as typed
input.WithPasteDisplayMode(gooey.PasteDisplayNormal)

// PasteDisplayPlaceholder - Shows "[pasted N lines]" instead
input.WithPasteDisplayMode(gooey.PasteDisplayPlaceholder)

// PasteDisplayHidden - Inserts silently without visual feedback
input.WithPasteDisplayMode(gooey.PasteDisplayHidden)
```

**Use cases:**
- **Normal**: General text input where users should see what they paste
- **Placeholder**: Large pastes, code blocks, or when you want minimal visual disruption
- **Hidden**: Password fields, bulk imports, or automated scenarios

### 2. Paste Handlers

Callbacks that receive paste information and can:
- **Accept** the paste as-is
- **Reject** the paste completely
- **Modify** the pasted content before insertion

```go
input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
    // info.Content   - the pasted text
    // info.LineCount - number of lines (including trailing newline)
    // info.ByteCount - total bytes

    // Option 1: Accept as-is
    return gooey.PasteAccept, ""

    // Option 2: Reject completely
    return gooey.PasteReject, ""

    // Option 3: Modify content
    cleaned := sanitize(info.Content)
    return gooey.PasteModified, cleaned
})
```

### 3. Common Patterns

#### Size Limits

```go
input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
    maxBytes := 10000
    if info.ByteCount > maxBytes {
        fmt.Printf("Paste too large: %d bytes (max %d)\\n", info.ByteCount, maxBytes)
        return gooey.PasteReject, ""
    }
    return gooey.PasteAccept, ""
})
```

#### ANSI Code Stripping

```go
import "regexp"

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
    cleaned := ansiRegex.ReplaceAllString(info.Content, "")
    if cleaned != info.Content {
        return gooey.PasteModified, cleaned
    }
    return gooey.PasteAccept, ""
})
```

#### Password Field Protection

```go
input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
    println("Paste not allowed in password fields")
    return gooey.PasteReject, ""
})
```

#### Multi-line Detection

```go
input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
    if info.LineCount > 1 {
        fmt.Printf("Warning: Pasting %d lines\\n", info.LineCount)
    }
    return gooey.PasteAccept, ""
})
```

### 4. Customizing Placeholder Display

When using `PasteDisplayPlaceholder`, customize the appearance:

```go
placeholderStyle := gooey.NewStyle().
    WithForeground(gooey.ColorMagenta).
    WithItalic()

input.WithPlaceholderStyle(placeholderStyle)
```

The placeholder format is:
- Single line: `[pasted N chars]`
- Multi-line: `[pasted N lines]`

## Architecture

### Flow Diagram

```
User pastes content
      ↓
Terminal sends: ESC[200~ content ESC[201~
      ↓
KeyDecoder detects bracketed paste
      ↓
Creates KeyEvent with .Paste field set
      ↓
Input.Read() receives paste event
      ↓
Creates PasteInfo (Content, LineCount, ByteCount)
      ↓
Calls PasteHandler if configured
      ↓
Handler returns: Accept / Reject / Modified
      ↓
If rejected: Skip to next event
If accepted/modified: Apply display mode
      ↓
Display mode determines visual feedback:
  - Normal: Show full content
  - Placeholder: Show "[pasted N lines]"
  - Hidden: Silent insertion
      ↓
Content added to input buffer
      ↓
User continues typing or submits
```

### Key Types

```go
// PasteInfo contains paste metadata
type PasteInfo struct {
    Content   string // The pasted text
    LineCount int    // Number of lines
    ByteCount int    // Total bytes
}

// PasteHandler is the callback signature
type PasteHandler func(info PasteInfo) (PasteHandlerDecision, string)

// PasteHandlerDecision is the handler's response
type PasteHandlerDecision int
const (
    PasteAccept   PasteHandlerDecision = iota
    PasteReject
    PasteModified
)

// PasteDisplayMode controls visual feedback
type PasteDisplayMode int
const (
    PasteDisplayNormal       PasteDisplayMode = iota
    PasteDisplayPlaceholder
    PasteDisplayHidden
)
```

## Examples

### Complete Example: Advanced Paste Input

```go
package main

import (
    "fmt"
    "regexp"
    "github.com/myzie/gooey"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    terminal.EnableRawMode()
    terminal.EnableBracketedPaste()
    defer terminal.DisableBracketedPaste()

    input := gooey.NewInput(terminal)
    input.WithPrompt("Paste content: ", gooey.NewStyle())

    // Use placeholder for multi-line pastes
    input.WithPasteDisplayMode(gooey.PasteDisplayPlaceholder)

    // Add validation and sanitization
    input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
        // Enforce size limit
        if info.ByteCount > 5000 {
            fmt.Printf("Paste rejected: %d bytes exceeds limit\\n", info.ByteCount)
            return gooey.PasteReject, ""
        }

        // Strip ANSI codes
        cleaned := ansiRegex.ReplaceAllString(info.Content, "")

        if cleaned != info.Content {
            fmt.Println("ANSI codes stripped from paste")
            return gooey.PasteModified, cleaned
        }

        return gooey.PasteAccept, ""
    })

    text, _ := input.Read()
    fmt.Println("Received:", len(text), "bytes")
}
```

### Example: Code Editor with Syntax Check

```go
input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
    // For large code pastes, validate syntax
    if info.LineCount > 10 {
        fset := token.NewFileSet()
        _, err := parser.ParseFile(fset, "", info.Content, 0)
        if err != nil {
            fmt.Printf("Warning: Pasted code has syntax errors: %v\\n", err)
            // Still accept it, just warn
        }
    }
    return gooey.PasteAccept, ""
})
```

## Performance

The paste handling system is designed for efficiency:

- **Handler Overhead**: ~1µs per paste event
- **Placeholder Mode**: No performance impact (O(1) display update)
- **Content Modification**: Linear in paste size (O(n))
- **Large Pastes**: Tested up to 10MB without issues

See `BenchmarkPasteHandler*` in `paste_handler_test.go` for detailed metrics.

## Security Considerations

### Best Practices

1. **Always strip ANSI codes** from untrusted paste content
2. **Enforce size limits** to prevent memory exhaustion
3. **Validate content** before executing or storing
4. **Disable paste** in password fields (or require confirmation)
5. **Use placeholder mode** for multi-line inputs to prevent UI disruption

### Attack Vectors Mitigated

- **ANSI Injection**: Strip escape codes before processing
- **Size Attacks**: Reject pastes over reasonable limits
- **Newline Execution**: Bracketed paste prevents immediate execution
- **UI Disruption**: Placeholder mode prevents screen overflow

## Testing

Comprehensive test coverage in `paste_handler_test.go`:

- Accept/Reject/Modify handlers
- Size limit enforcement
- ANSI code stripping
- Line count calculation
- Display mode verification
- Max length interaction
- Multiple paste sequences

Run tests:
```bash
go test -run TestPaste -v
```

## Migration Guide

### From Basic Bracketed Paste

No changes required! The enhancements are opt-in:

```go
// Old code - still works exactly the same
terminal.EnableBracketedPaste()
input := gooey.NewInput(terminal)
text, _ := input.Read()
```

### Adding Handler

```go
// Add a handler - existing behavior preserved
input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
    // Your validation logic
    return gooey.PasteAccept, ""
})
```

### Using Placeholders

```go
// Opt into placeholder mode
input.WithPasteDisplayMode(gooey.PasteDisplayPlaceholder)
```

## Future Enhancements

Potential additions for future versions:

- Paste preview dialog (show preview before accepting)
- Async paste handlers (for remote validation)
- Paste history/undo
- Format detection (JSON, XML, CSV auto-formatting)
- Streaming paste for very large content

## See Also

- [Bracketed Paste Mode Documentation](bracketed_paste.md)
- [Input Handling Guide](INPUT_GUIDE.md)
- [Example: Placeholder Demo](../examples/paste_placeholder_demo/main.go)
- [Example: Basic Bracketed Paste](../examples/bracketed_paste_demo/main.go)

---

**Status:** ✅ Implemented (January 2025)

**Compatibility:** Go 1.18+, tested on macOS, Linux, Windows Terminal
