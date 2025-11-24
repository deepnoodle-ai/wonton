# Claude Code Style Demo

This demo showcases a modern TUI interface similar to Claude Code, featuring:

- **Fixed input area at the bottom** - A persistent command prompt where you type
- **Scrollable message history above** - Shows conversation between you and the "assistant"
- **Clean, polished design** - Styled output with color-coded messages
- **Keyboard navigation** - Full keyboard support for input and scrolling

## Features

### Input Area
- Type your message and press Enter to send
- **Shift+Enter** to add a new line (multi-line input supported in iTerm2 and other modern terminals)
- **Alt+Enter** also adds a new line (for terminals that don't support Shift+Enter)
- Input area expands automatically as you add lines
- Backspace to delete characters
- Ctrl+C to exit
- Real-time cursor display

### Message Display
- User messages in cyan
- Assistant responses in default color
- Automatic text wrapping
- Spacing between messages

### Scrolling
- **Arrow Up/Down** - Scroll message history one line at a time
- **Page Up/Down** - Fast scroll (10 lines)
- Auto-scroll to bottom when sending new messages

## Running the Demo

```bash
go run examples/claude_style/main.go
```

## How It Works

The demo demonstrates several key Gooey library features:

1. **Frame-based rendering** - Uses `BeginFrame()`/`EndFrame()` for flicker-free updates
2. **KeyDecoder** - Reads individual keyboard events for interactive input
3. **Styled text** - Different colors for different message types
4. **Dynamic layouts** - Content area adjusts to terminal size
5. **Event loop** - Non-blocking input with continuous rendering

## Try These Commands

Once running, try typing:
- "hello" - Get a friendly greeting
- "help" - See what the assistant can help with
- "features" - Learn about Gooey library features
- "examples" - Get example commands to run

## Architecture

The demo is structured as:
- `ClaudeStyleDemo` struct holds all state
- `render()` - Draws the entire UI each frame
- `renderContent()` - Renders scrollable message history
- `renderInput()` - Renders the fixed input area at bottom
- `handleInput()` - Processes keyboard events
- `sendMessage()` - Adds messages and generates responses

This architecture can be adapted for chat clients, command interfaces, log viewers, and any application needing a fixed input with scrolling content.
