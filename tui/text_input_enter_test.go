package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// textInput reports Enter as handled only when it uses it. An Enter it does
// not use must reach the application.
func TestTextInputEnterHandledOnlyWhenUsed(t *testing.T) {
	for _, tc := range []struct {
		name          string
		event         KeyEvent
		multiline     bool
		submitOnEnter bool
		onSubmit      bool
		wantHandled   bool
		wantSubmit    bool
		wantValue     string
	}{
		{"enter without OnSubmit", KeyEvent{Key: KeyEnter}, false, true, false, false, false, "ab"},
		{"enter with OnSubmit", KeyEvent{Key: KeyEnter}, false, true, true, true, true, "ab"},
		{"enter with SubmitOnEnter off", KeyEvent{Key: KeyEnter}, false, false, true, false, false, "ab"},
		{"shift+enter single-line", KeyEvent{Key: KeyEnter, Shift: true}, false, true, false, false, false, "ab"},
		{"shift+enter multiline", KeyEvent{Key: KeyEnter, Shift: true}, true, true, false, true, false, "ab\n"},
		{"enter multiline without OnSubmit", KeyEvent{Key: KeyEnter}, true, true, false, false, false, "ab"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := newTextInput().WithMultilineMode(tc.multiline).WithSubmitOnEnter(tc.submitOnEnter)
			input.SetFocused(true)
			input.SetValue("ab")
			submitted := false
			if tc.onSubmit {
				input.OnSubmit = func(string) { submitted = true }
			}
			assert.Equal(t, tc.wantHandled, input.HandleKey(tc.event))
			assert.Equal(t, tc.wantSubmit, submitted)
			assert.Equal(t, tc.wantValue, input.Value())
		})
	}
}
