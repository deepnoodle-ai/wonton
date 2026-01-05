package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// Gap spacing tests (additional to view_test.go)

func TestGroup_GapWithTwoChildren(t *testing.T) {
	g := Group(
		Text("AA"),
		Text("BB"),
	).Gap(5)

	w, _ := g.size(100, 100)
	assert.Equal(t, 4+5, w) // 4 chars + 1 gap of 5
}

func TestGroup_GapWithSingleChild(t *testing.T) {
	g := Group(Text("Solo")).Gap(10)
	w, _ := g.size(100, 100)
	assert.Equal(t, 4, w) // no gap with single child
}

func TestGroup_GapZero(t *testing.T) {
	g := Group(
		Text("A"),
		Text("B"),
	).Gap(0)

	w, _ := g.size(100, 100)
	assert.Equal(t, 2, w) // no spacing
}

// Alignment tests

func TestGroup_AlignmentLeft(t *testing.T) {
	g := Group(
		Text("Short"),
		Stack(Text("A"), Text("B"), Text("C")),
	).Align(AlignLeft)

	_, h := g.size(100, 100)
	assert.Equal(t, 3, h) // tallest child
	assert.Equal(t, AlignLeft, g.alignment)
}

func TestGroup_AlignmentCenter(t *testing.T) {
	g := Group(
		Text("Short"),
		Stack(Text("A"), Text("B"), Text("C")),
	).Align(AlignCenter)

	_, h := g.size(100, 100)
	assert.Equal(t, 3, h)
	assert.Equal(t, AlignCenter, g.alignment)
}

func TestGroup_AlignmentRight(t *testing.T) {
	g := Group(
		Text("Short"),
		Stack(Text("A"), Text("B"), Text("C")),
	).Align(AlignRight)

	_, h := g.size(100, 100)
	assert.Equal(t, 3, h)
	assert.Equal(t, AlignRight, g.alignment)
}

func TestGroup_AlignmentWithVaryingHeights(t *testing.T) {
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
			g := Group(
				Text("X"),                              // 1 line tall
				Stack(Text("A"), Text("B")),            // 2 lines tall
				Stack(Text("1"), Text("2"), Text("3")), // 3 lines tall
			).Align(tt.align)

			_, h := g.size(100, 100)
			assert.Equal(t, 3, h) // height of tallest child
			assert.Equal(t, tt.align, g.alignment)
		})
	}
}

// Flexible children tests

func TestGroup_FlexibleChild(t *testing.T) {
	g := Group(
		Text("Left"),
		Spacer(),
		Text("Right"),
	)

	// With width constraint, spacer should expand
	w, h := g.size(50, 10)
	assert.True(t, w >= 10, "should have at least width for text")
	assert.Equal(t, 1, h)
}

func TestGroup_FlexibleChildWithSpace(t *testing.T) {
	g := Group(
		Text("A"),
		Spacer(),
		Text("B"),
	)

	// When given width constraint, spacer takes remaining space
	w, _ := g.size(20, 10)
	assert.Equal(t, 20, w) // should fill to maxWidth
}

func TestGroup_MultipleFlexChildren(t *testing.T) {
	g := Group(
		Text("A"),
		Spacer().Flex(1),
		Spacer().Flex(2),
		Text("B"),
	)

	// With width constraint
	w, _ := g.size(30, 10)
	assert.Equal(t, 30, w) // should fill available width
}

func TestGroup_FlexibleChildrenProportional(t *testing.T) {
	g := Group(
		Spacer().Flex(1),
		Spacer().Flex(3),
	)

	// Both flexible, should distribute proportionally
	w, _ := g.size(40, 10)
	assert.Equal(t, 40, w)
}

func TestGroup_OnlyFlexibleChildren(t *testing.T) {
	g := Group(
		Spacer(),
		Spacer(),
		Spacer(),
	)

	w, _ := g.size(60, 10)
	assert.Equal(t, 60, w) // should fill available width
}

func TestGroup_FlexibleChildNoConstraint(t *testing.T) {
	g := Group(
		Text("Fixed"),
		Spacer(),
	)

	// No max constraints - spacer has no space to fill
	w, _ := g.size(0, 0)
	assert.Equal(t, 5, w) // just the fixed text width
}

func TestGroup_FlexibleWithMinWidth(t *testing.T) {
	g := Group(
		Text("A"),
		Spacer().MinWidth(10),
		Text("B"),
	)

	// Spacer has minimum width
	w, _ := g.size(100, 10)
	assert.True(t, w >= 12, "should be at least text + min spacer width")
}

// Render behavior tests (additional to view_test.go)

func TestGroup_RenderWithGap(t *testing.T) {
	var buf strings.Builder
	g := Group(
		Text("Left"),
		Text("Right"),
	).Gap(3)

	err := Print(g, PrintConfig{Width: 80, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Left"), "should contain Left")
	assert.True(t, strings.Contains(output, "Right"), "should contain Right")
}

func TestGroup_RenderWithAlignment(t *testing.T) {
	var buf strings.Builder
	g := Group(
		Text("Short"),
		Stack(Text("Tall1"), Text("Tall2")),
	).Align(AlignCenter)

	err := Print(g, PrintConfig{Width: 80, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Short"), "should contain Short")
	assert.True(t, strings.Contains(output, "Tall1"), "should contain Tall1")
}

func TestGroup_RenderWithFlex(t *testing.T) {
	var buf strings.Builder
	g := Group(
		Text("Left"),
		Spacer(),
		Text("Right"),
	)

	err := Print(g, PrintConfig{Width: 40, Height: 5, Output: &buf})
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Left"), "should contain Left")
	assert.True(t, strings.Contains(output, "Right"), "should contain Right")
}

// Edge cases

func TestGroup_EmptyRender(t *testing.T) {
	var buf strings.Builder
	g := Group()

	err := Print(g, PrintConfig{Width: 80, Height: 10, Output: &buf})
	assert.NoError(t, err)
	// Should not error on empty group
}

func TestGroup_ZeroWidth(t *testing.T) {
	g := Group(Text("A"), Text("B"))
	w, h := g.size(0, 10)
	assert.Equal(t, 2, w) // natural width
	assert.Equal(t, 1, h)
}

func TestGroup_ZeroHeight(t *testing.T) {
	g := Group(Text("A"), Text("B"))
	w, h := g.size(10, 0)
	assert.Equal(t, 2, w)
	assert.Equal(t, 1, h)
}

func TestGroup_ClippingOverflow(t *testing.T) {
	var buf strings.Builder
	g := Group(
		Text("AAAAA"),
		Text("BBBBB"),
		Text("CCCCC"),
	)

	// Width constraint smaller than content
	err := Print(g, PrintConfig{Width: 10, Height: 5, Output: &buf})
	assert.NoError(t, err)
	// Should not error when clipping
}

func TestGroup_SingleChildClipping(t *testing.T) {
	var buf strings.Builder
	g := Group(Text("VeryLongText"))

	err := Print(g, PrintConfig{Width: 5, Height: 1, Output: &buf})
	assert.NoError(t, err)
	// Should clip content gracefully
}

func TestGroup_WithEmptyChildren(t *testing.T) {
	g := Group(
		Empty(),
		Text("A"),
		Empty(),
		Text("B"),
		Empty(),
	)

	w, h := g.size(100, 100)
	assert.Equal(t, 2, w) // only non-empty children contribute
	assert.Equal(t, 1, h)
}

func TestGroup_AllEmptyChildren(t *testing.T) {
	g := Group(Empty(), Empty(), Empty())
	w, h := g.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h) // all empty children result in 0x0 group
}

func TestGroup_GapWithEmptyView(t *testing.T) {
	// Empty views should not contribute to gap spacing
	g := Group(
		Text("A"),
		Empty(),
		Text("B"),
	).Gap(3)

	w, _ := g.size(100, 100)
	// Should be 2 chars + 1 gap (between A and B), not 2 gaps
	assert.Equal(t, 2+3, w) // 2 chars + 1 gap of 3
}

func TestGroup_GapWithIfFalse(t *testing.T) {
	// If(false, ...) returns Empty, should not add extra gap
	g := Group(
		Text("A"),
		If(false, Text("Hidden")),
		Text("B"),
	).Gap(2)

	w, _ := g.size(100, 100)
	// Should be 2 chars + 1 gap, not 2 gaps
	assert.Equal(t, 2+2, w)
}

func TestGroup_GapWithMultipleEmptyViews(t *testing.T) {
	// Multiple empty views should all be skipped
	g := Group(
		Empty(),
		Text("A"),
		Empty(),
		Empty(),
		Text("B"),
		Empty(),
	).Gap(5)

	w, _ := g.size(100, 100)
	// Should be 2 chars + 1 gap of 5
	assert.Equal(t, 2+5, w)
}

func TestGroup_AllEmptyViewsWithGap(t *testing.T) {
	g := Group(
		Empty(),
		Empty(),
		Empty(),
	).Gap(10)

	w, h := g.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestGroup_SingleVisibleWithEmpty(t *testing.T) {
	// Only one visible child means no gaps at all
	g := Group(
		Empty(),
		Text("Only"),
		Empty(),
	).Gap(10)

	w, _ := g.size(100, 100)
	assert.Equal(t, 4, w) // Just "Only", no gaps
}

// Chaining tests (additional to view_test.go)

func TestGroup_ChainingMultiple(t *testing.T) {
	g := Group(Text("X")).Gap(1).Align(AlignRight).Gap(5)

	// Last Gap call should override
	assert.Equal(t, 5, g.gap)
	assert.Equal(t, AlignRight, g.alignment)
}

// Complex layout tests

func TestGroup_NestedGroups(t *testing.T) {
	inner := Group(Text("A"), Text("B"))
	outer := Group(inner, Text("C"))

	w, h := outer.size(100, 100)
	assert.True(t, w >= 3, "should accommodate nested content")
	assert.Equal(t, 1, h)
}

func TestGroup_MixedFlexAndFixed(t *testing.T) {
	g := Group(
		Text("Fixed1"),
		Spacer().Flex(1),
		Text("Fixed2"),
		Spacer().Flex(2),
		Text("Fixed3"),
	)

	w, _ := g.size(100, 10)
	assert.Equal(t, 100, w) // should fill available width
}

func TestGroup_GapWithFlex(t *testing.T) {
	g := Group(
		Text("A"),
		Spacer(),
		Text("B"),
	).Gap(5)

	w, _ := g.size(50, 10)
	// Width should include gaps between all children
	assert.Equal(t, 50, w)
}

func TestGroup_WideChildrenWithConstraint(t *testing.T) {
	g := Group(
		Text("AAAAAAAAAA"), // 10 chars
		Text("BBBBBBBBBB"), // 10 chars
		Text("CCCCCCCCCC"), // 10 chars
	)

	// Total would be 30, but constraint is 20
	w, _ := g.size(20, 10)
	// Size should report actual total, not constrained
	assert.Equal(t, 30, w)
}

func TestGroup_RenderScalesChildrenProportionally(t *testing.T) {
	// Two fixed-width children that together exceed available space
	g := Group(
		Text("AAAAAAAAAA"),           // 10 chars
		Text("BBBBBBBBBBBBBBBBBBBB"), // 20 chars
	)

	// Render into a 15-wide frame (children want 30)
	screen := SprintScreen(g, PrintConfig{Width: 15, Height: 1})
	output := screen.Text()

	// Both children should be visible and scaled proportionally
	// Child 1 (10 chars) should get 10*15/30 = 5 chars
	// Child 2 (20 chars) should get 15-5 = 10 chars
	assert.True(t, strings.Contains(output, "AAAAA"), "First child should be scaled to ~5 chars")
	assert.True(t, strings.Contains(output, "BBBBBBBBBB"), "Second child should get remaining space")
}

func TestGroup_NoMaxConstraints(t *testing.T) {
	g := Group(
		Text("Test"),
		Text("Another"),
	)

	// No max constraints (0, 0)
	w, h := g.size(0, 0)
	assert.Equal(t, 11, w) // "Test" + "Another" = 4 + 7
	assert.Equal(t, 1, h)
}

// Modifier tests

func TestGroup_PaddingModifier(t *testing.T) {
	g := Group(Text("A"), Text("B"))
	padded := g.Padding(2)

	w, h := padded.size(100, 100)
	assert.Equal(t, 2+4, w) // 2 chars + 2 padding each side
	assert.Equal(t, 1+4, h) // 1 line + 2 padding each side
}

func TestGroup_PaddingHVModifier(t *testing.T) {
	g := Group(Text("X"))
	padded := g.PaddingHV(3, 1)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+6, w) // 1 char + 3 padding each side
	assert.Equal(t, 1+2, h) // 1 line + 1 padding each side
}

func TestGroup_PaddingLTRBModifier(t *testing.T) {
	g := Group(Text("Y"))
	padded := g.PaddingLTRB(1, 2, 3, 4)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+1+3, w) // 1 char + left 1 + right 3
	assert.Equal(t, 1+2+4, h) // 1 line + top 2 + bottom 4
}

func TestGroup_BorderedModifier(t *testing.T) {
	g := Group(Text("Border"))
	bordered := g.Bordered()

	assert.NotNil(t, bordered)
}

// Render context tests

func TestGroup_RenderZeroSize(t *testing.T) {
	var buf strings.Builder
	g := Group(Text("A"))

	// Zero width should not render
	err := Print(g, PrintConfig{Width: 0, Height: 10, Output: &buf})
	assert.NoError(t, err)
}

func TestGroup_RenderZeroHeight(t *testing.T) {
	var buf strings.Builder
	g := Group(Text("A"))

	// Zero height should not render
	err := Print(g, PrintConfig{Width: 10, Height: 0, Output: &buf})
	assert.NoError(t, err)
}

// Render tests using termtest with SprintScreen helper

func TestGroup_Render_Basic(t *testing.T) {
	g := Group(
		Text("Left"),
		Text("Middle"),
		Text("Right"),
	)
	screen := SprintScreen(g, PrintConfig{Width: 30})

	// All text should appear on the same row
	termtest.AssertRowContains(t, screen, 0, "Left")
	termtest.AssertRowContains(t, screen, 0, "Middle")
	termtest.AssertRowContains(t, screen, 0, "Right")
}

func TestGroup_Render_WithGap(t *testing.T) {
	g := Group(
		Text("A"),
		Text("B"),
	).Gap(3)
	screen := SprintScreen(g, PrintConfig{Width: 20})

	row := screen.Row(0)
	// "A" followed by 3 spaces, then "B"
	termtest.AssertRowContains(t, screen, 0, "A")
	termtest.AssertRowContains(t, screen, 0, "B")
	// Verify gap exists (at least 3 chars between A and B)
	assert.True(t, len(row) >= 5, "should have gap between items")
}

func TestGroup_Render_WithSpacer(t *testing.T) {
	g := Group(
		Text("Left"),
		Spacer(),
		Text("Right"),
	)
	screen := SprintScreen(g, PrintConfig{Width: 30})

	row := screen.Row(0)
	// Left and Right should be at opposite ends
	termtest.AssertRowContains(t, screen, 0, "Left")
	termtest.AssertRowContains(t, screen, 0, "Right")
	// Full width should be used
	assert.True(t, len(strings.TrimRight(row, " ")) >= 9, "should span width")
}

func TestGroup_Render_VaryingHeights(t *testing.T) {
	g := Group(
		Text("Short"),
		Stack(Text("Tall1"), Text("Tall2")),
	)
	screen := SprintScreen(g, PrintConfig{Width: 30})

	// Both columns should be visible
	termtest.AssertRowContains(t, screen, 0, "Short")
	termtest.AssertRowContains(t, screen, 0, "Tall1")
	termtest.AssertRowContains(t, screen, 1, "Tall2")
}

func TestGroup_Render_LeftAlign(t *testing.T) {
	// AlignLeft for Group means top-align children vertically
	g := Group(
		Text("A"),
		Stack(Text("1"), Text("2"), Text("3")),
	).Align(AlignLeft)
	screen := SprintScreen(g, PrintConfig{Width: 20})

	// "A" should be at the top (row 0)
	termtest.AssertRowContains(t, screen, 0, "A")
	termtest.AssertRowContains(t, screen, 0, "1")
}

func TestGroup_Render_CenterAlign(t *testing.T) {
	// AlignCenter for Group means center children vertically
	g := Group(
		Text("X"),
		Stack(Text("A"), Text("B"), Text("C")),
	).Align(AlignCenter)
	screen := SprintScreen(g, PrintConfig{Width: 20})

	// "X" should be vertically centered (row 1 for 3-row tall content)
	termtest.AssertRowContains(t, screen, 1, "X")
}

func TestGroup_Render_RightAlign(t *testing.T) {
	// AlignRight for Group means bottom-align children vertically
	g := Group(
		Text("Z"),
		Stack(Text("A"), Text("B"), Text("C")),
	).Align(AlignRight)
	screen := SprintScreen(g, PrintConfig{Width: 20})

	// "Z" should be at the bottom (row 2 for 3-row tall content)
	termtest.AssertRowContains(t, screen, 2, "Z")
}

func TestGroup_Render_WithStyles(t *testing.T) {
	g := Group(
		Text("Bold").Bold(),
		Text("Normal"),
	)
	screen := SprintScreen(g, PrintConfig{Width: 20})

	// Check content is present
	termtest.AssertRowContains(t, screen, 0, "Bold")
	termtest.AssertRowContains(t, screen, 0, "Normal")

	// Check that "Bold" text has bold style
	cell := screen.Cell(0, 0)
	assert.True(t, cell.Style.Bold)
}

func TestGroup_Render_Nested(t *testing.T) {
	g := Group(
		Text("["),
		Group(Text("A"), Text("B")),
		Text("]"),
	)
	screen := SprintScreen(g, PrintConfig{Width: 20})

	row := screen.Row(0)
	assert.True(t, strings.Contains(row, "["))
	assert.True(t, strings.Contains(row, "A"))
	assert.True(t, strings.Contains(row, "B"))
	assert.True(t, strings.Contains(row, "]"))
}
