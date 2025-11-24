package main

import (
	"fmt"
	"os"

	"github.com/deepnoodle-ai/gooey"
	"golang.org/x/term"
)

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		panic(err)
	}

	// Setup raw mode for mouse input
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() {
		term.Restore(int(os.Stdin.Fd()), oldState)
		terminal.DisableMouseTracking()
		terminal.DisableAlternateScreen()
		terminal.ShowCursor()
	}()

	terminal.EnableAlternateScreen()
	terminal.EnableMouseTracking()
	terminal.HideCursor()

	sm := gooey.NewScreenManager(terminal, 30)
	mouse := gooey.NewMouseHandler()

	width, _ := terminal.Size()

	// Title
	sm.DefineRegion("title", 0, 0, width, 2, false)
	sm.UpdateRegion("title", 0, "🖱️  Mouse Grid Demo", gooey.CreateRainbowText("🖱️  Mouse Grid Demo", 15))
	sm.UpdateRegion("title", 1, "Click cells to toggle colors! Press Ctrl+C to exit.", nil)

	// Grid Configuration
	gridW, gridH := 5, 5
	cellW, cellH := 6, 3
	startX, startY := 4, 4

	// State
	gridState := make([][]int, gridH)
	for i := range gridState {
		gridState[i] = make([]int, gridW)
	}

	colors := []gooey.Style{
		gooey.NewStyle().WithBackground(gooey.ColorBrightBlack).WithForeground(gooey.ColorWhite), // Off
		gooey.NewStyle().WithBackground(gooey.ColorRed).WithForeground(gooey.ColorWhite),
		gooey.NewStyle().WithBackground(gooey.ColorGreen).WithForeground(gooey.ColorBlack),
		gooey.NewStyle().WithBackground(gooey.ColorBlue).WithForeground(gooey.ColorWhite),
		gooey.NewStyle().WithBackground(gooey.ColorYellow).WithForeground(gooey.ColorBlack),
	}

	// Initialize Grid Regions and Mouse Handlers
	for y := 0; y < gridH; y++ {
		for x := 0; x < gridW; x++ {
			regionName := fmt.Sprintf("cell_%d_%d", x, y)
			screenX := startX + (x * (cellW + 1))
			screenY := startY + (y * (cellH + 1))

			sm.DefineRegion(regionName, screenX, screenY, cellW, cellH, false)

			// Capture loop variables
			cx, cy := x, y

			mouseRegion := &gooey.MouseRegion{
				X:      screenX,
				Y:      screenY,
				Width:  cellW,
				Height: cellH,
				Label:  regionName,
				Handler: func(e *gooey.MouseEvent) {
					if e.Type == gooey.MouseClick {
						// Toggle color
						gridState[cy][cx] = (gridState[cy][cx] + 1) % len(colors)
						updateCell(sm, regionName, gridState[cy][cx], colors, cellH)
					}
				},
			}
			mouse.AddRegion(mouseRegion)

			// Initial draw
			updateCell(sm, regionName, 0, colors, cellH)
		}
	}

	sm.Start()
	defer sm.Stop()

	// Event loop
	buf := make([]byte, 128)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}

		if n > 0 {
			if buf[0] == 3 { // Ctrl+C
				return
			}

			// Handle Mouse
			if buf[0] == 27 && n > 2 && buf[1] == '[' && buf[2] == '<' {
				event, err := gooey.ParseMouseEvent(buf[2:n])
				if err == nil {
					mouse.HandleEvent(event)
				}
			}
		}
	}
}

func updateCell(sm *gooey.ScreenManager, region string, colorIdx int, colors []gooey.Style, height int) {
	style := colors[colorIdx]

	// Create a block of color
	// We use full block characters or just background color with spaces
	content := "      " // 6 spaces matches cellW

	// Apply style to the content string (this is a bit of a hack since ScreenManager
	// expects content + optional animation, but we want static styled blocks.
	// ScreenManager supports PrintStyled but UpdateRegion takes string.
	// We'll rely on the fact that we can pass styled strings if we don't use animations,
	// OR we can use a dummy animation that just sets color?
	// Actually ScreenManager's UpdateRegion takes `content string`.
	// If we pass an ANSI string, it might work if length calc handles it.
	// But ScreenManager uses `utf8.RuneCountInString` which counts ANSI chars as runes usually if not stripped.
	// gooey's ScreenManager uses `PrintStyled` which applies style ON TOP of content.
	// But UpdateRegion doesn't take a Style, only Animation.
	//
	// Workaround: Use a static animation that returns the fixed style for all frames.

	staticAnim := &StaticStyleAnimation{Style: style}

	for i := 0; i < height; i++ {
		sm.UpdateRegion(region, i, content, staticAnim)
	}
}

// StaticStyleAnimation implements TextAnimation but returns a constant style
type StaticStyleAnimation struct {
	Style gooey.Style
}

func (a *StaticStyleAnimation) GetStyle(frame uint64, index, length int) gooey.Style {
	return a.Style
}
