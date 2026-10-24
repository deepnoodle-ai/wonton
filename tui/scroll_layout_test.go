package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// singleItems is a viewport list built from fixed views.
type singleItems struct{ views []View }

func (s *singleItems) Len() int        { return len(s.views) }
func (s *singleItems) Item(i int) View { return s.views[i] }

func screenLines(view View, width int) []string {
	return strings.Split(SprintScreen(view, WithWidth(width)).Text(), "\n")
}

func TestViewportFillBgStaysInsideItsItem(t *testing.T) {
	state := &ViewportState{}
	items := &singleItems{views: []View{
		Text("Item 0"),
		Text("Item 1").Bg(ColorBlue).FillBg(),
	}}
	lines := screenLines(Height(4, Viewport(state, items)), 20)
	assert.Equal(t, "Item 0", strings.TrimRight(lines[0], " "), "item 1's fill must not erase item 0")
	assert.Equal(t, "Item 1", strings.TrimRight(lines[1], " "))
}

func TestScrollFillBgStaysInsideItsChild(t *testing.T) {
	scrollY := 0
	content := Stack(
		Text("top"),
		Text("blue").Bg(ColorBlue).FillBg(),
		Text("a"), Text("b"), Text("c"),
	)
	lines := screenLines(Height(3, Scroll(content, &scrollY)), 10)
	assert.Equal(t, "top", strings.TrimRight(lines[0], " "))
	assert.Equal(t, "blue", strings.TrimRight(lines[1], " "))
	assert.Equal(t, "a", strings.TrimRight(lines[2], " "))
}

func TestViewportClipsChildrenToTheirColumns(t *testing.T) {
	state := &ViewportState{}
	items := &singleItems{views: []View{
		Group(Width(5, Text("1234567890")), Text("R")),
	}}
	lines := screenLines(Height(2, Viewport(state, items)), 20)
	assert.Equal(t, "12345R", strings.TrimRight(lines[0], " "))
}

func TestScrollClipsChildrenToTheirRows(t *testing.T) {
	scrollY := 0
	content := Stack(
		Height(1, Text("one\ntwo")),
		Text("n"),
		Text("a"), Text("b"),
	)
	lines := screenLines(Height(3, Scroll(content, &scrollY)), 10)
	assert.Equal(t, "one", strings.TrimRight(lines[0], " "))
	assert.Equal(t, "n", strings.TrimRight(lines[1], " "), "a 1-row child must not spill into the next row")
}

type clickApp struct {
	scrollY int
	clicked []string
}

func (a *clickApp) View() View {
	var rows []View
	for _, name := range []string{"r0", "r1", "r2", "r3", "r4"} {
		rows = append(rows, Clickable(name, func() { a.clicked = append(a.clicked, name) }))
	}
	return Stack(Text("header"), Height(3, Scroll(Stack(rows...), &a.scrollY)))
}

func (a *clickApp) HandleEvent(Event) []Cmd { return nil }

func TestScrollChildrenRegisterTheirOwnClickRegions(t *testing.T) {
	a := &clickApp{}
	r := NewRuntime(NewTestTerminal(20, 6, &bytes.Buffer{}), a, 30)
	assert.NoError(t, r.renderChecked())
	r.processEvent(MouseEvent{X: 0, Y: 2, Button: MouseButtonLeft, Type: MouseClick})
	assert.Equal(t, []string{"r1"}, a.clicked)

	a.scrollY = 2
	a.clicked = nil
	assert.NoError(t, r.renderChecked())
	r.processEvent(MouseEvent{X: 0, Y: 1, Button: MouseButtonLeft, Type: MouseClick})
	assert.Equal(t, []string{"r2"}, a.clicked)
}

func TestViewportScrollByReachesTheBottomOfTheLastItem(t *testing.T) {
	tests := []struct {
		name     string
		heights  []int
		scroll   int
		wantItem int
		wantLine int
	}{
		{name: "one tall item, exact span", heights: []int{30}, scroll: 30, wantItem: 0, wantLine: 20},
		{name: "one tall item, past the end", heights: []int{30}, scroll: 100, wantItem: 0, wantLine: 20},
		{name: "two items, past the end", heights: []int{5, 30}, scroll: 100, wantItem: 1, wantLine: 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state, _ := newViewport(t, tc.heights, 20, 10, 0)
			state.ScrollBy(tc.scroll)
			item, line := state.Anchor()
			assert.Equal(t, tc.wantItem, item, "anchor item")
			assert.Equal(t, tc.wantLine, line, "anchor line")
			assert.True(t, state.Follow, "reaching the bottom re-engages Follow")
		})
	}
}

func TestContainersKeepExhaustedConstraints(t *testing.T) {
	tall := Text("%s", strings.TrimSuffix(strings.Repeat("x\n", 500), "\n"))
	tests := []struct {
		name string
		view View
	}{
		{"Bordered", Bordered(Scroll(tall, nil))},
		{"Padding", Padding(1, Scroll(tall, nil))},
		{"AnimatedBordered", AnimatedBordered(Scroll(tall, nil), nil)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, h := Measure(tc.view, 80, 2)
			assert.Equal(t, 2, h, "a 2-row limit leaves the child no rows, not unlimited rows")
			w, _ := Measure(tc.view, 2, 10)
			assert.Equal(t, 2, w)
		})
	}
}

func TestGridSizeIsNeverNegative(t *testing.T) {
	w, h := Measure(CellGrid(0, 3).Gap(1), 0, 0)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
	w, h = Measure(ColorGrid(3, 0, nil, []Color{ColorRed}).Gap(1), 0, 0)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestColorGridClickNormalizesAnInvalidState(t *testing.T) {
	state := [][]int{{-5}}
	var got int
	grid := ColorGrid(1, 1, state, []Color{ColorRed, ColorGreen, ColorBlue}).
		OnStateChange(func(_, _, idx int) { got = idx })
	app := &staticApp{view: grid}
	r := NewRuntime(NewTestTerminal(10, 3, &bytes.Buffer{}), app, 30)
	assert.NoError(t, r.renderChecked())
	r.processEvent(MouseEvent{X: 0, Y: 0, Button: MouseButtonLeft, Type: MouseClick})
	// Render showed color 0 for the invalid state, so one click moves to 1.
	assert.Equal(t, 1, got)
	assert.Equal(t, 1, state[0][0])
}

type staticApp struct{ view View }

func (a *staticApp) View() View              { return a.view }
func (a *staticApp) HandleEvent(Event) []Cmd { return nil }

type scrollSourceItems struct {
	source  string
	scrollY *int
}

func (s *scrollSourceItems) Len() int                  { return 1 }
func (s *scrollSourceItems) Source(int) (string, bool) { return s.source, true }
func (s *scrollSourceItems) Item(int) View {
	return Stack(
		Text("Header").SourceOffset(0),
		Height(1, Scroll(Text("Line1\nLine2").SourceOffset(7), s.scrollY)),
		Text("Footer").SourceOffset(19),
	)
}

func TestScrollInsideSourceItemDoesNotOverwriteSiblings(t *testing.T) {
	scrollY := 1
	items := &scrollSourceItems{source: "Header\nLine1\nLine2\nFooter", scrollY: &scrollY}
	state := &ViewportState{}
	renderViewport(t, state, items, 20, 5, 0)

	state.SelectLine(1, 0)
	assert.Equal(t, "Header", state.SelectedText())
	state.SelectLine(1, 1)
	assert.Equal(t, "Line2", state.SelectedText())
	state.SelectLine(1, 2)
	assert.Equal(t, "Footer", state.SelectedText())
}

func TestDragToTheBottomEndsOnTheLastLineOfATallLastItem(t *testing.T) {
	// DragAutoScroll walks the endpoint down from the anchor with moveDown,
	// so a walk that runs past the end of the last item must stop on its last
	// line, not on the line it started from.
	items := &heightItems{heights: []int{3, 12}}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 5, 1)

	s.BeginSelection(0, 0)
	s.ExtendSelection(19, 9) // held below the bottom edge, past the line end
	for range 40 {
		if !s.DragAutoScroll() {
			break
		}
		renderViewport(t, s, items, 20, 5, 1)
	}
	s.EndSelection()

	assert.True(t, s.AtBottom, "the drag should have scrolled to the end")
	assert.True(t, strings.HasSuffix(s.SelectedText(), "\n1.11"),
		"a drag held past the bottom edge ends on the last line, got %q", s.SelectedText())
}

type quitApp struct {
	scrollY int
	clicked []string
}

func (a *quitApp) View() View {
	var rows []View
	for i := range 30 {
		name := fmt.Sprintf("row%d", i)
		rows = append(rows, Clickable(name, func() { a.clicked = append(a.clicked, name) }))
	}
	return Stack(
		Height(10, Scroll(Stack(rows...), &a.scrollY)),
		Clickable("Quit", func() { a.clicked = append(a.clicked, "Quit") }),
	)
}

func (a *quitApp) HandleEvent(Event) []Cmd { return nil }

func TestScrolledOutClickablesDoNotTakeClicksBelowTheScroll(t *testing.T) {
	a := &quitApp{}
	r := NewRuntime(NewTestTerminal(20, 12, &bytes.Buffer{}), a, 30)
	assert.NoError(t, r.renderChecked())
	r.processEvent(MouseEvent{X: 0, Y: 10, Button: MouseButtonLeft, Type: MouseClick})
	assert.Equal(t, []string{"Quit"}, a.clicked)

	a.scrollY = 5
	a.clicked = nil
	assert.NoError(t, r.renderChecked())
	r.processEvent(MouseEvent{X: 0, Y: 10, Button: MouseButtonLeft, Type: MouseClick})
	assert.Equal(t, []string{"Quit"}, a.clicked, "rows scrolled off the top must not shift onto the Quit row")
	r.processEvent(MouseEvent{X: 0, Y: 0, Button: MouseButtonLeft, Type: MouseClick})
	assert.Equal(t, []string{"Quit", "row5"}, a.clicked)
}

// focusRowsApp is a Scroll of one-row focusable widgets of several kinds,
// with a Button below it.
type focusRowsApp struct {
	scrollY int
	texts   [30]string
	sels    [30]int
	toggles [30]bool
}

func (a *focusRowsApp) View() View {
	var rows []View
	for i := range 30 {
		id := fmt.Sprintf("row%d", i)
		switch i % 6 {
		case 0:
			rows = append(rows, FocusableView(Text("%s", id), &FocusHandler{ID: id}))
		case 1:
			rows = append(rows, Button(id, func() {}).ID(id))
		case 2:
			rows = append(rows, SelectListStrings([]string{id}, &a.sels[i]).ID(id))
		case 3:
			rows = append(rows, Input(&a.texts[i]).ID(id))
		case 4:
			rows = append(rows, InputField(&a.texts[i]).ID(id))
		case 5:
			rows = append(rows, Toggle(&a.toggles[i]).ID(id))
		}
	}
	return Stack(
		Height(10, Scroll(Stack(rows...), &a.scrollY)),
		Button("Quit", func() {}).ID("quit"),
	)
}

func (a *focusRowsApp) HandleEvent(Event) []Cmd { return nil }

func TestScrolledOutFocusablesDoNotTakeFocusClicksBelowTheScroll(t *testing.T) {
	for _, scrollY := range []int{0, 5} {
		a := &focusRowsApp{scrollY: scrollY}
		r := NewRuntime(NewTestTerminal(20, 12, &bytes.Buffer{}), a, 30)
		// FocusManager.HandleClick walks a map, so repeat to cover its order.
		for range 20 {
			assert.NoError(t, r.renderChecked())
			r.focusMgr.SetFocus(fmt.Sprintf("row%d", scrollY))
			r.processEvent(MouseEvent{X: 0, Y: 10, Button: MouseButtonLeft, Type: MouseClick})
			assert.Equal(t, "quit", r.focusMgr.GetFocusedID(), "scrollY=%d: a click on Quit focuses Quit", scrollY)
		}
		// Every visible row, one of each focusable kind, still takes clicks.
		for y := range 10 {
			assert.NoError(t, r.renderChecked())
			r.processEvent(MouseEvent{X: 0, Y: y, Button: MouseButtonLeft, Type: MouseClick})
			assert.Equal(t, fmt.Sprintf("row%d", scrollY+y), r.focusMgr.GetFocusedID(), "scrollY=%d: a click on visible row %d focuses it", scrollY, y)
		}
	}
}
