package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

// ShiftEnterApp - demo for Shift+Enter detection
type ShiftEnterApp struct {
	terminal   *gooey.Terminal
	keyHistory []string
}

func (app *ShiftEnterApp) HandleEvent(event gooey.Event) []gooey.Cmd {
	switch e := event.(type) {
	case gooey.KeyEvent:
		// Debug: log all key events
		debug := fmt.Sprintf("Key=%d Rune=%q Shift=%v Ctrl=%v Alt=%v",
			e.Key, e.Rune, e.Shift, e.Ctrl, e.Alt)

		// Keep history of last 10 keys
		app.keyHistory = append(app.keyHistory, debug)
		if len(app.keyHistory) > 10 {
			app.keyHistory = app.keyHistory[1:]
		}

		// Only quit on Ctrl+C or Escape
		if e.Key == gooey.KeyCtrlC || e.Key == gooey.KeyEscape {
			return []gooey.Cmd{gooey.Quit()}
		}
	}
	return nil
}

func (app *ShiftEnterApp) Render(frame gooey.RenderFrame) {
	width, height := frame.Size()
	frame.FillStyled(0, 0, width, height, ' ', gooey.NewStyle())

	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()
	okStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen)
	highlightStyle := gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold()
	infoStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)
	dimStyle := gooey.NewStyle().WithForeground(gooey.ColorBrightBlack)

	y := 0

	frame.PrintStyled(0, y, "Shift+Enter Detection Demo", titleStyle)
	y += 2

	// Kitty protocol status
	supported := app.terminal.IsKittyProtocolSupported()
	enabled := app.terminal.IsKittyProtocolEnabled()
	frame.PrintStyled(0, y, fmt.Sprintf("Kitty Protocol: supported=%v enabled=%v", supported, enabled), dimStyle)
	y += 2

	// Instructions
	frame.PrintStyled(0, y, "Test Shift+Enter detection:", gooey.NewStyle().WithBold())
	y++
	frame.PrintStyled(2, y, "1. Press Shift+Enter (native - may not work in all terminals)", infoStyle)
	y++
	frame.PrintStyled(2, y, "2. Type \\ then Enter (backslash+Enter fallback - works everywhere)", infoStyle)
	y += 2

	// Check if Shift+Enter was detected
	shiftEnterWorks := false
	for _, k := range app.keyHistory {
		if strings.Contains(k, "Key=1") && strings.Contains(k, "Shift=true") {
			shiftEnterWorks = true
			break
		}
	}

	if shiftEnterWorks {
		frame.PrintStyled(0, y, "SUCCESS: Shift+Enter detected!", okStyle)
	} else {
		frame.PrintStyled(0, y, "Try typing: \\ then Enter (backslash followed by Enter)", infoStyle)
	}
	y += 2

	frame.PrintStyled(0, y, "Press Ctrl+C or Esc to quit", dimStyle)
	y += 2

	frame.PrintStyled(0, y, "Key History:", gooey.NewStyle().WithBold())
	y++

	for _, key := range app.keyHistory {
		style := gooey.NewStyle()
		// Highlight if Shift=true and Key=1 (Enter)
		if strings.Contains(key, "Key=1") && strings.Contains(key, "Shift=true") {
			style = highlightStyle
		}
		frame.PrintStyled(2, y, key, style)
		y++
	}
}

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create terminal: %v\n", err)
		os.Exit(1)
	}
	defer terminal.Close()

	app := &ShiftEnterApp{
		terminal:   terminal,
		keyHistory: []string{},
	}

	runtime := gooey.NewRuntime(terminal, app, 30)

	if err := runtime.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
