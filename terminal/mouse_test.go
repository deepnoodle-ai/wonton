package terminal

import (
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestParseMouseEvent_SGRFormat_LeftClick(t *testing.T) {
	// SGR format: <button;x;yM (press) or <button;x;ym (release)
	tests := []struct {
		name       string
		seq        []byte
		wantButton MouseButton
		wantType   MouseEventType
		wantX      int
		wantY      int
	}{
		{
			name:       "left click at 0,0",
			seq:        []byte("<0;1;1M"),
			wantButton: MouseButtonLeft,
			wantType:   MousePress,
			wantX:      0,
			wantY:      0,
		},
		{
			name:       "left click at 10,5",
			seq:        []byte("<0;11;6M"),
			wantButton: MouseButtonLeft,
			wantType:   MousePress,
			wantX:      10,
			wantY:      5,
		},
		{
			name:       "left click at large coords",
			seq:        []byte("<0;256;128M"),
			wantButton: MouseButtonLeft,
			wantType:   MousePress,
			wantX:      255,
			wantY:      127,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseMouseEvent(tt.seq)
			assert.NoError(t, err)
			assert.NotNil(t, event)
			assert.Equal(t, tt.wantButton, event.Button)
			assert.Equal(t, tt.wantType, event.Type)
			assert.Equal(t, tt.wantX, event.X)
			assert.Equal(t, tt.wantY, event.Y)
		})
	}
}

func TestParseMouseEvent_SGRFormat_Release(t *testing.T) {
	// Release uses lowercase 'm'
	seq := []byte("<0;10;5m")
	event, err := ParseMouseEvent(seq)
	assert.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, MouseButtonNone, event.Button)
	assert.Equal(t, MouseRelease, event.Type)
	assert.Equal(t, 9, event.X)
	assert.Equal(t, 4, event.Y)
}

func TestParseMouseEvent_SGRFormat_MiddleClick(t *testing.T) {
	// Button 1 = middle button
	seq := []byte("<1;10;10M")
	event, err := ParseMouseEvent(seq)
	assert.NoError(t, err)
	assert.Equal(t, MouseButtonMiddle, event.Button)
	assert.Equal(t, MousePress, event.Type)
}

func TestParseMouseEvent_SGRFormat_RightClick(t *testing.T) {
	// Button 2 = right button
	seq := []byte("<2;10;10M")
	event, err := ParseMouseEvent(seq)
	assert.NoError(t, err)
	assert.Equal(t, MouseButtonRight, event.Button)
	assert.Equal(t, MousePress, event.Type)
}

func TestParseMouseEvent_SGRFormat_NoButton(t *testing.T) {
	// Button 3 = no button (used in some scenarios)
	seq := []byte("<3;10;10M")
	event, err := ParseMouseEvent(seq)
	assert.NoError(t, err)
	assert.Equal(t, MouseButtonNone, event.Button)
	assert.Equal(t, MousePress, event.Type)
}

func TestParseMouseEvent_SGRFormat_Modifiers(t *testing.T) {
	tests := []struct {
		name     string
		seq      []byte
		wantMods MouseModifiers
	}{
		{
			name:     "shift modifier",
			seq:      []byte("<4;10;10M"), // bit 2 (4) = shift
			wantMods: ModShift,
		},
		{
			name:     "alt modifier",
			seq:      []byte("<8;10;10M"), // bit 3 (8) = alt
			wantMods: ModAlt,
		},
		{
			name:     "ctrl modifier",
			seq:      []byte("<16;10;10M"), // bit 4 (16) = ctrl
			wantMods: ModCtrl,
		},
		{
			name:     "shift+alt modifiers",
			seq:      []byte("<12;10;10M"), // 4 + 8 = shift + alt
			wantMods: ModShift | ModAlt,
		},
		{
			name:     "all modifiers",
			seq:      []byte("<28;10;10M"), // 4 + 8 + 16 = shift + alt + ctrl
			wantMods: ModShift | ModAlt | ModCtrl,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseMouseEvent(tt.seq)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantMods, event.Modifiers)
		})
	}
}

func TestParseMouseEvent_SGRFormat_WheelEvents(t *testing.T) {
	tests := []struct {
		name       string
		seq        []byte
		wantButton MouseButton
		wantDeltaX int
		wantDeltaY int
	}{
		{
			name:       "wheel up",
			seq:        []byte("<64;10;10M"), // bit 6 (64) = wheel, bit 0 = 0 for up
			wantButton: MouseButtonWheelUp,
			wantDeltaY: -1,
		},
		{
			name:       "wheel down",
			seq:        []byte("<65;10;10M"), // bit 6 (64) + bit 0 (1) = wheel down
			wantButton: MouseButtonWheelDown,
			wantDeltaY: 1,
		},
		{
			name:       "wheel left",
			seq:        []byte("<66;10;10M"), // bit 6 (64) + bit 1 (2) = horizontal wheel left
			wantButton: MouseButtonWheelLeft,
			wantDeltaX: -1,
		},
		{
			name:       "wheel right",
			seq:        []byte("<67;10;10M"), // bit 6 (64) + bit 1 (2) + bit 0 (1) = horizontal wheel right
			wantButton: MouseButtonWheelRight,
			wantDeltaX: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseMouseEvent(tt.seq)
			assert.NoError(t, err)
			assert.Equal(t, MouseScroll, event.Type)
			assert.Equal(t, tt.wantButton, event.Button)
			assert.Equal(t, tt.wantDeltaX, event.DeltaX)
			assert.Equal(t, tt.wantDeltaY, event.DeltaY)
		})
	}
}

func TestParseMouseEvent_SGRFormat_Drag(t *testing.T) {
	tests := []struct {
		name       string
		seq        []byte
		wantButton MouseButton
		wantType   MouseEventType
	}{
		{
			name:       "left button drag",
			seq:        []byte("<32;10;10M"), // bit 5 (32) = motion + button 0
			wantButton: MouseButtonLeft,
			wantType:   MouseDrag,
		},
		{
			name:       "middle button drag",
			seq:        []byte("<33;10;10M"), // bit 5 (32) + button 1
			wantButton: MouseButtonMiddle,
			wantType:   MouseDrag,
		},
		{
			name:       "right button drag",
			seq:        []byte("<34;10;10M"), // bit 5 (32) + button 2
			wantButton: MouseButtonRight,
			wantType:   MouseDrag,
		},
		{
			name:       "mouse move (no button)",
			seq:        []byte("<35;10;10M"), // bit 5 (32) + button 3 = motion no button
			wantButton: MouseButtonNone,
			wantType:   MouseMove,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseMouseEvent(tt.seq)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantButton, event.Button)
			assert.Equal(t, tt.wantType, event.Type)
		})
	}
}

func TestParseMouseEvent_LegacyFormat(t *testing.T) {
	// Legacy format: 3 bytes, values offset by 32
	tests := []struct {
		name       string
		seq        []byte
		wantButton MouseButton
		wantX      int
		wantY      int
	}{
		{
			name:       "left click at 0,0",
			seq:        []byte{32, 33, 33}, // button 0, x=1-32=0, y=1-32=0 (1-indexed, but we subtract 1 more in code)
			wantButton: MouseButtonLeft,
			wantX:      0,
			wantY:      0,
		},
		{
			name:       "right click at 10,5",
			seq:        []byte{34, 43, 38}, // button 2, x=11, y=6 (1-indexed)
			wantButton: MouseButtonRight,
			wantX:      10,
			wantY:      5,
		},
		{
			name:       "middle click",
			seq:        []byte{33, 33, 33}, // button 1
			wantButton: MouseButtonMiddle,
			wantX:      0,
			wantY:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseMouseEvent(tt.seq)
			assert.NoError(t, err)
			assert.NotNil(t, event)
			assert.Equal(t, tt.wantButton, event.Button)
			assert.Equal(t, tt.wantX, event.X)
			assert.Equal(t, tt.wantY, event.Y)
			assert.Equal(t, MousePress, event.Type) // Legacy is always press
		})
	}
}

func TestParseMouseEvent_LegacyFormat_Modifiers(t *testing.T) {
	// Legacy format with shift modifier (bit 2 = 4)
	seq := []byte{36, 33, 33} // 32 + 4 = shift + left button
	event, err := ParseMouseEvent(seq)
	assert.NoError(t, err)
	assert.Equal(t, ModShift, event.Modifiers)
}

func TestParseMouseEvent_Errors(t *testing.T) {
	tests := []struct {
		name string
		seq  []byte
	}{
		{
			name: "too short",
			seq:  []byte{1, 2},
		},
		{
			name: "empty",
			seq:  []byte{},
		},
		{
			name: "single byte",
			seq:  []byte{1},
		},
		{
			name: "SGR incomplete - no terminator",
			seq:  []byte("<0;10;10"),
		},
		{
			name: "SGR malformed - missing coords",
			seq:  []byte("<0M"),
		},
		{
			name: "legacy wrong length",
			seq:  []byte{32, 33, 33, 34}, // 4 bytes, should be 3
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMouseEvent(tt.seq)
			assert.Error(t, err)
		})
	}
}

func TestParseMouseEvent_Timestamp(t *testing.T) {
	seq := []byte("<0;1;1M")
	before := time.Now()
	event, err := ParseMouseEvent(seq)
	after := time.Now()

	assert.NoError(t, err)
	assert.True(t, !event.Time.Before(before), "event time should be >= before")
	assert.True(t, !event.Time.After(after), "event time should be <= after")
}

func TestMouseRegion_Contains(t *testing.T) {
	region := &MouseRegion{
		X:      10,
		Y:      5,
		Width:  20,
		Height: 10,
	}

	tests := []struct {
		name string
		x, y int
		want bool
	}{
		{"top-left corner", 10, 5, true},
		{"top-right corner (exclusive)", 30, 5, false}, // X+Width is exclusive
		{"bottom-left corner (exclusive)", 10, 15, false},
		{"inside center", 20, 10, true},
		{"just outside left", 9, 10, false},
		{"just outside right", 30, 10, false},
		{"just outside top", 20, 4, false},
		{"just outside bottom", 20, 15, false},
		{"last valid X", 29, 10, true},
		{"last valid Y", 20, 14, true},
		{"origin when region not at origin", 0, 0, false},
		{"negative coords", -1, -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := region.Contains(tt.x, tt.y)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMouseRegion_Contains_ZeroSize(t *testing.T) {
	region := &MouseRegion{
		X:      10,
		Y:      10,
		Width:  0,
		Height: 0,
	}

	// Zero-size region should contain nothing
	assert.False(t, region.Contains(10, 10))
	assert.False(t, region.Contains(9, 9))
}

func TestMouseRegion_Contains_SingleCell(t *testing.T) {
	region := &MouseRegion{
		X:      5,
		Y:      5,
		Width:  1,
		Height: 1,
	}

	assert.True(t, region.Contains(5, 5))
	assert.False(t, region.Contains(6, 5))
	assert.False(t, region.Contains(5, 6))
	assert.False(t, region.Contains(4, 5))
	assert.False(t, region.Contains(5, 4))
}

func TestMouseEvent_Timestamp(t *testing.T) {
	now := time.Now()
	event := MouseEvent{
		X:    10,
		Y:    5,
		Time: now,
	}

	// Test Event interface implementation
	assert.Equal(t, now, event.Timestamp())
}

func TestMouseButton_Values(t *testing.T) {
	// Verify button constants have expected values
	assert.Equal(t, MouseButton(0), MouseButtonLeft)
	assert.Equal(t, MouseButton(1), MouseButtonMiddle)
	assert.Equal(t, MouseButton(2), MouseButtonRight)
	assert.Equal(t, MouseButton(3), MouseButtonNone)
}

func TestMouseEventType_Values(t *testing.T) {
	// Verify event type constants have expected values
	assert.Equal(t, MouseEventType(0), MousePress)
	assert.Equal(t, MouseEventType(1), MouseRelease)
	assert.Equal(t, MouseEventType(2), MouseClick)
	assert.Equal(t, MouseEventType(3), MouseDoubleClick)
	assert.Equal(t, MouseEventType(4), MouseTripleClick)
	assert.Equal(t, MouseEventType(5), MouseDrag)
}

func TestMouseModifiers_Values(t *testing.T) {
	// Verify modifier constants are powers of 2
	assert.Equal(t, MouseModifiers(1), ModShift)
	assert.Equal(t, MouseModifiers(2), ModAlt)
	assert.Equal(t, MouseModifiers(4), ModCtrl)
	assert.Equal(t, MouseModifiers(8), ModMeta)
}

func TestCursorStyle_Values(t *testing.T) {
	// Verify cursor style constants
	assert.Equal(t, CursorStyle(0), CursorDefault)
	assert.Equal(t, CursorStyle(1), CursorPointer)
	assert.Equal(t, CursorStyle(2), CursorText)
}

func TestParseMouseEvent_SGRFormat_WheelWithModifiers(t *testing.T) {
	// Wheel up with shift modifier
	// 64 (wheel) + 4 (shift) = 68
	seq := []byte("<68;10;10M")
	event, err := ParseMouseEvent(seq)
	assert.NoError(t, err)
	assert.Equal(t, MouseScroll, event.Type)
	assert.Equal(t, ModShift, event.Modifiers)
}

func TestParseMouseEvent_SGRFormat_DragWithModifiers(t *testing.T) {
	// Left drag with ctrl modifier
	// 32 (motion) + 16 (ctrl) + 0 (left) = 48
	seq := []byte("<48;10;10M")
	event, err := ParseMouseEvent(seq)
	assert.NoError(t, err)
	assert.Equal(t, MouseDrag, event.Type)
	assert.Equal(t, MouseButtonLeft, event.Button)
	assert.Equal(t, ModCtrl, event.Modifiers)
}
