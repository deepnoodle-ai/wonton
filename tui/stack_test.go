package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestStack_Empty(t *testing.T) {
	s := Stack()
	w, h := s.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestStack_SingleChild(t *testing.T) {
	s := Stack(Text("Hello"))
	w, h := s.size(100, 100)
	assert.Equal(t, 5, w) // "Hello" is 5 chars
	assert.Equal(t, 1, h) // single line
}

func TestStack_MultipleChildren(t *testing.T) {
	s := Stack(
		Text("Line 1"),
		Text("Line 2"),
		Text("Line 3"),
	)
	w, h := s.size(100, 100)
	assert.Equal(t, 6, w) // "Line X" is 6 chars
	assert.Equal(t, 3, h) // 3 lines
}

func TestStack_Gap(t *testing.T) {
	s := Stack(
		Text("A"),
		Text("B"),
		Text("C"),
	).Gap(2)

	w, h := s.size(100, 100)
	assert.Equal(t, 1, w)     // single char width
	assert.Equal(t, 3+4, h)   // 3 lines + 2 gaps of 2
}

func TestStack_GapWithTwoChildren(t *testing.T) {
	s := Stack(
		Text("A"),
		Text("B"),
	).Gap(5)

	_, h := s.size(100, 100)
	assert.Equal(t, 2+5, h) // 2 lines + 1 gap of 5
}

func TestStack_Alignment(t *testing.T) {
	tests := []struct {
		name  string
		align Alignment
	}{
		{"left", AlignLeft},
		{"center", AlignCenter},
		{"right", AlignRight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Stack(
				Text("Short"),
				Text("Longer text"),
			).Align(tt.align)

			w, h := s.size(100, 100)
			assert.Equal(t, 11, w) // "Longer text" is widest
			assert.Equal(t, 2, h)
			assert.Equal(t, tt.align, s.alignment)
		})
	}
}

func TestStack_Render(t *testing.T) {
	var buf strings.Builder
	s := Stack(
		Text("Line 1"),
		Text("Line 2"),
	)

	err := Print(s, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Line 1"), "should contain Line 1")
	assert.True(t, strings.Contains(output, "Line 2"), "should contain Line 2")
}

func TestStack_RenderWithGap(t *testing.T) {
	var buf strings.Builder
	s := Stack(
		Text("A"),
		Text("B"),
	).Gap(1)

	err := Print(s, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "A"), "should contain A")
	assert.True(t, strings.Contains(output, "B"), "should contain B")
}

func TestStack_RenderWithAlignment(t *testing.T) {
	var buf strings.Builder
	s := Stack(
		Text("Hi"),
		Text("There"),
	).Align(AlignCenter)

	err := Print(s, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Hi"), "should contain Hi")
	assert.True(t, strings.Contains(output, "There"), "should contain There")
}

func TestStack_FlexibleChild(t *testing.T) {
	s := Stack(
		Text("Fixed"),
		Spacer(),
		Text("Bottom"),
	)

	// With height constraint, spacer should expand
	w, h := s.size(100, 10)
	assert.Equal(t, 6, w) // "Bottom" is widest
	// Total height should account for flexible spacer distribution
	assert.True(t, h >= 2, "should have at least height for text")
}

func TestStack_FlexibleChildWithSpace(t *testing.T) {
	s := Stack(
		Text("Top"),
		Spacer(),
		Text("Bottom"),
	)

	// When given height, spacer takes remaining space
	_, h := s.size(50, 20)
	// Height should equal maxHeight when there's a spacer
	assert.Equal(t, 20, h)
}

func TestStack_MultipleFlexChildren(t *testing.T) {
	s := Stack(
		Text("A"),
		Spacer().Flex(1),
		Spacer().Flex(2),
		Text("B"),
	)

	_, h := s.size(50, 30)
	// Should fill available height
	assert.Equal(t, 30, h)
}

func TestStack_PaddingModifier(t *testing.T) {
	s := Stack(Text("Hello"))
	padded := s.Padding(2)

	w, h := padded.size(100, 100)
	assert.Equal(t, 5+4, w) // 5 chars + 2 padding each side
	assert.Equal(t, 1+4, h) // 1 line + 2 padding each side
}

func TestStack_PaddingHVModifier(t *testing.T) {
	s := Stack(Text("Hi"))
	padded := s.PaddingHV(3, 1)

	w, h := padded.size(100, 100)
	assert.Equal(t, 2+6, w) // 2 chars + 3 padding each side
	assert.Equal(t, 1+2, h) // 1 line + 1 padding each side
}

func TestStack_PaddingLTRBModifier(t *testing.T) {
	s := Stack(Text("X"))
	padded := s.PaddingLTRB(1, 2, 3, 4)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+1+3, w) // 1 char + left 1 + right 3
	assert.Equal(t, 1+2+4, h) // 1 line + top 2 + bottom 4
}

func TestStack_BorderedModifier(t *testing.T) {
	s := Stack(Text("Box"))
	bordered := s.Bordered()

	assert.NotNil(t, bordered)
}

func TestStack_VaryingWidths(t *testing.T) {
	s := Stack(
		Text("A"),
		Text("ABC"),
		Text("ABCDE"),
		Text("AB"),
	)

	w, h := s.size(100, 100)
	assert.Equal(t, 5, w) // widest is "ABCDE"
	assert.Equal(t, 4, h) // 4 children
}

func TestStack_NoMaxConstraints(t *testing.T) {
	s := Stack(
		Text("Test"),
		Text("Another"),
	)

	// No max constraints (0, 0)
	w, h := s.size(0, 0)
	assert.Equal(t, 7, w) // "Another" is widest
	assert.Equal(t, 2, h)
}

func TestStack_Chaining(t *testing.T) {
	s := Stack(
		Text("A"),
		Text("B"),
	).Gap(1).Align(AlignCenter)

	assert.Equal(t, 1, s.gap)
	assert.Equal(t, AlignCenter, s.alignment)
}

func TestStack_GapWithEmptyView(t *testing.T) {
	// Empty views should not contribute to gap spacing
	s := Stack(
		Text("A"),
		Empty(),
		Text("B"),
	).Gap(2)

	_, h := s.size(100, 100)
	// Should be 2 lines + 1 gap (between A and B), not 2 gaps
	assert.Equal(t, 2+2, h) // 2 lines + 1 gap of 2
}

func TestStack_GapWithIfFalse(t *testing.T) {
	// If(false, ...) returns Empty, should not add extra gap
	s := Stack(
		Text("A"),
		If(false, Text("Hidden")),
		Text("B"),
	).Gap(1)

	_, h := s.size(100, 100)
	// Should be 2 lines + 1 gap, not 2 gaps
	assert.Equal(t, 2+1, h)
}

func TestStack_GapWithMultipleEmptyViews(t *testing.T) {
	// Multiple empty views should all be skipped
	s := Stack(
		Empty(),
		Text("A"),
		Empty(),
		Empty(),
		Text("B"),
		Empty(),
	).Gap(3)

	_, h := s.size(100, 100)
	// Should be 2 lines + 1 gap of 3
	assert.Equal(t, 2+3, h)
}

func TestStack_AllEmptyViews(t *testing.T) {
	s := Stack(
		Empty(),
		Empty(),
		Empty(),
	).Gap(5)

	w, h := s.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestStack_SingleVisibleWithEmpty(t *testing.T) {
	// Only one visible child means no gaps at all
	s := Stack(
		Empty(),
		Text("Only"),
		Empty(),
	).Gap(10)

	_, h := s.size(100, 100)
	assert.Equal(t, 1, h) // Just the text, no gaps
}

func TestStack_GapWithIfElse(t *testing.T) {
	// IfElse should work correctly too
	s := Stack(
		Text("A"),
		IfElse(false, Text("Then"), Empty()),
		Text("B"),
	).Gap(1)

	_, h := s.size(100, 100)
	// Should be 2 lines + 1 gap
	assert.Equal(t, 2+1, h)
}
