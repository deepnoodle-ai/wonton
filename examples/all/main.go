package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// ComprehensiveApp demonstrates all major Gooey features using the Runtime architecture.
// This includes animations, mouse handling, tab completion, and interactive components.
//
// Note: This example uses the old pattern (NewTerminal + NewRuntime) instead of gooey.Run()
// because it needs to call terminal.MoveCursor() in Render() to position the cursor for text input.
// Once RenderFrame supports cursor positioning, this can be simplified to use gooey.Run().
type ComprehensiveApp struct {
	terminal *gooey.Terminal
	runtime  *gooey.Runtime

	// Frame counter for animations
	frame uint64

	// Mouse handling
	mouseHandler *gooey.MouseHandler

	// Multiple input fields
	inputs        []string
	selectedInput int
	cursorPos     []int

	// Interactive components
	buttons      []*gooey.Button
	tabCompleter *gooey.TabCompleter
	clickedWords map[string]gooey.RGB

	// Demo data for metrics display
	cpuUsage       int
	memoryUsage    int
	networkSpeed   float64
	taskProgress   int
	spinnerFrame   int
	saveInProgress bool

	// Notifications
	notifications []string

	// Terminal size
	width  int
	height int
}

// HandleEvent processes events from the Runtime.
func (app *ComprehensiveApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.TickEvent:
		// Update animation state
		app.frame = e.Frame

		// Update spinner every ~150ms (at 30 FPS, every ~5 frames)
		if app.frame%5 == 0 {
			app.spinnerFrame++
		}

		// Update metrics periodically (~750ms at 30 FPS, every ~22 frames)
		if app.frame%22 == 0 {
			app.cpuUsage = 40 + int(time.Now().Unix()%40)
			app.memoryUsage = 50 + int(time.Now().Unix()%30)
			app.networkSpeed = 0.5 + float64(time.Now().Unix()%20)/10.0
			app.taskProgress = (app.taskProgress + 2) % 101
		}

	case gooey.KeyEvent:
		// Handle keyboard input
		app.handleKeyEvent(e)

	case gooey.MouseEvent:
		// Handle mouse clicks
		app.mouseHandler.HandleEvent(&e)

	case gooey.ResizeEvent:
		// Update terminal size and recreate components
		app.width = e.Width
		app.height = e.Height
		app.createButtons()
		app.updateClickableWords()
	}

	return nil
}

// handleKeyEvent processes keyboard events.
func (app *ComprehensiveApp) handleKeyEvent(e gooey.KeyEvent) {
	// Check for Ctrl+C to quit
	if e.Key == gooey.KeyCtrlC {
		// Stop the runtime
		app.runtime.Stop()
		return
	}

	// Arrow key handling
	if e.Key == gooey.KeyArrowUp || e.Key == gooey.KeyArrowDown ||
		e.Key == gooey.KeyArrowLeft || e.Key == gooey.KeyArrowRight {
		if app.tabCompleter.Visible {
			// Navigate in tab completer
			switch e.Key {
			case gooey.KeyArrowUp:
				app.tabCompleter.SelectPrev()
			case gooey.KeyArrowDown:
				app.tabCompleter.SelectNext()
			}
		} else {
			// Navigate between input fields or within field
			switch e.Key {
			case gooey.KeyArrowUp:
				if app.selectedInput > 0 {
					app.selectedInput--
				}
			case gooey.KeyArrowDown:
				if app.selectedInput < 3 {
					app.selectedInput++
				}
			case gooey.KeyArrowRight:
				if app.cursorPos[app.selectedInput] < len(app.inputs[app.selectedInput]) {
					app.cursorPos[app.selectedInput]++
				}
			case gooey.KeyArrowLeft:
				if app.cursorPos[app.selectedInput] > 0 {
					app.cursorPos[app.selectedInput]--
				}
			}
		}
		return
	}

	// Handle special keys
	switch e.Key {
	case gooey.KeyTab:
		if app.tabCompleter.Visible {
			selected := app.tabCompleter.GetSelected()
			if selected != "" {
				app.inputs[app.selectedInput] = selected + " "
				app.cursorPos[app.selectedInput] = len(app.inputs[app.selectedInput])
				app.tabCompleter.Hide()
			}
		} else if app.inputs[app.selectedInput] != "" {
			app.handleTabCompletion()
		} else {
			app.selectedInput = (app.selectedInput + 1) % 4
		}

	case gooey.KeyEscape:
		if app.tabCompleter.Visible {
			app.tabCompleter.Hide()
			app.updateClickableWords() // Re-add mouse regions
		} else {
			app.inputs[app.selectedInput] = ""
			app.cursorPos[app.selectedInput] = 0
		}

	case gooey.KeyEnter:
		if app.tabCompleter.Visible {
			selected := app.tabCompleter.GetSelected()
			if selected != "" {
				app.inputs[app.selectedInput] = selected + " "
				app.cursorPos[app.selectedInput] = len(app.inputs[app.selectedInput])
			}
			app.tabCompleter.Hide()
		} else if app.inputs[app.selectedInput] != "" {
			app.addNotification("Executed: " + app.inputs[app.selectedInput])
		}
		app.selectedInput = (app.selectedInput + 1) % 4

	case gooey.KeyBackspace:
		if app.cursorPos[app.selectedInput] > 0 {
			app.inputs[app.selectedInput] = app.inputs[app.selectedInput][:app.cursorPos[app.selectedInput]-1] +
				app.inputs[app.selectedInput][app.cursorPos[app.selectedInput]:]
			app.cursorPos[app.selectedInput]--
		}

	default:
		// Handle regular character input
		if e.Rune >= 32 && e.Rune < 127 {
			app.inputs[app.selectedInput] = app.inputs[app.selectedInput][:app.cursorPos[app.selectedInput]] +
				string(e.Rune) + app.inputs[app.selectedInput][app.cursorPos[app.selectedInput]:]
			app.cursorPos[app.selectedInput]++

			if app.tabCompleter.Visible {
				app.tabCompleter.Hide()
			}
		}
	}
}

// Render draws the current application state.
func (app *ComprehensiveApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()

	// Clear screen
	frame.Fill(' ', gooey.NewStyle())

	// Define layout regions
	y := 0

	// Title (2 lines)
	app.renderTitle(frame, y, width)
	y += 2

	// Status bar (2 lines)
	app.renderStatusBar(frame, y, width)
	y += 2

	// Clickable words (2 lines)
	app.renderClickableWords(frame, y, width)
	y += 2

	// Metrics (4 lines)
	app.renderMetrics(frame, y, width)
	y += 4

	// Buttons (2 lines)
	app.renderButtons(frame, y, width)
	y += 2

	// Input fields (4 lines)
	app.renderInputFields(frame, y, width)
	y += 4

	// Tab completions (if visible)
	if app.tabCompleter.Visible {
		app.tabCompleter.Draw(frame)
	}
	y += 4

	// Notifications (3 lines)
	app.renderNotifications(frame, y, width)
	y += 3

	// Footer
	if height-y > 2 {
		app.renderFooter(frame, height-3, width)
	}

	// Set cursor position for input
	cursorX := 10 + app.cursorPos[app.selectedInput]
	cursorY := 12 + app.selectedInput
	app.terminal.MoveCursor(cursorX, cursorY)
}

// renderTitle draws the animated title.
func (app *ComprehensiveApp) renderTitle(frame gooey.RenderFrame, y, width int) {
	spinner := app.getSpinner()
	titleText := spinner + " Gooey - Complete Demo with Mouse! " + spinner

	// Apply rainbow animation
	rainbow := gooey.CreateRainbowText(titleText, 15)
	for i, r := range titleText {
		if i < width {
			style := rainbow.GetStyle(app.frame, i, len(titleText))
			frame.SetCell(i, y, r, style)
		}
	}

	separator := strings.Repeat("═", width)
	frame.PrintStyled(0, y+1, separator, gooey.NewStyle())
}

// renderStatusBar draws the status information.
func (app *ComprehensiveApp) renderStatusBar(frame gooey.RenderFrame, y, width int) {
	modeColor := gooey.NewRGB(0, 255, 100)
	saveIndicator := ""
	if app.saveInProgress {
		saveIndicator = " " + app.getSpinner() + " Saving..."
	}

	statusText := fmt.Sprintf("Mode: NORMAL | Field: %d/4 | Time: %s%s",
		app.selectedInput+1,
		time.Now().Format("15:04:05"),
		saveIndicator)

	// Apply pulse animation
	pulse := gooey.CreatePulseText(modeColor, 30)
	for i, r := range statusText {
		if i < width {
			style := pulse.GetStyle(app.frame, i, len(statusText))
			frame.SetCell(i, y, r, style)
		}
	}

	// Progress bar
	progress := app.taskProgress % 101
	filled := progress / 5
	progressBar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
	progressText := fmt.Sprintf("Progress: [%s] %d%%", progressBar, progress)
	frame.PrintStyled(0, y+1, progressText, gooey.NewStyle())
}

// renderClickableWords draws the clickable word demo.
func (app *ComprehensiveApp) renderClickableWords(frame gooey.RenderFrame, y, width int) {
	words := []string{"Click", "these", "words", "to", "change", "colors!"}

	text := "🖱️ Clickable: "
	x := 0
	frame.PrintStyled(x, y, text, gooey.NewStyle())
	x += len(text)

	for i, word := range words {
		if color, exists := app.clickedWords[word]; exists {
			style := gooey.NewStyle().WithFgRGB(color)
			frame.PrintStyled(x, y, word, style)
		} else {
			frame.PrintStyled(x, y, word, gooey.NewStyle())
		}
		x += len(word)

		if i < len(words)-1 {
			frame.PrintStyled(x, y, " ", gooey.NewStyle())
			x++
		}
	}

	frame.PrintStyled(0, y+1, "   Try clicking the words above!", gooey.NewStyle())
}

// renderMetrics draws the system metrics display.
func (app *ComprehensiveApp) renderMetrics(frame gooey.RenderFrame, y, width int) {
	frame.PrintStyled(0, y, "📊 System Metrics:", gooey.NewStyle())

	cpuBar := app.createMeter(app.cpuUsage, 100)
	cpuText := fmt.Sprintf("   CPU: %s %3d%%", cpuBar, app.cpuUsage)
	frame.PrintStyled(0, y+1, cpuText, gooey.NewStyle())

	memBar := app.createMeter(app.memoryUsage, 100)
	memText := fmt.Sprintf("   RAM: %s %3d%%", memBar, app.memoryUsage)
	frame.PrintStyled(0, y+2, memText, gooey.NewStyle())

	netText := fmt.Sprintf("   NET: %s %.1f GB/s", app.getSpinner(), app.networkSpeed)
	frame.PrintStyled(0, y+3, netText, gooey.NewStyle())
}

// renderButtons draws the interactive buttons.
func (app *ComprehensiveApp) renderButtons(frame gooey.RenderFrame, y, width int) {
	frame.PrintStyled(0, y, "🔘 Interactive Buttons (click them!):", gooey.NewStyle())

	// Draw buttons
	for _, btn := range app.buttons {
		btn.Draw(frame)
	}
}

// renderInputFields draws the input fields.
func (app *ComprehensiveApp) renderInputFields(frame gooey.RenderFrame, y, width int) {
	labels := []string{"Name    ", "Email   ", "Project ", "Command "}

	for i := 0; i < 4; i++ {
		text := labels[i] + ": " + app.inputs[i]

		if i == app.selectedInput {
			style := gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack)
			frame.PrintStyled(0, y+i, text, style)
			// Pad the rest of the line
			if len(text) < width {
				padding := strings.Repeat(" ", width-len(text))
				frame.PrintStyled(len(text), y+i, padding, style)
			}
		} else {
			frame.PrintStyled(0, y+i, text, gooey.NewStyle())
		}
	}
}

// renderNotifications draws recent notifications.
func (app *ComprehensiveApp) renderNotifications(frame gooey.RenderFrame, y, width int) {
	for i := 0; i < 3; i++ {
		if i < len(app.notifications) {
			frame.PrintStyled(0, y+i, app.notifications[i], gooey.NewStyle())
		}
	}
}

// renderFooter draws the footer with help text.
func (app *ComprehensiveApp) renderFooter(frame gooey.RenderFrame, y, width int) {
	frame.PrintStyled(0, y, "🖱️ Click words/buttons | TAB: Completions | ↑↓: Navigate", gooey.NewStyle())
	frame.PrintStyled(0, y+1, "Enter: Accept | ESC: Cancel | Ctrl+C: Exit", gooey.NewStyle())
}

// createButtons creates the interactive button components.
func (app *ComprehensiveApp) createButtons() {
	// Create interactive buttons
	app.buttons = []*gooey.Button{
		gooey.NewButton(5, 11, "Execute", func() {
			app.addNotification("▶ Executing: " + app.inputs[app.selectedInput])
		}),
		gooey.NewButton(18, 11, "Clear", func() {
			for i := range app.inputs {
				app.inputs[i] = ""
				app.cursorPos[i] = 0
			}
			app.addNotification("✨ All fields cleared")
		}),
		gooey.NewButton(29, 11, "Save", func() {
			app.saveInProgress = true
			// Schedule async save completion
			go func() {
				time.Sleep(2 * time.Second)
				app.saveInProgress = false
				app.addNotification("💾 Configuration saved")
			}()
		}),
	}

	// Add button regions to mouse handler
	app.mouseHandler.ClearRegions()
	for _, btn := range app.buttons {
		app.mouseHandler.AddRegion(btn.GetRegion())
	}

	// Re-add clickable word regions
	app.updateClickableWords()
}

// setupTabCompleter initializes the tab completion component.
func (app *ComprehensiveApp) setupTabCompleter() {
	commands := []string{
		"build", "test", "run", "install", "clean",
		"deploy", "debug", "profile", "benchmark",
		"commit", "push", "pull", "merge", "rebase",
	}

	app.tabCompleter.OnSelect = func(suggestion string) {
		app.inputs[app.selectedInput] = suggestion + " "
		app.cursorPos[app.selectedInput] = len(app.inputs[app.selectedInput])
		app.tabCompleter.Hide()
	}

	app.tabCompleter.SetSuggestions(commands, "")
}

// updateClickableWords sets up the clickable word mouse regions.
func (app *ComprehensiveApp) updateClickableWords() {
	words := []string{"Click", "these", "words", "to", "change", "colors!"}

	// Add word regions (buttons are already in mouse handler)
	x := 14 // After "🖱️ Clickable: "
	y := 4

	for _, word := range words {
		wordCopy := word
		region := &gooey.MouseRegion{
			X:      x,
			Y:      y,
			Width:  len(word),
			Height: 1,
			Label:  word,
			OnClick: func(event *gooey.MouseEvent) {
				colors := []gooey.RGB{
					gooey.NewRGB(255, 100, 100),
					gooey.NewRGB(100, 255, 100),
					gooey.NewRGB(100, 100, 255),
					gooey.NewRGB(255, 255, 100),
				}

				currentIdx := 0
				if existingColor, exists := app.clickedWords[wordCopy]; exists {
					for i, c := range colors {
						if c == existingColor {
							currentIdx = (i + 1) % len(colors)
							break
						}
					}
				}

				app.clickedWords[wordCopy] = colors[currentIdx]
				app.addNotification(fmt.Sprintf("🎨 Colored '%s'", wordCopy))
			},
		}
		app.mouseHandler.AddRegion(region)
		x += len(word) + 1
	}

	// Also add tab completer regions if visible
	if app.tabCompleter.Visible {
		for _, region := range app.tabCompleter.GetRegions() {
			app.mouseHandler.AddRegion(region)
		}
	}
}

// handleTabCompletion shows tab completion suggestions.
func (app *ComprehensiveApp) handleTabCompletion() {
	if app.inputs[app.selectedInput] == "" {
		return
	}

	suggestions := []string{}
	commands := []string{
		"build", "test", "run", "install", "clean",
		"deploy", "debug", "profile", "benchmark",
		"commit", "push", "pull", "merge", "rebase",
	}

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, app.inputs[app.selectedInput]) {
			suggestions = append(suggestions, cmd)
		}
	}

	if len(suggestions) > 0 {
		app.tabCompleter.SetSuggestions(suggestions, app.inputs[app.selectedInput])
		app.tabCompleter.Show(10, 12+app.selectedInput, 30)

		for _, region := range app.tabCompleter.GetRegions() {
			app.mouseHandler.AddRegion(region)
		}
	} else {
		app.tabCompleter.Hide()
	}
}

// addNotification adds a notification message.
func (app *ComprehensiveApp) addNotification(msg string) {
	notification := fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), msg)

	app.notifications = append(app.notifications, notification)
	if len(app.notifications) > 3 {
		app.notifications = app.notifications[len(app.notifications)-3:]
	}
}

// createMeter creates a visual meter bar.
func (app *ComprehensiveApp) createMeter(value, max int) string {
	width := 20
	filled := (value * width) / max

	bar := "["
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "·"
		}
	}
	bar += "]"
	return bar
}

// getSpinner returns the current spinner character.
func (app *ComprehensiveApp) getSpinner() string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return spinners[app.spinnerFrame%len(spinners)]
}

// Init initializes the application.
func (app *ComprehensiveApp) Init() error {
	// Get terminal size
	app.width, app.height = app.terminal.Size()

	// Enable alternate screen and mouse tracking
	// Note: Runtime automatically enables raw mode
	app.terminal.EnableAlternateScreen()
	app.terminal.EnableMouseTracking()
	app.terminal.ShowCursor()

	// Initialize components
	app.createButtons()
	app.setupTabCompleter()
	app.updateClickableWords()

	return nil
}

// Destroy cleans up the application.
func (app *ComprehensiveApp) Destroy() {
	// Restore terminal state
	app.terminal.DisableMouseTracking()
	app.terminal.DisableAlternateScreen()
	app.terminal.ShowCursor()
	// Note: Runtime automatically disables raw mode
}

func main() {
	fmt.Println("\n🚀 Comprehensive Gooey Demo with Mouse Support!")
	fmt.Println("=" + strings.Repeat("=", 50))
	fmt.Println("\nFeatures:")
	fmt.Println("✅ Click on words to change their colors")
	fmt.Println("✅ Click on buttons (Execute, Clear, Save)")
	fmt.Println("✅ Tab completion (type 'b' then TAB)")
	fmt.Println("✅ Arrow keys for navigation")
	fmt.Println("✅ Real-time metrics and animations")
	fmt.Println("\nMake sure your terminal supports mouse events!")
	fmt.Println("Starting in 2 seconds...")
	time.Sleep(2 * time.Second)

	// Create terminal
	terminal, err := gooey.NewTerminal()
	if err != nil {
		log.Fatalf("Failed to create terminal: %v\n", err)
	}
	defer terminal.Close()

	// Create the application
	app := &ComprehensiveApp{
		terminal:      terminal,
		mouseHandler:  gooey.NewMouseHandler(),
		tabCompleter:  gooey.NewTabCompleter(),
		inputs:        make([]string, 4),
		cursorPos:     make([]int, 4),
		selectedInput: 0,
		notifications: []string{},
		clickedWords:  make(map[string]gooey.RGB),
		cpuUsage:      45,
		memoryUsage:   62,
		networkSpeed:  1.2,
		taskProgress:  0,
	}

	// Create and run the runtime with 30 FPS
	runtime := gooey.NewRuntime(terminal, app, 30)
	app.runtime = runtime // Store reference for stopping

	// Run the event loop (blocks until quit)
	if err := runtime.Run(); err != nil {
		log.Fatalf("Runtime error: %v\n", err)
	}

	fmt.Println("\n✨ Thanks for trying the comprehensive demo!")
}
