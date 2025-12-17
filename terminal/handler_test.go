package terminal

import (
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNewMouseHandler(t *testing.T) {
	h := NewMouseHandler()
	assert.NotNil(t, h)
	assert.Equal(t, 500*time.Millisecond, h.DoubleClickThreshold)
	assert.Equal(t, 500*time.Millisecond, h.TripleClickThreshold)
	assert.Equal(t, 2, h.ClickMoveThreshold)
	assert.Equal(t, 5, h.DragStartThreshold)
}

func TestMouseHandler_EnableDisableDebug(t *testing.T) {
	h := NewMouseHandler()

	h.EnableDebug()
	assert.True(t, h.debugMode)

	h.DisableDebug()
	assert.False(t, h.debugMode)
}

func TestMouseHandler_AddRegion(t *testing.T) {
	h := NewMouseHandler()
	region := &MouseRegion{X: 0, Y: 0, Width: 10, Height: 10}

	h.AddRegion(region)
	assert.Len(t, h.regions, 1)
}

func TestMouseHandler_RemoveRegion(t *testing.T) {
	h := NewMouseHandler()
	region1 := &MouseRegion{X: 0, Y: 0, Width: 10, Height: 10}
	region2 := &MouseRegion{X: 10, Y: 10, Width: 10, Height: 10}

	h.AddRegion(region1)
	h.AddRegion(region2)
	assert.Len(t, h.regions, 2)

	h.RemoveRegion(region1)
	assert.Len(t, h.regions, 1)
}

func TestMouseHandler_RemoveRegion_ClearsReferences(t *testing.T) {
	h := NewMouseHandler()
	region := &MouseRegion{X: 0, Y: 0, Width: 10, Height: 10}

	h.AddRegion(region)
	h.hoveredRegion = region
	h.capturedRegion = region
	h.dragRegion = region

	h.RemoveRegion(region)

	assert.Nil(t, h.hoveredRegion)
	assert.Nil(t, h.capturedRegion)
	assert.Nil(t, h.dragRegion)
}

func TestMouseHandler_ClearRegions(t *testing.T) {
	h := NewMouseHandler()
	h.AddRegion(&MouseRegion{X: 0, Y: 0, Width: 10, Height: 10})
	h.AddRegion(&MouseRegion{X: 10, Y: 10, Width: 10, Height: 10})
	h.hoveredRegion = &MouseRegion{}
	h.capturedRegion = &MouseRegion{}
	h.dragRegion = &MouseRegion{}
	h.isDragging = true

	h.ClearRegions()

	assert.Empty(t, h.regions)
	assert.Nil(t, h.hoveredRegion)
	assert.Nil(t, h.capturedRegion)
	assert.Nil(t, h.dragRegion)
	assert.False(t, h.isDragging)
}

func TestMouseHandler_SortsByZIndex(t *testing.T) {
	h := NewMouseHandler()
	region1 := &MouseRegion{X: 0, Y: 0, Width: 10, Height: 10, ZIndex: 1}
	region2 := &MouseRegion{X: 0, Y: 0, Width: 10, Height: 10, ZIndex: 10}
	region3 := &MouseRegion{X: 0, Y: 0, Width: 10, Height: 10, ZIndex: 5}

	h.AddRegion(region1)
	h.AddRegion(region2)
	h.AddRegion(region3)

	// Should be sorted by highest ZIndex first
	assert.Equal(t, region2, h.regions[0])
	assert.Equal(t, region3, h.regions[1])
	assert.Equal(t, region1, h.regions[2])
}

func TestMouseHandler_FindRegionAt(t *testing.T) {
	h := NewMouseHandler()
	region1 := &MouseRegion{X: 0, Y: 0, Width: 10, Height: 10, ZIndex: 1}
	region2 := &MouseRegion{X: 5, Y: 5, Width: 10, Height: 10, ZIndex: 10}

	h.AddRegion(region1)
	h.AddRegion(region2)

	// Point only in region1
	found := h.findRegionAt(2, 2)
	assert.Equal(t, region1, found)

	// Point only in region2
	found = h.findRegionAt(12, 12)
	assert.Equal(t, region2, found)

	// Overlapping point - should return higher ZIndex
	found = h.findRegionAt(7, 7)
	assert.Equal(t, region2, found)

	// Point outside all regions
	found = h.findRegionAt(100, 100)
	assert.Nil(t, found)
}

func TestMouseHandler_HandleEvent_EnterLeave(t *testing.T) {
	h := NewMouseHandler()
	var enterCalled, leaveCalled bool

	region := &MouseRegion{
		X: 0, Y: 0, Width: 10, Height: 10,
		OnEnter: func(e *MouseEvent) { enterCalled = true },
		OnLeave: func(e *MouseEvent) { leaveCalled = true },
	}
	h.AddRegion(region)

	// Move into region
	h.HandleEvent(&MouseEvent{Type: MouseMove, X: 5, Y: 5})
	assert.True(t, enterCalled)
	assert.False(t, leaveCalled)

	// Move out of region
	enterCalled = false
	h.HandleEvent(&MouseEvent{Type: MouseMove, X: 20, Y: 20})
	assert.True(t, leaveCalled)
}

func TestMouseHandler_HandleEvent_Click(t *testing.T) {
	h := NewMouseHandler()
	var pressCalled, clickCalled bool

	region := &MouseRegion{
		X: 0, Y: 0, Width: 10, Height: 10,
		OnPress: func(e *MouseEvent) { pressCalled = true },
		OnClick: func(e *MouseEvent) { clickCalled = true },
	}
	h.AddRegion(region)

	// Press
	h.HandleEvent(&MouseEvent{Type: MousePress, X: 5, Y: 5, Button: MouseButtonLeft, Time: time.Now()})
	assert.True(t, pressCalled)

	// Release - handler synthesizes click
	h.HandleEvent(&MouseEvent{Type: MouseRelease, X: 5, Y: 5, Button: MouseButtonNone, Time: time.Now()})
	assert.True(t, clickCalled)
}

func TestMouseHandler_HandleEvent_PreSynthesizedClick(t *testing.T) {
	h := NewMouseHandler()
	var clickCount int

	region := &MouseRegion{
		X: 0, Y: 0, Width: 10, Height: 10,
		OnClick: func(e *MouseEvent) { clickCount++ },
	}
	h.AddRegion(region)

	// Press
	h.HandleEvent(&MouseEvent{Type: MousePress, X: 5, Y: 5, Button: MouseButtonLeft, Time: time.Now()})

	// Pre-synthesized click (from Runtime)
	h.HandleEvent(&MouseEvent{Type: MouseClick, X: 5, Y: 5, Button: MouseButtonLeft, Time: time.Now()})
	assert.Equal(t, 1, clickCount)

	// Release - should not generate another click because clickSynthesized is true
	h.HandleEvent(&MouseEvent{Type: MouseRelease, X: 5, Y: 5, Button: MouseButtonNone, Time: time.Now()})
	assert.Equal(t, 1, clickCount) // Still 1, not 2
}

func TestMouseHandler_HandleEvent_DoubleClick(t *testing.T) {
	h := NewMouseHandler()
	var doubleClickCalled bool

	region := &MouseRegion{
		X: 0, Y: 0, Width: 10, Height: 10,
		OnDoubleClick: func(e *MouseEvent) { doubleClickCalled = true },
	}
	h.AddRegion(region)

	now := time.Now()

	// First click
	h.HandleEvent(&MouseEvent{Type: MousePress, X: 5, Y: 5, Button: MouseButtonLeft, Time: now})
	h.HandleEvent(&MouseEvent{Type: MouseRelease, X: 5, Y: 5, Time: now})

	// Second click within threshold
	h.HandleEvent(&MouseEvent{Type: MousePress, X: 5, Y: 5, Button: MouseButtonLeft, Time: now.Add(100 * time.Millisecond)})
	h.HandleEvent(&MouseEvent{Type: MouseRelease, X: 5, Y: 5, Time: now.Add(100 * time.Millisecond)})

	assert.True(t, doubleClickCalled)
}

func TestMouseHandler_HandleEvent_Drag(t *testing.T) {
	h := NewMouseHandler()
	var dragStartCalled, dragCalled bool

	region := &MouseRegion{
		X: 0, Y: 0, Width: 20, Height: 20,
		OnDragStart: func(e *MouseEvent) { dragStartCalled = true },
		OnDrag:      func(e *MouseEvent) { dragCalled = true },
	}
	h.AddRegion(region)

	// Press
	h.HandleEvent(&MouseEvent{Type: MousePress, X: 5, Y: 5, Button: MouseButtonLeft, Time: time.Now()})

	// Drag - small movement, not enough to trigger drag
	h.HandleEvent(&MouseEvent{Type: MouseDrag, X: 6, Y: 6})
	assert.False(t, dragStartCalled)

	// Drag - larger movement
	h.HandleEvent(&MouseEvent{Type: MouseDrag, X: 15, Y: 15})
	assert.True(t, dragStartCalled)

	// Continue dragging
	h.HandleEvent(&MouseEvent{Type: MouseDrag, X: 18, Y: 18})
	assert.True(t, dragCalled)
}

func TestMouseHandler_CancelDrag(t *testing.T) {
	h := NewMouseHandler()
	var dragCanceled bool

	region := &MouseRegion{
		X: 0, Y: 0, Width: 20, Height: 20,
		OnDragStart: func(e *MouseEvent) {},
	}
	h.AddRegion(region)

	// Start drag
	h.HandleEvent(&MouseEvent{Type: MousePress, X: 5, Y: 5, Button: MouseButtonLeft})
	h.HandleEvent(&MouseEvent{Type: MouseDrag, X: 15, Y: 15})

	assert.True(t, h.isDragging)
	h.CancelDrag()
	assert.False(t, h.isDragging)
	assert.False(t, dragCanceled) // OnDragCancel not set, but state should be cleared
}

func TestMouseHandler_HandleEvent_Scroll(t *testing.T) {
	h := NewMouseHandler()
	var scrollCalled bool

	region := &MouseRegion{
		X: 0, Y: 0, Width: 10, Height: 10,
		OnScroll: func(e *MouseEvent) { scrollCalled = true },
	}
	h.AddRegion(region)

	h.HandleEvent(&MouseEvent{Type: MouseScroll, X: 5, Y: 5, Button: MouseButtonWheelUp})
	assert.True(t, scrollCalled)
}
