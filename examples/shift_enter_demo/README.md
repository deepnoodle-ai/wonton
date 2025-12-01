# Shift+Enter Demo

This example demonstrates how to handle Shift+Enter key combinations in text input to add newlines.

## Features

- **Shift+Enter** or **Backslash+Enter**: Adds a newline character to the input
- **Enter**: Submits the input
- **Arrow keys**: Navigate cursor left/right through the text
- **Home/End**: Jump to start/end of input
- **Backspace/Delete**: Edit text
- **Ctrl+C or Esc**: Cancel input

## How It Works

The Runtime automatically handles Shift+Enter detection using two mechanisms:

### 1. Kitty Keyboard Protocol (Modern Terminals)

At startup, the Runtime probes the terminal to detect Kitty keyboard protocol support by sending `\x1b[?u\x1b[c`. If supported, it enables the protocol with `\x1b[>1u`.

When enabled, actual Shift+Enter sends `\x1b[13;2u` which is decoded as `KeyEvent{Key: KeyEnter, Shift: true}`.

**Supported terminals:**
- Kitty
- WezTerm
- foot
- ghostty
- iTerm2 (with "Report modifiers using CSI u" enabled in Preferences → Profiles → Keys)

### 2. Backslash+Enter Fallback (All Terminals)

For terminals that don't support the Kitty protocol, type `\` then press Enter. The backslash is consumed and converted to Shift+Enter. The Runtime uses a 100ms timeout to detect this sequence, so normal typing speed works fine.

This is the same approach used by the **Gemini CLI**.

| Terminal Support | User Action | Result |
|------------------|-------------|--------|
| Kitty protocol | Shift+Enter | `KeyEvent{Shift: true}` |
| No Kitty | `\` then Enter | `KeyEvent{Shift: true}` |

## Running the Example

```bash
go run examples/shift_enter_demo/main.go
```

## Code Highlights

```go
// In HandleEvent:
case tui.KeyEnter:
    if e.Shift {
        // Shift+Enter (or backslash+Enter): Add newline
        app.buffer = insertRune(app.buffer, app.cursor, '\n')
        app.cursor++
    } else {
        // Regular Enter: Submit input
        app.result = string(app.buffer)
        app.stage = 2
    }
```

The `KeyEvent` struct includes:
- `Key`: Special key identifier (Enter, Tab, Arrow keys, etc.)
- `Rune`: Regular character input
- `Shift`, `Ctrl`, `Alt`: Modifier key flags

## Implementation Details

The implementation follows the Gemini CLI approach:

1. **terminal.go**: `DetectKittyProtocol()` probes the terminal at startup
2. **terminal.go**: `EnableEnhancedKeyboard()` / `DisableEnhancedKeyboard()` manage the protocol
3. **runtime.go**: Automatically detects and enables Kitty protocol if supported
4. **runtime.go**: `inputReader()` implements backslash+Enter buffering with 100ms timeout
5. **key_decoder.go**: Parses CSI-u sequences (`\x1b[13;2u` → Enter with Shift)
