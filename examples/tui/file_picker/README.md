# File Picker Demo

This example demonstrates the **FilePicker** widget with a simple, straightforward implementation that's easy to understand and modify.

## Features Demonstrated

- **Fuzzy Filtering**: Type to filter files and directories in real-time
- **Directory Navigation**: Press Enter on folders to navigate into them
- **Keyboard Navigation**: Use arrow keys, Page Up/Down, Home/End to move through files
- **Mouse Support**: Click to select files and folders
- **Hidden Files Toggle**: Press `H` to show/hide dotfiles
- **File Details**: View file size and selection status

## How to Run

```bash
go run examples/file_picker_demo/main.go
```

## Controls

| Key/Action | Description |
|------------|-------------|
| Type characters | Filter files by name (fuzzy matching) |
| Arrow Up/Down | Navigate through the file list |
| Page Up/Down | Jump through the list quickly |
| Home | Jump to top of list |
| End | Jump to bottom of list |
| Enter | Select a file or open a directory |
| Mouse Click | Select with mouse |
| H | Toggle hidden files visibility |
| Backspace | Delete filter characters |
| Ctrl+C | Exit the demo |

## What You'll See

The interface has a simple 4-section layout:

1. **Title Bar** (top) - Shows "FILE PICKER DEMO"
2. **File List** (middle) - Interactive file browser with:
   - Filter input at the top
   - List of files/directories below
   - Current directory path shown on separator line
   - `[DIR]` markers for directories
3. **Status Bar** (bottom) - Shows:
   - Instructions and current status
   - File details when a file is selected (name and size)
   - Directory confirmations when navigating

## Code Structure

This demo uses a **simplified approach** without the composition system:

- **Direct rendering** - No containers or layout managers
- **Manual bounds** - Explicitly sets widget bounds before drawing
- **SubFrame usage** - Shows how to properly create clipped drawing areas
- **ASCII-only** - Avoids Unicode issues, works on all terminals

### Key Implementation Points

```go
// Set picker bounds (relative to its own coordinate space)
picker.SetBounds(image.Rect(0, 0, width, pickerHeight))

// Initialize picker
picker.Init()

// Create SubFrame for the picker at the correct screen position
pickerFrame := frame.SubFrame(image.Rect(0, 2, width, 2+pickerHeight))

// Draw picker into its SubFrame
picker.Draw(pickerFrame)
```

This pattern shows:
1. Widget bounds are relative to the widget (0,0 origin)
2. SubFrame positions the widget on screen (starting at row 2)
3. The widget draws into its SubFrame at (0,0) within that frame

## Why This Version?

The previous version was confusing because:
- Used Unicode emoji and symbols that didn't render on all terminals
- Complex layout with Container/VBox that obscured the basics
- No clear indication of what features were available
- Uncertainty in code comments about API usage

This simplified version:
- ✅ Uses only ASCII characters (works everywhere)
- ✅ Shows FilePicker without composition system overhead
- ✅ Clear, commented code showing each step
- ✅ Demonstrates SubFrame usage correctly
- ✅ Easy to understand and modify
- ✅ Shows all FilePicker features (filtering, navigation, hidden files)

## Educational Value

This example is great for learning:
- How to use the FilePicker widget
- How SubFrame creates clipped drawing areas
- Simple event loop patterns (goroutines + channels)
- Keyboard and mouse event handling
- Building simple TUI layouts without containers
