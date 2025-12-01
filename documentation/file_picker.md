# File Picker Component

## Overview

The File Picker component (`FilePicker`) provides a navigable, filterable file selection interface for the Gooey library. It allows users to browse directories, filter files using fuzzy search, and select files or navigate directories.

## Features

- **Directory Navigation**: Navigate up and down the directory tree.
- **Filtering**: Real-time fuzzy search filtering of file names.
- **Mouse Support**: Click to select files or navigate directories. (Basic structure ready, integration pending `MouseAware` implementation).
- **Keyboard Support**: Arrow keys for navigation, Enter to select/enter directory, Typing to filter.
- **Visual Feedback**: Icons for files and directories, selection highlighting.

## Architecture

The File Picker is built using the composition system (`ComposableWidget`) and composed of two reusable sub-components:

1.  **TextInput** (`text_input.go`): A single-line text input widget for typing filters.
2.  **List** (`list.go`): A scrollable list widget for displaying files.

### File Structure

- `file_picker.go`: Main `FilePicker` component logic.
- `list.go`: Generic `List` component.
- `text_input.go`: Generic `TextInput` component.
- `examples/file_picker_demo/main.go`: Interactive demo application.

## Usage

```go
package main

import (
    "os"
    "gooey"
)

func main() {
    // ... init terminal ...

    // Create picker starting in current directory
    pwd, _ := os.Getwd()
    picker := tui.NewFilePicker(pwd)

    // Handle selection
    picker.OnSelect = func(path string) {
        // Do something with selected file
    }

    // Add to container
    container.AddChild(picker)

    // ... event loop ...
    // Delegate keys
    picker.HandleKey(event)
}
```

## Future Improvements

- **Mouse Dragging**: Implement scrollbar dragging in the list.
- **Async Loading**: Load large directories asynchronously to avoid UI freeze.
- **Preview Pane**: Show file previews for selected items.
- **Multi-select**: Allow selecting multiple files.
- **Modal Integration**: Wrap `FilePicker` in a `Modal` for popup selection.
