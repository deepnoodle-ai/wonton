package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/deepnoodle-ai/gooey"
)

func main() {
	terminal, err := gooey.NewTerminal()
	if err != nil {
		log.Fatal(err)
	}
	defer terminal.Close()

	terminal.EnableAlternateScreen()
	terminal.HideCursor()
	defer terminal.ShowCursor()
	terminal.Clear()

	titleStyle := gooey.NewStyle().WithForeground(gooey.ColorCyan).WithBold()

	frame, _ := terminal.BeginFrame()
	frame.PrintStyled(0, 0, "╔════════════════════════════════════════════════════════════╗", titleStyle)
	frame.PrintStyled(0, 1, "║        Bracketed Paste Mode - Security Demonstration      ║", titleStyle)
	frame.PrintStyled(0, 2, "╚════════════════════════════════════════════════════════════╝", titleStyle)
	frame.PrintStyled(0, 4, "This demonstrates the security benefit of bracketed paste.", gooey.NewStyle())
	frame.PrintStyled(0, 5, "", gooey.NewStyle())
	frame.PrintStyled(0, 6, "Try pasting this malicious command:", gooey.NewStyle().WithForeground(gooey.ColorYellow))
	frame.PrintStyled(0, 7, "  echo 'safe command'", gooey.NewStyle().WithForeground(gooey.ColorGreen))
	frame.PrintStyled(0, 8, "  rm -rf /", gooey.NewStyle().WithForeground(gooey.ColorRed))
	frame.PrintStyled(0, 9, "", gooey.NewStyle())
	frame.PrintStyled(0, 10, "WITHOUT bracketed paste: Both commands execute immediately!", gooey.NewStyle().WithForeground(gooey.ColorRed))
	frame.PrintStyled(0, 11, "WITH bracketed paste: Newlines are preserved in input buffer.", gooey.NewStyle().WithForeground(gooey.ColorGreen))
	frame.PrintStyled(0, 12, "                       You can review before pressing Enter.", gooey.NewStyle().WithForeground(gooey.ColorGreen))
	terminal.EndFrame(frame)

	// Enable raw mode and bracketed paste
	terminal.EnableRawMode()
	terminal.EnableBracketedPaste()
	defer terminal.DisableBracketedPaste()

	input := gooey.NewInput(terminal)
	input.EnableMultiline()

	terminal.MoveCursor(0, 14)
	input.WithPrompt("Paste here (Ctrl+Enter to submit): ", gooey.NewStyle().WithForeground(gooey.ColorCyan))

	result, err := input.Read()
	if err != nil {
		return
	}

	// Show what was received
	terminal.Clear()
	frame2, _ := terminal.BeginFrame()
	frame2.PrintStyled(0, 0, "╔════════════════════════════════════════════════════════════╗", titleStyle)
	frame2.PrintStyled(0, 1, "║                    Paste Received                          ║", titleStyle)
	frame2.PrintStyled(0, 2, "╚════════════════════════════════════════════════════════════╝", titleStyle)
	frame2.PrintStyled(0, 4, "The following content was pasted:", gooey.NewStyle())

	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lineStyle := gooey.NewStyle().WithForeground(gooey.ColorWhite)
		if strings.Contains(line, "rm") {
			lineStyle = gooey.NewStyle().WithForeground(gooey.ColorRed).WithBold()
		}
		frame2.PrintStyled(0, 6+i, fmt.Sprintf("  Line %d: %s", i+1, line), lineStyle)
	}

	frame2.PrintStyled(0, 6+len(lines)+2, "✓ Bracketed paste prevented immediate execution!", gooey.NewStyle().WithForeground(gooey.ColorGreen).WithBold())
	frame2.PrintStyled(0, 6+len(lines)+3, "  You can now review the content before deciding to execute.", gooey.NewStyle())
	frame2.PrintStyled(0, 6+len(lines)+5, "Press any key to exit...", gooey.NewStyle().WithForeground(gooey.ColorBrightBlack))
	terminal.EndFrame(frame2)

	input.ReadKeyEvent()
}
