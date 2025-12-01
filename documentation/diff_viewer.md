# Diff Viewer

Gooey includes a powerful diff viewer component that displays file diffs with syntax highlighting, line numbers, and customizable color schemes. Perfect for building git diff viewers, code review tools, or any application that needs to display changes between files.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [API Reference](#api-reference)
- [Themes](#themes)
- [Examples](#examples)
- [Advanced Usage](#advanced-usage)

## Features

### Core Features
- **Unified Diff Parsing** - Parse standard unified diff format (git diff, svn diff, etc.)
- **Syntax Highlighting** - Full syntax highlighting for 200+ languages via Chroma
- **Line Numbers** - Side-by-side old/new line numbers with proper alignment
- **Color Coding** - Red for removed lines, green for added lines
- **Scrolling** - Smooth keyboard-driven scrolling through large diffs
- **Multiple Files** - Support for multi-file diffs
- **Multiple Hunks** - Handle diffs with multiple change blocks per file

### Display Features
- Customizable themes for diff colors
- Configurable line number width
- Optional syntax highlighting (can be disabled)
- Optional line numbers (can be hidden)
- Background colors for added/removed lines
- File headers with paths

## Quick Start

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/gooey/tui"
)

func main() {
    diffText := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,2 +1,2 @@
-old line
+new line`

    // Create diff viewer
    viewer, err := tui.NewDiffViewer(diffText, "go")
    if err != nil {
        panic(err)
    }

    // Set bounds and initialize
    viewer.SetBounds(image.Rect(0, 0, 80, 25))
    viewer.Init()

    // Render
    terminal, _ := tui.NewTerminal()
    frame, _ := terminal.BeginFrame()
    viewer.Draw(frame)
    terminal.EndFrame(frame)
}
```

### Complete Application

See `examples/diff_demo/main.go` for a full interactive diff viewer with scrolling.

```bash
go run examples/diff_demo/main.go
```

## API Reference

### DiffViewer

Main widget for displaying diffs.

```go
type DiffViewer struct {
    // Embedded BaseWidget for composition support
    BaseWidget
}
```

#### Constructor

```go
// Create from diff text
viewer, err := NewDiffViewer(diffText, "go")

// Create from parsed Diff object
diff, _ := ParseUnifiedDiff(diffText)
viewer := NewDiffViewerFromDiff(diff, "python")
```

#### Methods

**Configuration:**
- `SetLanguage(language string)` - Set programming language for syntax highlighting
- `SetTheme(theme DiffTheme)` - Set custom color theme
- `SetRenderer(renderer *DiffRenderer)` - Set custom renderer

**Scrolling:**
- `ScrollTo(line int)` - Scroll to specific line number
- `ScrollBy(delta int)` - Scroll by number of lines (positive = down, negative = up)
- `GetScrollPosition() int` - Get current scroll position
- `CanScrollUp() bool` - Check if can scroll up
- `CanScrollDown() bool` - Check if can scroll down

**Content:**
- `SetDiff(diff *Diff)` - Update diff content
- `SetDiffText(diffText string) error` - Update from diff text
- `GetDiff() *Diff` - Get underlying Diff object
- `GetLineCount() int` - Get total number of rendered lines

**Widget Lifecycle:**
- `Init()` - Initialize widget
- `Destroy()` - Clean up widget
- `HandleKey(event KeyEvent) bool` - Handle keyboard input

### Diff Parser

Parse unified diff format into structured data.

```go
diff, err := ParseUnifiedDiff(diffText)
// Returns *Diff with parsed files, hunks, and lines
```

### Diff Structure

```go
type Diff struct {
    Files []DiffFile
}

type DiffFile struct {
    OldPath string    // Path before change
    NewPath string    // Path after change
    Hunks   []DiffHunk
}

type DiffHunk struct {
    OldStart int    // Starting line in old file
    OldCount int    // Number of lines in old file
    NewStart int    // Starting line in new file
    NewCount int    // Number of lines in new file
    Header   string // @@ line
    Lines    []DiffLine
}

type DiffLine struct {
    Type       DiffLineType // Context, Added, Removed, Header, HunkHeader
    OldLineNum int          // Line number in old file (0 if added)
    NewLineNum int          // Line number in new file (0 if removed)
    Content    string       // Line content without +/- marker
    RawLine    string       // Original line including marker
}
```

### DiffRenderer

Lower-level renderer for custom rendering needs.

```go
renderer := NewDiffRenderer()
renderer.ShowLineNums = true
renderer.SyntaxHighlight = true
renderer.LineNumWidth = 5

renderedLines := renderer.RenderDiff(diff, "go")
```

## Themes

### Default Theme

```go
theme := tui.DefaultDiffTheme()
// Dark green background for added lines
// Dark red background for removed lines
// Cyan for file headers
// Blue for hunk headers
// Monokai syntax highlighting
```

### Custom Theme

```go
theme := tui.DiffTheme{
    AddedBg:      tui.RGB{R: 0, G: 100, B: 0},     // Dark green
    AddedFg:      tui.RGB{R: 150, G: 255, B: 150}, // Light green
    RemovedBg:    tui.RGB{R: 100, G: 0, B: 0},     // Dark red
    RemovedFg:    tui.RGB{R: 255, G: 150, B: 150}, // Light red
    ContextStyle: tui.NewStyle().WithForeground(tui.ColorWhite),
    HeaderStyle:  tui.NewStyle().WithForeground(tui.ColorCyan).WithBold(),
    HunkStyle:    tui.NewStyle().WithForeground(tui.ColorBlue).WithBold(),
    LineNumStyle: tui.NewStyle().WithForeground(tui.ColorBrightBlack),
    SyntaxTheme:  "dracula", // Any Chroma theme
}

viewer.SetTheme(theme)
```

### Available Syntax Themes

All Chroma themes are supported:
- `monokai` (default)
- `dracula`
- `github`
- `solarized-dark` / `solarized-light`
- `nord`
- `gruvbox`
- `one-dark`
- And 40+ more...

## Examples

### Basic Diff Viewing

```go
diffText := `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,3 +1,3 @@
 package main
-import "fmt"
+import "log"`

viewer, _ := tui.NewDiffViewer(diffText, "go")
viewer.SetBounds(image.Rect(0, 0, 80, 25))
viewer.Init()
```

### Scrolling Through Large Diffs

```go
viewer, _ := tui.NewDiffViewer(largeDiffText, "python")
viewer.SetBounds(image.Rect(0, 0, 80, 25))
viewer.Init()

// Keyboard navigation
viewer.HandleKey(tui.KeyEvent{Key: tui.KeyArrowDown}) // Scroll down 1 line
viewer.HandleKey(tui.KeyEvent{Key: tui.KeyArrowUp})   // Scroll up 1 line
viewer.HandleKey(tui.KeyEvent{Key: tui.KeyPageDown})  // Scroll down 1 page
viewer.HandleKey(tui.KeyEvent{Key: tui.KeyPageUp})    // Scroll up 1 page
viewer.HandleKey(tui.KeyEvent{Key: tui.KeyHome})      // Jump to top
viewer.HandleKey(tui.KeyEvent{Key: tui.KeyEnd})       // Jump to bottom

// Programmatic scrolling
viewer.ScrollTo(50)   // Scroll to line 50
viewer.ScrollBy(10)   // Scroll down 10 lines
viewer.ScrollBy(-5)   // Scroll up 5 lines
```

### Disable Syntax Highlighting

For very large diffs or when syntax highlighting isn't needed:

```go
viewer, _ := tui.NewDiffViewer(diffText, "")
// Empty language string disables syntax highlighting

// Or with custom renderer
renderer := tui.NewDiffRenderer()
renderer.SyntaxHighlight = false
viewer.SetRenderer(renderer)
```

### Hide Line Numbers

```go
renderer := tui.NewDiffRenderer()
renderer.ShowLineNums = false
viewer.SetRenderer(renderer)
```

### Git Integration

```go
import (
    "os/exec"
    "github.com/deepnoodle-ai/gooey/tui"
)

// Get diff from git
cmd := exec.Command("git", "diff", "HEAD~1", "HEAD")
diffBytes, err := cmd.Output()
if err != nil {
    panic(err)
}

// Display in viewer
viewer, err := tui.NewDiffViewer(string(diffBytes), "go")
if err != nil {
    panic(err)
}
```

### Multiple File Diffs

The viewer automatically handles diffs with multiple files:

```go
multiFileDiff := `diff --git a/file1.go b/file1.go
--- a/file1.go
+++ b/file1.go
@@ -1,1 +1,1 @@
-old
+new
diff --git a/file2.py b/file2.py
--- a/file2.py
+++ b/file2.py
@@ -1,1 +1,1 @@
-old
+new`

viewer, _ := tui.NewDiffViewer(multiFileDiff, "auto")
// Will show both files with proper headers
```

### Custom Rendering

For advanced use cases, use the renderer directly:

```go
diff, _ := tui.ParseUnifiedDiff(diffText)
renderer := tui.NewDiffRenderer()

renderedLines := renderer.RenderDiff(diff, "go")

for _, line := range renderedLines {
    // line.LineNumOld - old line number
    // line.LineNumNew - new line number
    // line.Segments   - styled text segments
    // line.BgColor    - background color (if any)

    // Custom rendering logic here
}
```

## Advanced Usage

### Integration with Composition System

```go
// Create a container with diff viewer and other widgets
container := tui.NewContainer(tui.NewVBoxLayout(1))

// Add header
header := tui.NewComposableLabel("Git Diff Viewer")
container.AddChild(header)

// Add diff viewer
viewer, _ := tui.NewDiffViewer(diffText, "go")
viewerParams := tui.DefaultLayoutParams()
viewerParams.Grow = 1 // Take all remaining space
viewer.SetLayoutParams(viewerParams)
container.AddChild(viewer)

// Add footer with stats
stats := fmt.Sprintf("Files: %d", len(viewer.GetDiff().Files))
footer := tui.NewComposableLabel(stats)
container.AddChild(footer)
```

### Language Detection

```go
func getLanguageFromPath(path string) string {
    switch {
    case strings.HasSuffix(path, ".go"):
        return "go"
    case strings.HasSuffix(path, ".py"):
        return "python"
    case strings.HasSuffix(path, ".js"):
        return "javascript"
    case strings.HasSuffix(path, ".rs"):
        return "rust"
    default:
        return ""
    }
}

diff, _ := tui.ParseUnifiedDiff(diffText)
language := getLanguageFromPath(diff.Files[0].NewPath)
viewer := tui.NewDiffViewerFromDiff(diff, language)
```

### Dynamic Diff Updates

```go
viewer, _ := tui.NewDiffViewer(initialDiff, "go")

// Later, update the diff
newDiffText := getNewDiff()
err := viewer.SetDiffText(newDiffText)
if err != nil {
    log.Printf("Error updating diff: %v", err)
}
// Viewer will re-render automatically
```

### Performance Optimization

For very large diffs:

```go
renderer := tui.NewDiffRenderer()

// Disable syntax highlighting for speed
renderer.SyntaxHighlight = false

// Reduce line number width to save space
renderer.LineNumWidth = 4

viewer.SetRenderer(renderer)
```

### Keyboard Shortcuts

The default HandleKey implementation supports:

| Key | Action |
|-----|--------|
| ↑ | Scroll up one line |
| ↓ | Scroll down one line |
| Page Up | Scroll up one page |
| Page Down | Scroll down one page |
| Home | Jump to beginning |
| End | Jump to end |

To add custom shortcuts:

```go
handled := viewer.HandleKey(event)
if !handled {
    // Add custom handling
    if event.Rune == 'j' {
        viewer.ScrollBy(1)
        return true
    }
    if event.Rune == 'k' {
        viewer.ScrollBy(-1)
        return true
    }
}
```

## Unified Diff Format

The parser supports standard unified diff format:

```
diff --git a/file.txt b/file.txt
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,3 @@
 context line
-removed line
+added line
```

### Format Details

- **File Header:** `diff --git a/path b/path`
- **Old File:** `--- a/path` or `--- path`
- **New File:** `+++ b/path` or `+++ path`
- **Hunk Header:** `@@ -oldStart,oldCount +newStart,newCount @@`
- **Context Line:** Starts with space ` ` or is empty
- **Removed Line:** Starts with minus `-`
- **Added Line:** Starts with plus `+`

### Supported Formats

- ✅ Git diffs (`git diff`)
- ✅ SVN diffs (`svn diff`)
- ✅ Patch files (`.patch`, `.diff`)
- ✅ GitHub/GitLab diff URLs (after downloading)
- ✅ Multiple files in single diff
- ✅ Multiple hunks per file

### Limitations

- ⚠️ Context diffs (not unified) are not supported
- ⚠️ Binary file diffs show as-is (no special handling)
- ⚠️ Merge conflicts markers are not parsed specially

## Performance Considerations

### Memory Usage

- Diff is parsed once and cached
- Rendered output is cached until diff changes
- Syntax highlighting is done on visible lines only (in viewport)

### Rendering Performance

- Only visible lines are rendered (virtualized scrolling)
- Syntax highlighting can be disabled for large files
- Line numbers use fixed-width formatting

### Optimization Tips

1. **Large Diffs:** Disable syntax highlighting
2. **Many Files:** Consider showing one file at a time
3. **Network Diffs:** Parse on background thread, display when ready

## Troubleshooting

### Diff Not Parsing

Check that your diff is in unified format:
```go
diff, err := tui.ParseUnifiedDiff(diffText)
if err != nil {
    log.Printf("Parse error: %v", err)
}
```

### Syntax Highlighting Not Working

1. Check language is supported: `lexers.Get(language) != nil`
2. Verify language name is correct (e.g., "go" not "golang")
3. Check if highlighting is enabled: `renderer.SyntaxHighlight == true`

### Colors Look Wrong

1. Verify terminal supports RGB colors
2. Try different syntax theme
3. Check custom theme values are valid RGB

### Line Numbers Misaligned

1. Adjust `LineNumWidth` for longer files
2. Check terminal font is monospace

## See Also

- [Markdown Renderer](markdown.md) - Render markdown with syntax highlighting
- [Composition Guide](composition_guide.md) - Using widgets in layouts
- [Styling Guide](styling.md) - Text styling and colors
- [Chroma](https://github.com/alecthomas/chroma) - Syntax highlighting library
