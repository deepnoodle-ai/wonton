# Hyperlink Support (OSC 8)

Gooey supports clickable hyperlinks in the terminal using the OSC 8 protocol. This allows you to create links that users can click to open in their browser, directly from your TUI application.

## Overview

OSC 8 is a terminal escape sequence standard for creating hyperlinks. When supported by the terminal emulator, clicking the link will open the URL in the user's default browser. In terminals that don't support OSC 8, the escape codes are ignored and only the text is displayed.

## Terminal Support

**Supported Terminals:**
- ✅ iTerm2 (macOS)
- ✅ WezTerm (cross-platform)
- ✅ kitty (cross-platform)
- ✅ foot (Linux/Wayland)
- ✅ Rio (cross-platform)
- ✅ Hyper (cross-platform)
- ✅ Konsole (KDE)
- ✅ GNOME Terminal (recent versions)

**Not Supported:**
- ❌ Most older terminals
- ❌ Windows Terminal (as of early 2025)
- ❌ Alacritty
- ❌ Many SSH clients (though OSC 8 can pass through)

## Usage

### Basic Usage

```go
// Create a hyperlink
link := tui.NewHyperlink("https://github.com/myzie/gooey", "Gooey on GitHub")

// Render it in a frame
frame, _ := term.BeginFrame()
frame.PrintHyperlink(10, 5, link)
term.EndFrame(frame)
```

### Custom Styling

By default, hyperlinks are styled blue and underlined. You can customize the style:

```go
link := tui.NewHyperlink("https://example.com", "Example")

// Custom style
customStyle := tui.NewStyle().
    WithForeground(tui.ColorMagenta).
    WithBold().
    WithUnderline()

link = link.WithStyle(customStyle)
```

### Multiple Links

You can render multiple links on the same line or throughout your UI:

```go
linkA := tui.NewHyperlink("https://go.dev", "Go")
linkB := tui.NewHyperlink("https://github.com", "GitHub")

frame.PrintHyperlink(5, 10, linkA)
frame.PrintStyled(5 + len("Go") + 2, 10, " | ", tui.NewStyle())
frame.PrintHyperlink(5 + len("Go") + 5, 10, linkB)
```

### Fallback Format

For terminals that don't support OSC 8, or when you want to explicitly show the URL, use the fallback format:

```go
link := tui.NewHyperlink("https://example.com", "Example")

// Renders as: "Example (https://example.com)"
frame.PrintHyperlinkFallback(10, 5, link)
```

## API Reference

### Types

#### `Hyperlink` struct

```go
type Hyperlink struct {
    URL   string  // The target URL
    Text  string  // The display text
    Style Style   // Optional styling
}
```

### Functions

#### `NewHyperlink(url, text string) Hyperlink`

Creates a new hyperlink with default blue underlined styling.

**Parameters:**
- `url`: The target URL (e.g., "https://example.com")
- `text`: The display text (e.g., "Click here")

**Returns:** A new `Hyperlink` with default styling.

**Example:**
```go
link := tui.NewHyperlink("https://github.com", "GitHub")
```

#### `Hyperlink.WithStyle(style Style) Hyperlink`

Sets custom styling for the hyperlink.

**Parameters:**
- `style`: The style to apply to the link text

**Returns:** The hyperlink with updated styling.

**Example:**
```go
link := tui.NewHyperlink("https://example.com", "Example")
link = link.WithStyle(tui.NewStyle().WithForeground(tui.ColorRed))
```

#### `Hyperlink.Validate() error`

Validates that the hyperlink has valid components.

**Returns:** An error if the URL or text is empty, or if the URL is malformed.

**Example:**
```go
link := tui.NewHyperlink("https://example.com", "Example")
if err := link.Validate(); err != nil {
    log.Printf("Invalid hyperlink: %v", err)
}
```

### RenderFrame Methods

#### `PrintHyperlink(x, y int, link Hyperlink) error`

Renders a clickable hyperlink using OSC 8 protocol.

**Parameters:**
- `x, y`: Starting coordinates
- `link`: The hyperlink to render

**Returns:** An error if the hyperlink is invalid or coordinates are out of bounds.

**Example:**
```go
frame, _ := term.BeginFrame()
link := tui.NewHyperlink("https://example.com", "Example")
frame.PrintHyperlink(10, 5, link)
term.EndFrame(frame)
```

#### `PrintHyperlinkFallback(x, y int, link Hyperlink) error`

Renders a hyperlink in fallback format: "Text (URL)".

**Parameters:**
- `x, y`: Starting coordinates
- `link`: The hyperlink to render

**Returns:** An error if the hyperlink is invalid or coordinates are out of bounds.

**Example:**
```go
frame, _ := term.BeginFrame()
link := tui.NewHyperlink("https://example.com", "Example")
frame.PrintHyperlinkFallback(10, 5, link)  // Renders: "Example (https://example.com)"
term.EndFrame(frame)
```

### Low-Level Functions

#### `OSC8Start(url string) string`

Returns the OSC 8 escape sequence to start a hyperlink.

**Format:** `\033]8;;URL\033\\`

**Example:**
```go
start := tui.OSC8Start("https://example.com")
// start = "\033]8;;https://example.com\033\\"
```

#### `OSC8End() string`

Returns the OSC 8 escape sequence to end a hyperlink.

**Format:** `\033]8;;\033\\`

**Example:**
```go
end := tui.OSC8End()
// end = "\033]8;;\033\\"
```

### Formatting Methods

#### `Hyperlink.Format() string`

Returns the formatted hyperlink with OSC 8 escape codes.

**Example:**
```go
link := tui.NewHyperlink("https://example.com", "Example")
formatted := link.Format()
// formatted = "\033]8;;https://example.com\033\\Example\033]8;;\033\\"
```

#### `Hyperlink.FormatFallback() string`

Returns the fallback representation: "Text (URL)".

**Example:**
```go
link := tui.NewHyperlink("https://example.com", "Example")
fallback := link.FormatFallback()
// fallback = "Example (https://example.com)"
```

#### `Hyperlink.FormatWithOption(useOSC8 bool) string`

Returns either OSC 8 format or fallback based on the parameter.

**Example:**
```go
link := tui.NewHyperlink("https://example.com", "Example")

// OSC 8 format
osc8 := link.FormatWithOption(true)

// Fallback format
fallback := link.FormatWithOption(false)
```

## Examples

### Basic Link

```go
package main

import "github.com/deepnoodle-ai/gooey/tui"

func main() {
    term, _ := tui.NewTerminal()
    defer term.Close()

    term.EnableRawMode()
    term.EnableAlternateScreen()

    frame, _ := term.BeginFrame()

    link := tui.NewHyperlink("https://github.com", "GitHub")
    frame.PrintHyperlink(0, 0, link)

    term.EndFrame(frame)
}
```

### Link Table

```go
type Resource struct {
    Name string
    URL  string
}

resources := []Resource{
    {"Documentation", "https://pkg.go.dev"},
    {"Source Code", "https://github.com"},
    {"Issue Tracker", "https://github.com/issues"},
}

y := 5
for _, res := range resources {
    // Print name
    frame.PrintStyled(5, y, fmt.Sprintf("%-20s", res.Name), tui.NewStyle())

    // Print link
    link := tui.NewHyperlink(res.URL, res.URL)
    frame.PrintHyperlink(26, y, link)

    y++
}
```

### Styled Links

```go
// Red link
redLink := tui.NewHyperlink("https://example.com/red", "Red").
    WithStyle(tui.NewStyle().WithForeground(tui.ColorRed).WithUnderline())

// Bold link
boldLink := tui.NewHyperlink("https://example.com/bold", "Bold").
    WithStyle(tui.NewStyle().WithBold().WithUnderline())

// Custom RGB link
customLink := tui.NewHyperlink("https://example.com/custom", "Custom").
    WithStyle(tui.NewStyle().WithFgRGB(tui.NewRGB(255, 100, 200)).WithUnderline())
```

## Security Considerations

### URL Validation

Always validate URLs before creating hyperlinks, especially if they come from user input:

```go
import "net/url"

func createSafeLink(rawURL, text string) (tui.Hyperlink, error) {
    // Parse and validate URL
    parsed, err := url.Parse(rawURL)
    if err != nil {
        return tui.Hyperlink{}, err
    }

    // Only allow http(s) URLs
    if parsed.Scheme != "http" && parsed.Scheme != "https" {
        return tui.Hyperlink{}, fmt.Errorf("invalid URL scheme: %s", parsed.Scheme)
    }

    return tui.NewHyperlink(rawURL, text), nil
}
```

### Phishing Prevention

Be careful with misleading link text:

```go
// ❌ BAD: Misleading link
badLink := tui.NewHyperlink("https://evil.com", "https://google.com")

// ✅ GOOD: Clear link text
goodLink := tui.NewHyperlink("https://example.com", "Example Website")

// ✅ GOOD: Show URL explicitly
frame.PrintHyperlinkFallback(x, y, link)  // Shows "Example Website (https://example.com)"
```

## Testing

Gooey includes comprehensive tests for hyperlink functionality. See `hyperlink_test.go` for examples.

Run the tests:
```bash
go test -run Hyperlink -v
```

## Demo

A full demonstration is available in `examples/hyperlink_demo/`:

```bash
go run examples/hyperlink_demo/main.go
```

The demo shows:
- Default styled links
- Custom styled links
- Multiple links on one line
- Links with emoji
- Fallback format
- Long URLs
- Different color styles
- Table of links

## Technical Details

### OSC 8 Protocol

The OSC 8 protocol uses the following escape sequences:

**Start a hyperlink:**
```
ESC ] 8 ; ; URL ST
```
Where ST (String Terminator) is `ESC \` or BEL (`\007`).

Gooey uses `ESC \` for better compatibility.

**End a hyperlink:**
```
ESC ] 8 ; ; ST
```

**Full example:**
```
\033]8;;https://example.com\033\\Click Here\033]8;;\033\\
```

### Implementation

Hyperlinks are implemented by:
1. Wrapping text with OSC 8 escape codes
2. Applying ANSI style codes between the OSC 8 start and end
3. Using `PrintStyled` internally for rendering

The escape codes are ignored by terminals that don't support OSC 8, so the text is displayed normally.

### Performance

Hyperlink rendering has minimal performance impact:
- OSC 8 codes are small (typically < 50 bytes per link)
- No additional rendering passes required
- Same performance characteristics as styled text

## Best Practices

1. **Use descriptive text:** Make link text clear and meaningful
   ```go
   // ✅ Good
   link := tui.NewHyperlink("https://github.com/myzie/gooey", "Gooey Repository")

   // ❌ Avoid
   link := tui.NewHyperlink("https://github.com/myzie/gooey", "Click here")
   ```

2. **Style consistently:** Use consistent styling for all links
   ```go
   linkStyle := tui.NewStyle().WithForeground(tui.ColorBlue).WithUnderline()

   link1 := tui.NewHyperlink(url1, text1).WithStyle(linkStyle)
   link2 := tui.NewHyperlink(url2, text2).WithStyle(linkStyle)
   ```

3. **Validate URLs:** Always validate user-provided URLs
   ```go
   if err := link.Validate(); err != nil {
       // Handle invalid link
   }
   ```

4. **Consider fallback:** For important URLs, consider using fallback format
   ```go
   // Shows URL explicitly
   frame.PrintHyperlinkFallback(x, y, link)
   ```

5. **Test across terminals:** Test your application in both supporting and non-supporting terminals

## Future Enhancements

Potential future additions:
- Terminal capability detection (auto-detect OSC 8 support)
- Link tooltips (using OSC 8 parameters)
- Custom link IDs (for tracking/analytics)
- Link hover states (if terminal supports)

## References

- [OSC 8 Specification](https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda)
- [Terminal Support Matrix](https://github.com/Alhadis/OSC8-Adoption)
- [iTerm2 Documentation](https://iterm2.com/documentation-escape-codes.html)

## Contributing

See `documentation/world_class_features_analysis.md` for the original design discussion and `hyperlink_test.go` for test examples.
