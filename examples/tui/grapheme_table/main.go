// Package main demonstrates grapheme-aware column alignment in the Table
// view. Every row mixes CJK ideographs, emoji, regional-indicator flags, and
// VS16 glyphs — exactly the values that make width-by-bytes or width-by-runes
// approaches misalign columns. With wonton's runewidth-backed measurement,
// the right border of each column should stay flush no matter which row is
// visible and which row is selected.
//
// What to look for:
//
//   - The "Name" and "City" columns should line up on the same terminal
//     column for every row, regardless of the emoji or CJK content.
//   - Selecting a row (arrow keys) must not shift the following columns.
//   - Resizing the terminal mid-run should keep the table flush.
package main

import (
	"log"

	"github.com/deepnoodle-ai/wonton/tui"
)

type App struct {
	columns  []tui.TableColumn
	rows     [][]string
	selected int
}

func (app *App) View() tui.View {
	return tui.Stack(
		tui.Text(" Grapheme-aware Table ").Bold().Bg(tui.ColorBlue).Fg(tui.ColorWhite),
		tui.Divider(),
		tui.Spacer().MinHeight(1),
		tui.Table(app.columns, &app.selected).
			Rows(app.rows).
			Height(len(app.rows)+3).
			UppercaseHeaders(true).
			SelectedBg(tui.ColorBlue).
			SelectedFg(tui.ColorWhite),
		tui.Spacer().MinHeight(1),
		tui.Text("Arrow keys to move selection. Each column's right edge "+
			"should stay flush across every row.").Dim(),
		tui.Text("Press q to quit.").Dim(),
	).Padding(1)
}

func (app *App) HandleEvent(event tui.Event) []tui.Cmd {
	if e, ok := event.(tui.KeyEvent); ok {
		if e.Rune == 'q' || e.Rune == 'Q' || e.Key == tui.KeyEscape || e.Key == tui.KeyCtrlC {
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

func main() {
	columns := []tui.TableColumn{
		{Title: "ID", Width: 4},
		{Title: "Name", Width: 18},
		{Title: "City", Width: 16},
		{Title: "Mood", Width: 8},
	}
	rows := [][]string{
		{"1", "Alice", "Tokyo 東京", "\u2764\uFE0F"},                         // VS16 heart
		{"2", "Björn", "Reykjavík", "\U0001F44B\U0001F3FD"},                // skin-toned wave
		{"3", "千尋", "京都 Kyōto", "\U0001F1EF\U0001F1F5"},                    // Japan flag
		{"4", "Chen 晨", "上海 Shanghai", "\U0001F3F3\uFE0F\u200D\U0001F308"}, // rainbow flag
		{"5", "Ravi", "चेन्नई", "\U0001F9D1\U0001F3FD\u200D\U0001F91D\u200D\U0001F9D1\U0001F3FE"},
		{"6", "Family", "Berlin 🇩🇪", "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"},
		{"7", "한글", "서울 Seoul", "\u2600\uFE0F"}, // VS16 sun
	}
	if err := tui.Run(&App{columns: columns, rows: rows}); err != nil {
		log.Fatal(err)
	}
}
