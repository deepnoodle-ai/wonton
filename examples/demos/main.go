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

type SynchronizedDemo struct {
	terminal      *gooey.Terminal
	screenManager *gooey.ScreenManager
	lines         [2]string
	selectedLine  int
	cursorPos     [2]int
	running       bool
	mu            sync.Mutex
}

func NewSynchronizedDemo() (*SynchronizedDemo, error) {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		return nil, err
	}

	screenManager := gooey.NewScreenManager(terminal, 30)

	return &SynchronizedDemo{
		terminal:      terminal,
		screenManager: screenManager,
		lines:         [2]string{"", ""},
		selectedLine:  0,
		cursorPos:     [2]int{0, 0},
		running:       true,
	}, nil
}

func (d *SynchronizedDemo) Setup() {
	// Enable alternate screen and show cursor
	d.terminal.EnableAlternateScreen()
	d.terminal.ShowCursor()

	width, height := d.terminal.Size()

	// Define screen regions
	// Header region (lines 0-1)
	d.screenManager.DefineRegion("header", 0, 0, width, 2, false)

	// Animated content region (lines 2-5)
	d.screenManager.DefineRegion("content", 0, 2, width, 4, false)

	// Input region (lines 6-7) - PROTECTED from animation overwrites
	d.screenManager.DefineRegion("input", 0, 6, width, 2, true)

	// Footer region (lines 8+)
	footerY := 8
	footerHeight := height - footerY - 1
	if footerHeight > 3 {
		footerHeight = 3
	}
	d.screenManager.DefineRegion("footer", 0, footerY, width, footerHeight, false)

	// Set initial content
	d.screenManager.UpdateRegion("header", 0, "🚀 Synchronized Gooey Demo", gooey.CreateRainbowText("🚀 Synchronized Gooey Demo", 20))
	d.screenManager.UpdateRegion("header", 1, "All updates properly synchronized!", gooey.CreateReverseRainbowText("All updates properly synchronized!", 25))

	d.screenManager.UpdateRegion("content", 0, "Status: Ready", gooey.CreatePulseText(gooey.NewRGB(0, 255, 100), 40))
	d.screenManager.UpdateRegion("content", 1, "Progress: ████████████████████", gooey.CreateRainbowText("Progress: ████████████████████", 15))
	d.screenManager.UpdateRegion("content", 2, "", nil)
	d.screenManager.UpdateRegion("content", 3, "System: Online", nil)

	d.screenManager.UpdateRegion("footer", 0, "↑↓ Switch lines | Type to add text | Ctrl+C to quit", nil)
	d.screenManager.UpdateRegion("footer", 1, "No more cursor jumping!", gooey.CreateRainbowText("No more cursor jumping!", 18))

	// Initial input lines
	d.UpdateInputLines()

	// Start the screen manager
	d.screenManager.Start()
}

func (d *SynchronizedDemo) UpdateInputLines() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i := 0; i < 2; i++ {
		prefix := fmt.Sprintf("Line %d: ", i+1)
		line := prefix + d.lines[i]

		if i == d.selectedLine {
			// Highlight selected line
			style := gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack)
			styledLine := style.Apply(line)
			d.screenManager.UpdateRegion("input", i, styledLine, nil)
		} else {
			// Normal line
			d.screenManager.UpdateRegion("input", i, line, nil)
		}
	}

	// Update cursor position in screen manager
	cursorX := 8 + d.cursorPos[d.selectedLine] // "Line N: " is 8 chars
	cursorY := 6 + d.selectedLine              // Input region starts at line 6
	d.screenManager.SetCursorPosition(cursorX, cursorY)
}

func (d *SynchronizedDemo) StartBackgroundUpdates() {
	go func() {
		counter := 0
		for d.running {
			time.Sleep(500 * time.Millisecond)
			counter++

			// Update status line
			statuses := []string{
				"Status: Initializing...",
				"Status: Loading...",
				"Status: Connecting...",
				"Status: Processing...",
				"Status: Optimizing...",
				"Status: Finalizing...",
				"Status: Complete!",
				"Status: Ready",
			}

			statusIdx := counter % len(statuses)
			statusText := statuses[statusIdx]

			var animation gooey.TextAnimation
			if statusIdx == 6 { // "Complete!" gets rainbow
				animation = gooey.CreateRainbowText(statusText, 15)
			} else {
				colors := []gooey.RGB{
					gooey.NewRGB(255, 255, 0),
					gooey.NewRGB(255, 165, 0),
					gooey.NewRGB(0, 255, 255),
					gooey.NewRGB(0, 255, 100),
					gooey.NewRGB(255, 0, 255),
					gooey.NewRGB(100, 255, 100),
					gooey.NewRGB(0, 255, 100),
					gooey.NewRGB(0, 255, 100),
				}
				animation = gooey.CreatePulseText(colors[statusIdx], 40)
			}

			d.screenManager.UpdateRegion("content", 0, statusText, animation)

			// Update progress bar
			progress := (counter % 20) + 1
			progressBar := strings.Repeat("█", progress) + strings.Repeat("░", 20-progress)
			progressText := fmt.Sprintf("Progress: %s %d%%", progressBar, progress*5)
			d.screenManager.UpdateRegion("content", 1, progressText, gooey.CreateRainbowText(progressText, 12))

			// Update system info
			connections := 40 + (counter % 10)
			d.screenManager.UpdateRegion("content", 3, fmt.Sprintf("System: %d connections", connections), nil)
		}
	}()
}

func (d *SynchronizedDemo) Run() error {
	// Set terminal to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
		d.terminal.DisableAlternateScreen()
		d.terminal.ShowCursor()
	}()

	// Clear and position cursor
	d.terminal.Clear()

	// Start background updates
	d.StartBackgroundUpdates()

	// Main input loop
	buf := make([]byte, 1)
	for d.running {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		switch buf[0] {
		case 3: // Ctrl+C
			d.running = false

		case 27: // Escape sequence (arrow keys)
			seq := make([]byte, 2)
			n, _ := os.Stdin.Read(seq)
			if n == 2 && seq[0] == '[' {
				switch seq[1] {
				case 'A': // Up arrow
					if d.selectedLine > 0 {
						d.selectedLine--
						d.UpdateInputLines()
					}
				case 'B': // Down arrow
					if d.selectedLine < 1 {
						d.selectedLine++
						d.UpdateInputLines()
					}
				case 'C': // Right arrow
					if d.cursorPos[d.selectedLine] < len(d.lines[d.selectedLine]) {
						d.cursorPos[d.selectedLine]++
						d.UpdateInputLines()
					}
				case 'D': // Left arrow
					if d.cursorPos[d.selectedLine] > 0 {
						d.cursorPos[d.selectedLine]--
						d.UpdateInputLines()
					}
				}
			}

		case 127, 8: // Backspace
			d.mu.Lock()
			line := d.lines[d.selectedLine]
			pos := d.cursorPos[d.selectedLine]
			if pos > 0 && len(line) > 0 {
				d.lines[d.selectedLine] = line[:pos-1] + line[pos:]
				d.cursorPos[d.selectedLine]--
			}
			d.mu.Unlock()
			d.UpdateInputLines()

		case 13: // Enter
			// Move to next line
			if d.selectedLine < 1 {
				d.selectedLine++
			} else {
				d.selectedLine = 0
			}
			d.UpdateInputLines()

		default:
			// Regular character input
			if buf[0] >= 32 && buf[0] < 127 {
				d.mu.Lock()
				line := d.lines[d.selectedLine]
				pos := d.cursorPos[d.selectedLine]
				d.lines[d.selectedLine] = line[:pos] + string(buf[0]) + line[pos:]
				d.cursorPos[d.selectedLine]++
				d.mu.Unlock()
				d.UpdateInputLines()
			}
		}
	}

	return nil
}

func (d *SynchronizedDemo) Cleanup() {
	d.running = false
	d.screenManager.Stop()
}

func main() {
	fmt.Println("\n🎨 Synchronized Gooey Demo")
	fmt.Println("This demo uses a ScreenManager to coordinate all drawing:")
	fmt.Println("✨ No cursor jumping")
	fmt.Println("🛡️  Protected input regions")
	fmt.Println("🎯 Proper synchronization")
	fmt.Println("\nStarting in 2 seconds...")
	time.Sleep(2 * time.Second)

	demo, err := NewSynchronizedDemo()
	if err != nil {
		fmt.Printf("Error creating demo: %v\n", err)
		return
	}

	demo.Setup()
	defer demo.Cleanup()

	if err := demo.Run(); err != nil {
		fmt.Printf("Error running demo: %v\n", err)
	}

	fmt.Println("\n👋 Thanks for trying the synchronized demo!")
}
