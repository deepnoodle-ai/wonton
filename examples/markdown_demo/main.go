package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
	"golang.org/x/term"
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

func main() {
	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Enable raw mode for key reading
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Enable alternate screen
	terminal.EnableAlternateScreen()

	// Get terminal size
	width, height := terminal.Size()

	// Create markdown viewer
	viewer := gooey.NewMarkdownViewer(sampleMarkdown)
	viewer.SetBounds(image.Rect(0, 0, width, height-2)) // Leave room for status line
	viewer.Init()

	// Create status message
	statusMsg := "Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump"

	// Create key decoder
	decoder := gooey.NewKeyDecoder(os.Stdin)

	// Main render loop
	for {
		// Begin frame
		frame, err := terminal.BeginFrame()
		if err != nil {
			break
		}

		// Clear screen
		frame.Fill(' ', gooey.NewStyle())

		// Draw the markdown viewer
		viewer.Draw(frame)

		// Draw status line at bottom
		statusStyle := gooey.NewStyle().
			WithBackground(gooey.ColorBlue).
			WithForeground(gooey.ColorWhite)

		statusLine := fmt.Sprintf(" %s ", statusMsg)
		// Pad to full width
		for len(statusLine) < width {
			statusLine += " "
		}
		frame.PrintStyled(0, height-1, statusLine[:width], statusStyle)

		// End frame
		terminal.EndFrame(frame)

		// Handle input
		key, err := decoder.ReadKeyEvent()
		if err != nil {
			break
		}

		// Handle quit
		if key.Rune == 'q' || key.Rune == 'Q' {
			return
		}

		// Let the viewer handle the key
		viewer.HandleKey(key)
	}
}
