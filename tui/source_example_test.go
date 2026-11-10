package tui_test

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/tui"
	"github.com/deepnoodle-ai/wonton/tui/tuitest"
)

type originalItems []string

func (i originalItems) Len() int                    { return len(i) }
func (i originalItems) Source(n int) (string, bool) { return i[n], true }
func (i originalItems) Item(n int) tui.View {
	return tui.PaddingLTRB(2, 0, 0, 0, tui.Markdown(i[n], nil).SourceOffset(0))
}

func ExampleViewportSourceItems() {
	items := originalItems{"**original**  世界"}
	state := &tui.ViewportState{}
	view := tui.Height(5, tui.Viewport(state, items))
	tuitest.Screen(view, tui.WithWidth(16))
	state.SelectLine(3, 0)
	fmt.Println(state.SelectedText())
	tuitest.Screen(view, tui.WithWidth(10))
	fmt.Println(state.SelectedText())
	// Output:
	// **original**  世界
	// **original**  世界
}
