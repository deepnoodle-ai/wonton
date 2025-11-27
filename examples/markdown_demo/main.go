package main

import (
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

// MarkdownDemoApp demonstrates the declarative Markdown view.
type MarkdownDemoApp struct {
	scrollY int
	width   int
	height  int
}

// HandleEvent processes events from the runtime.
func (app *MarkdownDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}

		// Handle scrolling
		pageSize := app.height - 3
		if pageSize < 1 {
			pageSize = 1
		}

		switch e.Key {
		case gooey.KeyArrowUp:
			if app.scrollY > 0 {
				app.scrollY--
			}
		case gooey.KeyArrowDown:
			app.scrollY++
		case gooey.KeyPageUp:
			app.scrollY -= pageSize
			if app.scrollY < 0 {
				app.scrollY = 0
			}
		case gooey.KeyPageDown:
			app.scrollY += pageSize
		case gooey.KeyHome:
			app.scrollY = 0
		case gooey.KeyEnd:
			app.scrollY = 1000 // will be clamped
		}

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// View returns the declarative UI for this app.
func (app *MarkdownDemoApp) View() gooey.View {
	markdownHeight := app.height - 2
	if markdownHeight < 1 {
		markdownHeight = 1
	}

	return gooey.VStack(
		gooey.Markdown(sampleMarkdown, &app.scrollY).
			Height(markdownHeight).
			MaxWidth(app.width),
		gooey.Text(" Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump ").
			Bg(gooey.ColorBlue).Fg(gooey.ColorWhite),
	)
}

func main() {
	if err := gooey.Run(&MarkdownDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
