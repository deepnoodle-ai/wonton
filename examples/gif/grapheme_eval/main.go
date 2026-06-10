// Example: grapheme_eval characterizes the gif terminal/emulator behavior for
// grapheme clusters and renders a few sample GIFs for manual inspection.
//
// Run with:
//
//	go run ./examples/gif/grapheme_eval
//
// GIFs are written to the current directory unless an output directory is
// given as the first argument:
//
//	go run ./examples/gif/grapheme_eval /tmp/gif-graphemes
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/deepnoodle-ai/wonton/gif"
)

type sample struct {
	label string
	text  string
}

func main() {
	outDir := "."
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "create output dir: %v\n", err)
		os.Exit(1)
	}

	samples := []sample{
		{label: "heart_vs16", text: "\u2764\uFE0F"},
		{label: "hash_keycap", text: "#\uFE0F\u20E3"},
		{label: "jp_flag", text: "\U0001F1EF\U0001F1F5"},
		{label: "rainbow_flag", text: "\U0001F3F3\uFE0F\u200D\U0001F308"},
		{label: "family", text: "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"},
		{label: "wave_skin_tone", text: "\U0001F44B\U0001F3FD"},
	}

	for _, s := range samples {
		if err := renderDirect(outDir, s); err != nil {
			fmt.Fprintf(os.Stderr, "direct %s: %v\n", s.label, err)
			os.Exit(1)
		}
		if err := renderEmulated(outDir, s); err != nil {
			fmt.Fprintf(os.Stderr, "emulated %s: %v\n", s.label, err)
			os.Exit(1)
		}
	}
}

func renderDirect(outDir string, s sample) error {
	screen := gif.NewTerminalScreen(24, 4)
	screen.WriteString("direct: ", gif.White, gif.Black)
	screen.WriteString(s.text, gif.Yellow, gif.Black)
	screen.WriteString(" <-", gif.White, gif.Black)

	fmt.Printf("[direct] %s cursor=(%d,%d)\n", s.label, screen.CursorX, screen.CursorY)
	dumpCells(screen, 0, 16)

	renderer := gif.NewTerminalRenderer(screen, 8)
	renderer.RenderFrame(10)

	filename := filepath.Join(outDir, s.label+"_direct.gif")
	if err := renderer.Save(filename); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", filename)
	return nil
}

func renderEmulated(outDir string, s sample) error {
	em := gif.NewEmulator(24, 4)
	em.ProcessOutput("\x1b[33memu:\x1b[0m " + s.text + " <-")

	screen := em.Screen()
	fmt.Printf("[emulator] %s cursor=(%d,%d)\n", s.label, screen.CursorX, screen.CursorY)
	dumpCells(screen, 0, 16)

	renderer := gif.NewTerminalRenderer(screen, 8)
	renderer.RenderFrame(10)

	filename := filepath.Join(outDir, s.label+"_emulator.gif")
	if err := renderer.Save(filename); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", filename)
	return nil
}

func dumpCells(screen *gif.TerminalScreen, row, count int) {
	if row < 0 || row >= screen.Height {
		return
	}
	if count > screen.Width {
		count = screen.Width
	}
	for i := 0; i < count; i++ {
		cell := screen.Cells[row][i]
		fmt.Printf("  %02d: %q U+%04X\n", i, cell.Char, cell.Char)
	}
}
