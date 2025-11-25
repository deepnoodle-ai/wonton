package main

import (
	"fmt"
	"image"
	"os"

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
		// Quit on 'q'
		if e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

		// Pass other keys to viewer for scrolling
		app.viewer.HandleKey(e)

	case gooey.ResizeEvent:
		// Update dimensions and viewer bounds on resize
		app.width = e.Width
		app.height = e.Height
		app.viewer.SetBounds(image.Rect(0, 0, e.Width, e.Height-2))
	}

	return nil
}

// Render draws the current application state.
func (app *MarkdownDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Draw the markdown viewer
	app.viewer.Draw(frame)

	// Draw status line at bottom
	statusStyle := gooey.NewStyle().
		WithBackground(gooey.ColorBlue).
		WithForeground(gooey.ColorWhite)

	statusMsg := "Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump"
	statusLine := fmt.Sprintf(" %s ", statusMsg)

	// Pad to full width
	for len(statusLine) < width {
		statusLine += " "
	}
	if len(statusLine) > width {
		statusLine = statusLine[:width]
	}

	frame.PrintStyled(0, height-1, statusLine, statusStyle)
}

func main() {
	// Create and initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Get initial terminal size
	width, height := terminal.Size()

	// Create the application
	app := &MarkdownDemoApp{
		width:  width,
		height: height,
	}

	// Create and run the runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
