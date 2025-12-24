package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

const sampleMarkdown = `# Wonton Markdown Renderer

Welcome to the **Wonton** markdown rendering demo! This showcases the rich text and markdown rendering capabilities of the Wonton TUI library.

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
    fmt.Println("Hello, Wonton!")

    // Create a terminal
    terminal, err := tui.NewTerminal()
    if err != nil {
        panic(err)
    }
    defer terminal.Restore()

    // Render markdown
    renderer := tui.NewMarkdownRenderer()
    result, _ := renderer.Render("# Hello World")
}
` + "```" + `

### Links

The renderer supports clickable hyperlinks using OSC 8:

- [Wonton on GitHub](https://github.com/deepnoodle-ai/wonton)
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

*This is a demonstration of the Wonton markdown rendering system.*
`

// MarkdownDemoApp demonstrates the declarative Markdown view.
type MarkdownDemoApp struct {
	scrollY int
}

// HandleEvent processes events from the runtime.
func (app *MarkdownDemoApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}

	return nil
}

// View returns the declarative UI for this app.
func (app *MarkdownDemoApp) View() tui.View {
	return tui.Stack(
		tui.Bordered(
			tui.Scroll(
				tui.Markdown(sampleMarkdown, nil),
				&app.scrollY,
			),
		).BorderFg(tui.ColorCyan).Title("Markdown Viewer"),
		tui.Text(" Press q to quit | ↑↓ to scroll | PgUp/PgDn for pages | Home/End to jump ").
			Bg(tui.ColorBlue).Fg(tui.ColorWhite),
	)
}

func main() {
	if err := tui.Run(&MarkdownDemoApp{}); err != nil {
		log.Fatal(err)
	}
}
