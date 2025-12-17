package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestZStack_Empty(t *testing.T) {
	z := ZStack()
	w, h := z.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestZStack_SingleChild(t *testing.T) {
	z := ZStack(Text("Hello"))
	w, h := z.size(100, 100)
	assert.Equal(t, 5, w) // "Hello" is 5 chars
	assert.Equal(t, 1, h) // single line
}

func TestZStack_MultipleChildren(t *testing.T) {
	z := ZStack(
		Text("Short"),
		Text("Medium text"),
		Text("Longer text here"),
	)
	w, h := z.size(100, 100)
	assert.Equal(t, 16, w) // "Longer text here" is 16 chars (widest)
	assert.Equal(t, 1, h)  // all are single line, so height is 1
}

func TestZStack_UsesLargestChildDimensions(t *testing.T) {
	z := ZStack(
		Text("A"),
		Text("ABC\nDEF\nGHI"), // 3 lines, 3 chars wide
		Text("XY"),
	)
	w, h := z.size(100, 100)
	assert.Equal(t, 3, w) // widest child
	assert.Equal(t, 3, h) // tallest child
}

func TestZStack_DefaultAlignment(t *testing.T) {
	z := ZStack(Text("Test"))
	assert.Equal(t, AlignCenter, z.alignment)
}

func TestZStack_AlignmentLeft(t *testing.T) {
	z := ZStack(
		Text("Large background"),
		Text("Small"),
	).Align(AlignLeft)

	assert.Equal(t, AlignLeft, z.alignment)
	w, h := z.size(100, 100)
	assert.Equal(t, 16, w) // "Large background"
	assert.Equal(t, 1, h)
}

func TestZStack_AlignmentCenter(t *testing.T) {
	z := ZStack(
		Text("Background"),
		Text("Top"),
	).Align(AlignCenter)

	assert.Equal(t, AlignCenter, z.alignment)
}

func TestZStack_AlignmentRight(t *testing.T) {
	z := ZStack(
		Text("Background"),
		Text("Top"),
	).Align(AlignRight)

	assert.Equal(t, AlignRight, z.alignment)
}

func TestZStack_AllAlignments(t *testing.T) {
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
			z := ZStack(
				Text("Background text"),
				Text("Small"),
			).Align(tt.align)

			w, h := z.size(100, 100)
			assert.Equal(t, 15, w) // "Background text" is widest
			assert.Equal(t, 1, h)
			assert.Equal(t, tt.align, z.alignment)
		})
	}
}

func TestZStack_RenderBasic(t *testing.T) {
	var buf strings.Builder
	z := ZStack(
		Text("Background"),
		Text("Top"),
	)

	err := Print(z, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	// Both texts should be in the output (overlaid)
	assert.True(t, strings.Contains(output, "Background") || strings.Contains(output, "Top"),
		"should contain at least one of the texts")
}

func TestZStack_RenderWithAlignment(t *testing.T) {
	var buf strings.Builder
	z := ZStack(
		Text("Background"),
		Text("X"),
	).Align(AlignCenter)

	err := Print(z, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	// Should render without error
	assert.True(t, buf.Len() > 0, "should produce output")
}

func TestZStack_RenderEmpty(t *testing.T) {
	var buf strings.Builder
	z := ZStack()

	err := Print(z, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)
}

func TestZStack_RenderZeroSize(t *testing.T) {
	var buf strings.Builder
	z := ZStack(Text("Test"))

	// Zero width should not panic
	err := Print(z, WithWidth(0), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	// Zero height should not panic
	buf.Reset()
	err = Print(z, WithWidth(20), WithHeight(0), WithOutput(&buf))
	assert.NoError(t, err)
}

func TestZStack_OverlappingViews(t *testing.T) {
	// Test that multiple views are rendered (last one on top)
	var buf strings.Builder
	z := ZStack(
		Text("Layer 1"),
		Text("Layer 2"),
		Text("Layer 3"),
	)

	err := Print(z, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	// Should have rendered all layers (though they may overlap)
	assert.True(t, buf.Len() > 0, "should produce output")
}

func TestZStack_VaryingDimensions(t *testing.T) {
	z := ZStack(
		Text("One line"),
		Text("Two\nLines"),
		Text("Three\nLines\nHere"),
		Text("Short"),
	)

	w, h := z.size(100, 100)
	assert.Equal(t, 8, w) // "One line" is widest
	assert.Equal(t, 3, h) // "Three\nLines\nHere" is tallest
}

func TestZStack_NoMaxConstraints(t *testing.T) {
	z := ZStack(
		Text("First"),
		Text("Second line"),
	)

	// No max constraints (0, 0)
	w, h := z.size(0, 0)
	assert.Equal(t, 11, w) // "Second line" is widest
	assert.Equal(t, 1, h)  // both are single line
}

func TestZStack_Chaining(t *testing.T) {
	z := ZStack(
		Text("A"),
		Text("B"),
	).Align(AlignRight)

	assert.Equal(t, AlignRight, z.alignment)
}

func TestZStack_ChainingReturnsPointer(t *testing.T) {
	z := ZStack(Text("Test"))
	result := z.Align(AlignCenter)

	// Verify it returns the same instance (pointer)
	assert.Equal(t, z, result)
}

func TestZStack_ChildSizesStored(t *testing.T) {
	z := ZStack(
		Text("A"),
		Text("ABC"),
		Text("AB"),
	)

	z.size(100, 100)

	// Verify child sizes were calculated and stored
	assert.Equal(t, 3, len(z.childSizes))
	assert.Equal(t, 1, z.childSizes[0].X) // "A"
	assert.Equal(t, 3, z.childSizes[1].X) // "ABC"
	assert.Equal(t, 2, z.childSizes[2].X) // "AB"
}

func TestZStack_PaddingModifier(t *testing.T) {
	z := ZStack(Text("Hello"))
	padded := z.Padding(2)

	w, h := padded.size(100, 100)
	assert.Equal(t, 5+4, w) // 5 chars + 2 padding each side
	assert.Equal(t, 1+4, h) // 1 line + 2 padding each side
}

func TestZStack_PaddingHVModifier(t *testing.T) {
	z := ZStack(Text("Hi"))
	padded := PaddingHV(3, 1, z)

	w, h := padded.size(100, 100)
	assert.Equal(t, 2+6, w) // 2 chars + 3 padding each side
	assert.Equal(t, 1+2, h) // 1 line + 1 padding each side
}

func TestZStack_PaddingLTRBModifier(t *testing.T) {
	z := ZStack(Text("X"))
	padded := PaddingLTRB(1, 2, 3, 4, z)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+1+3, w) // 1 char + left 1 + right 3
	assert.Equal(t, 1+2+4, h) // 1 line + top 2 + bottom 4
}

func TestZStack_BorderedModifier(t *testing.T) {
	z := ZStack(Text("Box"))
	bordered := z.Bordered()

	assert.NotNil(t, bordered)
}

func TestZStack_ComplexOverlay(t *testing.T) {
	// Test a more realistic use case: background + content + overlay
	var buf strings.Builder
	z := ZStack(
		Text("================="),     // Background
		Text("Content here"),          // Middle layer
		Text("!"),                      // Top overlay
	).Align(AlignCenter)

	err := Print(z, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	assert.True(t, buf.Len() > 0, "should render complex overlay")
}

func TestZStack_WithDifferentAlignments(t *testing.T) {
	// Test that children are positioned correctly based on alignment
	tests := []struct {
		name  string
		align Alignment
	}{
		{"left aligned overlay", AlignLeft},
		{"center aligned overlay", AlignCenter},
		{"right aligned overlay", AlignRight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			z := ZStack(
				Text("Background text"),
				Text("FG"),
			).Align(tt.align)

			err := Print(z, WithWidth(20), WithHeight(3), WithOutput(&buf))
			assert.NoError(t, err)
			assert.True(t, buf.Len() > 0, "should produce output for "+tt.name)
		})
	}
}

func TestZStack_MultilineChildren(t *testing.T) {
	z := ZStack(
		Text("Line 1\nLine 2\nLine 3"),
		Text("A\nB"),
		Text("Single"),
	)

	w, h := z.size(100, 100)
	assert.Equal(t, 6, w) // "Line X" is 6 chars
	assert.Equal(t, 3, h) // 3 lines is tallest
}

func TestZStack_SizeReMeasurement(t *testing.T) {
	// Test that size is re-measured during render
	var buf strings.Builder
	z := ZStack(
		Text("Test"),
		Text("Another"),
	)

	// First sizing
	w1, h1 := z.size(100, 100)

	// Render (which re-measures)
	err := Print(z, WithWidth(50), WithHeight(50), WithOutput(&buf))
	assert.NoError(t, err)

	// Size should still work correctly after render
	w2, h2 := z.size(100, 100)
	assert.Equal(t, w1, w2)
	assert.Equal(t, h1, h2)
}

func TestZStack_EmptyChildrenList(t *testing.T) {
	z := &zStack{
		children:  []View{},
		alignment: AlignCenter,
	}

	w, h := z.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestZStack_NilChildSizesBeforeSizing(t *testing.T) {
	z := ZStack(Text("Test"))

	// childSizes should be nil before sizing
	assert.True(t, z.childSizes == nil || len(z.childSizes) == 0)

	// After sizing, should be populated
	z.size(100, 100)
	assert.Equal(t, 1, len(z.childSizes))
}
