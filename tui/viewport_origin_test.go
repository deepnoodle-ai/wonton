package tui

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// framedViewportApp lays a viewport out the way examples/tui/viewport does:
// padding, two header rows, and a border, so its first row is well below the
// top of the screen.
type framedViewportApp struct {
	vp    ViewportState
	items *textItems
}

func (a *framedViewportApp) View() View {
	return Stack(
		Text("title"),
		Text("help"),
		Bordered(Viewport(&a.vp, a.items)),
	).Padding(1)
}

func (a *framedViewportApp) HandleEvent(e Event) []Cmd {
	if m, ok := e.(MouseEvent); ok {
		m.X -= a.vp.X
		m.Y -= a.vp.Y
		a.vp.HandleMouse(m)
	}
	return nil
}

func TestViewportRecordsItsScreenPosition(t *testing.T) {
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("L%02d", i))
	}
	a := &framedViewportApp{items: &textItems{text: lines}}
	r := NewRuntime(NewTestTerminal(30, 12, &bytes.Buffer{}), a, 30)
	assert.NoError(t, r.renderChecked())

	// Padding 1, two header rows and the top border put the first row at
	// screen row 4; padding and the left border put column 0 at column 2.
	assert.Equal(t, [2]int{a.vp.X, a.vp.Y}, [2]int{2, 4})

	// Drag across the second and third visible rows in screen coordinates.
	r.processEvent(MouseEvent{X: 2, Y: 5, Button: MouseButtonLeft, Type: MousePress})
	r.processEvent(MouseEvent{X: 4, Y: 6, Button: MouseButtonLeft, Type: MouseDrag})
	assert.Equal(t, a.vp.SelectedText(), "L01\nL0", "the press selects the row under the pointer")
	r.processEvent(MouseEvent{X: 4, Y: 6, Button: MouseButtonLeft, Type: MouseRelease})
}

// layoutViewportApp puts a viewport in a caller-chosen layout and feeds it
// mouse events converted to viewport coordinates.
type layoutViewportApp struct {
	vp     ViewportState
	items  *textItems
	layout func(View) View
}

func (a *layoutViewportApp) View() View { return a.layout(Viewport(&a.vp, a.items)) }

func (a *layoutViewportApp) HandleEvent(e Event) []Cmd {
	if m, ok := e.(MouseEvent); ok {
		m.X -= a.vp.X
		m.Y -= a.vp.Y
		a.vp.HandleMouse(m)
	}
	return nil
}

// dragScrolls scrolls the viewport to the middle of its content, drags from
// screen row from to screen row to, and reports which way one auto-scroll
// step moved the viewport: -1 up, +1 down, 0 not at all.
func dragScrolls(t *testing.T, layout func(View) View, from, to int) int {
	t.Helper()
	var lines []string
	for i := range 40 {
		lines = append(lines, fmt.Sprintf("L%02d", i))
	}
	a := &layoutViewportApp{items: &textItems{text: lines}, layout: layout}
	r := NewRuntime(NewTestTerminal(30, 12, &bytes.Buffer{}), a, 30)
	assert.NoError(t, r.renderChecked())
	a.vp.ScrollToItem(20)
	assert.NoError(t, r.renderChecked())

	before, _ := a.vp.Anchor()
	r.processEvent(MouseEvent{X: a.vp.X, Y: from, Button: MouseButtonLeft, Type: MousePress})
	r.processEvent(MouseEvent{X: a.vp.X, Y: to, Button: MouseButtonLeft, Type: MouseDrag})
	a.vp.DragAutoScroll()
	after, _ := a.vp.Anchor()
	switch {
	case after < before:
		return -1
	case after > before:
		return 1
	}
	return 0
}

func TestEdgeRowsScrollOnlyWhenThePointerCannotLeave(t *testing.T) {
	// The example's layout: rows above and below the viewport (padding,
	// header, border) are reachable, so only those rows pull. The viewport
	// spans screen rows 4 to 9.
	framed := func(v View) View {
		return Stack(Text("title"), Text("help"), Bordered(v)).Padding(1)
	}
	assert.Equal(t, dragScrolls(t, framed, 6, 4), 0, "inside first row: no scroll")
	assert.Equal(t, dragScrolls(t, framed, 6, 3), -1, "border row above: scroll up")
	assert.Equal(t, dragScrolls(t, framed, 6, 9), 0, "inside last row: no scroll")
	assert.Equal(t, dragScrolls(t, framed, 6, 10), 1, "border row below: scroll down")

	// Touching the top of the screen, with a footer below: the first row
	// pulls, the last does not.
	top := func(v View) View { return Stack(v, Text("footer")) }
	assert.Equal(t, dragScrolls(t, top, 5, 0), -1, "first row at the screen top pulls")
	assert.Equal(t, dragScrolls(t, top, 5, 10), 0, "last row above the footer does not")
	assert.Equal(t, dragScrolls(t, top, 5, 11), 1, "the footer row does")

	// Touching the bottom of the screen, with a header above.
	bottom := func(v View) View { return Stack(Text("header"), v) }
	assert.Equal(t, dragScrolls(t, bottom, 5, 11), 1, "last row at the screen bottom pulls")
	assert.Equal(t, dragScrolls(t, bottom, 5, 1), 0, "first row below the header does not")
	assert.Equal(t, dragScrolls(t, bottom, 5, 0), -1, "the header row does")
}

func TestViewportInsideAScrollRecordsItsScreenPosition(t *testing.T) {
	// Three header rows, then a Scroll scrolled down one row whose content
	// is two rows of text and the viewport. The viewport starts at content
	// row 2, which is screen row 3 + 2 - 1 = 4.
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("L%02d", i))
	}
	offset := 1
	a := &layoutViewportApp{items: &textItems{text: lines}, layout: func(v View) View {
		return Stack(
			Text("h1"), Text("h2"), Text("h3"),
			Scroll(Stack(Text("pre1"), Text("pre2"), Height(12, v)), &offset),
		)
	}}
	r := NewRuntime(NewTestTerminal(30, 12, &bytes.Buffer{}), a, 30)
	assert.NoError(t, r.renderChecked())

	assert.Equal(t, [2]int{a.vp.X, a.vp.Y}, [2]int{0, 4})
	r.processEvent(MouseEvent{X: 0, Y: 5, Button: MouseButtonLeft, Type: MousePress})
	r.processEvent(MouseEvent{X: 2, Y: 6, Button: MouseButtonLeft, Type: MouseDrag})
	assert.Equal(t, a.vp.SelectedText(), "L01\nL0", "the drag selects the rows under the pointer")
}
