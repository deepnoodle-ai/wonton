package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/deepnoodle-ai/gooey"
	"golang.org/x/term"
)

// Simplified File Picker Demo
//
// This demonstrates the FilePicker widget with minimal complexity.
// Features:
// - Type to filter files (fuzzy matching)
// - Arrow keys to navigate
// - Enter to select file or open directory
// - H to toggle hidden files
// - Mouse click to select
// - Ctrl+C to exit

func main() {
	// Initialize terminal
	t, err := gooey.NewTerminal()
	if err != nil {
		log.Fatalf("Failed to initialize terminal: %v", err)
	}
	defer t.Close()

	// Set up raw mode for input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Fatalf("Failed to enter raw mode: %v", err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Enable terminal features
	t.EnableAlternateScreen()
	defer t.DisableAlternateScreen()
	t.HideCursor()
	defer t.ShowCursor()
	t.EnableMouseTracking()
	defer t.DisableMouseTracking()

	width, height := t.Size()

	// Create file picker
	pwd, _ := os.Getwd()
	picker := gooey.NewFilePicker(pwd)

	// Make the input more visible with a clear background and bold text
	picker.SetInputStyle(gooey.NewStyle().
		WithBackground(gooey.ColorBlue).
		WithForeground(gooey.ColorYellow).
		WithBold())

	picker.Init() // Initialize once at startup

	// Explicitly ensure input is focused
	picker.FocusInput()

	// Set initial bounds for picker
	pickerHeight := height - 4 // Leave room for title, separator, and status
	if pickerHeight < 5 {
		pickerHeight = 5
	}
	picker.SetBounds(image.Rect(0, 0, width, pickerHeight))

	// Status message
	statusMsg := "Ready - Type to filter, arrows to navigate, Enter to select, H to toggle hidden files, Ctrl+C to exit"

	// Update status when file is selected
	picker.OnSelect = func(path string) {
		info, err := os.Stat(path)
		if err != nil {
			statusMsg = fmt.Sprintf("Error: %v", err)
		} else {
			if info.IsDir() {
				statusMsg = fmt.Sprintf("Opened directory: %s", path)
			} else {
				size := info.Size()
				var sizeStr string
				if size < 1024 {
					sizeStr = fmt.Sprintf("%d B", size)
				} else if size < 1024*1024 {
					sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
				} else {
					sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
				}
				statusMsg = fmt.Sprintf("Selected: %s (%s)", filepath.Base(path), sizeStr)
			}
		}
	}

	// Draw function
	draw := func() {
		frame, err := t.BeginFrame()
		if err != nil {
			return
		}

		// Draw title (line 0)
		title := "FILE PICKER DEMO"
		titleX := (width - len(title)) / 2
		if titleX < 0 {
			titleX = 0
		}
		frame.PrintStyled(titleX, 0, title, gooey.NewStyle().WithBold().WithForeground(gooey.ColorCyan))

		// Draw separator (line 1)
		frame.PrintStyled(0, 1, fmt.Sprintf("%"+fmt.Sprintf("%d", width)+"s", ""), gooey.NewStyle().WithBackground(gooey.ColorBrightBlack))

		// Draw file picker (lines 2 to height-3)
		// Note: bounds are set once at startup, not every frame
		bounds := picker.GetBounds()
		pickerFrame := frame.SubFrame(image.Rect(0, 2, width, 2+bounds.Dy()))
		picker.Draw(pickerFrame)

		// Draw separator before status (line height-2)
		frame.PrintStyled(0, height-2, fmt.Sprintf("%"+fmt.Sprintf("%d", width)+"s", ""), gooey.NewStyle().WithBackground(gooey.ColorBrightBlack))

		// Draw status (line height-1)
		statusText := statusMsg
		if len(statusText) > width {
			statusText = statusText[:width-3] + "..."
		}

		// Add filter debug info
		inputValue := picker.GetInputValue()
		inputFocused := picker.GetInputFocused()
		inputCursor := picker.GetInputCursorPos()
		filterDebug := fmt.Sprintf(" | Focused:%v Cursor:%d Val:'%s' Flt:'%s'",
			inputFocused, inputCursor, inputValue, picker.Filter)
		if len(statusText)+len(filterDebug) < width {
			statusText += filterDebug
		}

		frame.PrintStyled(0, height-1, statusText, gooey.NewStyle().WithForeground(gooey.ColorGreen))

		t.EndFrame(frame)
	}

	// Initial draw
	draw()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Input channel
	inputChan := make(chan []byte, 10)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				inputChan <- data
			}
		}
	}()

	// Main event loop
	for {
		select {
		case <-sigChan:
			return

		case data := <-inputChan:
			if len(data) == 0 {
				continue
			}

			// Ctrl+C
			if data[0] == 3 {
				return
			}

			// Handle mouse events
			if len(data) > 2 && data[0] == 0x1b && data[1] == '[' && data[2] == '<' {
				event, err := gooey.ParseMouseEvent(data[2:])
				if err == nil {
					picker.HandleMouse(*event)
					draw()
				}
				continue
			}

			// Handle keyboard events
			var key gooey.KeyEvent

			if len(data) == 1 {
				switch data[0] {
				case 13: // Enter
					key = gooey.KeyEvent{Key: gooey.KeyEnter}
				case 127: // Backspace
					key = gooey.KeyEvent{Key: gooey.KeyBackspace}
				case 'h', 'H': // Toggle hidden files
					picker.ShowHidden = !picker.ShowHidden
					picker.Refresh()
					if picker.ShowHidden {
						statusMsg = "Hidden files: ON"
					} else {
						statusMsg = "Hidden files: OFF"
					}
					draw()
					continue
				default:
					if data[0] >= 32 {
						key = gooey.KeyEvent{Rune: rune(data[0])}
					}
				}
			} else if len(data) >= 3 && data[0] == 0x1b && data[1] == '[' {
				switch data[2] {
				case 'A':
					key = gooey.KeyEvent{Key: gooey.KeyArrowUp}
				case 'B':
					key = gooey.KeyEvent{Key: gooey.KeyArrowDown}
				case 'H':
					key = gooey.KeyEvent{Key: gooey.KeyHome}
				case 'F':
					key = gooey.KeyEvent{Key: gooey.KeyEnd}
				case '5':
					if len(data) >= 4 && data[3] == '~' {
						key = gooey.KeyEvent{Key: gooey.KeyPageUp}
					}
				case '6':
					if len(data) >= 4 && data[3] == '~' {
						key = gooey.KeyEvent{Key: gooey.KeyPageDown}
					}
				}
			}

			if key.Key != 0 || key.Rune != 0 {
				handled := picker.HandleKey(key)

				// WORKAROUND: Manually update the input value since HandleKey isn't working
				if key.Rune != 0 && key.Rune >= 32 {
					currentVal := picker.GetInputValue()
					newVal := currentVal + string(key.Rune)
					picker.SetInputValueDirect(newVal)
					statusMsg = fmt.Sprintf("MANUAL UPDATE: added '%c' to '%s' = '%s'", key.Rune, currentVal, newVal)
				} else if key.Key == gooey.KeyBackspace {
					currentVal := picker.GetInputValue()
					if len(currentVal) > 0 {
						newVal := currentVal[:len(currentVal)-1]
						picker.SetInputValueDirect(newVal)
						statusMsg = fmt.Sprintf("MANUAL UPDATE: backspace '%s' = '%s'", currentVal, newVal)
					}
				} else if handled {
					statusMsg = fmt.Sprintf("Key HANDLED: keycode %d", key.Key)
				} else {
					statusMsg = fmt.Sprintf("Key NOT handled")
				}
				draw()
			}
		}
	}
}
