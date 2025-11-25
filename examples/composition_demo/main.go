package main

import (
	"fmt"
	"image"
	"os"

	"github.com/deepnoodle-ai/gooey"
)

// This example demonstrates the new composition system in Gooey.
// It shows:
// 1. Nested containers with different layout managers
// 2. VBox and HBox layouts
// 3. FlexLayout with various settings
// 4. Composable buttons and labels
// 5. Parent-child relationships and event handling
// 6. Message-driven Runtime architecture

// CompositionDemoApp implements the Application interface for the Runtime.
type CompositionDemoApp struct {
	mainContainer *gooey.Container
	statusLabel   *gooey.ComposableLabel
	clickCount    int
}

// NewCompositionDemoApp creates and initializes the application.
func NewCompositionDemoApp(terminal *gooey.Terminal) *CompositionDemoApp {
	app := &CompositionDemoApp{clickCount: 0}

	width, height := terminal.Size()

	// Create the main container with VBox layout
	// Use LayoutAlignStretch to make all children fill the width
	app.mainContainer = gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(1).WithAlignment(gooey.LayoutAlignStretch),
		&gooey.SingleBorder,
	)
	app.mainContainer.SetStyle(gooey.NewStyle().WithBackground(gooey.ColorBlack))

	// Set layout params for padding
	params := gooey.DefaultLayoutParams()
	params.PaddingTop = 1
	params.PaddingBottom = 1
	params.PaddingLeft = 2
	params.PaddingRight = 2
	app.mainContainer.SetLayoutParams(params)

	// Create header
	header := gooey.NewComposableLabel("Gooey Composition Demo")
	header.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold())
	header.WithAlign(gooey.AlignCenter)

	// Create description
	description := gooey.NewComposableMultiLineLabel([]string{
		"This showcases the composition system:",
		"• Nested containers with layouts",
		"• VBox, HBox, and Flex layouts",
		"• Bounds-based positioning",
		"• Parent-child event delegation",
	})
	description.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorWhite))

	// Create a horizontal button bar using HBox
	buttonBar := gooey.NewContainer(gooey.NewHBoxLayout(2))
	buttonBarParams := gooey.DefaultLayoutParams()
	buttonBarParams.MarginTop = 1
	buttonBar.SetLayoutParams(buttonBarParams)

	app.statusLabel = gooey.NewComposableLabel("Status: Ready")
	app.statusLabel.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	// Create buttons with different styles
	btn1 := gooey.NewComposableButton("Click Me!", func() {
		app.clickCount++
		app.statusLabel.SetText(fmt.Sprintf("Status: Button 1 clicked! (Count: %d)", app.clickCount))
	})

	btn2 := gooey.NewComposableButton("Press Here", func() {
		app.statusLabel.SetText("Status: Button 2 pressed!")
	})
	btn2.Style = gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack)
	btn2.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorBrightGreen).WithForeground(gooey.ColorBlack).WithBold()

	btn3 := gooey.NewComposableButton("Quit (q)", func() {
		app.statusLabel.SetText("Status: Exiting...")
	})
	btn3.Style = gooey.NewStyle().WithBackground(gooey.ColorRed).WithForeground(gooey.ColorWhite)
	btn3.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorBrightRed).WithForeground(gooey.ColorWhite).WithBold()

	buttonBar.AddChild(btn1)
	buttonBar.AddChild(btn2)
	buttonBar.AddChild(btn3)

	// Create a flex layout demo section
	flexSection := gooey.NewContainerWithBorder(
		gooey.NewFlexLayout().
			WithDirection(gooey.FlexRow).
			WithJustify(gooey.FlexJustifySpaceBetween).
			WithAlignItems(gooey.FlexAlignItemsCenter).
			WithSpacing(1),
		&gooey.RoundedBorder,
	)
	flexSectionParams := gooey.DefaultLayoutParams()
	flexSectionParams.MarginTop = 1
	flexSectionParams.PaddingTop = 1
	flexSectionParams.PaddingBottom = 1
	flexSectionParams.PaddingLeft = 2
	flexSectionParams.PaddingRight = 2
	flexSection.SetLayoutParams(flexSectionParams)

	flexLabel1 := gooey.NewComposableLabel("Flex Item 1")
	flexLabel1.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorMagenta))

	flexLabel2 := gooey.NewComposableLabel("Flex Item 2")
	flexLabel2.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorCyan))
	flexLabel2Params := gooey.DefaultLayoutParams()
	flexLabel2Params.Grow = 1 // This will take extra space
	flexLabel2.SetLayoutParams(flexLabel2Params)

	flexLabel3 := gooey.NewComposableLabel("Flex Item 3")
	flexLabel3.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorYellow))

	flexSection.AddChild(flexLabel1)
	flexSection.AddChild(flexLabel2)
	flexSection.AddChild(flexLabel3)

	// Create a nested container demo
	nestedSection := gooey.NewContainerWithBorder(
		gooey.NewVBoxLayout(0),
		&gooey.DoubleBorder,
	)
	nestedSectionParams := gooey.DefaultLayoutParams()
	nestedSectionParams.MarginTop = 1
	nestedSectionParams.PaddingTop = 0
	nestedSectionParams.PaddingBottom = 0
	nestedSectionParams.PaddingLeft = 1
	nestedSectionParams.PaddingRight = 1
	nestedSection.SetLayoutParams(nestedSectionParams)

	nestedTitle := gooey.NewComposableLabel("Nested Container")
	nestedTitle.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold())
	nestedTitle.WithAlign(gooey.AlignCenter)

	nestedContent := gooey.NewComposableLabel("Container inside container!")
	nestedContent.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorWhite))
	nestedContent.WithAlign(gooey.AlignCenter)

	nestedSection.AddChild(nestedTitle)
	nestedSection.AddChild(nestedContent)

	// Add instructions
	instructions := gooey.NewComposableMultiLineLabel([]string{
		"",
		"Instructions:",
		"• Click buttons with your mouse!",
		"• Watch the status label update",
		"• Press 'q' to quit",
	})
	instructions.WithStyle(gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))

	// Add all sections to main container
	app.mainContainer.AddChild(header)
	app.mainContainer.AddChild(description)
	app.mainContainer.AddChild(buttonBar)
	app.mainContainer.AddChild(app.statusLabel)
	app.mainContainer.AddChild(flexSection)
	app.mainContainer.AddChild(nestedSection)
	app.mainContainer.AddChild(instructions)

	// Set the main container bounds to fill the terminal
	app.mainContainer.SetBounds(image.Rect(0, 0, width, height))

	// Initialize the container (calls Init on all children)
	app.mainContainer.Init()

	return app
}

// HandleEvent processes events from the Runtime.
func (app *CompositionDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Handle 'q' to quit
		if e.Rune == 'q' || e.Rune == 'Q' {
			return []gooey.Cmd{gooey.Quit()}
		}

	case gooey.MouseEvent:
		// Pass mouse events to the container
		if mouseAware, ok := interface{}(app.mainContainer).(gooey.MouseAware); ok {
			mouseAware.HandleMouse(e)
		}

	case gooey.ResizeEvent:
		// Update container bounds when terminal is resized
		app.mainContainer.SetBounds(image.Rect(0, 0, e.Width, e.Height))
	}

	return nil
}

// Render draws the current application state.
func (app *CompositionDemoApp) Render(frame gooey.RenderFrame) {
	// The container handles all rendering of its children
	app.mainContainer.Draw(frame)
}

func main() {
	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	// Create the application
	app := NewCompositionDemoApp(terminal)

	// Create and run the Runtime
	// 30 FPS is good for UI responsiveness
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run blocks until the application quits
	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
