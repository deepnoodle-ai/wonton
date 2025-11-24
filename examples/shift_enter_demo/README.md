# Shift+Enter Demo

This example demonstrates how to handle Shift+Enter key combinations in text input to add newlines to the input string.

## Features

- **Shift+Enter**: Adds a newline character to the input
- **Enter**: Submits the input
- **Arrow keys**: Navigate cursor left/right through the text
- **Home/End**: Jump to start/end of input
- **Backspace/Delete**: Edit text
- **Ctrl+C or Esc**: Cancel input

## Running the Example

```bash
go run examples/shift_enter_demo/main.go
```

## How It Works

The example demonstrates custom key event handling using the `ReadKeyEvent()` method from the Input component:

1. **Key Event Detection**: Uses `input.ReadKeyEvent()` to read individual key events with modifier flags
2. **Shift Modifier Detection**: Checks `event.Shift` to determine if Shift is pressed
3. **Newline Insertion**: When Shift+Enter is detected, inserts a `\n` character into the buffer
4. **Multi-line Display**: The display function handles rendering multiple lines with proper indentation

## Code Highlights

```go
// Read key event with modifiers
event := input.ReadKeyEvent()

// Check for Shift+Enter
if event.Key == gooey.KeyEnter && event.Shift {
    // Add newline to buffer
    buffer = insertRune(buffer, cursorPos, '\n')
    cursorPos++
}
```

The `KeyEvent` struct includes:
- `Key`: Special key identifier (Enter, Tab, Arrow keys, etc.)
- `Rune`: Regular character input
- `Shift`, `Ctrl`, `Alt`: Modifier key flags

This pattern can be extended to handle any key combination, such as:
- Ctrl+S for save
- Ctrl+K for custom actions
- Alt+arrow keys for word navigation
- And more!
