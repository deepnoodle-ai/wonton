package tui

import (
	"image"
	"math"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// renderSplit draws the split at width x height, labelling each pane by
// filling it with the first letter of its ID.
func renderSplit(s *SplitState, width, height int) *termtest.Screen {
	view := Split(s, func(id string) View {
		return CanvasContext(func(ctx *RenderContext) {
			ctx.Fill(rune(id[0]), NewStyle())
		})
	})
	return SprintScreen(view, WithWidth(width), WithHeight(height))
}

func TestSplitNilStateRendersNothing(t *testing.T) {
	SprintScreen(Split(nil, func(string) View { return nil }), WithWidth(10), WithHeight(3))
}

func TestSplitIgnoresNonFiniteWeights(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	s.Root.Children[0].Weight = math.NaN()
	s.Root.Children[1].Weight = math.Inf(1)
	screen := renderSplit(s, 21, 2)
	assert.Equal(t, screen.Row(0), "aaaaaaaaaa bbbbbbbbbb")
}

func TestSplitAndAppendDropStaleGutters(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	renderSplit(s, 21, 5)
	s.HandleMouse(MouseEvent{Type: MouseMove, X: 10, Y: 2})
	assert.NotNil(t, s.hover)

	s.Split("a", "c", SplitRows)
	assert.Nil(t, s.hover, "the gutter belongs to the old tree")
	renderSplit(s, 21, 5)
	s.HandleMouse(MouseEvent{Type: MouseMove, X: 10, Y: 2})
	assert.NotNil(t, s.hover)
	s.Append("d", SplitRows)
	assert.Nil(t, s.hover)
}

func TestSplitLayoutColumnsAndRows(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	s.Split("b", "c", SplitRows)

	screen := renderSplit(s, 21, 5)
	assert.Equal(t, screen.Row(0), "aaaaaaaaaa bbbbbbbbbb")
	assert.Equal(t, screen.Row(1), "aaaaaaaaaa bbbbbbbbbb")
	assert.Equal(t, screen.Row(2), "aaaaaaaaaa")
	assert.Equal(t, screen.Row(3), "aaaaaaaaaa cccccccccc")

	assert.Equal(t, s.Leaves(), []string{"a", "b", "c"})
	b, ok := s.Bounds("b")
	assert.True(t, ok)
	assert.Equal(t, b, image.Rect(11, 0, 21, 2))
	c, _ := s.Bounds("c")
	assert.Equal(t, c, image.Rect(11, 3, 21, 5))

	id, ok := s.PaneAt(15, 4)
	assert.True(t, ok)
	assert.Equal(t, id, "c")
	_, ok = s.PaneAt(10, 0) // the gutter
	assert.False(t, ok)

	branch, index, ok := s.DividerAt(10, 2)
	assert.True(t, ok)
	assert.Equal(t, branch, s.Root)
	assert.Equal(t, index, 0)
	branch, index, ok = s.DividerAt(15, 2)
	assert.True(t, ok)
	assert.Equal(t, branch.Axis, SplitRows)
	assert.Equal(t, index, 0)
}

func TestSplitJoinsExistingRun(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Split("a", "b", SplitColumns)
	s.Split("a", "c", SplitColumns)
	// a's neighbors sit in the same row rather than in nested branches.
	assert.Equal(t, len(s.Root.Children), 3)
	assert.Equal(t, s.Leaves(), []string{"a", "c", "b"})
	assert.False(t, s.Split("missing", "d", SplitRows))
}

func TestSplitRemoveCollapsesBranches(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	s.Split("b", "c", SplitRows)
	s.Split("c", "d", SplitColumns)

	assert.True(t, s.Remove("c"))
	// The rows branch held b and (c|d); with c gone, d stands alone in it.
	assert.Equal(t, s.Leaves(), []string{"a", "b", "d"})
	assert.True(t, s.Remove("b"))
	// Now d is the only child of the rows branch, which collapses into the
	// root's columns.
	assert.Equal(t, len(s.Root.Children), 2)
	assert.True(t, s.Root.Children[1].IsLeaf())
	assert.True(t, s.Remove("a"))
	assert.True(t, s.Root.IsLeaf())
	assert.Equal(t, s.Root.ID, "d")
	assert.True(t, s.Remove("d"))
	assert.Nil(t, s.Root)
	assert.False(t, s.Remove("d"))
}

func TestSplitRemoveFlattensSameAxis(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	s.Split("b", "c", SplitRows)
	s.Split("c", "d", SplitColumns) // columns: a | rows(b, columns(c, d))
	s.Remove("b")
	// rows(columns(c, d)) collapses to columns(c, d), which merges into the
	// root's columns.
	assert.Equal(t, len(s.Root.Children), 3)
	assert.Equal(t, s.Leaves(), []string{"a", "c", "d"})
	for _, c := range s.Root.Children {
		assert.True(t, c.IsLeaf())
	}
}

func TestSplitDragGutter(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	renderSplit(s, 21, 3)

	press := MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: 10, Y: 1}
	assert.True(t, s.HandleMouse(press))
	assert.True(t, s.Dragging())
	assert.True(t, s.HandleMouse(MouseEvent{Type: MouseDrag, Button: MouseButtonLeft, X: 5, Y: 1}))
	// A click synthesized while dragging is the drag's, not the app's.
	assert.True(t, s.HandleMouse(MouseEvent{Type: MouseClick, Button: MouseButtonLeft, X: 5, Y: 1}))
	assert.True(t, s.HandleMouse(MouseEvent{Type: MouseRelease, Button: MouseButtonLeft, X: 5, Y: 1}))
	assert.False(t, s.Dragging())

	screen := renderSplit(s, 21, 3)
	assert.Equal(t, screen.Row(0), "aaaaa bbbbbbbbbbbbbbb")

	// Dragging past the edge stops at the minimum size.
	s.HandleMouse(MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: 5, Y: 0})
	s.HandleMouse(MouseEvent{Type: MouseDrag, Button: MouseButtonLeft, X: 0, Y: 0})
	s.HandleMouse(MouseEvent{Type: MouseRelease, Button: MouseButtonLeft, X: 0, Y: 0})
	screen = renderSplit(s, 21, 3)
	assert.Equal(t, screen.Row(0), "aaa bbbbbbbbbbbbbbbbb")

	// Presses elsewhere, and other buttons, are not the split's.
	assert.False(t, s.HandleMouse(MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: 8, Y: 0}))
	assert.False(t, s.HandleMouse(MouseEvent{Type: MousePress, Button: MouseButtonRight, X: 3, Y: 0}))
}

func TestSplitPressAfterRemoveIgnoresOldGutters(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Split("a", "b", SplitRows)
	s.Append("c", SplitColumns)
	renderSplit(s, 21, 5)
	branch, _, ok := s.DividerAt(5, 2)
	assert.True(t, ok)

	// Removing b turns the rows branch into the leaf a. A press on its old
	// gutter before the next render must not read the gone children.
	s.Remove("b")
	assert.True(t, branch.IsLeaf())
	assert.False(t, s.HandleMouse(MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: 5, Y: 2}))
	_, _, ok = s.DividerAt(5, 2)
	assert.False(t, ok)
}

func TestSplitDragWithNoRoomKeepsWeights(t *testing.T) {
	s := &SplitState{Tight: true}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	// Both panes were drawn with no width, so the gutter has nowhere to go.
	s.drag = &splitDivider{node: s.Root, index: 0}
	s.dragTo(5, 0)
	assert.Equal(t, s.Root.Children[0].Weight, 1.0)
	assert.Equal(t, s.Root.Children[1].Weight, 1.0)
}

func TestSplitHoverHighlightsGutter(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	renderSplit(s, 9, 2)
	assert.False(t, s.HandleMouse(MouseEvent{Type: MouseMove, X: 4, Y: 0}))
	screen := renderSplit(s, 9, 2)
	assert.Equal(t, screen.Row(0), "aaaa│bbbb")
	s.HandleMouse(MouseEvent{Type: MouseMove, X: 0, Y: 0})
	screen = renderSplit(s, 9, 2)
	assert.Equal(t, screen.Row(0), "aaaa bbbb")
}

func TestSplitNeighbor(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	s.Split("b", "c", SplitRows)
	renderSplit(s, 21, 6)

	next, ok := s.Neighbor("a", 1, 0)
	assert.True(t, ok)
	assert.Equal(t, next, "b")
	next, _ = s.Neighbor("c", -1, 0)
	assert.Equal(t, next, "a")
	next, _ = s.Neighbor("b", 0, 1)
	assert.Equal(t, next, "c")
	_, ok = s.Neighbor("a", 0, -1)
	assert.False(t, ok)
}

func TestSplitEqualizeAndWeights(t *testing.T) {
	s := &SplitState{}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	s.Append("c", SplitColumns)
	s.Root.Children[0].Weight = 4
	screen := renderSplit(s, 14, 1)
	assert.Equal(t, screen.Row(0), "aaaaaaaa bb cc")
	s.Equalize()
	screen = renderSplit(s, 14, 1)
	assert.Equal(t, screen.Row(0), "aaaa bbbb cccc")
}

func TestSplitEmpty(t *testing.T) {
	s := &SplitState{}
	screen := renderSplit(s, 5, 2)
	assert.Equal(t, screen.Row(0), "")
	_, ok := s.PaneAt(0, 0)
	assert.False(t, ok)
	assert.Len(t, s.Leaves(), 0)
}

func TestSplitTight(t *testing.T) {
	s := &SplitState{Tight: true}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	s.Split("b", "c", SplitRows)

	screen := renderSplit(s, 20, 4)
	assert.Equal(t, screen.Row(0), "aaaaaaaaaabbbbbbbbbb")
	assert.Equal(t, screen.Row(1), "aaaaaaaaaabbbbbbbbbb")
	assert.Equal(t, screen.Row(2), "aaaaaaaaaacccccccccc")
	b, _ := s.Bounds("b")
	assert.Equal(t, b, image.Rect(10, 0, 20, 2))

	// The cells either side of an edge are its gutter.
	for _, x := range []int{9, 10} {
		branch, _, ok := s.DividerAt(x, 0)
		assert.True(t, ok)
		assert.Equal(t, branch, s.Root)
	}
	_, _, ok := s.DividerAt(8, 0)
	assert.False(t, ok)
	branch, _, ok := s.DividerAt(15, 1)
	assert.True(t, ok)
	assert.Equal(t, branch.Axis, SplitRows)

	// Taking the gutter by either cell moves the edge with the pointer,
	// without a jump.
	for _, x := range []int{9, 10} {
		s.HandleMouse(MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: x, Y: 0})
		s.HandleMouse(MouseEvent{Type: MouseDrag, Button: MouseButtonLeft, X: x, Y: 0})
		s.HandleMouse(MouseEvent{Type: MouseRelease, Button: MouseButtonLeft, X: x, Y: 0})
		screen = renderSplit(s, 20, 4)
		assert.Equal(t, screen.Row(0), "aaaaaaaaaabbbbbbbbbb")
	}
	s.HandleMouse(MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: 9, Y: 0})
	s.HandleMouse(MouseEvent{Type: MouseDrag, Button: MouseButtonLeft, X: 5, Y: 0})
	s.HandleMouse(MouseEvent{Type: MouseRelease, Button: MouseButtonLeft, X: 5, Y: 0})
	screen = renderSplit(s, 20, 4)
	assert.Equal(t, screen.Row(0), "aaaaaabbbbbbbbbbbbbb")
}

func TestSplitTightHighlightsBorders(t *testing.T) {
	s := &SplitState{Tight: true}
	s.Append("a", SplitColumns)
	s.Append("b", SplitColumns)
	active := NewStyle().WithForeground(ColorRed)
	render := func() *termtest.Screen {
		view := Split(s, func(id string) View {
			return Bordered(Text("%s", id)).Title(id)
		}).ActiveGutterStyle(active)
		return SprintScreen(view, WithWidth(12), WithHeight(3))
	}
	screen := render()
	assert.Equal(t, screen.Row(1), "│a   ││b   │")

	red := termtest.Color{Type: termtest.ColorBasic, Value: 1}
	s.HandleMouse(MouseEvent{Type: MouseMove, X: 6, Y: 1})
	screen = render()
	assert.Equal(t, screen.Row(1), "│a   ││b   │")
	for y := 0; y < 3; y++ {
		assert.Equal(t, screen.Cell(5, y).Style.Foreground, red)
		assert.Equal(t, screen.Cell(6, y).Style.Foreground, red)
		assert.NotEqual(t, screen.Cell(0, y).Style.Foreground, red)
	}

	// Stacked, the edge runs through the lower pane's title, which keeps
	// its style.
	s = &SplitState{Tight: true}
	s.Append("a", SplitRows)
	s.Append("b", SplitRows)
	render = func() *termtest.Screen {
		view := Split(s, func(id string) View {
			return Bordered(Text("%s", id)).Title(id)
		}).ActiveGutterStyle(active)
		return SprintScreen(view, WithWidth(8), WithHeight(6))
	}
	render()
	s.HandleMouse(MouseEvent{Type: MouseMove, X: 4, Y: 3})
	screen = render()
	assert.Equal(t, screen.Row(3), "┌─b────┐")
	assert.Equal(t, screen.Cell(0, 3).Style.Foreground, red)
	assert.Equal(t, screen.Cell(0, 2).Style.Foreground, red)
	assert.NotEqual(t, screen.Cell(2, 3).Style.Foreground, red)
}
