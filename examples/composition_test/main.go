package main

import (
	"image"
	"log"

	"github.com/deepnoodle-ai/gooey"
)

// Minimal composition test with deeply nested containers.
type App struct {
	root   *gooey.Container
	status *gooey.ComposableLabel
}

func (app *App) setupUI(width, height int) {

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
}

func (app *App) Init() error {
	return nil
}

func (app *App) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.ResizeEvent:
		if app.root == nil {
			// Initial setup
			app.setupUI(e.Width, e.Height)
		} else {
			// Just resize
			app.root.SetBounds(image.Rect(0, 0, e.Width, e.Height))
		}
	case gooey.KeyEvent:
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == gooey.KeyEscape || e.Key == gooey.KeyCtrlC {
			return []gooey.Cmd{gooey.Quit()}
		}
	case gooey.MouseEvent:
		if mouseAware, ok := interface{}(app.root).(gooey.MouseAware); ok {
			mouseAware.HandleMouse(e)
		}
	}
	return nil
}

func (app *App) Render(frame gooey.RenderFrame) {
	app.root.Draw(frame)
}

func main() {
	if err := gooey.Run(&App{}, gooey.WithMouseTracking(true)); err != nil {
		log.Fatal(err)
	}
}
