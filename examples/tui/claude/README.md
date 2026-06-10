# Claude Code Style Demo

This demo showcases a modern TUI interface similar to Claude Code, featuring:

- **Fixed input area at the bottom** - A persistent command prompt where you type
- **Scrollable message history above** - Shows conversation between you and the "assistant"
- **Clean, polished design** - Styled output with color-coded messages
- **Keyboard navigation** - Full keyboard support for input, history, and scrolling

## Features

### Input Area
- Type your message and press Enter to send
- **Shift+Enter** to add a new line (in terminals supporting the Kitty keyboard protocol)
- **\ then Enter** also adds a new line (fallback for other terminals)
- Input area expands automatically as you add lines
- Backspace to delete characters
- **Arrow Up/Down** - Browse command history
- Ctrl+C to exit
- Real-time cursor display

### Message Display
- User messages in cyan
- Assistant responses in default color
- Automatic text wrapping
- Spacing between messages

### Scrolling
- **Page Up/Down** - Scroll message history (10 lines)
- **Mouse wheel** - Scroll message history (3 lines)
- Auto-scroll to bottom when sending new messages

## Running the Demo

```bash
go run ./examples/tui/claude
```

## How It Works

The demo demonstrates several key Wonton library features:

1. **Declarative views** - `View()` describes the UI; the runtime diffs and renders it
2. **Event handling** - `HandleEvent()` processes keyboard and mouse events
3. **Scroll views** - `tui.Scroll(...).Bottom()` anchors chat history to the bottom
4. **Styled text** - Different colors for different message types
5. **Dynamic layouts** - The footer grows as the multi-line input expands

## Try These Commands

Once running, try typing:
- "hello" - Get a friendly greeting
- "help" - See what the assistant can help with
- "features" - Learn about Wonton library features
- "examples" - Get example commands to run

## Architecture

The demo is structured as:
- `ClaudeStyleDemo` struct holds all state
- `View()` - Returns the declarative view tree each frame
- `renderMessages()` - Builds the scrollable message history view
- `renderInputArea()` - Builds the fixed input area at the bottom
- `handleKeyEvent()` / `handleMouseEvent()` - Process input events
- `sendMessage()` - Adds messages and generates responses

This architecture can be adapted for chat clients, command interfaces, log viewers, and any application needing a fixed input with scrolling content.
