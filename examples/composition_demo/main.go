package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// Simple composition demo showing nested layouts with buttons and labels.
type App struct {
	terminal   *gooey.Terminal
	root       *gooey.Container
	counter    int
	counterLbl *gooey.ComposableLabel
}

func NewApp(terminal *gooey.Terminal) *App {
	app := &App{terminal: terminal}
	width, height := terminal.Size()

	// Root container with vertical layout - stretch makes children fill width
	app.root = gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1).WithAlignment(gooey.LayoutAlignStretch),
		&gooey.RoundedBorder,
	)

	// Title
	title := gooey.NewComposableLabel("Composition Demo")
	title.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())
	title.WithAlign(gooey.AlignCenter)

	// Counter display
	app.counterLbl = gooey.NewComposableLabel("Counter: 0")
	app.counterLbl.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Horizontal button row
	buttonRow := gooey.NewContainer(gooey.NewHBoxLayout(2))

	incBtn := gooey.NewComposableButton(" + ", func() {
		app.counter++
		app.counterLbl.SetText(fmt.Sprintf("Counter: %d", app.counter))
	})
	incBtn.Style = gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack)

	decBtn := gooey.NewComposableButton(" - ", func() {
		app.counter--
		app.counterLbl.SetText(fmt.Sprintf("Counter: %d", app.counter))
	})
	decBtn.Style = gooey.NewStyle().WithBackground(gooey.ColorRed).WithForeground(gooey.ColorWhite)

	resetBtn := gooey.NewComposableButton("Reset", func() {
		app.counter = 0
		app.counterLbl.SetText("Counter: 0")
	})

	buttonRow.AddChild(incBtn)
	buttonRow.AddChild(decBtn)
	buttonRow.AddChild(resetBtn)

	// Nested container with its own border
	nested := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(0),
		&gooey.SingleBorder,
	)
	nestedParams := gooey.DefaultLayoutParams()
	nestedParams.MarginTop = 1
	nested.SetLayoutParams(nestedParams)

	nestedTitle := gooey.NewComposableLabel("Nested VBox")
	nestedTitle.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorMagenta))

	nestedItem1 := gooey.NewComposableLabel("Item 1")
	nestedItem2 := gooey.NewComposableLabel("Item 2")
	nestedItem3 := gooey.NewComposableLabel("Item 3")

	nested.AddChild(nestedTitle)
	nested.AddChild(nestedItem1)
	nested.AddChild(nestedItem2)
	nested.AddChild(nestedItem3)

	// Instructions
	instructions := gooey.NewComposableLabel("Click buttons or press 'q' to quit")
	instructions.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	// Build layout
	app.root.AddChild(title)
	app.root.AddChild(app.counterLbl)
	app.root.AddChild(buttonRow)
	app.root.AddChild(nested)
	app.root.AddChild(instructions)

	app.root.SetBounds(image.Rect(0, 0, width, height))
	app.root.Init()

	return app
}

func (app *App) Init() error {
	app.terminal.EnableMouseTracking()
	return nil
}

func (app *App) Destroy() {
	app.terminal.DisableMouseTracking()
}

func (app *App) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
	case gooey.MouseEvent:
		if mouseAware, ok := interface{}(app.root).(gooey.MouseAware); ok {
			mouseAware.HandleMouse(e)
		}
	case gooey.ResizeEvent:
		app.root.SetBounds(image.Rect(0, 0, e.Width, e.Height))
	}
	return nil
}

func (app *App) Render(frame gooey.RenderFrame) {
	app.root.Draw(frame)
}

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	app := NewApp(terminal)

	runtime := gooey.NewRuntime(terminal, app, 30)
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
