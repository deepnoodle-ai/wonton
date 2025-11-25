package gooey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test modifiers parsing
func TestParseMouseEvent_Modifiers(t *testing.T) {
	tests := []struct {
		name      string
		seq       []byte
		wantShift bool
		wantAlt   bool
		wantCtrl  bool
	}{
		{"no modifiers", []byte("<0;10;5M"), false, false, false},
		{"shift", []byte("<4;10;5M"), true, false, false},
		{"alt", []byte("<8;10;5M"), false, true, false},
		{"ctrl", []byte("<16;10;5M"), false, false, true},
		{"shift+alt", []byte("<12;10;5M"), true, true, false},
		{"shift+ctrl", []byte("<20;10;5M"), true, false, true},
		{"alt+ctrl", []byte("<24;10;5M"), false, true, true},
		{"all modifiers", []byte("<28;10;5M"), true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseMouseEvent(tt.seq)
			require.NoError(t, err)
			require.NotNil(t, event)

			assert.Equal(t, tt.wantShift, event.Modifiers&ModShift != 0, "Shift modifier")
			assert.Equal(t, tt.wantAlt, event.Modifiers&ModAlt != 0, "Alt modifier")
			assert.Equal(t, tt.wantCtrl, event.Modifiers&ModCtrl != 0, "Ctrl modifier")
		})
	}
}

// Test horizontal wheel
func TestParseMouseEvent_HorizontalWheel(t *testing.T) {
	// Wheel left: button 66 (64 + 2)
	seq := []byte("<66;10;5M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, MouseButtonWheelLeft, event.Button)
	assert.Equal(t, MouseScroll, event.Type)
	assert.Equal(t, -1, event.DeltaX)
	assert.Equal(t, 0, event.DeltaY)

	// Wheel right: button 67 (64 + 2 + 1)
	seq = []byte("<67;10;5M")
	event, err = ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, MouseButtonWheelRight, event.Button)
	assert.Equal(t, MouseScroll, event.Type)
	assert.Equal(t, 1, event.DeltaX)
	assert.Equal(t, 0, event.DeltaY)
}

// Test vertical wheel deltas
func TestParseMouseEvent_VerticalWheel(t *testing.T) {
	// Wheel up
	seq := []byte("<64;10;5M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	assert.Equal(t, MouseButtonWheelUp, event.Button)
	assert.Equal(t, -1, event.DeltaY)
	assert.Equal(t, 0, event.DeltaX)

	// Wheel down
	seq = []byte("<65;10;5M")
	event, err = ParseMouseEvent(seq)

	require.NoError(t, err)
	assert.Equal(t, MouseButtonWheelDown, event.Button)
	assert.Equal(t, 1, event.DeltaY)
	assert.Equal(t, 0, event.DeltaX)
}

// Test double-click detection
func TestMouseHandler_DoubleClick(t *testing.T) {
	handler := NewMouseHandler()
	doubleClickCount := 0

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnDoubleClick: func(event *MouseEvent) {
			doubleClickCount++
		},
	}

	handler.AddRegion(region)

	// First press
	pressEvent := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonLeft,
		Type:      MousePress,
		Time: time.Now(),
	}
	handler.HandleEvent(pressEvent)

	// First release (generates click)
	releaseEvent := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonNone,
		Type:      MouseRelease,
		Time: time.Now(),
	}
	handler.HandleEvent(releaseEvent)

	assert.Equal(t, 0, doubleClickCount, "Should not have double-click yet")

	// Second press (within threshold)
	time.Sleep(10 * time.Millisecond)
	pressEvent2 := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonLeft,
		Type:      MousePress,
		Time: time.Now(),
	}
	handler.HandleEvent(pressEvent2)

	// Second release (generates double-click)
	releaseEvent2 := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonNone,
		Type:      MouseRelease,
		Time: time.Now(),
	}
	handler.HandleEvent(releaseEvent2)

	assert.Equal(t, 1, doubleClickCount, "Should have one double-click")
}

// Test triple-click detection
func TestMouseHandler_TripleClick(t *testing.T) {
	handler := NewMouseHandler()
	tripleClickCount := 0

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnTripleClick: func(event *MouseEvent) {
			tripleClickCount++
		},
	}

	handler.AddRegion(region)

	// Perform three quick clicks
	for i := 0; i < 3; i++ {
		pressEvent := &MouseEvent{
			X:         15,
			Y:         15,
			Button:    MouseButtonLeft,
			Type:      MousePress,
			Time: time.Now(),
		}
		handler.HandleEvent(pressEvent)

		releaseEvent := &MouseEvent{
			X:         15,
			Y:         15,
			Button:    MouseButtonNone,
			Type:      MouseRelease,
			Time: time.Now(),
		}
		handler.HandleEvent(releaseEvent)

		time.Sleep(10 * time.Millisecond)
	}

	assert.Equal(t, 1, tripleClickCount, "Should have one triple-click")
}

// Test drag state machine
func TestMouseHandler_DragLifecycle(t *testing.T) {
	handler := NewMouseHandler()
	var events []MouseEventType

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnPress: func(event *MouseEvent) {
			events = append(events, event.Type)
		},
		OnDragStart: func(event *MouseEvent) {
			events = append(events, event.Type)
		},
		OnDrag: func(event *MouseEvent) {
			events = append(events, event.Type)
		},
		OnDragEnd: func(event *MouseEvent) {
			events = append(events, event.Type)
		},
	}

	handler.AddRegion(region)

	// Press
	pressEvent := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonLeft,
		Type:      MousePress,
		Time: time.Now(),
	}
	handler.HandleEvent(pressEvent)

	// Move a little (not enough to trigger drag)
	dragEvent1 := &MouseEvent{
		X:         16,
		Y:         16,
		Button:    MouseButtonLeft,
		Type:      MouseDrag,
		Time: time.Now(),
	}
	handler.HandleEvent(dragEvent1)

	// Move more (triggers drag start)
	dragEvent2 := &MouseEvent{
		X:         25,
		Y:         25,
		Button:    MouseButtonLeft,
		Type:      MouseDrag,
		Time: time.Now(),
	}
	handler.HandleEvent(dragEvent2)

	// Continue dragging
	dragEvent3 := &MouseEvent{
		X:         30,
		Y:         30,
		Button:    MouseButtonLeft,
		Type:      MouseDrag,
		Time: time.Now(),
	}
	handler.HandleEvent(dragEvent3)

	// Release (ends drag)
	releaseEvent := &MouseEvent{
		X:         30,
		Y:         30,
		Button:    MouseButtonNone,
		Type:      MouseRelease,
		Time: time.Now(),
	}
	handler.HandleEvent(releaseEvent)

	// Verify event sequence
	require.Len(t, events, 4, "Should have 4 events")
	assert.Equal(t, MousePress, events[0])
	assert.Equal(t, MouseDragStart, events[1])
	assert.Equal(t, MouseDrag, events[2])
	assert.Equal(t, MouseDragEnd, events[3])
}

// Test pointer capture
func TestMouseHandler_PointerCapture(t *testing.T) {
	handler := NewMouseHandler()
	region1Events := 0
	region2Events := 0

	region1 := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnPress: func(event *MouseEvent) {
			region1Events++
		},
		OnDrag: func(event *MouseEvent) {
			region1Events++
		},
		OnDragEnd: func(event *MouseEvent) {
			region1Events++
		},
	}

	region2 := &MouseRegion{
		X:      40,
		Y:      40,
		Width:  20,
		Height: 20,
		OnPress: func(event *MouseEvent) {
			region2Events++
		},
		OnDrag: func(event *MouseEvent) {
			region2Events++
		},
	}

	handler.AddRegion(region1)
	handler.AddRegion(region2)

	// Press in region1
	pressEvent := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonLeft,
		Type:      MousePress,
		Time: time.Now(),
	}
	handler.HandleEvent(pressEvent)

	// Drag outside region1 and into region2
	dragEvent := &MouseEvent{
		X:         45, // Inside region2
		Y:         45,
		Button:    MouseButtonLeft,
		Type:      MouseDrag,
		Time: time.Now(),
	}
	handler.HandleEvent(dragEvent)

	// Release in region2
	releaseEvent := &MouseEvent{
		X:         45,
		Y:         45,
		Button:    MouseButtonNone,
		Type:      MouseRelease,
		Time: time.Now(),
	}
	handler.HandleEvent(releaseEvent)

	// Region1 should receive all events due to capture
	assert.Greater(t, region1Events, 0, "Region1 should receive events")
	// Region2 should not receive any press events since it was captured by region1
	assert.Equal(t, 0, region2Events, "Region2 should not receive events during capture")
}

// Test enter/leave events
func TestMouseHandler_EnterLeave(t *testing.T) {
	handler := NewMouseHandler()
	var enterLeaveEvents []MouseEventType

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnEnter: func(event *MouseEvent) {
			enterLeaveEvents = append(enterLeaveEvents, event.Type)
		},
		OnLeave: func(event *MouseEvent) {
			enterLeaveEvents = append(enterLeaveEvents, event.Type)
		},
	}

	handler.AddRegion(region)

	// Move outside region (no event)
	moveEvent1 := &MouseEvent{
		X:         5,
		Y:         5,
		Button:    MouseButtonNone,
		Type:      MouseMove,
		Time: time.Now(),
	}
	handler.HandleEvent(moveEvent1)
	assert.Len(t, enterLeaveEvents, 0, "No events outside region")

	// Move into region (enter event + move event dispatched)
	moveEvent2 := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonNone,
		Type:      MouseMove,
		Time: time.Now(),
	}
	handler.HandleEvent(moveEvent2)
	assert.Len(t, enterLeaveEvents, 1, "Should have enter event")
	assert.Equal(t, MouseEnter, enterLeaveEvents[0])

	// Move within region (only move event, no enter/leave)
	moveEvent3 := &MouseEvent{
		X:         16,
		Y:         16,
		Button:    MouseButtonNone,
		Type:      MouseMove,
		Time: time.Now(),
	}
	handler.HandleEvent(moveEvent3)
	assert.Len(t, enterLeaveEvents, 1, "No new enter/leave events")

	// Move out of region (leave event)
	moveEvent4 := &MouseEvent{
		X:         5,
		Y:         5,
		Button:    MouseButtonNone,
		Type:      MouseMove,
		Time: time.Now(),
	}
	handler.HandleEvent(moveEvent4)
	assert.Len(t, enterLeaveEvents, 2, "Should have leave event")
	assert.Equal(t, MouseLeave, enterLeaveEvents[1])
}

// Test z-index ordering
func TestMouseHandler_ZIndex(t *testing.T) {
	handler := NewMouseHandler()
	clickedRegion := ""

	// Create overlapping regions with different z-indices
	region1 := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  30,
		Height: 30,
		ZIndex: 1,
		OnClick: func(event *MouseEvent) {
			clickedRegion = "region1"
		},
	}

	region2 := &MouseRegion{
		X:      15,
		Y:      15,
		Width:  30,
		Height: 30,
		ZIndex: 2, // Higher z-index
		OnClick: func(event *MouseEvent) {
			clickedRegion = "region2"
		},
	}

	handler.AddRegion(region1)
	handler.AddRegion(region2)

	// Click in overlapping area
	pressEvent := &MouseEvent{
		X:         20,
		Y:         20,
		Button:    MouseButtonLeft,
		Type:      MousePress,
		Time: time.Now(),
	}
	handler.HandleEvent(pressEvent)

	releaseEvent := &MouseEvent{
		X:         20,
		Y:         20,
		Button:    MouseButtonNone,
		Type:      MouseRelease,
		Time: time.Now(),
	}
	handler.HandleEvent(releaseEvent)

	// Region2 should receive the click due to higher z-index
	assert.Equal(t, "region2", clickedRegion, "Higher z-index region should receive click")
}

// Test drag cancel
func TestMouseHandler_CancelDrag(t *testing.T) {
	handler := NewMouseHandler()
	dragStarted := false
	dragCancelled := false

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnDragStart: func(event *MouseEvent) {
			dragStarted = true
		},
		OnDragEnd: func(event *MouseEvent) {
			// Should not be called on cancel
			t.Error("OnDragEnd should not be called when drag is cancelled")
		},
	}

	handler.AddRegion(region)

	// Press
	pressEvent := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonLeft,
		Type:      MousePress,
		Time: time.Now(),
	}
	handler.HandleEvent(pressEvent)

	// Drag to trigger drag start
	dragEvent := &MouseEvent{
		X:         25,
		Y:         25,
		Button:    MouseButtonLeft,
		Type:      MouseDrag,
		Time: time.Now(),
	}
	handler.HandleEvent(dragEvent)

	assert.True(t, dragStarted, "Drag should have started")

	// Cancel drag (e.g., user presses Escape)
	handler.CancelDrag()

	// Release should not trigger drag end since we cancelled
	releaseEvent := &MouseEvent{
		X:         25,
		Y:         25,
		Button:    MouseButtonNone,
		Type:      MouseRelease,
		Time: time.Now(),
	}
	handler.HandleEvent(releaseEvent)

	assert.False(t, dragCancelled, "Drag was cancelled")
}

// Test legacy handler compatibility
func TestMouseHandler_LegacyHandlers(t *testing.T) {
	handler := NewMouseHandler()
	clicked := false
	hovered := false

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		Handler: func(event *MouseEvent) {
			clicked = true
		},
		HoverHandler: func(hovering bool) {
			hovered = hovering
		},
	}

	handler.AddRegion(region)

	// Test hover (enter)
	moveEvent := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonNone,
		Type:      MouseMove,
		Time: time.Now(),
	}
	handler.HandleEvent(moveEvent)
	assert.True(t, hovered, "Legacy HoverHandler should be called on enter")

	// Test click
	pressEvent := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonLeft,
		Type:      MousePress,
		Time: time.Now(),
	}
	handler.HandleEvent(pressEvent)

	releaseEvent := &MouseEvent{
		X:         15,
		Y:         15,
		Button:    MouseButtonNone,
		Type:      MouseRelease,
		Time: time.Now(),
	}
	handler.HandleEvent(releaseEvent)

	assert.True(t, clicked, "Legacy Handler should be called on click")

	// Test hover (leave)
	moveEvent2 := &MouseEvent{
		X:         5,
		Y:         5,
		Button:    MouseButtonNone,
		Type:      MouseMove,
		Time: time.Now(),
	}
	handler.HandleEvent(moveEvent2)
	assert.False(t, hovered, "Legacy HoverHandler should be called on leave with false")
}
