package tui

import "testing"

func newPlaceholderInput(t *testing.T) *textInput {
	t.Helper()
	ti := newTextInput().WithPastePlaceholderMode(true)
	ti.SetFocused(true)
	ti.HandleKey(KeyEvent{Rune: 'a'})
	ti.HandlePaste("line1\nline2")
	if ti.DisplayText() != "a[pasted 2 lines]" {
		t.Fatalf("unexpected display %q", ti.DisplayText())
	}
	return ti
}

// The cursor must never rest inside a paste placeholder.
func TestTextInputCursorSkipsPastePlaceholder(t *testing.T) {
	ti := newPlaceholderInput(t)
	ti.HandleKey(KeyEvent{Key: KeyHome})
	ti.HandleKey(KeyEvent{Key: KeyArrowRight})
	ti.HandleKey(KeyEvent{Key: KeyArrowRight})
	if ti.CursorPos != len(ti.DisplayText()) {
		t.Fatalf("Right should jump over the placeholder, cursor=%d", ti.CursorPos)
	}
	ti.HandleKey(KeyEvent{Key: KeyArrowLeft})
	if ti.CursorPos != 1 {
		t.Fatalf("Left should jump over the placeholder, cursor=%d", ti.CursorPos)
	}
}

// Editing with the cursor inside a placeholder must not panic or leave a
// negative cursor (reached here by setting CursorPos directly).
func TestTextInputEditInsidePastePlaceholder(t *testing.T) {
	ti := newPlaceholderInput(t)
	ti.CursorPos = 2
	ti.HandleKey(KeyEvent{Key: KeyBackspace})
	if ti.CursorPos != 1 || ti.Value() != "a" {
		t.Fatalf("backspace: cursor=%d value=%q", ti.CursorPos, ti.Value())
	}
	ti.HandleKey(KeyEvent{Rune: 'b'})
	if ti.Value() != "ab" {
		t.Fatalf("typing after backspace: value=%q", ti.Value())
	}

	ti = newPlaceholderInput(t)
	ti.CursorPos = 2
	ti.HandleKey(KeyEvent{Key: KeyDelete})
	if ti.CursorPos != 1 || ti.Value() != "a" {
		t.Fatalf("delete: cursor=%d value=%q", ti.CursorPos, ti.Value())
	}

	ti = newPlaceholderInput(t)
	ti.CursorPos = 2
	ti.HandlePaste("z")
	if ti.Value() != "aline1\nline2z" || ti.CursorPos != len(ti.DisplayText()) {
		t.Fatalf("paste: cursor=%d value=%q", ti.CursorPos, ti.Value())
	}

	ti = newPlaceholderInput(t)
	ti.CursorPos = 2
	ti.HandleKey(KeyEvent{Rune: 'b'})
	if ti.Value() != "aline1\nline2b" || ti.CursorPos != len(ti.DisplayText()) {
		t.Fatalf("insert: cursor=%d value=%q", ti.CursorPos, ti.Value())
	}
}
