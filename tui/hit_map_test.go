package tui

import (
	"fmt"
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

func TestHitMapFindsAreas(t *testing.T) {
	var hits HitMap
	view := Stack(
		hits.Area("header", Text("Header")),
		hits.Area("list", Stack(
			hits.Area("row:1", Text("one")),
			hits.Area("row:2", Text("two")),
		)),
	)
	SprintScreen(view, WithWidth(20), WithHeight(5))

	key, ok := hits.At(2, 0)
	assert.True(t, ok)
	assert.Equal(t, key, "header")
	// Nested areas: the innermost wins.
	key, _ = hits.At(1, 2)
	assert.Equal(t, key, "row:2")
	b, ok := hits.Bounds("row:1")
	assert.True(t, ok)
	assert.Equal(t, b.Min, image.Pt(0, 1))
	_, ok = hits.At(0, 4)
	assert.False(t, ok)

	hits.Reset()
	_, ok = hits.At(2, 0)
	assert.False(t, ok)
	_, ok = hits.Bounds("header")
	assert.False(t, ok)
}

func TestHitMapKeepsFlex(t *testing.T) {
	var hits HitMap
	view := Stack(
		Text("top"),
		hits.Area("fill", Stack(Text("middle")).Flex(1)),
		Text("bottom"),
	)
	screen := SprintScreen(view, WithWidth(10), WithHeight(5))
	assert.Equal(t, screen.Row(4), "bottom")
	b, ok := hits.Bounds("fill")
	assert.True(t, ok)
	assert.Equal(t, b.Dy(), 3)
}

// hitSourceItems wraps each source-bound text item in a HitMap area.
type hitSourceItems struct {
	hits *HitMap
	text []string
}

func (h *hitSourceItems) Len() int                    { return len(h.text) }
func (h *hitSourceItems) Source(i int) (string, bool) { return h.text[i], true }
func (h *hitSourceItems) Item(i int) View {
	return h.hits.Area(fmt.Sprintf("row:%d", i), Text("%s", h.text[i]).SourceOffset(0))
}

func TestHitMapAreaInsideViewportWithSelection(t *testing.T) {
	var hits HitMap
	items := &hitSourceItems{hits: &hits, text: []string{"alpha", "beta"}}
	state := &ViewportState{}
	view := Stack(Height(5, Text("")), Height(3, Viewport(state, items)))
	SprintScreen(view, WithWidth(20), WithHeight(8))
	state.SelectLine(0, 0) // viewport-local coordinates
	assert.Equal(t, state.SelectedText(), "alpha", "an area around a source-bound view keeps its text")

	// Rendering with a selection paints it off-screen; that must not move
	// the areas recorded on screen.
	hits.Reset()
	state.Invalidate(0) // the item changed, so the paint must render it again
	SprintScreen(view, WithWidth(20), WithHeight(8))
	b, ok := hits.Bounds("row:0")
	assert.True(t, ok)
	assert.Equal(t, b, image.Rect(0, 5, 20, 6))
	key, _ := hits.At(0, 0)
	assert.Equal(t, key, "")
}

func TestCellsView(t *testing.T) {
	plain := NewStyle()
	red := NewStyle().WithForeground(ColorRed)
	rows := [][]Cell{
		{{Char: 'h', Width: 1, Style: red}, {Char: 'i', Width: 1, Style: plain}, {Char: ' ', Width: 1}},
		{{Char: '日', Width: 2, Style: plain}, {Continuation: true}, {Char: 'x', Width: 1}},
		{{Char: 'e', Trailing: "́", Width: 1}, {}, {Char: '本', Width: 2}},
	}
	screen := SprintScreen(Cells(rows).Cursor(1, 0), WithWidth(3), WithHeight(3))
	assert.Equal(t, screen.Row(0), "hi")
	assert.Equal(t, screen.Row(1), "日x")
	// A cell with no character is a space, and half a wide character that
	// does not fit is left blank.
	assert.Equal(t, screen.Row(2), "e\u0301")
	assert.Equal(t, screen.Cell(0, 0).Style.Foreground, termtest.Color{Type: termtest.ColorBasic, Value: 1})
	assert.True(t, screen.Cell(1, 0).Style.Reverse)

	w, h := Measure(Cells(rows), 0, 0)
	assert.Equal(t, w, 3)
	assert.Equal(t, h, 3)
	w, h = Measure(Cells(rows), 2, 1)
	assert.Equal(t, w, 2)
	assert.Equal(t, h, 1)
}

func TestHitMapAreaAroundSpacerKeepsGroupFixed(t *testing.T) {
	var hits HitMap
	// A Group holding a Spacer is not flexible; wrapping the Spacer must not
	// change that, or the bar would take half the Stack.
	bar := Group(Text("left"), hits.Area("gap", Spacer()), Text("right"))
	view := Stack(bar, Stack(Text("body")).Flex(1), Text("footer"))
	screen := SprintScreen(view, WithWidth(20), WithHeight(6))
	assert.Equal(t, screen.Row(0), "left           right")
	assert.Equal(t, screen.Row(1), "body")
	assert.Equal(t, screen.Row(5), "footer")
}

func TestHitMapAreaAroundWrappedTextKeepsReflow(t *testing.T) {
	var hits HitMap
	// Wrapped text in a flex row reflows to the width left over. Wrapping it
	// in an area must not make it clip at its natural width instead.
	text := func() *TextView { return Text("alpha beta gamma delta").Wrap() }
	plain := SprintScreen(Group(Text("label: "), text().Flex(1)), WithWidth(20), WithHeight(4))
	wrapped := SprintScreen(Group(Text("label: "), hits.Area("text", text().Flex(1))), WithWidth(20), WithHeight(4))
	assert.Equal(t, wrapped.Text(), plain.Text())
	assert.Equal(t, plain.Row(1), "       gamma delta")
}
