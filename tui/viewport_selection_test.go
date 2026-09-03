package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// textItems is a list of items whose content is given verbatim, one item per
// string. Newlines inside a string make a multi-line item.
type textItems struct{ text []string }

func (t *textItems) Len() int { return len(t.text) }

func (t *textItems) Item(i int) View {
	if t.text[i] == "" {
		return nil
	}
	return Text("%s", t.text[i])
}

// newTextViewport renders one frame of a list of text items and returns the
// state and the screen, so a test can drag over what it can see.
func newTextViewport(t *testing.T, lines []string, width, height int) (*ViewportState, *textItems, *termtest.Screen) {
	t.Helper()
	items := &textItems{text: lines}
	s := &ViewportState{}
	screen := renderViewport(t, s, items, width, height, 0)
	return s, items, screen
}

// reversedRuns returns the text of each run of reverse-video cells on screen,
// which is where the selection highlight lands.
func reversedRuns(screen *termtest.Screen, width, height int) []string {
	var runs []string
	for y := range height {
		var run strings.Builder
		for x := range width {
			cell := screen.Cell(x, y)
			if cell.Style.Reverse {
				run.WriteRune(cell.Char)
			}
		}
		if run.Len() > 0 {
			runs = append(runs, run.String())
		}
	}
	return runs
}

func TestSelectionStartsEmptyUntilTheDragMoves(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"hello world"}, 20, 5)

	s.BeginSelection(2, 0)
	assert.True(t, s.SelectionActive(), "a press starts a drag")
	assert.False(t, s.HasSelection(), "a press alone selects nothing")

	s.ExtendSelection(7, 0)
	assert.True(t, s.HasSelection())
	assert.Equal(t, s.SelectedText(), "llo w")
}

func TestSelectionAcrossItems(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"first line", "second line", "third line"}, 20, 5)

	s.BeginSelection(6, 0)
	s.ExtendSelection(6, 2)
	s.EndSelection()

	assert.Equal(t, s.SelectedText(), "line\nsecond line\nthird")
}

func TestSelectionReadsTheSameWayDraggedBackwards(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"first line", "second line"}, 20, 5)

	s.BeginSelection(6, 1)
	s.ExtendSelection(6, 0)
	s.EndSelection()

	start, end, ok := s.Selection()
	assert.True(t, ok)
	assert.True(t, start.before(end), "Selection always reports content order")
	assert.Equal(t, s.SelectedText(), "line\nsecond")
}

func TestSelectionDropsTrailingSpaces(t *testing.T) {
	// The middle line of a multi-line selection runs to the end of the
	// viewport, but the blanks past the text are not part of the text.
	s, _, _ := newTextViewport(t, []string{"ab", "a much longer line"}, 40, 5)

	s.BeginSelection(0, 0)
	s.ExtendSelection(6, 1)
	s.EndSelection()

	assert.Equal(t, s.SelectedText(), "ab\na much")
}

func TestSelectionKeepsWholeCharacters(t *testing.T) {
	// A wide character occupies two cells; a column landing on the second must
	// not split it into a replacement rune.
	s, _, _ := newTextViewport(t, []string{"世界 hello"}, 20, 5)

	s.BeginSelection(0, 0)
	s.ExtendSelection(4, 0)
	s.EndSelection()

	assert.Equal(t, s.SelectedText(), "世界")
}

func TestSelectionHighlightsExactlyTheSelectedCells(t *testing.T) {
	items := &textItems{text: []string{"hello world", "second line"}}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 5, 0)

	s.BeginSelection(6, 0)
	s.ExtendSelection(6, 1)
	s.EndSelection()

	screen := renderViewport(t, s, items, 20, 5, 0)
	assert.Equal(t, reversedRuns(screen, 20, 5), []string{"world", "second"})
}

func TestSelectionHighlightStopsAtTheEndOfTheText(t *testing.T) {
	// A row in the middle of the selection must not trail a reversed bar of
	// blanks across the rest of the viewport.
	items := &textItems{text: []string{"ab", "cd", "ef"}}
	s := &ViewportState{}
	renderViewport(t, s, items, 30, 5, 0)

	s.BeginSelection(0, 0)
	s.ExtendSelection(1, 2)
	s.EndSelection()

	screen := renderViewport(t, s, items, 30, 5, 0)
	assert.Equal(t, reversedRuns(screen, 30, 5), []string{"ab", "cd", "e"})
}

func TestSelectionSurvivesContentAppendedAbove(t *testing.T) {
	// The endpoints are item-relative, so a streaming reply growing above the
	// selection must not drag the highlight onto different words.
	items := &textItems{text: []string{"one", "two", "target text"}}
	s := &ViewportState{Follow: false}
	renderViewport(t, s, items, 20, 10, 0)

	s.BeginSelection(0, 2)
	s.ExtendSelection(6, 2)
	s.EndSelection()
	assert.Equal(t, s.SelectedText(), "target")

	// The item above grows by two lines, as a streaming message would.
	items.text[1] = "two\nmore\nlines"
	s.Invalidate(1)
	renderViewport(t, s, items, 20, 10, 0)

	assert.Equal(t, s.SelectedText(), "target", "the selection stays on its own words")
}

func TestSelectionSurvivesAResize(t *testing.T) {
	items := &textItems{text: []string{"alpha", "bravo charlie"}}
	s := &ViewportState{}
	renderViewport(t, s, items, 40, 6, 0)

	s.BeginSelection(0, 1)
	s.ExtendSelection(5, 1)
	s.EndSelection()
	assert.Equal(t, s.SelectedText(), "bravo")

	renderViewport(t, s, items, 24, 6, 0)
	assert.Equal(t, s.SelectedText(), "bravo", "a narrower screen keeps the same endpoints")
}

func TestSelectionOverAGapRowContinuesFromTheItemAbove(t *testing.T) {
	items := &textItems{text: []string{"first", "second"}}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 6, 1)

	// Row 1 is the gap between the two items.
	s.BeginSelection(0, 0)
	s.ExtendSelection(3, 1)
	s.EndSelection()

	assert.Equal(t, s.SelectedText(), "first")
}

func TestDoubleClickSelectsAWord(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"run make test now"}, 30, 5)

	s.SelectWord(6, 0) // inside "make"
	assert.Equal(t, s.SelectedText(), "make")
}

func TestDoubleClickKeepsAPathTogether(t *testing.T) {
	// A path or an identifier is what a user double-clicks in a terminal, so
	// the separators inside one count as part of the word.
	s, _, _ := newTextViewport(t, []string{"see tui/viewport_selection.go:42 for it"}, 50, 5)

	s.SelectWord(10, 0)
	assert.Equal(t, s.SelectedText(), "tui/viewport_selection.go:42")
}

func TestDoubleClickOnBlankSelectsNothing(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"ab cd"}, 20, 5)

	s.SelectWord(2, 0) // the space
	assert.False(t, s.HasSelection())
}

func TestTripleClickSelectsTheLine(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"first line\nsecond line"}, 30, 5)

	s.SelectLine(3, 1)
	assert.Equal(t, s.SelectedText(), "second line")
}

func TestSelectingSuspendsFollowAndReleasingRestoresIt(t *testing.T) {
	// Follow would pull the content out from under the pointer while a reply
	// streams, so a drag has to pin the viewport for its duration.
	s, _, _ := newTextViewport(t, []string{"one", "two"}, 20, 5)
	s.Follow = true

	s.BeginSelection(0, 0)
	assert.False(t, s.Follow, "a drag pins the viewport")

	s.ExtendSelection(2, 0)
	s.EndSelection()
	assert.False(t, s.Follow, "and it stays pinned while the selection stands")

	s.ClearSelection()
	assert.True(t, s.Follow, "dismissing the selection resumes following")
}

func TestHandleMouseIgnoresWhatIsNotASelectionGesture(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"hello"}, 20, 5)

	wheel := MouseEvent{X: 1, Y: 1, Button: MouseButtonWheelDown, Type: MousePress}
	assert.False(t, s.HandleMouse(wheel), "the wheel is the application's to handle")

	right := MouseEvent{X: 1, Y: 1, Button: MouseButtonRight, Type: MousePress}
	assert.False(t, s.HandleMouse(right))
}

func TestHandleMouseDragSelects(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"hello world"}, 20, 5)

	assert.True(t, s.HandleMouse(MouseEvent{X: 0, Y: 0, Button: MouseButtonLeft, Type: MousePress}))
	assert.True(t, s.HandleMouse(MouseEvent{X: 5, Y: 0, Button: MouseButtonLeft, Type: MouseDrag}))
	assert.True(t, s.HandleMouse(MouseEvent{X: 5, Y: 0, Button: MouseButtonLeft, Type: MouseRelease}))

	assert.Equal(t, s.SelectedText(), "hello")
}

func TestASingleClickDismissesTheSelection(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"hello world"}, 20, 5)

	s.BeginSelection(0, 0)
	s.ExtendSelection(5, 0)
	s.EndSelection()
	assert.True(t, s.HasSelection())

	s.HandleMouse(MouseEvent{X: 8, Y: 0, Button: MouseButtonLeft, Type: MousePress})
	s.HandleMouse(MouseEvent{X: 8, Y: 0, Button: MouseButtonLeft, Type: MouseClick, ClickCount: 1})
	assert.False(t, s.HasSelection())
}

func TestDragPastTheBottomScrollsAndKeepsSelecting(t *testing.T) {
	// Ten single-line items in a five-row viewport: dragging below the bottom
	// edge has somewhere to go.
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = string(rune('a'+i)) + "-line"
	}
	items := &textItems{text: lines}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 5, 0)

	s.HandleMouse(MouseEvent{X: 0, Y: 0, Button: MouseButtonLeft, Type: MousePress})
	s.HandleMouse(MouseEvent{X: 5, Y: 6, Button: MouseButtonLeft, Type: MouseDrag}) // past the bottom

	before, _ := s.Anchor()
	assert.True(t, s.DragAutoScroll(), "a drag held past the edge keeps scrolling")
	renderViewport(t, s, items, 20, 5, 0)
	after, _ := s.Anchor()
	assert.True(t, after > before, "the viewport moved toward the content being selected")

	s.HandleMouse(MouseEvent{X: 5, Y: 4, Button: MouseButtonLeft, Type: MouseRelease})
	assert.True(t, strings.HasPrefix(s.SelectedText(), "a-line\n"),
		"the selection still starts where the drag did, got %q", s.SelectedText())
}

func TestDragAutoScrollDoesNothingInsideTheViewport(t *testing.T) {
	s, _, _ := newTextViewport(t, []string{"one", "two", "three"}, 20, 10)

	s.HandleMouse(MouseEvent{X: 0, Y: 0, Button: MouseButtonLeft, Type: MousePress})
	s.HandleMouse(MouseEvent{X: 2, Y: 1, Button: MouseButtonLeft, Type: MouseDrag})

	assert.False(t, s.DragAutoScroll())
}

func TestSelectionStyleIsConfigurable(t *testing.T) {
	items := &textItems{text: []string{"hello"}}
	s := &ViewportState{SelectionStyle: NewStyle().WithBackground(ColorBlue)}
	renderViewport(t, s, items, 20, 3, 0)

	s.BeginSelection(0, 0)
	s.ExtendSelection(5, 0)
	s.EndSelection()

	screen := renderViewport(t, s, items, 20, 3, 0)
	assert.Equal(t, screen.Cell(0, 0).Style.Background.Value, uint8(4), "blue")
	assert.False(t, screen.Cell(0, 0).Style.Reverse, "an explicit style replaces the default")
}

func TestNoSelectionPaintsNothing(t *testing.T) {
	items := &textItems{text: []string{"hello", "world"}}
	s := &ViewportState{}
	screen := renderViewport(t, s, items, 20, 5, 0)

	assert.Equal(t, len(reversedRuns(screen, 20, 5)), 0)
}

// countingItems is one item that reports how many times it has been rendered,
// so a test can hold the per-frame cost of a selection to what it should be.
type countingItems struct {
	text  []string
	count *int
}

func (c *countingItems) Len() int { return len(c.text) }

func (c *countingItems) Item(i int) View {
	return &countingView{text: c.text[i], count: c.count}
}

type countingView struct {
	text  string
	count *int
}

func (v *countingView) render(ctx *RenderContext) {
	*v.count++
	for y, line := range strings.Split(v.text, "\n") {
		ctx.PrintStyled(0, y, line, NewStyle())
	}
}

func (v *countingView) size(maxWidth, maxHeight int) (int, int) {
	return maxWidth, len(strings.Split(v.text, "\n"))
}

func TestSelectionCostsNoExtraRendersPerFrame(t *testing.T) {
	// The highlight asks for the end of every selected row on every frame. If
	// each of those questions re-rendered the item, a selection would undo the
	// virtualization the viewport exists for.
	body := strings.TrimSuffix(strings.Repeat("hello world\n", 10), "\n")
	renders := 0
	items := &countingItems{text: []string{body}, count: &renders}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 10, 0)

	renders = 0
	renderViewport(t, s, items, 20, 10, 0)
	unselected := renders

	s.BeginSelection(0, 0)
	s.ExtendSelection(5, 9)
	renderViewport(t, s, items, 20, 10, 0) // warms the line cache
	renders = 0
	renderViewport(t, s, items, 20, 10, 0)

	assert.Equal(t, renders, unselected,
		"a frame with a ten-line selection must render the item as often as one without")
}

func TestSelectionLineCacheFollowsInvalidate(t *testing.T) {
	// The cached rows hang off the same entry as the height, so the call that
	// drops one has to drop the other — otherwise a streaming item would be
	// selected by its stale text.
	items := &textItems{text: []string{"before"}}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 5, 0)
	s.BeginSelection(0, 0)
	s.ExtendSelection(6, 0)
	assert.Equal(t, s.SelectedText(), "before")

	items.text[0] = "after edit"
	s.Invalidate(0)
	renderViewport(t, s, items, 20, 5, 0)
	s.BeginSelection(0, 0)
	s.ExtendSelection(10, 0)
	assert.Equal(t, s.SelectedText(), "after edit",
		"Invalidate must drop the cached rows along with the cached height")
}

func TestAutoScrollToTheBottomDoesNotRestoreFollow(t *testing.T) {
	// ScrollBy sets Follow when it reaches the end of the content. Mid-drag
	// that would re-pin the viewport and let a streaming reply move the very
	// words being selected.
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = "line"
	}
	items := &textItems{text: lines}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 5, 0)

	// Follow as a chat would have it when the drag begins: on, with the drag
	// about to pin it. BeginSelection is what captures the value to restore.
	s.Follow = true
	s.BeginSelection(0, 0)
	s.ExtendSelection(3, 9) // held below the bottom edge
	for range 40 {
		if !s.DragAutoScroll() {
			break
		}
		renderViewport(t, s, items, 20, 5, 0)
	}

	assert.True(t, s.AtBottom, "the drag should have scrolled to the end")
	assert.False(t, s.Follow, "Follow stays off while the selection is live")
	assert.True(t, s.HasSelection(), "and the selection survives the scroll")

	s.EndSelection()
	s.ClearSelection()
	assert.True(t, s.Follow, "dismissing it restores the Follow the app had")
}

func TestTripleClickOnABlankLineLeavesFollowAlone(t *testing.T) {
	// Suspending Follow for a selection that never happens pins the viewport
	// with nothing to show for it and nothing for the user to dismiss.
	s, _, _ := newTextViewport(t, []string{"hello\n\nworld"}, 20, 5)
	s.Follow = true

	s.SelectLine(0, 1) // the blank middle line

	assert.False(t, s.HasSelection(), "a blank line selects nothing")
	assert.True(t, s.Follow, "and must not leave Follow suspended")
}

func TestHighlightKeepsMultiRuneGraphemesIntact(t *testing.T) {
	// A cell is restyled by rewriting it, and a rewrite carries one rune. The
	// combining mark on "é" and the VS16 that makes an emoji wide live in the
	// cell's Trailing, and dropping them corrupts what the user is looking at.
	const text = "éx ❤️ y"
	s, items, _ := newTextViewport(t, []string{text}, 20, 5)

	s.BeginSelection(0, 0)
	s.ExtendSelection(10, 0)
	screen := renderViewport(t, s, items, 20, 5, 0)

	var onScreen strings.Builder
	for x := range 20 {
		cell := screen.Cell(x, 0)
		if cell.Continuation {
			continue
		}
		if cell.Char != 0 {
			onScreen.WriteRune(cell.Char)
		}
		onScreen.WriteString(cell.Trailing)
	}

	assert.Equal(t, strings.TrimRight(onScreen.String(), " "), text,
		"highlighting must not rewrite the text it highlights")
	assert.Equal(t, s.SelectedText(), text)

	// Intact is only half of it: the clusters have to be highlighted too, or
	// the accented word is the one part of the selection that looks unselected.
	for x := range len([]rune("éx ❤️ y")) {
		if screen.Cell(x, 0).Char == 0 {
			continue
		}
		assert.True(t, screen.Cell(x, 0).Style.Reverse,
			"every selected cell is highlighted, clusters included")
	}
}

func TestHighlightCoversBothHalvesOfAWideCharacter(t *testing.T) {
	// A wide character owns two cells. Restyling only the lead would draw the
	// highlight half-width and leave the right half on the old background.
	items := &textItems{text: []string{"世界"}}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 5, 0)

	s.BeginSelection(0, 0)
	s.ExtendSelection(4, 0)
	s.EndSelection()

	screen := renderViewport(t, s, items, 20, 5, 0)
	for x := range 4 {
		assert.True(t, screen.Cell(x, 0).Style.Reverse,
			"a wide character highlights across both of its cells")
	}
	assert.False(t, screen.Cell(4, 0).Style.Reverse, "and stops there")
}

func TestAClickWithNothingToDismissFallsThroughToTheApp(t *testing.T) {
	// A Viewport that swallowed every left click would make a clickable
	// element inside one impossible.
	s, _, _ := newTextViewport(t, []string{"hello world"}, 20, 5)

	press := MouseEvent{Type: MousePress, Button: MouseButtonLeft, X: 2, Y: 0}
	click := MouseEvent{Type: MouseClick, Button: MouseButtonLeft, X: 2, Y: 0, ClickCount: 1}

	assert.True(t, s.HandleMouse(press), "a press anchors a drag that may yet happen")
	assert.False(t, s.HandleMouse(click), "the click that follows belongs to the app")

	// With a selection standing, the same click is a dismissal and is consumed.
	s.BeginSelection(0, 0)
	s.ExtendSelection(5, 0)
	s.EndSelection()
	assert.True(t, s.HasSelection())
	assert.True(t, s.HandleMouse(click), "dismissing a selection is the whole event")
	assert.False(t, s.HasSelection())
}

func TestItemAtNamesWhatWasClicked(t *testing.T) {
	// Gap 1, so rows are: item 0, gap, item 1 line 0, item 1 line 1, gap, item 2.
	s := &ViewportState{}
	items := &textItems{text: []string{"first", "second\nsecond line two", "third"}}
	renderViewport(t, s, items, 20, 6, 1)

	item, line, ok := s.ItemAt(0, 0)
	assert.True(t, ok, "row 0 is the first item")
	assert.Equal(t, item, 0)
	assert.Equal(t, line, 0)

	item, line, ok = s.ItemAt(3, 3)
	assert.True(t, ok, "row 3 is the second line of the second item")
	assert.Equal(t, item, 1)
	assert.Equal(t, line, 1)

	_, _, ok = s.ItemAt(0, 1)
	assert.False(t, ok, "the gap between items belongs to neither")

	_, _, ok = s.ItemAt(0, 40)
	assert.False(t, ok, "below the last item there is nothing to click")

	_, _, ok = s.ItemAt(25, 0)
	assert.False(t, ok, "past the viewport's width there is nothing to click")
}

func TestDragToTheBottomReachesTheLastLine(t *testing.T) {
	// The auto-scroll endpoint used to be mapped through the layout recorded by
	// the previous render, which the scroll had just invalidated. That left it
	// one row behind on every step, and since the scroll stops before the
	// endpoint catches up, the last line could not be selected by dragging.
	var lines []string
	for i := range 20 {
		lines = append(lines, fmt.Sprintf("L%02d", i))
	}
	items := &textItems{text: lines}
	s := &ViewportState{}
	renderViewport(t, s, items, 20, 5, 0)

	s.BeginSelection(0, 0)
	s.ExtendSelection(1, 9) // held below the bottom edge
	for range 40 {
		if !s.DragAutoScroll() {
			break
		}
		renderViewport(t, s, items, 20, 5, 0)
	}
	s.EndSelection()

	assert.True(t, s.AtBottom, "the drag should have scrolled to the end")
	assert.True(t, strings.HasSuffix(s.SelectedText(), "\nL19"),
		"a drag held past the bottom edge reaches the last line")
}

func TestADragThatSelectsNothingRestoresFollow(t *testing.T) {
	// A press that never became a selection leaves nothing for the user to
	// dismiss, so releasing has to put Follow back itself.
	s, _, _ := newTextViewport(t, []string{"hello world"}, 20, 5)
	s.Follow = true

	s.BeginSelection(2, 0)
	assert.False(t, s.Follow, "the press pins the viewport")

	s.EndSelection()

	assert.False(t, s.HasSelection())
	assert.True(t, s.Follow, "and releasing with nothing selected unpins it")
}
