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

	// The edge rows line up too: a drag held on the viewport's first screen
	// row scrolls up once the viewport is scrolled down.
	a.vp.ClearSelection()
	a.vp.ScrollToBottom()
	assert.NoError(t, r.renderChecked())
	r.processEvent(MouseEvent{X: 2, Y: 5, Button: MouseButtonLeft, Type: MousePress})
	r.processEvent(MouseEvent{X: 2, Y: 4, Button: MouseButtonLeft, Type: MouseDrag})
	assert.True(t, a.vp.DragAutoScroll())
}
