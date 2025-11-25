package gooey

import (
	"bytes"
	"io"
	"testing"
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Key != tt.expected.Key {
				t.Errorf("expected key %v, got %v", tt.expected.Key, event.Key)
			}
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Key != tt.expected {
				t.Errorf("expected key %v, got %v", tt.expected, event.Key)
			}

			if !event.Ctrl {
				t.Errorf("expected Ctrl modifier to be true")
			}
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Key != tt.expected {
				t.Errorf("expected key %v, got %v", tt.expected, event.Key)
			}
		})
	}
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Key != tt.expected {
				t.Errorf("expected key %v, got %v", tt.expected, event.Key)
			}
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Key != tt.expected {
				t.Errorf("expected key %v, got %v", tt.expected, event.Key)
			}
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Rune != tt.expected {
				t.Errorf("expected rune %q, got %q", tt.expected, event.Rune)
			}
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Rune != tt.expected {
				t.Errorf("expected rune %q, got %q", tt.expected, event.Rune)
			}
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Rune != tt.expected {
				t.Errorf("expected rune %q, got %q", tt.expected, event.Rune)
			}

			if !event.Alt {
				t.Errorf("expected Alt modifier to be true")
			}
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

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if event.Key != tt.expectedKey {
				t.Errorf("expected key %v, got %v", tt.expectedKey, event.Key)
			}

			if event.Ctrl != tt.expectCtrl {
				t.Errorf("expected Ctrl=%v, got %v", tt.expectCtrl, event.Ctrl)
			}

			if event.Alt != tt.expectAlt {
				t.Errorf("expected Alt=%v, got %v", tt.expectAlt, event.Alt)
			}
		})
	}
}

// TestKeyDecoder_EOF tests EOF error handling
func TestKeyDecoder_EOF(t *testing.T) {
	decoder := NewKeyDecoder(bytes.NewReader([]byte{}))
	_, err := decoder.ReadKeyEvent()

	if err != io.EOF {
		t.Errorf("expected EOF error, got %v", err)
	}
}

// TestKeyDecoder_UnknownSequence tests handling of unknown escape sequences
func TestKeyDecoder_UnknownSequence(t *testing.T) {
	// Some random escape sequence that we don't recognize
	input := []byte{0x1B, '[', '9', '9', 'Z'}
	decoder := NewKeyDecoder(bytes.NewReader(input))
	event, err := decoder.ReadKeyEvent()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should return KeyUnknown for unrecognized sequences
	if event.Key != KeyUnknown {
		t.Errorf("expected KeyUnknown for unrecognized sequence, got %v", event.Key)
	}
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
		if err != nil {
			t.Fatalf("test %d (%s): unexpected error: %v", i, tt.name, err)
		}

		switch expected := tt.expected.(type) {
		case rune:
			if event.Rune != expected {
				t.Errorf("test %d (%s): expected rune %q, got %q", i, tt.name, expected, event.Rune)
			}
		case Key:
			if event.Key != expected {
				t.Errorf("test %d (%s): expected key %v, got %v", i, tt.name, expected, event.Key)
			}
		}
	}

	// Should be EOF now
	_, err := decoder.ReadKeyEvent()
	if err != io.EOF {
		t.Errorf("expected EOF after all events, got %v", err)
	}
}


