package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// Minimal composition test with deeply nested containers.
type App struct {
	terminal *gooey.Terminal
	root     *gooey.Container
	status   *gooey.ComposableLabel
}

func NewApp(terminal *gooey.Terminal, width, height int) *App {
	app := &App{terminal: terminal}

	// Root: VBox with border
	app.root = gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1),
		&gooey.DoubleBorder,
	)

	// Header
	header := gooey.NewComposableLabel("Nested Layout Test")
	header.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())

	// Status label
	app.status = gooey.NewComposableLabel("Click a button...")
	app.status.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Two-column layout using HBox
	columns := gooey.NewContainer(gooey.NewHBoxLayout(2))

	// Left column: VBox with buttons
	leftCol := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1),
		&gooey.SingleBorder,
	)
	leftTitle := gooey.NewComposableLabel("Buttons")
	leftTitle.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen))

	btn1 := gooey.NewComposableButton("Button 1", func() {
		app.status.SetText("Button 1 clicked!")
	})
	btn2 := gooey.NewComposableButton("Button 2", func() {
		app.status.SetText("Button 2 clicked!")
	})
	btn3 := gooey.NewComposableButton("Button 3", func() {
		app.status.SetText("Button 3 clicked!")
	})

	leftCol.AddChild(leftTitle)
	leftCol.AddChild(btn1)
	leftCol.AddChild(btn2)
	leftCol.AddChild(btn3)

	// Right column: VBox with labels
	rightCol := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(0),
		&gooey.RoundedBorder,
	)
	rightTitle := gooey.NewComposableLabel("Labels")
	rightTitle.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorMagenta))

	label1 := gooey.NewComposableLabel("Label A")
	label2 := gooey.NewComposableLabel("Label B")
	label3 := gooey.NewComposableLabel("Label C")

	rightCol.AddChild(rightTitle)
	rightCol.AddChild(label1)
	rightCol.AddChild(label2)
	rightCol.AddChild(label3)

	columns.AddChild(leftCol)
	columns.AddChild(rightCol)

	// Footer
	footer := gooey.NewComposableLabel("Press 'q' to quit")
	footer.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	// Assemble
	app.root.AddChild(header)
	app.root.AddChild(app.status)
	app.root.AddChild(columns)
	app.root.AddChild(footer)

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

	width, height := terminal.Size()
	app := NewApp(terminal, width, height)

	runtime := gooey.NewRuntime(terminal, app, 30)
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
