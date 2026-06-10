package tui

import (
	"image"
	"sync"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func newFlexTestWidget(width, height, grow int) *MockLayoutWidget {
	w := NewMockLayoutWidget(width, height)
	params := DefaultLayoutParams()
	params.Grow = grow
	w.SetLayoutParams(params)
	return w
}

// Wrapping must use the items' computed sizes; previously items were split
// into lines before any sizes were calculated, so nothing ever wrapped.
func TestFlexLayout_Wrap(t *testing.T) {
	layout := NewFlexLayout().WithWrap(FlexWrapOn)

	w1 := newFlexTestWidget(60, 10, 0)
	w2 := newFlexTestWidget(60, 10, 0)
	w3 := newFlexTestWidget(60, 10, 0)

	// Container fits only two 60-wide items per row
	container := image.Rect(0, 0, 130, 100)
	layout.Layout(container, []ComposableWidget{w1, w2, w3})

	assert.Equal(t, 0, w1.GetBounds().Min.Y, "first item on first line")
	assert.Equal(t, 0, w2.GetBounds().Min.Y, "second item on first line")
	assert.Equal(t, 10, w3.GetBounds().Min.Y, "third item should wrap to second line")
	assert.Equal(t, 0, w3.GetBounds().Min.X, "wrapped item starts at line beginning")
}

func TestFlexLayout_WrapWithLineSpacing(t *testing.T) {
	layout := NewFlexLayout().WithWrap(FlexWrapOn).WithLineSpacing(2)

	w1 := newFlexTestWidget(60, 10, 0)
	w2 := newFlexTestWidget(60, 10, 0)

	container := image.Rect(0, 0, 80, 100)
	layout.Layout(container, []ComposableWidget{w1, w2})

	assert.Equal(t, 0, w1.GetBounds().Min.Y)
	assert.Equal(t, 12, w2.GetBounds().Min.Y, "second line offset by line height + spacing")
}

// Growing items must consume all remaining space; integer division
// remainders go to the last flexible item.
func TestFlexLayout_GrowDistributesRemainder(t *testing.T) {
	layout := NewFlexLayout()

	w1 := newFlexTestWidget(0, 10, 1)
	w2 := newFlexTestWidget(0, 10, 1)
	w3 := newFlexTestWidget(0, 10, 1)

	// 100 / 3 = 33 remainder 1; the last item should absorb the extra cell
	container := image.Rect(0, 0, 100, 10)
	layout.Layout(container, []ComposableWidget{w1, w2, w3})

	total := w1.GetBounds().Dx() + w2.GetBounds().Dx() + w3.GetBounds().Dx()
	assert.Equal(t, 100, total, "grow should use the full container width")
	assert.Equal(t, 100, w3.GetBounds().Max.X, "last item should end at the container edge")
}

func TestFlexLayout_ShrinkDistributesRemainder(t *testing.T) {
	layout := NewFlexLayout()

	w1 := NewMockLayoutWidget(50, 10)
	w2 := NewMockLayoutWidget(50, 10)
	w3 := NewMockLayoutWidget(50, 10)
	for _, w := range []*MockLayoutWidget{w1, w2, w3} {
		params := DefaultLayoutParams()
		params.Shrink = 1
		w.SetLayoutParams(params)
	}

	// 150 preferred into 100: shrink 50 across 3 items (16 each + remainder 2)
	container := image.Rect(0, 0, 100, 10)
	layout.Layout(container, []ComposableWidget{w1, w2, w3})

	total := w1.GetBounds().Dx() + w2.GetBounds().Dx() + w3.GetBounds().Dx()
	assert.Equal(t, 100, total, "shrink should fit exactly into the container width")
}

// GetProgress/GetPosition previously acquired p.mu recursively, which can
// deadlock when a writer is waiting between the two acquisitions.
func TestPlaybackController_ConcurrentProgressAndSeek(t *testing.T) {
	p := &PlaybackController{
		events: []RecordingEvent{
			{Time: 0.0, Type: "o", Data: "a"},
			{Time: 1.0, Type: "o", Data: "b"},
			{Time: 2.0, Type: "o", Data: "c"},
		},
		speed: 1.0,
	}

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				p.GetProgress()
				p.GetPosition()
				p.GetDuration()
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				p.Seek(float64(i % 3))
				p.SetLoop(i%2 == 0)
			}
		}()
	}
	wg.Wait()
}

func TestListViewItemHeightIgnoresInvalid(t *testing.T) {
	items := []ListItem{{Label: "a"}, {Label: "b"}, {Label: "c"}}
	selected := 0

	l := FilterableList(items, &selected).ItemHeight(0)
	assert.Equal(t, 1, l.itemHeight, "zero item height must be ignored")

	l = FilterableList(items, &selected).ItemHeight(-5)
	assert.Equal(t, 1, l.itemHeight, "negative item height must be ignored")

	l = FilterableList(items, &selected).ItemHeight(3)
	assert.Equal(t, 3, l.itemHeight)
}
