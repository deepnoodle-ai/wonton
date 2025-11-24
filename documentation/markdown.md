# Markdown Rendering

Gooey includes a comprehensive markdown rendering system that converts markdown text into beautifully styled terminal output with syntax highlighting, hyperlinks, and customizable themes.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Themes](#themes)
- [Widgets](#widgets)
- [Examples](#examples)
- [Advanced Usage](#advanced-usage)

## Features

The markdown renderer supports:

### Text Formatting
- **Bold** text using `**text**` or `__text__`
- *Italic* text using `*text*` or `_text_`
- `Inline code` using backticks
- Combined formatting (bold + italic)

### Structure
- Headings (H1-H6) with customizable styles
- Paragraphs with automatic word wrapping
- Horizontal rules (`---`)
- Blockquotes (coming soon)

### Lists
- Unordered lists with bullet points
- Ordered lists with numbers
- Nested lists (coming soon)

### Code
- Fenced code blocks with language specification
- Syntax highlighting via [Chroma](https://github.com/alecthomas/chroma)
- Support for 200+ programming languages
- Customizable syntax highlighting themes

### Links
- Clickable hyperlinks using OSC 8 protocol
- Graceful fallback for unsupported terminals
- Custom link styling

## Quick Start

### Basic Rendering

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/gooey"
)

func main() {
    // Create renderer
    renderer := gooey.NewMarkdownRenderer()

    // Render markdown
    markdown := "# Hello World\n\nThis is **bold** and this is *italic*."
    result, err := renderer.Render(markdown)
    if err != nil {
        panic(err)
    }

    // Access rendered lines
    for _, line := range result.Lines {
        for _, seg := range line.Segments {
            fmt.Print(seg.Text)
        }
        fmt.Println()
    }
}
```

### Using the Widget

```go
package main

import (
    "image"
    "github.com/deepnoodle-ai/gooey"
)

func main() {
    terminal, _ := gooey.NewTerminal()
    defer terminal.Close()

    // Create markdown widget
    widget := gooey.NewMarkdownWidget("# My Document\n\nContent here...")
    widget.SetBounds(image.Rect(0, 0, 80, 25))
    widget.Init()

    // Render
    frame, _ := terminal.BeginFrame()
    widget.Draw(frame)
    terminal.EndFrame(frame)
}
```

### Scrollable Viewer

```go
viewer := gooey.NewMarkdownViewer(longMarkdownContent)
viewer.SetBounds(image.Rect(0, 0, 80, 25))
viewer.Init()

// Handle scrolling with keyboard
viewer.HandleKey(gooey.KeyEvent{Key: gooey.KeyArrowDown}) // Scroll down
viewer.HandleKey(gooey.KeyEvent{Key: gooey.KeyArrowUp})   // Scroll up
viewer.HandleKey(gooey.KeyEvent{Key: gooey.KeyPageDown})  // Page down
viewer.HandleKey(gooey.KeyEvent{Key: gooey.KeyPageUp})    // Page up
viewer.HandleKey(gooey.KeyEvent{Key: gooey.KeyHome})      // Jump to top
viewer.HandleKey(gooey.KeyEvent{Key: gooey.KeyEnd})       // Jump to bottom
```

## API Reference

### MarkdownRenderer

```go
type MarkdownRenderer struct {
    Theme    MarkdownTheme
    MaxWidth int  // Maximum width for text wrapping (0 = no limit)
    TabWidth int  // Width of tab character in spaces
}
```

#### Methods

- `NewMarkdownRenderer() *MarkdownRenderer` - Creates a new renderer with default theme
- `WithTheme(theme MarkdownTheme) *MarkdownRenderer` - Sets a custom theme
- `WithMaxWidth(width int) *MarkdownRenderer` - Sets maximum width for wrapping
- `Render(markdown string) (*RenderedMarkdown, error)` - Renders markdown to styled output

### RenderedMarkdown

```go
type RenderedMarkdown struct {
    Lines []StyledLine
}
```

Represents fully rendered markdown content as a series of styled lines.

### StyledLine

```go
type StyledLine struct {
    Segments []StyledSegment
    Indent   int // Indentation level in spaces
}
```

A single line of rendered content with styled segments.

### StyledSegment

```go
type StyledSegment struct {
    Text      string
    Style     Style
    Hyperlink *Hyperlink // Optional hyperlink
}
```

A portion of text with a specific style and optional hyperlink.

## Themes

### Default Theme

```go
theme := gooey.DefaultMarkdownTheme()
// Returns a theme with good contrast and readability:
// - Cyan headings (bold + underline for H1)
// - Yellow inline code
// - Blue hyperlinks (underlined)
// - Green code blocks
// - Monokai syntax highlighting
```

### Custom Theme

```go
theme := gooey.DefaultMarkdownTheme()

// Customize heading styles
theme.H1Style = gooey.NewStyle().
    WithForeground(gooey.ColorRed).
    WithBold().
    WithUnderline()

// Customize code style
theme.CodeStyle = gooey.NewStyle().
    WithForeground(gooey.ColorMagenta)

// Customize list markers
theme.BulletChar = "→"
theme.NumberFmt = "%d) "

// Change syntax highlighting theme
theme.SyntaxTheme = "dracula" // or "monokai", "github", "solarized", etc.

// Apply theme
renderer := gooey.NewMarkdownRenderer()
renderer.WithTheme(theme)
```

### Available Theme Options

```go
type MarkdownTheme struct {
    // Heading styles by level (1-6)
    H1Style, H2Style, H3Style Style
    H4Style, H5Style, H6Style Style

    // Text styles
    BoldStyle          Style
    ItalicStyle        Style
    CodeStyle          Style // Inline code
    LinkStyle          Style
    StrikethroughStyle Style

    // Block styles
    BlockQuoteStyle Style
    CodeBlockStyle  Style // Fenced code blocks

    // List styles
    BulletChar string // e.g., "•", "→", "*"
    NumberFmt  string // e.g., "%d. ", "%d) "

    // Horizontal rules
    HorizontalRuleChar  string // e.g., "─", "-", "="
    HorizontalRuleStyle Style

    // Syntax highlighting theme name
    SyntaxTheme string // "monokai", "dracula", "github", etc.
}
```

## Widgets

### MarkdownWidget

A composable widget that renders markdown content.

```go
widget := gooey.NewMarkdownWidget(content)
widget.SetBounds(image.Rect(0, 0, 80, 25))
widget.Init()

// Update content
widget.SetContent("# New Content")

// Set custom renderer
customRenderer := gooey.NewMarkdownRenderer().WithMaxWidth(60)
widget.SetRenderer(customRenderer)

// Scrolling
widget.ScrollTo(10)      // Scroll to line 10
widget.ScrollBy(5)       // Scroll down 5 lines
widget.ScrollBy(-3)      // Scroll up 3 lines
pos := widget.GetScrollPosition()

// Get line count
lineCount := widget.GetLineCount()
```

### MarkdownViewer

A higher-level widget with built-in keyboard scrolling support.

```go
viewer := gooey.NewMarkdownViewer(content)
viewer.SetBounds(image.Rect(0, 0, 80, 25))
viewer.Init()

// The viewer handles keyboard events automatically
viewer.HandleKey(event)

// Supported keys:
// - Arrow Up/Down: Scroll one line
// - Page Up/Down: Scroll one page
// - Home: Jump to beginning
// - End: Jump to end
```

## Examples

### Complete Application

See `examples/markdown_demo/main.go` for a complete, interactive markdown viewer application.

```bash
go run examples/markdown_demo/main.go
```

### Rendering Different Elements

```go
renderer := gooey.NewMarkdownRenderer()

// Headings
result, _ := renderer.Render("# Heading 1\n## Heading 2")

// Lists
result, _ := renderer.Render(`
- Item 1
- Item 2
- Item 3
`)

// Code blocks
result, _ := renderer.Render("```go\nfunc main() {\n    println(\"Hello\")\n}\n```")

// Links
result, _ := renderer.Render("[Click here](https://example.com)")

// Complex document
markdown := `
# My Document

This is a paragraph with **bold** and *italic* text.

## Features

1. First feature
2. Second feature
3. Third feature

### Code Example

` + "```go\npackage main\n\nfunc main() {\n    println(\"Hello, World!\")\n}\n```" + `

[Learn more](https://example.com)
`
result, _ := renderer.Render(markdown)
```

## Advanced Usage

### Custom Rendering Loop

```go
renderer := gooey.NewMarkdownRenderer().WithMaxWidth(80)
result, _ := renderer.Render(markdown)

for lineNum, line := range result.Lines {
    // Apply indentation
    for i := 0; i < line.Indent; i++ {
        frame.Print(x+i, y+lineNum, " ", gooey.NewStyle())
    }

    x := line.Indent
    for _, seg := range line.Segments {
        if seg.Hyperlink != nil {
            // Render as hyperlink
            frame.PrintHyperlink(x, y+lineNum, *seg.Hyperlink)
        } else {
            // Render as styled text
            frame.PrintStyled(x, y+lineNum, seg.Text, seg.Style)
        }
        x += runewidth.StringWidth(seg.Text)
    }
}
```

### Word Wrapping

```go
// Set maximum width for wrapping
renderer := gooey.NewMarkdownRenderer()
renderer.WithMaxWidth(60) // Wrap at 60 columns

// Wrapping respects:
// - Word boundaries
// - Indentation (lists, code blocks)
// - Styled segments (maintains styling across lines)
```

### Syntax Highlighting Themes

Available themes from Chroma (https://github.com/alecthomas/chroma):

- `monokai` - Default, high contrast
- `dracula` - Dark purple theme
- `github` - Light GitHub style
- `solarized-dark` / `solarized-light` - Solarized theme
- `nord` - Cool blue theme
- `gruvbox` - Warm retro theme
- `one-dark` - Atom One Dark
- And many more...

```go
theme := gooey.DefaultMarkdownTheme()
theme.SyntaxTheme = "dracula"
renderer.WithTheme(theme)
```

### Integration with Composition System

```go
// Create a container with markdown and other widgets
container := gooey.NewContainer(gooey.NewVBoxLayout(1))

// Add header
header := gooey.NewComposableLabel("Documentation Viewer")
container.AddChild(header)

// Add markdown content
mdWidget := gooey.NewMarkdownWidget(content)
mdParams := gooey.DefaultLayoutParams()
mdParams.Grow = 1 // Take all remaining space
mdWidget.SetLayoutParams(mdParams)
container.AddChild(mdWidget)

// Add footer
footer := gooey.NewComposableLabel("Press q to quit")
container.AddChild(footer)
```

## Performance Considerations

### Rendering Large Documents

- The renderer uses a single-pass AST walker for O(n) performance
- Syntax highlighting is applied lazily only to visible code blocks
- Word wrapping uses efficient segmentation

### Memory Usage

- Rendered output is stored as lightweight styled segments
- No duplication of source text
- Hyperlink objects are only created for actual links

### Optimization Tips

1. **Limit MaxWidth**: Set a reasonable maximum width to avoid very long lines
2. **Use MarkdownViewer**: It only renders visible content
3. **Update Incrementally**: Only re-render when content changes
4. **Disable Syntax Highlighting**: For very large code blocks, use plain code block style

## Browser Compatibility

### Terminal Requirements

| Feature | Requirement | Fallback |
|---------|-------------|----------|
| Basic formatting | Any ANSI terminal | N/A |
| Syntax highlighting | 256-color support | Single color |
| Hyperlinks | OSC 8 support | URL shown in parentheses |
| Emoji | UTF-8 + wide char support | Replacement char (�) |

### Tested Terminals

- ✅ iTerm2 (macOS) - Full support
- ✅ WezTerm - Full support
- ✅ kitty - Full support
- ✅ Alacritty - Full support
- ✅ Windows Terminal - Full support
- ⚠️ Terminal.app (macOS) - No hyperlinks
- ⚠️ xterm - No hyperlinks
- ⚠️ tmux - Hyperlinks require recent version

## Limitations and Future Work

### Current Limitations

- No nested lists support
- Blockquotes styling is basic
- Tables not yet supported
- No image rendering (terminals vary widely)
- Task lists (checkboxes) not yet supported

### Planned Features

- [ ] Nested list support
- [ ] Better blockquote styling with left border
- [ ] Table rendering with column alignment
- [ ] Task list support with checkboxes
- [ ] Footnotes
- [ ] Definition lists
- [ ] Better ANSI parsing for syntax highlighting

## See Also

- [Hyperlinks Documentation](hyperlinks.md) - OSC 8 hyperlink support
- [Composition Guide](composition_guide.md) - Using widgets in layouts
- [Styling Guide](styling.md) - Text styling and colors
- [goldmark](https://github.com/yuin/goldmark) - Markdown parser used
- [chroma](https://github.com/alecthomas/chroma) - Syntax highlighter used
