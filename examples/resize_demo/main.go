package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// ResizeDemoApp demonstrates automatic terminal resize handling with containers.
type ResizeDemoApp struct {
	// Layout components
	layout        *gooey.Layout
	mainContainer *gooey.Container

	// Labels
	titleLabel       *gooey.ComposableLabel
	sizeLabel        *gooey.ComposableLabel
	instructionLabel *gooey.ComposableLabel

	// State
	clickCount  int
	resizeCount int
	width       int
	height      int
}

// Init initializes the application components.
func (app *ResizeDemoApp) Init() error {
	// Layout will be initialized in first render
	return nil
}

// HandleEvent processes events from the runtime.
func (app *ResizeDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle keyboard input
		if e.Key == gooey.KeyCtrlC || e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.ResizeEvent:
		// Update dimensions and resize count
		app.width = e.Width
		app.height = e.Height
		app.resizeCount++

		// Update footer with new resize count
		if app.layout != nil {
			app.layout.SetFooter(&gooey.Footer{
				StatusBar: true,
				StatusItems: []gooey.StatusItem{
					{Key: "Resizes", Value: fmt.Sprintf("%d", app.resizeCount), Icon: "📏"},
					{Key: "Size", Value: fmt.Sprintf("%dx%d", app.width, app.height), Icon: "📐"},
					{Key: "Controls", Value: "Press Q or Ctrl+C to exit", Icon: "⌨"},
				},
				Border:      true,
				BorderStyle: gooey.RoundedBorder,
				Height:      3,
			})
		}
	}

	return nil
}

// Render draws the current application state.
func (app *ResizeDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Update dimensions
	app.width = width
	app.height = height

	// Lazy initialization of layout and container on first render
	if app.layout == nil {
		app.initializeComponents(frame)
	}

	// Update size label
	cy, ch := app.layout.ContentArea()
	app.sizeLabel.SetText(fmt.Sprintf("Terminal: %dx%d | Content: Y=%d H=%d", width, height, cy, ch))

	// Update container bounds
	app.mainContainer.SetBounds(image.Rect(0, cy, width, cy+ch))

	// Draw layout and container
	app.layout.DrawTo(frame)
	app.mainContainer.Draw(frame)
}

// initializeComponents creates the layout and container on first render.
func (app *ResizeDemoApp) initializeComponents(frame gooey.RenderFrame) {
	// Get terminal from frame (we need it for layout)
	// Since we don't have direct access, we'll create components without it

	// Create layout (it will adapt to frame size)
	app.layout = &gooey.Layout{}

	// Configure header
	headerStyle := gooey.NewStyle().
		WithBold().
		WithForeground(gooey.ColorCyan)
	app.layout.SetHeader(&gooey.Header{
		Center:      "Terminal Resize Demo with Containers",
		Style:       headerStyle,
		Border:      true,
		BorderStyle: gooey.RoundedBorder,
		Height:      3,
	})

	// Configure footer with status bar
	app.layout.SetFooter(&gooey.Footer{
		StatusBar: true,
		StatusItems: []gooey.StatusItem{
			{Key: "Status", Value: "Ready", Icon: "✓"},
			{Key: "Controls", Value: "Press Q or Ctrl+C to exit", Icon: "⌨"},
		},
		Border:      true,
		BorderStyle: gooey.RoundedBorder,
		Height:      3,
	})

	// Create main container
	app.mainContainer = gooey.NewContainer(gooey.NewVBoxLayout(1))

	// Add title label
	app.titleLabel = gooey.NewComposableLabel("Container with Auto-Resize").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()).
		WithAlign(gooey.AlignCenter)
	app.mainContainer.AddChild(app.titleLabel)

	// Add size label
	app.sizeLabel = gooey.NewComposableLabel("Size: ...").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow)).
		WithAlign(gooey.AlignCenter)
	app.mainContainer.AddChild(app.sizeLabel)

	// Add instruction label
	app.instructionLabel = gooey.NewComposableLabel("Try resizing your terminal window!").
		WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen)).
		WithAlign(gooey.AlignCenter)
	app.mainContainer.AddChild(app.instructionLabel)

	// Create button container
	buttonContainer := gooey.NewContainer(gooey.NewHBoxLayout(2))

	button1 := gooey.NewComposableButton("Click Me!", func() {
		app.clickCount++
		app.titleLabel.SetText(fmt.Sprintf("Clicked %d times!", app.clickCount))
	})
	button1.Style = gooey.NewStyle().
		WithForeground(gooey.ColorBlack).
		WithBackground(gooey.ColorCyan)
	buttonContainer.AddChild(button1)

	button2 := gooey.NewComposableButton("Reset", func() {
		app.clickCount = 0
		app.titleLabel.SetText("Container with Auto-Resize")
	})
	button2.Style = gooey.NewStyle().
		WithForeground(gooey.ColorBlack).
		WithBackground(gooey.ColorYellow)
	buttonContainer.AddChild(button2)

	app.mainContainer.AddChild(buttonContainer)

	// Initialize container
	width, _ := frame.Size()
	contentY, contentHeight := app.layout.ContentArea()
	app.mainContainer.SetBounds(image.Rect(0, contentY, width, contentY+contentHeight))
	app.mainContainer.Init()
}

// Destroy cleans up application resources.
func (app *ResizeDemoApp) Destroy() {
	if app.mainContainer != nil {
		app.mainContainer.Destroy()
	}
}

func main() {
	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Get initial size
	width, height := terminal.Size()

	// Create application
	app := &ResizeDemoApp{
		width:  width,
		height: height,
	}

	// Create runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run the event loop
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
