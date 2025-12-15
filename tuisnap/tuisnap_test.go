package tuisnap

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// mockView is a simple view for testing
type mockView struct {
	text string
}

func (v *mockView) render(frame interface {
	PrintStyled(x, y int, text string, style interface{})
}, bounds struct{ Min, Max struct{ X, Y int } }) {
	// This is a simplified render - actual implementation uses tui types
}

func (v *mockView) size(maxWidth, maxHeight int) (int, int) {
	return len(v.text), 1
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"TestSimple", "TestSimple"},
		{"Test/SubTest", "Test_SubTest"},
		{"Test With Spaces", "Test_With_Spaces"},
		{"Test:Colon", "Test_Colon"},
	}

	for _, tt := range tests {
		result := sanitizeName(tt.input)
		assert.Equal(t, tt.expected, result, "sanitizeName(%q)", tt.input)
	}
}

func TestGenerateDiff(t *testing.T) {
	expected := "line1\nline2\nline3"
	actual := "line1\nmodified\nline3"

	diff := generateDiff(expected, actual)

	assert.Contains(t, diff, "--- Expected")
	assert.Contains(t, diff, "+++ Actual")
	assert.Contains(t, diff, "line2")
	assert.Contains(t, diff, "modified")
}

func TestGenerateDiffIdentical(t *testing.T) {
	text := "line1\nline2"
	diff := generateDiff(text, text)

	// Should only contain headers, no actual diff content
	assert.True(t, strings.HasPrefix(diff, "--- Expected"))
	assert.True(t, strings.Contains(diff, "+++ Actual"))
	// No @@ markers for identical content
	assert.False(t, strings.Contains(diff, "@@"))
}
