package tui

import (
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
