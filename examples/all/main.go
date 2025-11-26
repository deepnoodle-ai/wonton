package main

import (
	"fmt"
	"image"
	"log"
	"strings"
	"time"

	"github.com/deepnoodle-ai/gooey"
)

// ComprehensiveApp demonstrates all major Gooey features using the declarative View architecture.
// This includes animations, mouse handling, tab completion, and interactive components.
type ComprehensiveApp struct {
	// Frame counter for animations
	frame uint64

	// Multiple input fields
	inputs        []string
	selectedInput int
	cursorPos     []int

	// Tab completion
	tabCompleter      *gooey.TabCompleter
	clickedWords      map[string]gooey.RGB
	clickableWordRefs []string // Track word order

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
		return app.handleKeyEvent(e)

	case gooey.ResizeEvent:
		// Update terminal size
		app.width = e.Width
		app.height = e.Height
	}

	return nil
}

// handleKeyEvent processes keyboard events.
func (app *ComprehensiveApp) handleKeyEvent(e gooey.KeyEvent) []gooey.Cmd {
	// Check for Ctrl+C to quit
	if e.Key == gooey.KeyCtrlC {
		return []gooey.Cmd{gooey.Quit()}
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
		return nil
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
	return nil
}

// View returns the declarative view structure.
func (app *ComprehensiveApp) View() gooey.View {
	spinner := app.getSpinner()

	// Build clickable words
	clickableWordsView := app.buildClickableWords()

	// Build buttons
	buttonsView := app.buildButtons()

	// Build input fields
	inputFieldsView := app.buildInputFields()

	// Build notifications
	notificationsView := app.buildNotifications()

	return gooey.VStack(
		// Title with animated rainbow text
		app.buildAnimatedTitle(spinner),
		gooey.Text(strings.Repeat("═", 80)),
		gooey.Spacer().MinHeight(1),

		// Status bar
		app.buildStatusBar(spinner),
		gooey.Spacer().MinHeight(1),

		// Clickable words demo
		clickableWordsView,
		gooey.Text("   Try clicking the words above!"),
		gooey.Spacer().MinHeight(1),

		// Metrics
		app.buildMetrics(spinner),
		gooey.Spacer().MinHeight(1),

		// Buttons
		gooey.Text("🔘 Interactive Buttons (click them!):"),
		buttonsView,
		gooey.Spacer().MinHeight(1),

		// Input fields
		inputFieldsView,
		gooey.Spacer().MinHeight(1),

		// Notifications
		notificationsView,
		gooey.Spacer(),

		// Footer
		gooey.Text("🖱️ Click words/buttons | TAB: Completions | ↑↓: Navigate"),
		gooey.Text("Enter: Accept | ESC: Cancel | Ctrl+C: Exit"),
	)
}

// buildAnimatedTitle creates the animated rainbow title.
func (app *ComprehensiveApp) buildAnimatedTitle(spinner string) gooey.View {
	titleText := spinner + " Gooey - Complete Demo with Mouse! " + spinner
	return gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
		rainbow := gooey.CreateRainbowText(titleText, 15)
		for i, r := range titleText {
			if i < bounds.Dx() {
				style := rainbow.GetStyle(app.frame, i, len(titleText))
				frame.SetCell(bounds.Min.X+i, bounds.Min.Y, r, style)
			}
		}
	})
}

// buildStatusBar creates the status bar with pulse animation.
func (app *ComprehensiveApp) buildStatusBar(spinner string) gooey.View {
	saveIndicator := ""
	if app.saveInProgress {
		saveIndicator = " " + spinner + " Saving..."
	}

	statusText := fmt.Sprintf("Mode: NORMAL | Field: %d/4 | Time: %s%s",
		app.selectedInput+1,
		time.Now().Format("15:04:05"),
		saveIndicator)

	progress := app.taskProgress % 101
	filled := progress / 5
	progressBar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
	progressText := fmt.Sprintf("Progress: [%s] %d%%", progressBar, progress)

	return gooey.VStack(
		gooey.Canvas(func(frame gooey.RenderFrame, bounds image.Rectangle) {
			modeColor := gooey.NewRGB(0, 255, 100)
			pulse := gooey.CreatePulseText(modeColor, 30)
			for i, r := range statusText {
				if i < bounds.Dx() {
					style := pulse.GetStyle(app.frame, i, len(statusText))
					frame.SetCell(bounds.Min.X+i, bounds.Min.Y, r, style)
				}
			}
		}),
		gooey.Text(progressText),
	)
}

// buildClickableWords creates the clickable words demo.
func (app *ComprehensiveApp) buildClickableWords() gooey.View {
	words := []string{"Click", "these", "words", "to", "change", "colors!"}
	app.clickableWordRefs = words // Store for reference

	wordViews := []gooey.View{gooey.Text("🖱️ Clickable: ")}

	for i, word := range words {
		wordCopy := word
		wordView := gooey.Clickable(word, func() {
			app.cycleWordColor(wordCopy)
		}).Bold()

		wordViews = append(wordViews, wordView)

		if i < len(words)-1 {
			wordViews = append(wordViews, gooey.Text(" "))
		}
	}

	return gooey.HStack(wordViews...)
}

// cycleWordColor cycles through colors for a clicked word.
func (app *ComprehensiveApp) cycleWordColor(word string) {
	colors := []gooey.RGB{
		gooey.NewRGB(255, 100, 100),
		gooey.NewRGB(100, 255, 100),
		gooey.NewRGB(100, 100, 255),
		gooey.NewRGB(255, 255, 100),
	}

	currentIdx := 0
	if existingColor, exists := app.clickedWords[word]; exists {
		for i, c := range colors {
			if c == existingColor {
				currentIdx = (i + 1) % len(colors)
				break
			}
		}
	}

	app.clickedWords[word] = colors[currentIdx]
	app.addNotification(fmt.Sprintf("🎨 Colored '%s'", word))
}

// buildMetrics creates the metrics display.
func (app *ComprehensiveApp) buildMetrics(spinner string) gooey.View {
	cpuBar := app.createMeter(app.cpuUsage, 100)
	cpuText := fmt.Sprintf("   CPU: %s %3d%%", cpuBar, app.cpuUsage)

	memBar := app.createMeter(app.memoryUsage, 100)
	memText := fmt.Sprintf("   RAM: %s %3d%%", memBar, app.memoryUsage)

	netText := fmt.Sprintf("   NET: %s %.1f GB/s", spinner, app.networkSpeed)

	return gooey.VStack(
		gooey.Text("📊 System Metrics:"),
		gooey.Text(cpuText),
		gooey.Text(memText),
		gooey.Text(netText),
	)
}

// buildButtons creates the interactive buttons.
func (app *ComprehensiveApp) buildButtons() gooey.View {
	return gooey.HStack(
		gooey.Clickable("[Execute]", func() {
			app.addNotification("▶ Executing: " + app.inputs[app.selectedInput])
		}).Fg(gooey.ColorGreen),
		gooey.Spacer().MinWidth(2),
		gooey.Clickable("[Clear]", func() {
			for i := range app.inputs {
				app.inputs[i] = ""
				app.cursorPos[i] = 0
			}
			app.addNotification("✨ All fields cleared")
		}).Fg(gooey.ColorYellow),
		gooey.Spacer().MinWidth(2),
		gooey.Clickable("[Save]", func() {
			app.saveInProgress = true
			go func() {
				time.Sleep(2 * time.Second)
				app.saveInProgress = false
				app.addNotification("💾 Configuration saved")
			}()
		}).Fg(gooey.ColorBlue),
	)
}

// buildInputFields creates the input field display.
func (app *ComprehensiveApp) buildInputFields() gooey.View {
	labels := []string{"Name    ", "Email   ", "Project ", "Command "}
	fields := []gooey.View{}

	for i := 0; i < 4; i++ {
		text := labels[i] + ": " + app.inputs[i]

		if i == app.selectedInput {
			fields = append(fields, gooey.Text(text).Bg(gooey.ColorGreen).Fg(gooey.ColorBlack))
		} else {
			fields = append(fields, gooey.Text(text))
		}
	}

	return gooey.VStack(fields...)
}

// buildNotifications creates the notifications display.
func (app *ComprehensiveApp) buildNotifications() gooey.View {
	notifViews := []gooey.View{}

	for i := 0; i < 3; i++ {
		if i < len(app.notifications) {
			notifViews = append(notifViews, gooey.Text(app.notifications[i]))
		} else {
			notifViews = append(notifViews, gooey.Text(""))
		}
	}

	return gooey.VStack(notifViews...)
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

func main() {
	fmt.Println("\n🚀 Comprehensive Gooey Demo with Declarative Views!")
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

	// Create the application
	app := &ComprehensiveApp{
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
		width:         80,
		height:        24,
	}

	// Run with mouse tracking and 30 FPS
	if err := gooey.Run(app, gooey.WithMouseTracking(true), gooey.WithFPS(30)); err != nil {
		log.Fatalf("Runtime error: %v\n", err)
	}

	fmt.Println("\n✨ Thanks for trying the comprehensive demo!")
}
