package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/require"
)

func TestParseMouseEvent_SGR_LeftClick(t *testing.T) {
	// SGR format: <0;10;5M for left click at (10,5)
	seq := []byte("<0;10;5M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, 9, event.X) // 0-based, so 10-1
	assert.Equal(t, 4, event.Y) // 0-based, so 5-1
	assert.Equal(t, MouseButtonLeft, event.Button)
	assert.Equal(t, MousePress, event.Type) // ParseMouseEvent returns Press events
}

func TestParseMouseEvent_SGR_RightClick(t *testing.T) {
	seq := []byte("<2;15;20M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, 14, event.X)
	assert.Equal(t, 19, event.Y)
	assert.Equal(t, MouseButtonRight, event.Button)
	assert.Equal(t, MousePress, event.Type)
}

func TestParseMouseEvent_SGR_MiddleClick(t *testing.T) {
	seq := []byte("<1;25;30M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, 24, event.X)
	assert.Equal(t, 29, event.Y)
	assert.Equal(t, MouseButtonMiddle, event.Button)
	assert.Equal(t, MousePress, event.Type)
}

func TestParseMouseEvent_SGR_Release(t *testing.T) {
	// Lowercase 'm' indicates release
	seq := []byte("<0;10;5m")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, MouseButtonNone, event.Button)
	assert.Equal(t, MouseRelease, event.Type)
}

func TestParseMouseEvent_SGR_WheelUp(t *testing.T) {
	// Button 64 (0x40) indicates wheel event
	seq := []byte("<64;10;5M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, MouseButtonWheelUp, event.Button)
	assert.Equal(t, MouseScroll, event.Type)
}

func TestParseMouseEvent_SGR_WheelDown(t *testing.T) {
	// Button 65 (0x41) indicates wheel down
	seq := []byte("<65;10;5M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, MouseButtonWheelDown, event.Button)
	assert.Equal(t, MouseScroll, event.Type)
}

func TestParseMouseEvent_SGR_Drag(t *testing.T) {
	// Button with bit 5 set (32) indicates drag
	seq := []byte("<32;10;5M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, MouseDrag, event.Type)
}

func TestParseMouseEvent_InvalidSequence_TooShort(t *testing.T) {
	seq := []byte("<0")
	event, err := ParseMouseEvent(seq)

	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestParseMouseEvent_InvalidSequence_NoTerminator(t *testing.T) {
	seq := []byte("<0;10;5")
	event, err := ParseMouseEvent(seq)

	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestParseMouseEvent_InvalidSequence_BadFormat(t *testing.T) {
	seq := []byte("<abc;def;ghiM")
	_, err := ParseMouseEvent(seq)

	assert.Error(t, err)
}

func TestParseMouseEvent_LargeCoordinates(t *testing.T) {
	// SGR format supports coordinates > 223
	seq := []byte("<0;250;300M")
	event, err := ParseMouseEvent(seq)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, 249, event.X)
	assert.Equal(t, 299, event.Y)
}

func TestMouseRegion_Contains(t *testing.T) {
	region := &MouseRegion{
		X:      10,
		Y:      5,
		Width:  20,
		Height: 10,
	}

	// Inside
	assert.True(t, region.Contains(10, 5))
	assert.True(t, region.Contains(15, 10))
	assert.True(t, region.Contains(29, 14))

	// Outside
	assert.False(t, region.Contains(9, 5))   // Left of region
	assert.False(t, region.Contains(10, 4))  // Above region
	assert.False(t, region.Contains(30, 10)) // Right of region
	assert.False(t, region.Contains(15, 15)) // Below region
}

func TestMouseRegion_Contains_EdgeCases(t *testing.T) {
	region := &MouseRegion{
		X:      0,
		Y:      0,
		Width:  10,
		Height: 10,
	}

	// Top-left corner
	assert.True(t, region.Contains(0, 0))

	// Bottom-right corner (exclusive)
	assert.False(t, region.Contains(10, 10))
	assert.True(t, region.Contains(9, 9))
}

func TestMouseHandler_NewMouseHandler(t *testing.T) {
	handler := NewMouseHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.regions)
	assert.Equal(t, 0, len(handler.regions))
	assert.Nil(t, handler.hoveredRegion)
}

func TestMouseHandler_AddRegion(t *testing.T) {
	handler := NewMouseHandler()
	region := &MouseRegion{
		X:      10,
		Y:      5,
		Width:  20,
		Height: 10,
		Label:  "Test",
	}

	handler.AddRegion(region)
	assert.Equal(t, 1, len(handler.regions))
	assert.Equal(t, region, handler.regions[0])
}

func TestMouseHandler_ClearRegions(t *testing.T) {
	handler := NewMouseHandler()
	handler.AddRegion(&MouseRegion{X: 0, Y: 0, Width: 10, Height: 10})
	handler.AddRegion(&MouseRegion{X: 10, Y: 10, Width: 10, Height: 10})

	assert.Equal(t, 2, len(handler.regions))

	handler.ClearRegions()
	assert.Equal(t, 0, len(handler.regions))
	assert.Nil(t, handler.hoveredRegion)
}

func TestMouseHandler_HandleEvent_Click(t *testing.T) {
	handler := NewMouseHandler()
	clicked := false

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnClick: func(event *MouseEvent) {
			clicked = true
		},
	}

	handler.AddRegion(region)

	event := &MouseEvent{
		X:      15,
		Y:      15,
		Button: MouseButtonLeft,
		Type:   MouseClick,
	}

	handler.HandleEvent(event)
	assert.True(t, clicked, "OnClick should be called for click inside region")
}

func TestMouseHandler_HandleEvent_OutsideRegion(t *testing.T) {
	handler := NewMouseHandler()
	clicked := false

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnClick: func(event *MouseEvent) {
			clicked = true
		},
	}

	handler.AddRegion(region)

	event := &MouseEvent{
		X:      5,
		Y:      5,
		Button: MouseButtonLeft,
		Type:   MouseClick,
	}

	handler.HandleEvent(event)
	assert.False(t, clicked, "OnClick should not be called for click outside region")
}

func TestMouseHandler_HandleEvent_Hover(t *testing.T) {
	handler := NewMouseHandler()
	hoverState := false

	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  20,
		Height: 20,
		OnEnter: func(event *MouseEvent) {
			hoverState = true
		},
		OnLeave: func(event *MouseEvent) {
			hoverState = false
		},
	}

	handler.AddRegion(region)

	// Trigger hover with MouseMove (the correct event type for hover)
	event := &MouseEvent{
		X:    15,
		Y:    15,
		Type: MouseMove,
	}

	handler.HandleEvent(event)
	assert.True(t, hoverState, "Hover should be true when entering region")

	// Leave region
	event = &MouseEvent{
		X:    5,
		Y:    5,
		Type: MouseMove,
	}

	handler.HandleEvent(event)
	assert.False(t, hoverState, "Hover should be false when leaving region")
}

func TestMouseHandler_HandleEvent_MultipleRegions(t *testing.T) {
	handler := NewMouseHandler()
	region1Clicked := false
	region2Clicked := false

	region1 := &MouseRegion{
		X:      0,
		Y:      0,
		Width:  10,
		Height: 10,
		OnClick: func(event *MouseEvent) {
			region1Clicked = true
		},
	}

	region2 := &MouseRegion{
		X:      20,
		Y:      20,
		Width:  10,
		Height: 10,
		OnClick: func(event *MouseEvent) {
			region2Clicked = true
		},
	}

	handler.AddRegion(region1)
	handler.AddRegion(region2)

	// Click in region 1
	event := &MouseEvent{
		X:      5,
		Y:      5,
		Button: MouseButtonLeft,
		Type:   MouseClick,
	}

	handler.HandleEvent(event)
	assert.True(t, region1Clicked)
	assert.False(t, region2Clicked)

	region1Clicked = false

	// Click in region 2
	event = &MouseEvent{
		X:      25,
		Y:      25,
		Button: MouseButtonLeft,
		Type:   MouseClick,
	}

	handler.HandleEvent(event)
	assert.False(t, region1Clicked)
	assert.True(t, region2Clicked)
}

func TestMouseButton_Constants(t *testing.T) {
	assert.Equal(t, MouseButton(0), MouseButtonLeft)
	assert.Equal(t, MouseButton(1), MouseButtonMiddle)
	assert.Equal(t, MouseButton(2), MouseButtonRight)
	assert.Equal(t, MouseButton(3), MouseButtonNone)
	assert.Equal(t, MouseButton(4), MouseButtonWheelUp)
	assert.Equal(t, MouseButton(5), MouseButtonWheelDown)
	assert.Equal(t, MouseButton(6), MouseButtonWheelLeft)
	assert.Equal(t, MouseButton(7), MouseButtonWheelRight)
}

func TestMouseEventType_Constants(t *testing.T) {
	assert.Equal(t, MouseEventType(0), MousePress)
	assert.Equal(t, MouseEventType(1), MouseRelease)
	assert.Equal(t, MouseEventType(2), MouseClick)
	assert.Equal(t, MouseEventType(3), MouseDoubleClick)
	assert.Equal(t, MouseEventType(4), MouseTripleClick)
	assert.Equal(t, MouseEventType(5), MouseDrag)
	assert.Equal(t, MouseEventType(6), MouseDragStart)
	assert.Equal(t, MouseEventType(7), MouseDragEnd)
	assert.Equal(t, MouseEventType(8), MouseDragCancel)
	assert.Equal(t, MouseEventType(9), MouseMove)
	assert.Equal(t, MouseEventType(10), MouseEnter)
	assert.Equal(t, MouseEventType(11), MouseLeave)
	assert.Equal(t, MouseEventType(12), MouseScroll)
}
