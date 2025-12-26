# Inline Input for Wonton TUI

This document describes the inline input functionality in Wonton's TUI package, designed to support "inline CLI" applications that preserve terminal scrollback while providing rich input handling.

## Overview

Wonton now supports two modes of operation:

1. **Full-screen TUI** (`tui.Run()`) - Takes over terminal with alternate screen, continuous event loop
2. **Inline CLI mode** - Preserves scrollback, discrete input/output cycles, live updating regions

The inline mode is ideal for chat applications, REPL interfaces, and other CLI tools that need:
- Preserved conversation/interaction history in terminal scrollback
- Rich input features (history, autocomplete, multi-line)
- Live updating regions during async operations
- Normal terminal behavior between interactions

## Core API: `tui.Prompt()`

The `tui.Prompt()` function provides a simple, blocking way to read user input with rich editing features.

### Basic Usage

```go
input, err := tui.Prompt("> ")
if err != nil {
    if err == tui.ErrInterrupted {
        // User pressed Ctrl+C
        return
    }
    if err == tui.ErrEOF {
        // User pressed Ctrl+D on empty input
        return
    }
    // Other error
    return
}
fmt.Println("You entered:", input)
```

### With History

```go
var history []string

for {
    input, err := tui.Prompt("> ", tui.WithHistory(&history))
    if err != nil {
        break
    }
    // Process input...
}
// User can navigate history with up/down arrows
```

### With Autocomplete

```go
input, err := tui.Prompt("> ",
    tui.WithAutocomplete(func(input string, cursorPos int) ([]string, int) {
        // Find @ before cursor
        atIdx := strings.LastIndex(input[:cursorPos], "@")
        if atIdx == -1 {
            return nil, 0
        }
        prefix := input[atIdx+1 : cursorPos]

        // Return matching files
        matches := findFiles(prefix)
        return matches, atIdx
    }),
)
```

### Multi-line Input

```go
input, err := tui.Prompt("> ",
    tui.WithMultiLine(true),
)
// User can press Shift+Enter or Ctrl+J for newlines
// Regular Enter submits
```

### Complete Example

```go
var history []string

input, err := tui.Prompt(" > ",
    tui.WithHistory(&history),
    tui.WithAutocomplete(fileAutocomplete),
    tui.WithMultiLine(true),
    tui.WithPlaceholder("Type a message..."),
    tui.WithPromptStyle(tui.NewStyle().WithForeground(tui.ColorCyan)),
    tui.WithInputStyle(tui.NewStyle().WithForeground(tui.ColorWhite)),
)
```

## Available Options

### `WithHistory(*[]string)`
Enables command history navigation with up/down arrows. New entries are automatically added to the slice.

### `WithAutocomplete(AutocompleteFunc)`
Enables Tab completion. The function receives current input and cursor position, returns matches and replacement start index.

```go
type AutocompleteFunc func(input string, cursorPos int) (completions []string, replaceFrom int)
```

### `WithMultiLine(bool)`
When enabled:
- Shift+Enter or Ctrl+J inserts a newline
- Enter submits the input

### `WithPlaceholder(string)`
Shows hint text when input is empty (displayed in dim style).

### `WithValidator(func(string) error)`
Validates input before submission. Return non-nil error to prevent submission and show error message.

### `WithMaxLength(int)`
Limits the maximum input length in characters.

### `WithMask(string)`
Masks input characters (for passwords). Pass "*" to show asterisks, "" to hide completely.

### `WithPromptStyle(Style)`
Styles the prompt text.

### `WithInputStyle(Style)`
Styles the user's input text.

### `WithPromptOutput(io.Writer)`
Sets the output writer. Default is `os.Stdout`.

## Keyboard Shortcuts

During input, the following shortcuts are available:

| Key                      | Action                                                       |
| ------------------------ | ------------------------------------------------------------ |
| `Enter`                  | Submit input (or newline in multi-line mode with Shift)      |
| `Ctrl+C`                 | Cancel (returns `ErrInterrupted`)                            |
| `Ctrl+D`                 | EOF on empty input (returns `ErrEOF`), otherwise delete char |
| `Backspace`              | Delete character before cursor                               |
| `Delete`                 | Delete character at cursor                                   |
| `Left/Right Arrow`       | Move cursor                                                  |
| `Home` / `Ctrl+A`        | Move to start of line                                        |
| `End` / `Ctrl+E`         | Move to end of line                                          |
| `Ctrl+U`                 | Clear entire line                                            |
| `Ctrl+W`                 | Delete word backward                                         |
| `Ctrl+K`                 | Delete to end of line                                        |
| `Up Arrow`               | Previous history entry                                       |
| `Down Arrow`             | Next history entry                                           |
| `Tab`                    | Trigger autocomplete                                         |
| `Shift+Enter` / `Ctrl+J` | Insert newline (multi-line mode)                             |

## Autocomplete Dropdown

When autocomplete is triggered:
- Shows up to 5 matches below the input
- Use Up/Down arrows to navigate
- Press Tab or Enter to apply selection
- Press Escape to dismiss
- Any other key dismisses and processes the key

Example dropdown:
```
> @app
────────────────────
 → app.go
   app_test.go
   application.go
   ... +2 more
```

## Integration with LivePrinter

Combine `tui.Prompt()` with `LivePrinter` for a complete inline CLI experience:

```go
func chatLoop() {
    var history []string

    for {
        // Get input
        input, err := tui.Prompt(" > ",
            tui.WithHistory(&history),
            tui.WithMultiLine(true),
        )
        if err != nil {
            break
        }

        // Print user message to scrollback
        tui.Print(tui.Stack(
            tui.Text("You:").Bold(),
            tui.Padding(1, tui.Text(input)),
        ))

        // Show live streaming response
        live := tui.NewLivePrinter()
        go streamResponse(input, live)

        // Wait for completion...
        live.Stop() // Prints to scrollback
    }
}
```

## Graceful Degradation

When stdin is not a terminal (piped input, redirected from file), `Prompt()` falls back to simple line reading without rich features.

## Example Application

See `examples/inline_input/main.go` for a complete chat-like example demonstrating:
- History navigation
- File autocomplete with @ syntax
- Multi-line input
- Live streaming responses
- Integration with `LivePrinter`

Run with:
```bash
go run ./examples/inline_input
```

## Comparison to Full-Screen Mode

| Feature          | `tui.Run()` (Full-screen)   | `tui.Prompt()` (Inline) |
| ---------------- | --------------------------- | ----------------------- |
| Scrollback       | ❌ Uses alternate screen     | ✅ Preserved             |
| Event loop       | ✅ Continuous                | ❌ Discrete (per input)  |
| Input widgets    | ✅ TextInput, TextArea       | ✅ Built-in with Prompt  |
| Focus management | ✅ FocusManager              | ❌ Single input          |
| Live updates     | ✅ Re-render on state change | ✅ Via LivePrinter       |
| Mouse support    | ✅ Full                      | ❌ Not in Prompt         |
| Use case         | Full TUI apps               | Chat CLIs, REPLs        |

## Migration from Manual Input Handling

Before (manual implementation):
```go
// 170 lines of raw mode handling, cursor manipulation, etc.
func (a *App) readInput() (string, error) {
    fd := int(os.Stdin.Fd())
    oldState, err := term.MakeRaw(fd)
    // ... manual key event handling
    // ... manual autocomplete rendering
    // ... manual history navigation
    return input, nil
}
```

After (using Prompt):
```go
func (a *App) readInput() (string, error) {
    return tui.Prompt(" > ",
        tui.WithHistory(&a.history),
        tui.WithAutocomplete(a.fileAutocomplete),
        tui.WithMultiLine(true),
    )
}
```

## Architecture

The inline input system:

1. **Terminal State Management** - Temporarily enables raw mode, restores on exit
2. **Input Decoding** - Uses `terminal.KeyDecoder` for proper handling of all key events
3. **State Tracking** - Manages input buffer, cursor position, history navigation
4. **Rendering** - Uses ANSI escape codes for cursor positioning and line clearing
5. **Autocomplete** - Renders dropdown below input, cleans up on dismiss

All components reuse existing Wonton infrastructure:
- `terminal.KeyDecoder` for input parsing
- `terminal.KeyEvent` for key representation
- `LivePrinter`-style rendering for dropdown
- `Style` for text styling

## Testing

See `tui/inline_input_test.go` for unit tests covering:
- Input insertion and deletion
- History navigation
- Autocomplete triggering and application
- Word deletion
- Maximum length enforcement
- Input masking
- Validation

## Future Enhancements

Potential additions for Phase 2+:
- `tui.InlineInput` - Struct-based widget for more control
- `tui.RunInline()` - Full inline application model
- Vim mode for input editing
- Custom key bindings
- Paste handlers
- Input suggestions (distinct from autocomplete)
