# Enhanced Progress Bar Demo

This example demonstrates the enhanced features of the Progress component in the wonton TUI package.

## New Features

### 1. Empty Pattern Support

Instead of using a single character for the unfilled portion, you can now use repeating patterns:

```go
// Use a repeating dot-dash pattern
Progress(50, 100).EmptyPattern("·-")

// Use a repeating block pattern
Progress(30, 100).EmptyPattern("░▒")

// Use custom unicode patterns
Progress(75, 100).EmptyPattern("●○")
```

The pattern will repeat to fill the entire unfilled portion of the progress bar.

### 2. Percentage Styling

You can now customize the style of the percentage text independently from the progress bar:

```go
// Set percentage text to a different color
Progress(50, 100).PercentFg(tui.ColorMagenta)

// Use a complete custom style for the percentage
customStyle := tui.NewStyle().WithForeground(tui.ColorCyan).WithBold()
Progress(50, 100).PercentStyle(customStyle)
```

### 3. Existing Features (Enhanced)

The Progress component also supports these existing features:

- **ShowPercent()** / **HidePercent()**: Control percentage display (percentage is shown by default)
- **ShowFraction()**: Show "current/total" instead of percentage
- **EmptyChar()**: Set a single character for unfilled portion (clears any pattern)
- **EmptyFg()**: Set color for the unfilled portion
- **FilledChar()**: Set character for filled portion
- **Fg()**: Set color for filled portion
- **Width()**: Set the width of the progress bar
- **Label()**: Add a label before the progress bar

## Example Usage

```go
// Standard progress with percentage
Progress(50, 100).Width(40).Fg(tui.ColorGreen)

// Progress with custom empty pattern
Progress(75, 100).
    Width(40).
    Fg(tui.ColorCyan).
    EmptyPattern("·-").
    EmptyFg(tui.ColorBrightBlack)

// Progress with custom percentage color
Progress(30, 100).
    Width(40).
    Fg(tui.ColorGreen).
    PercentFg(tui.ColorMagenta)

// Progress without percentage
Progress(90, 100).Width(40).Fg(tui.ColorBlue).HidePercent()

// Progress with fraction display
Progress(45, 100).Width(40).Fg(tui.ColorRed).ShowFraction()
```

## Running the Demo

```bash
cd examples/tui/progress_enhanced
go run main.go
```

Press 'q' to quit the demo.
