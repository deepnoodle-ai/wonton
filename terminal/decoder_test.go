package terminal

import (
	"bytes"
	"io"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// TestKeyDecoder_SingleByteKeys tests simple single-byte key detection
func TestKeyDecoder_SingleByteKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected KeyEvent
	}{
		{"Enter (CR)", []byte{0x0D}, KeyEvent{Key: KeyEnter}},
		{"Enter (LF)", []byte{0x0A}, KeyEvent{Key: KeyEnter}},
		{"Tab", []byte{0x09}, KeyEvent{Key: KeyTab}},
		{"Backspace (DEL)", []byte{0x7F}, KeyEvent{Key: KeyBackspace}},
		{"Backspace (BS)", []byte{0x08}, KeyEvent{Key: KeyBackspace}},
		{"Escape", []byte{0x1B}, KeyEvent{Key: KeyEscape}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Key, tt.expected.Key)
		})
	}
}

// TestKeyDecoder_CtrlKeys tests Ctrl+Letter combinations
func TestKeyDecoder_CtrlKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Key
	}{
		{"Ctrl+A", []byte{0x01}, KeyCtrlA},
		{"Ctrl+B", []byte{0x02}, KeyCtrlB},
		{"Ctrl+C", []byte{0x03}, KeyCtrlC},
		{"Ctrl+D", []byte{0x04}, KeyCtrlD},
		{"Ctrl+E", []byte{0x05}, KeyCtrlE},
		{"Ctrl+F", []byte{0x06}, KeyCtrlF},
		{"Ctrl+G", []byte{0x07}, KeyCtrlG},
		{"Ctrl+K", []byte{0x0B}, KeyCtrlK},
		{"Ctrl+L", []byte{0x0C}, KeyCtrlL},
		{"Ctrl+N", []byte{0x0E}, KeyCtrlN},
		{"Ctrl+O", []byte{0x0F}, KeyCtrlO},
		{"Ctrl+P", []byte{0x10}, KeyCtrlP},
		{"Ctrl+Q", []byte{0x11}, KeyCtrlQ},
		{"Ctrl+R", []byte{0x12}, KeyCtrlR},
		{"Ctrl+S", []byte{0x13}, KeyCtrlS},
		{"Ctrl+T", []byte{0x14}, KeyCtrlT},
		{"Ctrl+U", []byte{0x15}, KeyCtrlU},
		{"Ctrl+V", []byte{0x16}, KeyCtrlV},
		{"Ctrl+W", []byte{0x17}, KeyCtrlW},
		{"Ctrl+X", []byte{0x18}, KeyCtrlX},
		{"Ctrl+Y", []byte{0x19}, KeyCtrlY},
		{"Ctrl+Z", []byte{0x1A}, KeyCtrlZ},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Key, tt.expected)
			assert.True(t, event.Ctrl)
		})
	}
}

// TestKeyDecoder_ArrowKeys tests arrow key escape sequences
func TestKeyDecoder_ArrowKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Key
	}{
		{"Arrow Up", []byte{0x1B, '[', 'A'}, KeyArrowUp},
		{"Arrow Down", []byte{0x1B, '[', 'B'}, KeyArrowDown},
		{"Arrow Right", []byte{0x1B, '[', 'C'}, KeyArrowRight},
		{"Arrow Left", []byte{0x1B, '[', 'D'}, KeyArrowLeft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Key, tt.expected)
		})
	}
}

// TestKeyDecoder_ShiftTab tests Shift+Tab (BackTab) detection
func TestKeyDecoder_ShiftTab(t *testing.T) {
	// Shift+Tab sends ESC [ Z
	input := []byte{0x1B, '[', 'Z'}
	decoder := NewKeyDecoder(bytes.NewReader(input))
	event, err := decoder.ReadKeyEvent()

	assert.NoError(t, err)
	assert.Equal(t, event.Key, KeyTab)
	assert.True(t, event.Shift)
}

// TestKeyDecoder_FunctionKeys tests F1-F12 function keys
func TestKeyDecoder_FunctionKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Key
	}{
		// ESC [ 1 N ~ format
		{"F1 (format 1)", []byte{0x1B, '[', '1', '1', '~'}, KeyF1},
		{"F2 (format 1)", []byte{0x1B, '[', '1', '2', '~'}, KeyF2},
		{"F3 (format 1)", []byte{0x1B, '[', '1', '3', '~'}, KeyF3},
		{"F4 (format 1)", []byte{0x1B, '[', '1', '4', '~'}, KeyF4},
		{"F5 (format 1)", []byte{0x1B, '[', '1', '5', '~'}, KeyF5},
		{"F6", []byte{0x1B, '[', '1', '7', '~'}, KeyF6},
		{"F7", []byte{0x1B, '[', '1', '8', '~'}, KeyF7},
		{"F8", []byte{0x1B, '[', '1', '9', '~'}, KeyF8},
		{"F9", []byte{0x1B, '[', '2', '0', '~'}, KeyF9},
		{"F10", []byte{0x1B, '[', '2', '1', '~'}, KeyF10},
		{"F11", []byte{0x1B, '[', '2', '3', '~'}, KeyF11},
		{"F12", []byte{0x1B, '[', '2', '4', '~'}, KeyF12},
		// ESC O P format (alternate)
		{"F1 (format 2)", []byte{0x1B, 'O', 'P'}, KeyF1},
		{"F2 (format 2)", []byte{0x1B, 'O', 'Q'}, KeyF2},
		{"F3 (format 2)", []byte{0x1B, 'O', 'R'}, KeyF3},
		{"F4 (format 2)", []byte{0x1B, 'O', 'S'}, KeyF4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Key, tt.expected)
		})
	}
}

// TestKeyDecoder_NavigationKeys tests Home, End, PageUp, PageDown, Insert, Delete
func TestKeyDecoder_NavigationKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Key
	}{
		{"Home (format 1)", []byte{0x1B, '[', 'H'}, KeyHome},
		{"Home (format 2)", []byte{0x1B, '[', '1', '~'}, KeyHome},
		{"Home (format 3)", []byte{0x1B, 'O', 'H'}, KeyHome},
		{"End (format 1)", []byte{0x1B, '[', 'F'}, KeyEnd},
		{"End (format 2)", []byte{0x1B, '[', '4', '~'}, KeyEnd},
		{"End (format 3)", []byte{0x1B, 'O', 'F'}, KeyEnd},
		{"Insert", []byte{0x1B, '[', '2', '~'}, KeyInsert},
		{"Delete", []byte{0x1B, '[', '3', '~'}, KeyDelete},
		{"PageUp", []byte{0x1B, '[', '5', '~'}, KeyPageUp},
		{"PageDown", []byte{0x1B, '[', '6', '~'}, KeyPageDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Key, tt.expected)
		})
	}
}

// TestKeyDecoder_PrintableCharacters tests ASCII printable characters
func TestKeyDecoder_PrintableCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected rune
	}{
		{"Space", []byte{0x20}, ' '},
		{"Letter a", []byte{'a'}, 'a'},
		{"Letter Z", []byte{'Z'}, 'Z'},
		{"Digit 0", []byte{'0'}, '0'},
		{"Digit 9", []byte{'9'}, '9'},
		{"Symbol @", []byte{'@'}, '@'},
		{"Symbol ~", []byte{'~'}, '~'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Rune, tt.expected)
		})
	}
}

// TestKeyDecoder_UTF8Characters tests multi-byte UTF-8 characters
func TestKeyDecoder_UTF8Characters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected rune
	}{
		{"Emoji 😀", "😀", '😀'},
		{"Chinese 你", "你", '你'},
		{"Greek α", "α", 'α'},
		{"Cyrillic Д", "Д", 'Д'},
		{"Arabic ا", "ا", 'ا'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader([]byte(tt.input)))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Rune, tt.expected)
		})
	}
}

// TestKeyDecoder_AltModifier tests Alt+key combinations
func TestKeyDecoder_AltModifier(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected rune
	}{
		{"Alt+a", []byte{0x1B, 'a'}, 'a'},
		{"Alt+z", []byte{0x1B, 'z'}, 'z'},
		{"Alt+1", []byte{0x1B, '1'}, '1'},
		{"Alt+@", []byte{0x1B, '@'}, '@'},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Rune, tt.expected)
			assert.True(t, event.Alt)
		})
	}
}

// TestKeyDecoder_ModifiedArrowKeys tests arrow keys with modifiers
func TestKeyDecoder_ModifiedArrowKeys(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedKey Key
		expectCtrl  bool
		expectAlt   bool
	}{
		{"Ctrl+ArrowUp", []byte{0x1B, '[', '1', ';', '5', 'A'}, KeyArrowUp, true, false},
		{"Alt+ArrowDown", []byte{0x1B, '[', '1', ';', '3', 'B'}, KeyArrowDown, false, true},
		{"Ctrl+Alt+ArrowRight", []byte{0x1B, '[', '1', ';', '7', 'C'}, KeyArrowRight, true, true},
		{"Shift+Ctrl+Alt+ArrowLeft", []byte{0x1B, '[', '1', ';', '8', 'D'}, KeyArrowLeft, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadKeyEvent()

			assert.NoError(t, err)
			assert.Equal(t, event.Key, tt.expectedKey)
			assert.Equal(t, event.Ctrl, tt.expectCtrl)
			assert.Equal(t, event.Alt, tt.expectAlt)
		})
	}
}

// TestKeyDecoder_EOF tests EOF error handling
func TestKeyDecoder_EOF(t *testing.T) {
	decoder := NewKeyDecoder(bytes.NewReader([]byte{}))
	_, err := decoder.ReadKeyEvent()

	assert.ErrorIs(t, err, io.EOF)
}

// TestKeyDecoder_UnknownSequence tests handling of unknown escape sequences
func TestKeyDecoder_UnknownSequence(t *testing.T) {
	// Some random escape sequence that we don't recognize
	input := []byte{0x1B, '[', '9', '9', 'Z'}
	decoder := NewKeyDecoder(bytes.NewReader(input))
	event, err := decoder.ReadKeyEvent()

	assert.NoError(t, err)
	assert.Equal(t, event.Key, KeyUnknown)
}

// TestKeyDecoder_SequentialReads tests reading multiple key events in sequence
// Note: This test reads complete escape sequences separately to simulate
// how terminal input actually works (escape sequences arrive as atomic units)
func TestKeyDecoder_SequentialReads(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected interface{} // either rune or Key
	}{
		{"letter a", []byte{'a'}, 'a'},
		{"arrow up", []byte{0x1B, '[', 'A'}, KeyArrowUp},
		{"letter b", []byte{'b'}, 'b'},
		{"enter", []byte{0x0D}, KeyEnter},
	}

	// Concatenate all inputs
	var allInput []byte
	for _, tt := range tests {
		allInput = append(allInput, tt.input...)
	}

	decoder := NewKeyDecoder(bytes.NewReader(allInput))

	for i, tt := range tests {
		event, err := decoder.ReadKeyEvent()
		assert.NoError(t, err, "test %d (%s)", i, tt.name)

		switch expected := tt.expected.(type) {
		case rune:
			assert.Equal(t, event.Rune, expected, "test %d (%s)", i, tt.name)
		case Key:
			assert.Equal(t, event.Key, expected, "test %d (%s)", i, tt.name)
		}
	}

	// Should be EOF now
	_, err := decoder.ReadKeyEvent()
	assert.ErrorIs(t, err, io.EOF)
}

// TestKeyDecoder_ReadEvent_MouseEvent tests that ReadEvent can decode SGR mouse events
func TestKeyDecoder_ReadEvent_MouseEvent(t *testing.T) {
	tests := []struct {
		name           string
		input          []byte
		expectedX      int
		expectedY      int
		expectedButton MouseButton
		expectedType   MouseEventType
	}{
		{
			name:           "Left click at (10, 5)",
			input:          []byte{0x1B, '[', '<', '0', ';', '1', '1', ';', '6', 'M'},
			expectedX:      10,
			expectedY:      5,
			expectedButton: MouseButtonLeft,
			expectedType:   MousePress,
		},
		{
			name:           "Right click at (20, 15)",
			input:          []byte{0x1B, '[', '<', '2', ';', '2', '1', ';', '1', '6', 'M'},
			expectedX:      20,
			expectedY:      15,
			expectedButton: MouseButtonRight,
			expectedType:   MousePress,
		},
		{
			name:           "Mouse release",
			input:          []byte{0x1B, '[', '<', '0', ';', '5', ';', '5', 'm'},
			expectedX:      4,
			expectedY:      4,
			expectedButton: MouseButtonNone,
			expectedType:   MouseRelease,
		},
		{
			name:           "Mouse wheel up",
			input:          []byte{0x1B, '[', '<', '6', '4', ';', '1', '0', ';', '1', '0', 'M'},
			expectedX:      9,
			expectedY:      9,
			expectedButton: MouseButtonWheelUp,
			expectedType:   MouseScroll,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadEvent()

			assert.NoError(t, err)
			mouseEvent, ok := event.(MouseEvent)
			assert.True(t, ok)

			assert.Equal(t, mouseEvent.X, tt.expectedX)
			assert.Equal(t, mouseEvent.Y, tt.expectedY)
			assert.Equal(t, mouseEvent.Button, tt.expectedButton)
			assert.Equal(t, mouseEvent.Type, tt.expectedType)
		})
	}
}

// TestKeyDecoder_ReadEvent_KeyEvent tests that ReadEvent still returns KeyEvents properly
func TestKeyDecoder_ReadEvent_KeyEvent(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedKey Key
		expectedR   rune
	}{
		{"Enter", []byte{0x0D}, KeyEnter, 0},
		{"Letter a", []byte{'a'}, KeyUnknown, 'a'},
		{"Arrow Up", []byte{0x1B, '[', 'A'}, KeyArrowUp, 0},
		{"Ctrl+C", []byte{0x03}, KeyCtrlC, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadEvent()

			assert.NoError(t, err)
			keyEvent, ok := event.(KeyEvent)
			assert.True(t, ok)

			if tt.expectedKey != KeyUnknown {
				assert.Equal(t, keyEvent.Key, tt.expectedKey)
			}
			if tt.expectedR != 0 {
				assert.Equal(t, keyEvent.Rune, tt.expectedR)
			}
		})
	}
}

// TestKeyDecoder_ReadEvent_MixedInput tests reading mixed mouse and key events
func TestKeyDecoder_ReadEvent_MixedInput(t *testing.T) {
	// Simulate: 'a', mouse click at (5,5), 'b', mouse release
	input := []byte{
		'a',                                          // Key 'a'
		0x1B, '[', '<', '0', ';', '6', ';', '6', 'M', // Mouse left press at (5,5)
		'b',                                          // Key 'b'
		0x1B, '[', '<', '0', ';', '6', ';', '6', 'm', // Mouse release
	}

	decoder := NewKeyDecoder(bytes.NewReader(input))

	// Event 1: Key 'a'
	event, err := decoder.ReadEvent()
	assert.NoError(t, err, "event 1")
	keyEvent, ok := event.(KeyEvent)
	assert.True(t, ok, "event 1")
	assert.Equal(t, keyEvent.Rune, 'a', "event 1")

	// Event 2: Mouse press
	event, err = decoder.ReadEvent()
	assert.NoError(t, err, "event 2")
	mouseEvent, ok := event.(MouseEvent)
	assert.True(t, ok, "event 2")
	assert.Equal(t, mouseEvent.Type, MousePress, "event 2")
	assert.Equal(t, mouseEvent.X, 5, "event 2")
	assert.Equal(t, mouseEvent.Y, 5, "event 2")

	// Event 3: Key 'b'
	event, err = decoder.ReadEvent()
	assert.NoError(t, err, "event 3")
	keyEvent, ok = event.(KeyEvent)
	assert.True(t, ok, "event 3")
	assert.Equal(t, keyEvent.Rune, 'b', "event 3")

	// Event 4: Mouse release
	event, err = decoder.ReadEvent()
	assert.NoError(t, err, "event 4")
	mouseEvent, ok = event.(MouseEvent)
	assert.True(t, ok, "event 4")
	assert.Equal(t, mouseEvent.Type, MouseRelease, "event 4")
}

func TestKeyDecoder_ReadEvent_BracketedPaste(t *testing.T) {
	input := []byte{
		0x1B, '[', '2', '0', '0', '~',
		'l', 'i', 'n', 'e', '1', '\r', '\n',
		'l', 'i', 'n', 'e', '2', '\t', 'e', 'n', 'd',
		'\r', 'l', 'i', 'n', 'e', '3',
		0x1B, '[', '2', '0', '1', '~',
	}
	decoder := NewKeyDecoder(bytes.NewReader(input))
	decoder.SetPasteTabWidth(2)

	event, err := decoder.ReadEvent()
	assert.NoError(t, err)

	keyEvent, ok := event.(KeyEvent)
	assert.True(t, ok)

	expected := "line1\nline2  end\nline3"
	assert.Equal(t, keyEvent.Paste, expected)
}

func TestKeyDecoder_ReadEvent_KittyCSIuModifiers(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedKey Key
		expectCtrl  bool
		expectShift bool
	}{
		{
			name:        "Ctrl+Enter via CSI u",
			input:       []byte{0x1B, '[', '1', '3', ';', '5', 'u'},
			expectedKey: KeyEnter,
			expectCtrl:  true,
			expectShift: true,
		},
		{
			name:        "Ctrl+C via CSI u",
			input:       []byte{0x1B, '[', '9', '9', ';', '5', 'u'},
			expectedKey: KeyCtrlC,
			expectCtrl:  true,
			expectShift: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadEvent()
			assert.NoError(t, err)

			keyEvent, ok := event.(KeyEvent)
			assert.True(t, ok)

			assert.Equal(t, keyEvent.Key, tt.expectedKey)
			assert.Equal(t, keyEvent.Ctrl, tt.expectCtrl)
			assert.Equal(t, keyEvent.Shift, tt.expectShift)
		})
	}
}

func TestKeyDecoder_ReadEvent_EscapeStandalone(t *testing.T) {
	decoder := NewKeyDecoder(bytes.NewReader([]byte{0x1B}))
	event, err := decoder.ReadEvent()
	assert.NoError(t, err)
	keyEvent, ok := event.(KeyEvent)
	assert.True(t, ok)
	assert.Equal(t, keyEvent.Key, KeyEscape)
}

// TestKeyDecoder_CursorPositionResponse tests parsing of cursor position responses
// Cursor position response format: ESC [ row ; col R
func TestKeyDecoder_CursorPositionResponse(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expectedRow int
		expectedCol int
	}{
		{"Position 1,1", []byte{0x1B, '[', '1', ';', '1', 'R'}, 1, 1},
		{"Position 10,20", []byte{0x1B, '[', '1', '0', ';', '2', '0', 'R'}, 10, 20},
		{"Position 24,80", []byte{0x1B, '[', '2', '4', ';', '8', '0', 'R'}, 24, 80},
		{"Position 100,200", []byte{0x1B, '[', '1', '0', '0', ';', '2', '0', '0', 'R'}, 100, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewKeyDecoder(bytes.NewReader(tt.input))
			event, err := decoder.ReadEvent()
			assert.NoError(t, err)

			// Should be a CursorPositionEvent, not a KeyEvent
			cpEvent, ok := event.(CursorPositionEvent)
			assert.True(t, ok, "expected CursorPositionEvent, got %T", event)
			assert.Equal(t, tt.expectedRow, cpEvent.Row)
			assert.Equal(t, tt.expectedCol, cpEvent.Col)
		})
	}
}

// TestKeyDecoder_ReadKeyEvent_CursorPositionReturnsUnknown tests that ReadKeyEvent
// returns KeyUnknown for cursor position responses (since it can only return KeyEvent)
func TestKeyDecoder_ReadKeyEvent_CursorPositionReturnsUnknown(t *testing.T) {
	// Cursor position response: ESC [ 10 ; 20 R
	input := []byte{0x1B, '[', '1', '0', ';', '2', '0', 'R'}
	decoder := NewKeyDecoder(bytes.NewReader(input))
	event, err := decoder.ReadKeyEvent()
	assert.NoError(t, err)
	assert.Equal(t, KeyUnknown, event.Key)
}
