# File Picker Demo

This example demonstrates the declarative **FilePicker** view: a filterable,
keyboard-driven file browser built from a `tui.ListItem` slice bound to the
application's state.

## Features Demonstrated

- **Fuzzy filtering**: type to filter files and directories in real-time
- **Directory navigation**: press Enter on a directory (or `..`) to navigate
- **Keyboard navigation**: arrows, PageUp/PageDown, Home/End
- **Mouse support**: click to select files and folders
- **Hidden files toggle**: press F2 to show/hide dotfiles
- **File details**: status line shows the selected file's name and size

## How to Run

```bash
go run ./examples/tui/file_picker
```

## Controls

| Key/Action      | Description                              |
| --------------- | ---------------------------------------- |
| Type characters | Filter entries by name (fuzzy matching)  |
| Arrow Up/Down   | Navigate through the list                |
| Page Up/Down    | Jump through the list quickly            |
| Home / End      | Jump to top / bottom of the list         |
| Enter           | Select a file or open a directory        |
| Mouse Click     | Select with the mouse                    |
| F2              | Toggle hidden files visibility           |
| Backspace       | Delete filter characters                 |
| Esc / Ctrl+C    | Exit the demo                            |

Note: because typed letters go to the filter, the hidden-files toggle is a
function key (F2) rather than a letter.

## Code Structure

The app keeps plain Go state (`currentDir`, `files`, `filter`, `selected`)
and renders a `tui.FilePicker` bound to it:

```go
tui.FilePicker(app.files, &app.filter, &app.selected).
    CurrentPath(app.currentDir).
    Height(pickerHeight).
    OnSelect(func(item tui.ListItem, index int) {
        app.handleSelect()
    })
```

- `refreshFiles()` reads the directory, sorts directories first, prepends a
  `..` entry, and rebuilds the `ListItem` slice. Each item's `Value` carries
  the absolute path.
- `handleSelect()` stats the chosen path: directories update `currentDir`
  and refresh the listing; files report their size in the status line.
- `HandleEvent` only needs the keys the picker doesn't consume: F2 to toggle
  hidden files and Escape/Ctrl+C to quit. Navigation, filtering, and
  selection are handled by the focused FilePicker itself.

## Educational Value

This example is useful for learning:

- How to bind application state to a declarative view
- How a focused component's key handling composes with app-level
  `HandleEvent` (the app sees only unconsumed keys, plus Ctrl+C)
- Building a small multi-pane layout with `Stack`, `Spacer`, and `Text`
