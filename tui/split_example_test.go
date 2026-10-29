package tui_test

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/tui"
)

// ExampleSplitState lays out an editor beside a shell, with logs below the
// shell. Tight lets the bordered panes touch.
func ExampleSplitState() {
	split := &tui.SplitState{Tight: true}
	split.Append("editor", tui.SplitColumns)
	split.Append("shell", tui.SplitColumns)     // a new column on the right
	split.Split("shell", "logs", tui.SplitRows) // logs below the shell

	view := tui.Split(split, func(id string) tui.View {
		return tui.Bordered(tui.Text("%s", id)).Border(&tui.RoundedBorder)
	})
	screen := tui.SprintScreen(view, tui.WithWidth(24), tui.WithHeight(6))
	for y := range 6 {
		fmt.Println(screen.Row(y))
	}
	b, _ := split.Bounds("logs")
	fmt.Println("logs:", b)
	// Output:
	// ╭──────────╮╭──────────╮
	// │editor    ││shell     │
	// │          │╰──────────╯
	// │          │╭──────────╮
	// │          ││logs      │
	// ╰──────────╯╰──────────╯
	// logs: (12,3)-(24,6)
}
