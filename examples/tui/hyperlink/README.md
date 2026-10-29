# Hyperlink Demo

This demo showcases Wonton's OSC 8 hyperlink support, which enables clickable links in the terminal.

## Features Demonstrated

1. **Default Styled Links** - Blue underlined links (standard web style)
2. **Custom Styled Links** - Links with custom colors and attributes
3. **Multiple Links** - Several links on the same line
4. **Links with Emoji** - Unicode characters in link text
5. **Fallback Format** - Explicit URL display for non-supporting terminals
6. **Long URLs** - Handling of complex URLs with parameters
7. **Different Styles** - Red, green, yellow links showing color variation
8. **Table of Links** - Organized layout of multiple links

## Running the Demo

```bash
go run ./examples/tui/hyperlink
```

The demo exits when you press any key, or on its own after 30 seconds.

## Terminal Support

This demo works best in terminals that support OSC 8:

### Supported Terminals
- ✅ iTerm2 (macOS)
- ✅ WezTerm (cross-platform)
- ✅ kitty (cross-platform)
- ✅ foot (Linux/Wayland)
- ✅ Rio (cross-platform)

### Partially Supported
- ⚠️ Some terminals will show the text but links won't be clickable
- ⚠️ OSC 8 escape codes are safely ignored

## What You'll See

The demo creates a full-screen TUI with:

- A title and description at the top
- Eight different examples of hyperlink usage
- Information about terminal support at the bottom
- All examples use actual clickable links (in supporting terminals)

## Example Links

The demo includes links to:
- GitHub (main repo)
- Example.com (demonstration)
- Go programming language
- Anthropic
- Stack Overflow
- And more...

## Code Highlights

### Basic Link Creation
```go
tui.Link("https://github.com", "GitHub")
```

### Custom Styling
```go
tui.Link("https://example.com", "Example.com").Fg(tui.ColorMagenta).Bold()
```

### Fallback Format
```go
// Shows: "Go Programming (https://golang.org)"
tui.Link("https://golang.org", "Go Programming").ShowURL()
```

## Testing in Different Terminals

Try running this demo in different terminals to see how OSC 8 support varies:

1. **iTerm2** - Full support, links are clickable and styled
2. **kitty** - Full support with additional features
3. **Terminal.app** - Shows text only, escape codes ignored
4. **VS Code Terminal** - Usually shows text only

## Learn More

See the `tui.Link`, `tui.InlineLinks`, and `tui.LinkRow` documentation in the
`tui` package, and the OSC 8 encoding in `terminal/hyperlink.go`.

## Source Code

The demo source is thoroughly commented to show best practices for using hyperlinks in Wonton applications.
