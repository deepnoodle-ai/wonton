# Double-Buffered Rendering System

Gooey v0.2 introduces a robust double-buffered rendering system to eliminate screen flicker and improve rendering performance. This document details the architecture, usage, and best practices for using this system.

## 🚀 Overview

### The Problem: Screen Flicker
In traditional terminal applications, clearing the screen and redrawing content sequentially causes "flicker." This happens because the user sees the intermediate blank state before the new content is fully drawn.

### The Solution: Double Buffering
Double buffering uses two virtual screens in memory:

1.  **Back Buffer:** An off-screen grid where all drawing operations take place.
2.  **Front Buffer:** A record of what is currently displayed on the real terminal.

When you call `terminal.Flush()`, the engine compares the Back Buffer with the Front Buffer cell-by-cell. It then generates the minimal ANSI escape sequences required to update only the changed characters on the real screen.

**Benefits:**
*   **Zero Flicker:** The screen updates atomically.
*   **Performance:** Redundant writes are eliminated (e.g., redrawing a static border doesn't send data to the terminal).
*   **Thread Safety:** The buffers are protected by mutexes, allowing concurrent updates from different goroutines.

## 🏗 Architecture

### Core Components

#### `Cell`
The fundamental unit of the buffer.
```go
type Cell struct {
    Char  rune  // The character to display
    Style Style // The style attributes (color, bold, etc.)
}
```

#### `Terminal`
The `Terminal` struct now manages the state of the buffers.

*   **`backBuffer`**: The canvas you draw on.
*   **`frontBuffer`**: The state of the actual terminal screen.
*   **`currentStyle`**: The active style applied to subsequent `Print` calls.

## 🛠 Usage Guide

### 1. Initialization
The buffering system is enabled by default when you create a new terminal.

```go
terminal, err := gooey.NewTerminal()
if err != nil {
    panic(err)
}
defer terminal.Close() // Important: Cleans up raw mode and buffers
```

### 2. Drawing Text
Instead of using `fmt.Print` or embedding ANSI codes directly in strings, use the `Terminal` methods.

**❌ Old Way (Direct Output):**
```go
fmt.Print("\033[31mHello\033[0m World")
```

**✅ New Way (Buffered):**
```go
// Set the style for subsequent writes
terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorRed))
terminal.Print("Hello")

// Reset style (or set a new one)
terminal.Reset()
terminal.Print(" World")
```

### 3. Committing Changes (`Flush`)
Nothing appears on the screen until you call `Flush()`.

```go
terminal.Print("Loading...")
terminal.Flush() // Now "Loading..." appears on screen
```

*Note: If you use `ScreenManager`, it handles the loop and `Flush()` calls for you.*

### 4. Handling Styles
Because the buffer stores `Style` objects separate from characters, you must separate your text from your styling logic.

**Complex Example:**
```go
// Draw a box with a title
terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorBlue))
terminal.Print("┌── ")

terminal.SetStyle(gooey.NewStyle().WithBold().WithForeground(gooey.ColorWhite))
terminal.Print("My Title")

terminal.SetStyle(gooey.NewStyle().WithForeground(gooey.ColorBlue))
terminal.Println(" ──┐")
```

## ⚡ Performance Tips

1.  **Minimize Flushes:** Call `Flush()` once per "frame" or logical update, rather than after every print statement.
2.  **Use `ScreenManager`:** For complex UIs, use the `ScreenManager`. It manages regions and automatically batches updates into a consistent frame rate (e.g., 30 FPS).
3.  **Avoid Direct Writes:** Do not use `fmt.Println`, `os.Stdout.Write`, or `log.Println` while the double buffer is active. These will bypass the buffer and corrupt the screen layout.

## 🔄 Migration Guide

If you are updating older Gooey code or standard Go CLI code:

| Old Pattern | New Pattern |
| :--- | :--- |
| `fmt.Print("text")` | `terminal.Print("text")` |
| `fmt.Printf("\033[31m%s\033[0m", text)` | `t.SetStyle(Red); t.Print(text); t.Reset()` |
| `terminal.Clear()` | `terminal.Clear()` (now clears buffer, not screen immediately) |
| `terminal.MoveCursor(x, y)` | `terminal.MoveCursor(x, y)` (updates virtual cursor) |

## 🧩 Internal Logic (Diffing Algorithm)

When `Flush()` runs:

1.  Iterates through every cell `(x, y)` of the `backBuffer`.
2.  Compares `backBuffer[y][x]` with `frontBuffer[y][x]`.
3.  If they differ:
    *   Moves the real cursor to `(x, y)` (optimizing for sequential writes).
    *   Updates the terminal's active ANSI color if the style changed.
    *   Writes the character.
    *   Updates `frontBuffer` to match `backBuffer`.
4.  Restores the virtual cursor position.

This ensures that if you redraw a full-screen UI but only a clock in the corner changed, only the characters for the clock are sent to the terminal.
