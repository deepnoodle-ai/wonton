package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// heightItems is a list whose items are exactly the heights given: item i is a
// block of h(i) lines, each labelled "i.0", "i.1", … so a rendered screen says
// unambiguously which line of which item is on it. A height of 0 means the item
// renders nothing.
type heightItems struct {
	heights []int
	built   []int // indices Item was called for, in order
}

func (h *heightItems) Len() int { return len(h.heights) }

func (h *heightItems) Item(i int) View {
	h.built = append(h.built, i)
	n := h.heights[i]
	if n == 0 {
		return nil
	}
	lines := make([]string, n)
	for l := range lines {
		lines[l] = fmt.Sprintf("%d.%d", i, l)
	}
	return Text("%s", strings.Join(lines, "\n"))
}

// heightsOf returns n items of the same height.
func heightsOf(n, h int) []int {
	heights := make([]int, n)
	for i := range heights {
		heights[i] = h
	}
	return heights
}

func newViewport(t *testing.T, heights []int, width, height, gap int) (*ViewportState, *heightItems) {
	t.Helper()
	items := &heightItems{heights: heights}
	state := &ViewportState{}
	renderViewport(t, state, items, width, height, gap)
	return state, items
}

// renderViewport renders one frame and returns the visible screen.
func renderViewport(t *testing.T, s *ViewportState, items ViewportItems, width, height, gap int) *termtest.Screen {
	t.Helper()
	view := Height(height, Viewport(s, items).Gap(gap))
	return SprintScreen(view, WithWidth(width))
}

// visible returns the non-blank lines on screen, trimmed.
func visible(screen *termtest.Screen) []string {
	var out []string
	for _, line := range strings.Split(screen.Text(), "\n") {
		out = append(out, strings.TrimRight(line, " "))
	}
	// Drop trailing blank rows so tests can talk about content, not padding.
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

func TestViewportShowsTheTopWhenContentFits(t *testing.T) {
	state, _ := newViewport(t, []int{2, 2}, 20, 10, 1)

	item, line := state.Anchor()
	assert.Equal(t, 0, item)
	assert.Equal(t, 0, line)
	assert.True(t, state.AtBottom, "short content is entirely visible")
	assert.Equal(t, 0, state.LinesBelow)
}

func TestViewportFollowsTheEnd(t *testing.T) {
	// Six items of 3 lines with a 1-line gap: 3+1+3+1+3+1+3+1+3+1+3 = 23 lines
	// in a 5-line viewport. Following the end shows the last item's 3 lines
	// plus the gap and the tail of the item before it.
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{3, 3, 3, 3, 3, 3}}
	screen := renderViewport(t, state, items, 20, 5, 1)

	assert.Equal(t, []string{"4.2", "", "5.0", "5.1", "5.2"}, visible(screen))
	assert.True(t, state.AtBottom)
	assert.Equal(t, 0, state.LinesBelow)

	item, line := state.Anchor()
	assert.Equal(t, 4, item)
	assert.Equal(t, 2, line)
}

func TestViewportOnlyBuildsTheItemsItShows(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: make([]int, 500)}
	for i := range items.heights {
		items.heights[i] = 3
	}
	renderViewport(t, state, items, 20, 10, 1)

	// A 10-row viewport over 500 items of 3 lines each needs three items to
	// fill it. maxAnchor walks back from the end, so a handful more may be
	// touched — but nothing close to 500.
	built := map[int]bool{}
	for _, i := range items.built {
		built[i] = true
	}
	assert.True(t, len(built) <= 10, "built %d items to fill a 10-row viewport", len(built))
	assert.True(t, built[499], "the last item must be measured to follow the end")
	assert.False(t, built[0], "item 0 is 1500 lines above the viewport")
}

func TestViewportScrollByWalksLinesAcrossItems(t *testing.T) {
	tests := []struct {
		name       string
		heights    []int
		gap        int
		height     int
		scroll     int
		wantItem   int
		wantLine   int
		wantFollow bool
	}{
		{
			name: "within one item", heights: []int{10, 10}, gap: 0, height: 5,
			scroll: -3, wantItem: 1, wantLine: 2,
		},
		{
			name: "up across an item boundary", heights: []int{10, 10}, gap: 0, height: 5,
			scroll: -8, wantItem: 0, wantLine: 7,
		},
		{
			name: "up across a gap", heights: []int{10, 10}, gap: 2, height: 5,
			// Bottom anchor is (1, 5). Up 6 lands on the last gap row above item 1,
			// which belongs to item 0's span: line 10 is the first gap row, 11 the second.
			scroll: -6, wantItem: 0, wantLine: 11,
		},
		{
			name: "up past the start clamps", heights: []int{4, 4}, gap: 1, height: 3,
			scroll: -1000, wantItem: 0, wantLine: 0,
		},
		{
			name: "down past the end clamps and re-follows", heights: []int{4, 4}, gap: 1, height: 3,
			scroll: 1000, wantItem: 1, wantLine: 1, wantFollow: true,
		},
		{
			name: "an empty item is skipped, gap and all", heights: []int{5, 0, 5}, gap: 1, height: 3,
			// Bottom anchor is (2, 2). Up 3: (2,1), (2,0), then the gap row after
			// item 0 — item 1 contributes nothing at all.
			scroll: -3, wantItem: 0, wantLine: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := &ViewportState{Follow: true}
			items := &heightItems{heights: tc.heights}
			renderViewport(t, state, items, 20, tc.height, tc.gap)

			state.ScrollBy(tc.scroll)
			item, line := state.Anchor()
			assert.Equal(t, tc.wantItem, item, "anchor item")
			assert.Equal(t, tc.wantLine, line, "anchor line")
			assert.Equal(t, tc.wantFollow, state.Follow, "Follow")
		})
	}
}

func TestViewportScrollingUpStopsFollowingAndReportsWhatIsBelow(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{4, 4, 4, 4}}
	renderViewport(t, state, items, 20, 5, 0)

	state.ScrollBy(-6)
	assert.False(t, state.Follow, "scrolling up releases the bottom")

	renderViewport(t, state, items, 20, 5, 0)
	assert.False(t, state.AtBottom)
	assert.Equal(t, 6, state.LinesBelow)

	state.ScrollBy(6)
	assert.True(t, state.Follow, "scrolling back down re-attaches to the bottom")
	renderViewport(t, state, items, 20, 5, 0)
	assert.Equal(t, 0, state.LinesBelow)
}

func TestViewportAppendDoesNotMoveAScrolledUpView(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{4, 4, 4, 4}}
	renderViewport(t, state, items, 20, 5, 0)
	state.ScrollBy(-5)

	before := visible(renderViewport(t, state, items, 20, 5, 0))

	items.heights = append(items.heights, 4, 4)
	after := visible(renderViewport(t, state, items, 20, 5, 0))

	assert.Equal(t, before, after, "appending below the anchor must not move it")
	assert.Equal(t, 13, state.LinesBelow)
}

func TestViewportGrowingTheAnchoredItemDoesNotMoveIt(t *testing.T) {
	// The streaming case: the item at the top of the viewport gets taller.
	// The anchor is (item, line), so the same line stays on the first row.
	state := &ViewportState{}
	items := &heightItems{heights: []int{20, 6}}
	renderViewport(t, state, items, 20, 5, 0)
	state.ScrollToItem(0)
	state.ScrollBy(3)
	before := visible(renderViewport(t, state, items, 20, 5, 0))
	assert.Equal(t, []string{"0.3", "0.4", "0.5", "0.6", "0.7"}, before)

	items.heights[0] = 40
	state.Invalidate(0)
	after := visible(renderViewport(t, state, items, 20, 5, 0))
	assert.Equal(t, before, after)
}

func TestViewportRemeasuresOnWidthChange(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{3, 3}}
	renderViewport(t, state, items, 20, 10, 1)
	assert.Equal(t, 20, state.Width)

	items.built = nil
	renderViewport(t, state, items, 40, 10, 1)
	assert.Equal(t, 40, state.Width)
	assert.True(t, len(items.built) > 0, "a width change must drop cached heights")
}

func TestViewportBuildsEachItemOnceUntilInvalidated(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{3, 3}}
	renderViewport(t, state, items, 20, 10, 1)

	items.built = nil
	renderViewport(t, state, items, 20, 10, 1)
	renderViewport(t, state, items, 20, 10, 1)
	assert.Equal(t, 0, len(items.built), "cached items must not be rebuilt every frame")

	state.Invalidate(1)
	renderViewport(t, state, items, 20, 10, 1)
	assert.Equal(t, []int{1}, items.built)
}

func TestViewportScrollToTopAndBottom(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{4, 4, 4}}
	renderViewport(t, state, items, 20, 4, 0)

	state.ScrollToTop()
	assert.False(t, state.Follow)
	assert.Equal(t, []string{"0.0", "0.1", "0.2", "0.3"}, visible(renderViewport(t, state, items, 20, 4, 0)))

	state.ScrollToItem(1)
	assert.Equal(t, []string{"1.0", "1.1", "1.2", "1.3"}, visible(renderViewport(t, state, items, 20, 4, 0)))

	state.ScrollToBottom()
	assert.True(t, state.Follow)
	assert.Equal(t, []string{"2.0", "2.1", "2.2", "2.3"}, visible(renderViewport(t, state, items, 20, 4, 0)))
}

func TestViewportPagingOverlapsOneLine(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{30}}
	renderViewport(t, state, items, 20, 5, 0)
	assert.Equal(t, []string{"0.25", "0.26", "0.27", "0.28", "0.29"}, visible(renderViewport(t, state, items, 20, 5, 0)))

	state.PageUp()
	assert.Equal(t, []string{"0.21", "0.22", "0.23", "0.24", "0.25"}, visible(renderViewport(t, state, items, 20, 5, 0)))

	state.PageDown()
	assert.Equal(t, []string{"0.25", "0.26", "0.27", "0.28", "0.29"}, visible(renderViewport(t, state, items, 20, 5, 0)))
}

func TestViewportScrollMethodsAreNoOpsBeforeFirstRender(t *testing.T) {
	state := &ViewportState{}
	state.ScrollBy(-5)
	state.PageUp()
	state.ScrollToItem(3)
	item, line := state.Anchor()
	assert.Equal(t, 0, item)
	assert.Equal(t, 0, line)
}

func TestViewportHandlesAnEmptyList(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{}
	screen := renderViewport(t, state, items, 20, 5, 1)
	assert.Equal(t, 0, len(visible(screen)))
	assert.True(t, state.AtBottom)
}

func TestViewportClampsAnAnchorPastAShrunkList(t *testing.T) {
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{4, 4, 4, 4}}
	renderViewport(t, state, items, 20, 4, 0)
	state.ScrollToItem(3)

	// /clear: the list is replaced and every index means something new.
	items.heights = []int{2}
	state.InvalidateAll()
	screen := renderViewport(t, state, items, 20, 4, 0)

	assert.Equal(t, []string{"0.0", "0.1"}, visible(screen))
	item, _ := state.Anchor()
	assert.Equal(t, 0, item)
}

func TestViewportFillsTheSpaceAStackLeaves(t *testing.T) {
	// The point of flex() == 1: the footer keeps its natural height and the
	// viewport takes exactly the rest, with no arithmetic in the app.
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{2, 2, 2, 2}}
	view := Height(6, Stack(
		Viewport(state, items).Gap(0),
		Text("footer-a\nfooter-b"),
	).Gap(0))
	screen := SprintScreen(view, WithWidth(20))

	assert.Equal(t, []string{"2.0", "2.1", "3.0", "3.1", "footer-a", "footer-b"}, visible(screen))
	assert.Equal(t, 4, state.Height, "viewport height is the stack's leftovers")
}

func TestViewportKeepsFollowingAsContentIsAppended(t *testing.T) {
	// The bug this replaces: Scroll(content, &offset) writes the clamped offset
	// back, so once it has been clamped to the bottom it stays at that absolute
	// line while the content grows past it — the view silently stops following.
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{3}}
	assert.Equal(t, []string{"0.0", "0.1", "0.2"}, visible(renderViewport(t, state, items, 20, 4, 0)))

	for i := 0; i < 20; i++ {
		items.heights = append(items.heights, 3)
		screen := renderViewport(t, state, items, 20, 4, 0)
		last := len(items.heights) - 1
		assert.Equal(t, fmt.Sprintf("%d.2", last), visible(screen)[3],
			"append %d: the newest line must be on the bottom row", i)
		assert.True(t, state.AtBottom)
	}
}

func TestViewportFollowSurvivesTheAnchoredItemStreaming(t *testing.T) {
	// The other half of following: the last item grows in place, one flush at a
	// time, and each frame must show its newest line.
	state := &ViewportState{Follow: true}
	items := &heightItems{heights: []int{2, 1}}
	for h := 1; h <= 12; h++ {
		items.heights[1] = h
		state.Invalidate(1)
		rows := visible(renderViewport(t, state, items, 20, 5, 1))
		assert.Equal(t, fmt.Sprintf("1.%d", h-1), rows[len(rows)-1],
			"height %d: the newest line must be the last one shown", h)
		if h >= 3 {
			assert.Equal(t, 5, len(rows), "height %d: a full viewport", h)
		}
	}
}

func TestViewportPositionIsCurrentBeforeTheNextRender(t *testing.T) {
	// An app draws its scroll indicator from AtBottom and LinesBelow in the
	// same frame that handles the key, so scrolling has to update them at once
	// rather than leaving it to the render that follows.
	state, _ := newViewport(t, heightsOf(40, 1), 20, 10, 0)
	state.ScrollToBottom()
	assert.True(t, state.AtBottom)

	state.PageUp()
	assert.False(t, state.AtBottom)
	assert.Equal(t, 9, state.LinesBelow)

	state.ScrollToTop()
	assert.False(t, state.AtBottom)
	assert.Equal(t, 30, state.LinesBelow)

	state.ScrollToItem(35)
	assert.True(t, state.AtBottom, "item 35 of 40 is inside the last screenful")
	assert.Equal(t, 0, state.LinesBelow)
}
