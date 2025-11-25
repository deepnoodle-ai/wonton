package main

import (
	"fmt"

	"github.com/deepnoodle-ai/gooey"
)

// AnimationDemoApp demonstrates various text animation effects using the Runtime.
// It cycles through different animation styles with smooth transitions.
type AnimationDemoApp struct {
	frame        uint64
	width        int
	height       int
	currentStyle int // Current animation style index
	styleNames   []string
	animations   []gooey.TextAnimation
}

// Init initializes the animation styles
func (app *AnimationDemoApp) Init() error {
	app.styleNames = []string{
		"Rainbow Animation",
		"Reverse Rainbow",
		"Pulse Animation (Cyan)",
		"Pulse Animation (Orange)",
		"Fast Rainbow",
	}

	app.animations = []gooey.TextAnimation{
		gooey.CreateRainbowText("", 20),
		gooey.CreateReverseRainbowText("", 20),
		gooey.CreatePulseText(gooey.NewRGB(0, 255, 255), 30),
		gooey.CreatePulseText(gooey.NewRGB(255, 100, 0), 30),
		gooey.CreateRainbowText("", 10), // Faster rainbow
	}

	return nil
}

// HandleEvent processes events from the Runtime.
func (app *AnimationDemoApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		// Update animation frame
		app.frame = e.Frame

		// Cycle through animation styles every 5 seconds (150 frames at 30 FPS)
		if app.frame%150 == 0 && app.frame > 0 {
			app.currentStyle = (app.currentStyle + 1) % len(app.animations)
		}

	case gooey.KeyEvent:
		switch e.Rune {
		case 'q', 'Q':
			return []gooey.Cmd{gooey.Quit()}
		case 'n', 'N':
			// Next animation style
			app.currentStyle = (app.currentStyle + 1) % len(app.animations)
		case 'p', 'P':
			// Previous animation style
			app.currentStyle = (app.currentStyle - 1 + len(app.animations)) % len(app.animations)
		}

	case gooey.ResizeEvent:
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// Render draws the current animation frame.
func (app *AnimationDemoApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	// Draw title
	titleStyle := gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan)
	title := "Gooey Animation Demo"
	titleX := (width - len(title)) / 2
	if titleX < 0 {
		titleX = 0
	}
	frame.PrintStyled(titleX, 1, title, titleStyle)

	// Draw current style name
	currentStyleName := app.styleNames[app.currentStyle]
	styleNameX := (width - len(currentStyleName)) / 2
	if styleNameX < 0 {
		styleNameX = 0
	}
	styleNameStyle := gooey.NewStyle().WithForeground(gooey.ColorYellow)
	frame.PrintStyled(styleNameX, 3, currentStyleName, styleNameStyle)

	// Draw animated text
	demoText := "Hello, Gooey Animations!"
	textX := (width - len(demoText)) / 2
	if textX < 0 {
		textX = 0
	}
	textY := height / 2

	animation := app.animations[app.currentStyle]
	for i, ch := range demoText {
		style := animation.GetStyle(app.frame, i, len(demoText))
		frame.SetCell(textX+i, textY, ch, style)
	}

	// Draw feature list with gradient colors
	features := []string{
		"Multi-line animated content",
		"Rainbow text animation",
		"Reverse rainbow animation",
		"Pulsing brightness effects",
		"RGB color gradients",
		"30+ FPS animation engine",
	}

	startY := textY + 3
	for i, feature := range features {
		if startY+i >= height-2 {
			break
		}

		// Apply rainbow gradient to feature text
		featureX := 2
		rainbowAnim := gooey.CreateRainbowText("", 15)
		for j, ch := range feature {
			// Offset animation based on feature index for variety
			frameOffset := app.frame + uint64(i*10)
			style := rainbowAnim.GetStyle(frameOffset, j, len(feature))
			frame.SetCell(featureX+j, startY+i, ch, style)
		}
	}

	// Draw help text at bottom
	helpStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
	help := "[N]ext  [P]revious  [Q]uit"
	helpX := (width - len(help)) / 2
	if helpX < 0 {
		helpX = 0
	}
	frame.PrintStyled(helpX, height-2, help, helpStyle)

	// Draw frame counter
	frameText := fmt.Sprintf("Frame: %d", app.frame)
	frameStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	frame.PrintStyled(width-len(frameText)-1, height-1, frameText, frameStyle)
}

func main() {
	fmt.Println("Gooey Animation Demo")
	fmt.Println("====================")
	fmt.Println()
	fmt.Println("Testing animation features before starting interactive demo...")
	fmt.Println()

	// Test RGB color generation
	fmt.Println("1. RGB Color Generation:")
	colors := gooey.SmoothRainbow(10)
	for i, color := range colors {
		text := fmt.Sprintf("Color %d ", i)
		fmt.Print(color.Apply(text, false))
	}
	fmt.Println()

	// Test rainbow gradient
	fmt.Println("\n2. Rainbow Gradient:")
	rainbow := gooey.RainbowGradient(20)
	for i, color := range rainbow {
		fmt.Print(color.Apply("█", false))
		if i%10 == 9 {
			fmt.Print(" ")
		}
	}
	fmt.Println()

	// Test multi-gradient
	fmt.Println("\n3. Multi-Color Gradient:")
	stops := []gooey.RGB{
		gooey.NewRGB(255, 0, 0), // Red
		gooey.NewRGB(0, 255, 0), // Green
		gooey.NewRGB(0, 0, 255), // Blue
	}
	multiGrad := gooey.MultiGradient(stops, 15)
	for _, color := range multiGrad {
		fmt.Print(color.Apply("●", false))
	}
	fmt.Println()
	fmt.Println()
	fmt.Println("Starting interactive animation demo...")
	fmt.Println("Press any key to continue...")

	// Wait for user input
	var input string
	fmt.Scanln(&input)

	// Initialize terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Printf("Error initializing terminal: %v\n", err)
		return
	}
	defer terminal.Close()

	// Get initial terminal size
	width, height := terminal.Size()

	// Create the application
	app := &AnimationDemoApp{
		width:        width,
		height:       height,
		currentStyle: 0,
	}

	// Create and run the runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)

	// Run the event loop (blocks until quit)
	if err := runtime.Run(); err != nil {
		fmt.Printf("Runtime error: %v\n", err)
		return
	}

	fmt.Println("\n✨ Animation demo finished!")
	fmt.Println("Gooey supports all requested animation capabilities!")
}
