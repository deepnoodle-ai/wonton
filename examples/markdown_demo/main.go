package main

import (
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

const sampleMarkdown = `# Gooey Markdown Renderer

Welcome to the **Gooey** markdown rendering demo! This showcases the rich text and markdown rendering capabilities of the Gooey TUI library.

## Features

The markdown renderer supports a wide range of formatting options:

### Text Formatting

- **Bold text** using double asterisks
- *Italic text* using single asterisks
- ` + "`inline code`" + ` using backticks
- Combined **_bold and italic_** text

### Lists

#### Unordered Lists

- First item
- Second item
  - Nested item (coming soon)
- Third item

#### Ordered Lists

1. First step
2. Second step
3. Third step

### Code Blocks

Here's an example of syntax-highlighted Go code:

` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello, Gooey!")

    // Create a terminal
    terminal, err := gooey.NewTerminal()
    if err != nil {
        panic(err)
    }
    defer terminal.Restore()

    // Render markdown
    renderer := gooey.NewMarkdownRenderer()
    result, _ := renderer.Render("# Hello World")
}
` + "```" + `

### Links

The renderer supports clickable hyperlinks using OSC 8:

- [Gooey on GitHub](https://github.com/deepnoodle-ai/gooey)
- [Learn more about terminals](https://en.wikipedia.org/wiki/Terminal_emulator)

---

## Themes

The markdown renderer supports customizable themes. You can change:

- Heading colors and styles (H1-H6)
- Text formatting (bold, italic, code)
- Link styling
- Code block highlighting
- List markers
- And more!

## Controls

- **Arrow Up/Down**: Scroll the document
- **Page Up/Down**: Page through the document
- **Home**: Jump to the beginning
- **End**: Jump to the end
- **q**: Quit

---

*This is a demonstration of the Gooey markdown rendering system.*
`

// MarkdownDemoApp demonstrates markdown rendering using the Runtime architecture.
// It shows how to use the MarkdownViewer widget with scrolling support.
type MarkdownDemoApp struct {
	viewer *gooey.MarkdownViewer
	width  int
	height int
}

// Init initializes the application by creating the markdown viewer.
func (app *MarkdownDemoApp) Init() error {
	app.viewer = gooey.NewMarkdownViewer(sampleMarkdown)
	// Set initial bounds (will be updated on first resize event)
	app.viewer.SetBounds(image.Rect(0, 0, app.width, app.height-2))
	app.viewer.Init()
	return nil
}

// HandleEvent processes events from the runtime.
func (app *MarkdownDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
		app.viewer.HandleKey(e)

	case gooey.ResizeEvent:
		// Update dimensions and viewer bounds on resize
		app.width = e.Width
		app.height = e.Height
		app.viewer.SetBounds(image.Rect(0, 0, e.Width, e.Height-2))
	}

	return nil
}

// View returns the declarative UI for this app.
func (app *MarkdownDemoApp) View() gooey.View {
	// Status bar style
	statusStyle := gooey.NewStyle().
		WithBackground(gooey.ColorBlue).
		WithForeground(gooey.ColorWhite)

	return gooey.VStack(
		// Markdown viewer canvas (fills available space)
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			app.viewer.Draw(frame)
		}),
		// Status line at bottom
		gooey.Text(" Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump ").Style(statusStyle),
	)
}

func main() {
	if err := gooey.Run(&MarkdownDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
