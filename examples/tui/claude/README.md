# Claude Code Style Demo

This demo showcases a modern TUI interface similar to Claude Code, featuring:

- **Fixed input area at the bottom** - A persistent command prompt where you type
- **Scrollable message history above** - Shows conversation between you and the "assistant"
- **Clean, polished design** - Styled output with color-coded messages
- **Keyboard navigation** - Full keyboard support for input, history, and scrolling

## Features

### Input Area
- Type your message and press Enter to send
- **Shift+Enter** to add a new line (multi-line input supported in iTerm2 and other modern terminals)
- **\\+Enter** (a backslash, then Enter) also adds a new line, for terminals that don't report Shift+Enter
- Input area expands automatically as you add lines
- Backspace to delete characters
- **Arrow Up/Down** - Browse previously sent messages
- Ctrl+C to exit
- Real-time cursor display

### Message Display
- User messages in cyan
- Assistant responses in default color
- Automatic text wrapping
- Spacing between messages

### Scrolling
- **Page Up/Down** - Scroll the message history one screen at a time
- **Home/End** - Jump to the oldest or newest message
- **Mouse wheel** - Scroll three lines at a time
- Auto-scroll to bottom when sending new messages

## Running the Demo

```bash
go run ./examples/tui/claude
```

## How It Works

The demo demonstrates several key Wonton library features:

1. **Declarative views** - `View()` returns the whole UI and `tui.Run` redraws only what changed
2. **Viewport** - `tui.Viewport` builds only the messages on screen and follows the newest one
3. **Styled text** - Different colors for different message types
4. **Dynamic layouts** - The message area takes whatever height the input area leaves
5. **Event handling** - `HandleEvent` receives key and mouse events from the runtime

## Try These Commands

Once running, try typing:
- "hello" - Get a friendly greeting
- "help" - See what the assistant can help with
- "features" - Learn about Wonton library features
- "examples" - Get example commands to run

## Architecture

The demo is structured as:
- `ClaudeStyleDemo` struct holds all state
- `View()` - Builds the entire UI from that state
- `Len()`/`Item()` - Feed messages to the viewport, one view per message
- `renderMessage()` - Renders a single message
- `renderInputArea()` - Renders the fixed input area at bottom
- `handleKeyEvent()`/`handleMouseEvent()` - Process input events
- `sendMessage()` - Adds messages and generates responses

This architecture can be adapted for chat clients, command interfaces, log viewers, and any application needing a fixed input with scrolling content.
