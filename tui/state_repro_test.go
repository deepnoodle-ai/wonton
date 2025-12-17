package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTextArea_StateLoss(t *testing.T) {
	// Scenario: A TextArea is created without an external scroll binding.
	// We simulate an event that scrolls it, then "re-render" it (create a new instance).
	// We expect the state to be lost in the current implementation.

	// Frame 1: Create TextArea
	ta1 := TextArea(nil).Content("Line 1\nLine 2\nLine 3\nLine 4\nLine 5").Height(2)
	ta1.ID("my-textarea")

	// Simulate rendering Frame 1 (which would register focus handlers etc)
	// For this test, we just look at the struct behavior.
	// Internally, TextArea uses 'internal' field for scroll if binding is nil.
	
	// Simulate user scrolling down
	handler := &textAreaFocusHandler{area: ta1}
	handler.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	
	// Verify scroll increased in ta1
	assert.Equal(t, 1, ta1.getScrollY())

	// Frame 2: Re-create TextArea (as happens in View())
	ta2 := TextArea(nil).Content("Line 1\nLine 2\nLine 3\nLine 4\nLine 5").Height(2)
	ta2.ID("my-textarea")

	// Verify scroll is PERSISTED in new instance
	// This confirms that with the registry, state is preserved
	assert.Equal(t, 1, ta2.getScrollY())
}
