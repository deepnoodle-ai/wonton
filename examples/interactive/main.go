package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/deepnoodle-ai/gooey"
	"golang.org/x/term"
)

type InteractiveDemo struct {
	terminal      *gooey.Terminal
	screenManager *gooey.ScreenManager
	mouseHandler  *gooey.MouseHandler

	// UI Components
	buttons      []*gooey.Button
	tabCompleter *gooey.TabCompleter
	radioGroup   *gooey.RadioGroup

	// State
	currentInput   string
	cursorPos      int
	commandHistory []string
	notifications  []string
	clickedWords   map[string]gooey.RGB
	selectedTheme  string

	running bool
	mu      sync.Mutex
}

func NewInteractiveDemo() (*InteractiveDemo, error) {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		return nil, err
	}

	return &InteractiveDemo{
		terminal:       terminal,
		screenManager:  gooey.NewScreenManager(terminal, 15),
		mouseHandler:   gooey.NewMouseHandler(),
		tabCompleter:   gooey.NewTabCompleter(),
		clickedWords:   make(map[string]gooey.RGB),
		commandHistory: make([]string, 0),
		notifications:  make([]string, 0),
		running:        true,
		selectedTheme:  "Default",
	}, nil
}

func (d *InteractiveDemo) Setup() {
	// Enable alternate screen and mouse tracking
	d.terminal.EnableAlternateScreen()
	d.terminal.EnableMouseTracking()
	d.terminal.ShowCursor()

	width, height := d.terminal.Size()

	// Function to set up regions
	setupRegions := func(w, h int) {
		// Define screen regions
		d.screenManager.DefineRegion("title", 0, 0, w, 2, false)
		d.screenManager.DefineRegion("clickable", 0, 2, w, 4, false)
		d.screenManager.DefineRegion("buttons", 0, 6, w, 3, false)
		d.screenManager.DefineRegion("radio", 0, 9, w, 4, false)
		d.screenManager.DefineRegion("input", 0, 13, w, 2, true)
		d.screenManager.DefineRegion("completions", 0, 15, w, 7, false)
		d.screenManager.DefineRegion("notifications", 0, 22, w, 3, false)

		// Footer at bottom
		footerY := h - 2
		if footerY > 25 {
			footerY = 25
		}
		d.screenManager.DefineRegion("footer", 0, footerY, w, 2, false)

		// Reinitialize display
		d.initializeDisplay()
	}

	// Initial setup
	setupRegions(width, height)

	// Enable automatic resize handling
	d.terminal.WatchResize()
	d.terminal.OnResize(func(w, h int) {
		setupRegions(w, h)
		d.createButtons()
		d.createRadioGroup()
		d.updateClickableText()
	})

	// Create UI components
	d.createButtons()
	d.createRadioGroup()
	d.setupTabCompleter()

	// Start screen manager
	d.screenManager.Start()
}

func (d *InteractiveDemo) createButtons() {
	// Create interactive buttons
	d.buttons = []*gooey.Button{
		gooey.NewButton(5, 7, "Run Command", func() {
			d.addNotification("▶ Running command: " + d.currentInput)
			d.commandHistory = append(d.commandHistory, d.currentInput)
			d.currentInput = ""
			d.cursorPos = 0
			d.updateInput()
		}),
		gooey.NewButton(22, 7, "Clear Input", func() {
			d.currentInput = ""
			d.cursorPos = 0
			d.tabCompleter.Hide()
			d.updateInput()
			d.addNotification("✨ Input cleared")
		}),
		gooey.NewButton(39, 7, "Show Help", func() {
			d.addNotification("ℹ️  Use TAB for completions, click words to color them!")
		}),
	}

	// Set custom draw callback for buttons region
	d.screenManager.SetRegionDrawCallback("buttons", func(frame gooey.RenderFrame, x, y, w, h int) {
		// Draw region title
		frame.PrintStyled(x, y, "🔘 Interactive Buttons:", gooey.NewStyle())

		// Draw buttons
		// Note: Buttons use absolute coordinates (e.g. 5, 7) which matches the screen layout
		for _, btn := range d.buttons {
			btn.Draw(frame)
		}
	})

	// Add button regions to mouse handler
	for _, btn := range d.buttons {
		d.mouseHandler.AddRegion(btn.GetRegion())
	}
}

func (d *InteractiveDemo) createRadioGroup() {
	themes := []string{"Default", "Dark Mode", "High Contrast", "Solarized"}
	d.radioGroup = gooey.NewRadioGroup(5, 10, themes)
	d.radioGroup.OnChange = func(index int, theme string) {
		d.selectedTheme = theme
		d.addNotification(fmt.Sprintf("🎨 Theme changed to: %s", theme))
		d.updateColors()
	}

	// Set custom draw callback for radio region
	d.screenManager.SetRegionDrawCallback("radio", func(frame gooey.RenderFrame, x, y, w, h int) {
		// Draw region title
		frame.PrintStyled(x, y, "🎨 Theme Selection:", gooey.NewStyle())

		// Draw radio group
		d.radioGroup.Draw(frame)
	})

	// Add radio button regions
	for _, region := range d.radioGroup.GetRegions() {
		d.mouseHandler.AddRegion(region)
	}
}

func (d *InteractiveDemo) setupTabCompleter() {
	// Commands for tab completion
	commands := []string{
		"build", "test", "run", "install", "clean",
		"deploy", "debug", "profile", "benchmark",
		"commit", "push", "pull", "merge", "rebase",
		"status", "diff", "log", "branch", "checkout",
	}

	d.tabCompleter.OnSelect = func(suggestion string) {
		d.currentInput = suggestion + " "
		d.cursorPos = len(d.currentInput)
		d.tabCompleter.Hide()
		d.updateInput()
	}

	// Store for later use
	d.tabCompleter.SetSuggestions(commands, "")

	// Set custom draw callback for completions region
	// This ensures the dropdown is drawn as part of the screen refresh cycle
	d.screenManager.SetRegionDrawCallback("completions", func(frame gooey.RenderFrame, x, y, w, h int) {
		if d.tabCompleter.Visible {
			d.tabCompleter.Draw(frame)
		}
	})
}

func (d *InteractiveDemo) initializeDisplay() {
	// Title
	d.screenManager.UpdateRegion("title", 0,
		"🖱️  Interactive Gooey Demo - Mouse, Tab Completion & Buttons",
		gooey.CreateRainbowText("🖱️  Interactive Gooey Demo - Mouse, Tab Completion & Buttons", 12))
	d.screenManager.UpdateRegion("title", 1,
		"═══════════════════════════════════════════════════════════",
		nil)

	// Clickable text area
	d.updateClickableText()

	// Buttons
	d.updateButtons()

	// Radio buttons
	d.updateRadioButtons()

	// Input area
	d.updateInput()

	// Footer
	d.screenManager.UpdateRegion("footer", 0,
		"TAB: Show completions | Click words to color | Click buttons to interact",
		nil)
	d.screenManager.UpdateRegion("footer", 1,
		"↑↓: Navigate completions | Enter: Accept | ESC: Cancel | Ctrl+C: Exit",
		nil)
}

func (d *InteractiveDemo) updateClickableText() {
	// Create clickable words
	words := []string{"Click", "these", "words", "to", "change", "their", "colors!"}

	// Set custom draw callback for clickable region
	d.screenManager.SetRegionDrawCallback("clickable", func(frame gooey.RenderFrame, x, y, w, h int) {
		// Draw title
		frame.PrintStyled(x, y, "📝 Clickable Text Area:", gooey.NewStyle())

		// Draw colored words
		currentX := x + 3
		currentY := y + 1

		for i, word := range words {
			style := gooey.NewStyle()

			// Check if word has been clicked and has a color
			if color, exists := d.clickedWords[word]; exists {
				style = style.WithFgRGB(color)
			}

			frame.PrintStyled(currentX, currentY, word, style)
			currentX += len(word)

			if i < len(words)-1 {
				frame.PrintStyled(currentX, currentY, " ", gooey.NewStyle())
				currentX += 1
			}
		}

		// Draw example sentence
		frame.PrintStyled(x, y+2, "   Try clicking the words above to see them change color!", gooey.NewStyle())
	})

	// Add mouse regions for each word
	x := 3 // Starting x position
	y := 3 // Line position (title is at y, words at y+1)

	for _, word := range words {
		wordCopy := word // Capture for closure
		region := &gooey.MouseRegion{
			X:      x,
			Y:      y,
			Width:  len(word),
			Height: 1,
			Label:  word,
			Handler: func(event *gooey.MouseEvent) {
				// Cycle through colors on click
				colors := []gooey.RGB{
					gooey.NewRGB(255, 100, 100),
					gooey.NewRGB(100, 255, 100),
					gooey.NewRGB(100, 100, 255),
					gooey.NewRGB(255, 255, 100),
					gooey.NewRGB(255, 100, 255),
					gooey.NewRGB(100, 255, 255),
				}

				// Get next color
				currentIdx := 0
				if existingColor, exists := d.clickedWords[wordCopy]; exists {
					for i, c := range colors {
						if c == existingColor {
							currentIdx = (i + 1) % len(colors)
							break
						}
					}
				}

				d.clickedWords[wordCopy] = colors[currentIdx]
				d.updateClickableText()
				d.addNotification(fmt.Sprintf("🎨 Colored '%s'", wordCopy))
			},
		}
		d.mouseHandler.AddRegion(region)
		x += len(word) + 1
	}

	// Trigger redraw
	d.screenManager.UpdateRegion("clickable", 0, "refresh", nil)
}

func (d *InteractiveDemo) drawComponent(c interface{ Draw(gooey.RenderFrame) }) {
	frame, err := d.terminal.BeginFrame()
	if err != nil {
		return
	}
	c.Draw(frame)
	d.terminal.EndFrame(frame)
}

func (d *InteractiveDemo) updateButtons() {
	// Trigger redraw of button region
	// The DrawCallback set in createButtons will handle the actual rendering
	d.screenManager.UpdateRegion("buttons", 0, "🔘 Interactive Buttons:", nil)
}

func (d *InteractiveDemo) updateRadioButtons() {
	// Trigger redraw of radio region
	// The DrawCallback set in createRadioGroup will handle the actual rendering
	d.screenManager.UpdateRegion("radio", 0, "🎨 Theme Selection:", nil)
}

func (d *InteractiveDemo) updateInput() {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Input prompt
	prompt := "$ "
	inputLine := prompt + d.currentInput

	// Show cursor position
	d.screenManager.UpdateRegion("input", 0, inputLine, nil)

	// Update cursor position
	d.screenManager.SetCursorPosition(len(prompt)+d.cursorPos, 13)
}

// drawOverlays is no longer needed as overlays are handled by ScreenManager callbacks
func (d *InteractiveDemo) unused_drawOverlays() {
	// Kept for reference but unused
}

func (d *InteractiveDemo) handleTabCompletion() {
	if d.currentInput == "" {
		return
	}

	// Find matching suggestions
	suggestions := []string{}
	for _, cmd := range []string{
		"build", "test", "run", "install", "clean",
		"deploy", "debug", "profile", "benchmark",
		"commit", "push", "pull", "merge", "rebase",
		"status", "diff", "log", "branch", "checkout",
	} {
		if strings.HasPrefix(cmd, d.currentInput) {
			suggestions = append(suggestions, cmd)
		}
	}

	if len(suggestions) > 0 {
		d.tabCompleter.SetSuggestions(suggestions, d.currentInput)
		d.tabCompleter.Show(2, 13, 40)

		// Add clickable regions for suggestions
		for _, region := range d.tabCompleter.GetRegions() {
			d.mouseHandler.AddRegion(region)
		}
	} else {
		d.tabCompleter.Hide()
	}
}

func (d *InteractiveDemo) updateColors() {
	// Update UI colors based on theme
	switch d.selectedTheme {
	case "Dark Mode":
		for _, btn := range d.buttons {
			btn.Style = gooey.NewStyle().WithBackground(gooey.ColorBlack).WithForeground(gooey.ColorWhite)
			btn.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorCyan)
		}
	case "High Contrast":
		for _, btn := range d.buttons {
			btn.Style = gooey.NewStyle().WithBackground(gooey.ColorYellow).WithForeground(gooey.ColorBlack)
			btn.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorWhite).WithForeground(gooey.ColorBlack)
		}
	case "Solarized":
		for _, btn := range d.buttons {
			btn.Style = gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack)
			btn.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorCyan).WithForeground(gooey.ColorBlack)
		}
	default:
		for _, btn := range d.buttons {
			btn.Style = gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite)
			btn.HoverStyle = gooey.NewStyle().WithBackground(gooey.ColorCyan).WithForeground(gooey.ColorBlack)
		}
	}
	d.updateButtons()
}

func (d *InteractiveDemo) addNotification(msg string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	notification := fmt.Sprintf("[%s] %s", timestamp, msg)

	d.notifications = append(d.notifications, notification)
	if len(d.notifications) > 3 {
		d.notifications = d.notifications[len(d.notifications)-3:]
	}

	// Update display
	for i := 0; i < 3; i++ {
		if i < len(d.notifications) {
			d.screenManager.UpdateRegion("notifications", i, d.notifications[i], nil)
		} else {
			d.screenManager.UpdateRegion("notifications", i, "", nil)
		}
	}
}

func (d *InteractiveDemo) Run() error {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
		d.terminal.DisableMouseTracking()
		d.terminal.DisableAlternateScreen()
		d.terminal.ShowCursor()
	}()

	// Clear screen
	d.terminal.Clear()

	// Refresh buttons and radio initially
	d.updateButtons()
	d.updateRadioButtons()

	// Main event loop
	buf := make([]byte, 20)
	for d.running {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		// Check for mouse events
		if buf[0] == 27 && n > 2 && buf[1] == '[' {
			if buf[2] == '<' {
				// SGR mouse event
				event, err := gooey.ParseMouseEvent(buf[2:n])
				if err == nil {
					d.mouseHandler.HandleEvent(event)

					// Redraw buttons after mouse event
					d.updateButtons()
					d.updateRadioButtons()
					continue
				}
			}

			// Arrow keys in tab completion
			if d.tabCompleter.Visible && n >= 3 {
				switch buf[2] {
				case 'A': // Up
					d.tabCompleter.SelectPrev()
				case 'B': // Down
					d.tabCompleter.SelectNext()
				}
				continue
			}
		}

		// Regular keyboard input
		switch buf[0] {
		case 3: // Ctrl+C
			d.running = false

		case 9: // TAB
			if d.tabCompleter.Visible {
				// Accept current selection
				selected := d.tabCompleter.GetSelected()
				if selected != "" {
					d.currentInput = selected + " "
					d.cursorPos = len(d.currentInput)
					d.tabCompleter.Hide()
					d.updateInput()
				}
			} else {
				// Show completions
				d.handleTabCompletion()
			}

		case 27: // ESC
			if n == 1 {
				// Just ESC - hide completions
				d.tabCompleter.Hide()
				// Clear completion regions
				d.mouseHandler.ClearRegions()
				// Re-add persistent regions
				d.createButtons()
				d.createRadioGroup()
				d.updateClickableText()
			}

		case 13: // Enter
			if d.tabCompleter.Visible {
				// Accept completion
				selected := d.tabCompleter.GetSelected()
				if selected != "" {
					d.currentInput = selected + " "
					d.cursorPos = len(d.currentInput)
				}
				d.tabCompleter.Hide()
			} else if d.currentInput != "" {
				// Execute command
				d.commandHistory = append(d.commandHistory, d.currentInput)
				d.addNotification("▶ Executed: " + d.currentInput)
				d.currentInput = ""
				d.cursorPos = 0
			}
			d.updateInput()

		case 127, 8: // Backspace
			if d.cursorPos > 0 {
				d.currentInput = d.currentInput[:d.cursorPos-1] + d.currentInput[d.cursorPos:]
				d.cursorPos--
				d.updateInput()

				// Update completions if Visible
				if d.tabCompleter.Visible {
					d.handleTabCompletion()
				}
			}

		default:
			// Regular character
			if buf[0] >= 32 && buf[0] < 127 {
				d.currentInput = d.currentInput[:d.cursorPos] + string(buf[0]) + d.currentInput[d.cursorPos:]
				d.cursorPos++
				d.updateInput()

				// Hide completions on new input
				if d.tabCompleter.Visible {
					d.tabCompleter.Hide()
				}
			}
		}
	}

	return nil
}

func (d *InteractiveDemo) Cleanup() {
	d.running = false
	d.terminal.StopWatchResize()
	d.screenManager.Stop()
}

func main() {
	fmt.Println("\n🖱️  Interactive Gooey Demo")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\nThis demo showcases:")
	fmt.Println("✅ Mouse support - click on words to change colors")
	fmt.Println("✅ Interactive buttons with hover effects")
	fmt.Println("✅ Tab completion with dropdown suggestions")
	fmt.Println("✅ Radio button groups for selection")
	fmt.Println("✅ Real-time notifications")
	fmt.Println("\nStarting in 2 seconds...")
	time.Sleep(2 * time.Second)

	demo, err := NewInteractiveDemo()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	demo.Setup()
	defer demo.Cleanup()

	if err := demo.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\n✨ Thanks for trying the interactive features!")
}
