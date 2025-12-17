package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Empty view tests

func TestEmpty(t *testing.T) {
	e := Empty()
	assert.NotNil(t, e)
}

func TestEmpty_Size(t *testing.T) {
	e := Empty()
	w, h := e.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestEmpty_SizeNoConstraints(t *testing.T) {
	e := Empty()
	w, h := e.size(0, 0)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestEmpty_Render(t *testing.T) {
	var buf strings.Builder
	err := Print(Empty(), WithWidth(80), WithOutput(&buf))
	assert.NoError(t, err)
}

func TestEmpty_InStack(t *testing.T) {
	s := Stack(
		Text("Hello"),
		Empty(),
		Text("World"),
	)

	w, h := s.size(100, 100)
	assert.Equal(t, 5, w) // "Hello" or "World"
	assert.Equal(t, 2, h) // Empty contributes 0 height
}

// Spacer view tests

func TestSpacer(t *testing.T) {
	s := Spacer()
	assert.NotNil(t, s)
	assert.Equal(t, 1, s.flexFactor)
}

func TestSpacer_Size(t *testing.T) {
	s := Spacer()
	w, h := s.size(100, 100)
	// Default spacer has no minimum size
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestSpacer_Flex(t *testing.T) {
	s := Spacer().Flex(3)
	assert.Equal(t, 3, s.flexFactor)
	assert.Equal(t, 3, s.flex())
}

func TestSpacer_MinWidth(t *testing.T) {
	s := Spacer().MinWidth(10)
	assert.Equal(t, 10, s.minWidth)
	w, _ := s.size(100, 100)
	assert.Equal(t, 10, w)
}

func TestSpacer_MinHeight(t *testing.T) {
	s := Spacer().MinHeight(5)
	assert.Equal(t, 5, s.minHeight)
	_, h := s.size(100, 100)
	assert.Equal(t, 5, h)
}

func TestSpacer_Chaining(t *testing.T) {
	s := Spacer().Flex(2).MinWidth(10).MinHeight(5)
	assert.Equal(t, 2, s.flexFactor)
	assert.Equal(t, 10, s.minWidth)
	assert.Equal(t, 5, s.minHeight)
}

func TestSpacer_Render(t *testing.T) {
	var buf strings.Builder
	err := Print(Spacer(), WithWidth(80), WithHeight(10), WithOutput(&buf))
	assert.NoError(t, err)
}

func TestSpacer_InStack(t *testing.T) {
	s := Stack(
		Text("Top"),
		Spacer(),
		Text("Bottom"),
	)

	// Without height constraint, spacer gets minimum (0)
	w, h := s.size(100, 0)
	assert.Equal(t, 6, w)  // "Bottom" is 6 chars
	assert.True(t, h >= 2) // At least top and bottom text

	// With height constraint, spacer fills space
	_, h = s.size(100, 20)
	assert.Equal(t, 20, h)
}

func TestSpacer_InGroup(t *testing.T) {
	g := Group(
		Text("Left"),
		Spacer(),
		Text("Right"),
	)

	// Without width constraint, spacer gets minimum (0)
	w, h := g.size(0, 100)
	assert.True(t, w >= 9) // "Left" + "Right" = 9
	assert.Equal(t, 1, h)

	// With width constraint, spacer fills space
	w, _ = g.size(50, 100)
	assert.Equal(t, 50, w)
}

func TestSpacer_MultipleFlex(t *testing.T) {
	s := Stack(
		Text("A"),
		Spacer().Flex(1),
		Text("B"),
		Spacer().Flex(2),
		Text("C"),
	)

	// With height constraint, spacers fill proportionally
	_, h := s.size(100, 30)
	assert.Equal(t, 30, h)
}

func TestSpacer_WithMinSizeInStack(t *testing.T) {
	s := Stack(
		Text("Top"),
		Spacer().MinHeight(3),
		Text("Bottom"),
	)

	// Spacer should contribute at least its min height
	_, h := s.size(100, 0)
	assert.True(t, h >= 5) // top(1) + min(3) + bottom(1)
}

func TestSpacer_FlexInGroup(t *testing.T) {
	g := Group(
		Text("A"),
		Spacer().Flex(1),
		Spacer().Flex(3),
		Text("B"),
	)

	// With width constraint
	w, _ := g.size(40, 10)
	assert.Equal(t, 40, w)
}

// Group (HStack) tests

func TestGroup_Empty(t *testing.T) {
	g := Group()
	w, h := g.size(100, 100)
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestGroup_SingleChild(t *testing.T) {
	g := Group(Text("Hello"))
	w, h := g.size(100, 100)
	assert.Equal(t, 5, w)
	assert.Equal(t, 1, h)
}

func TestGroup_MultipleChildren(t *testing.T) {
	g := Group(
		Text("A"),
		Text("BB"),
		Text("CCC"),
	)

	w, h := g.size(100, 100)
	assert.Equal(t, 6, w) // 1+2+3
	assert.Equal(t, 1, h)
}

func TestGroup_Gap(t *testing.T) {
	g := Group(
		Text("A"),
		Text("B"),
		Text("C"),
	).Gap(2)

	w, h := g.size(100, 100)
	assert.Equal(t, 3+4, w) // 3 chars + 2 gaps of 2
	assert.Equal(t, 1, h)
}

func TestGroup_Alignment(t *testing.T) {
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
				Text("X"),
				Stack(Text("A"), Text("B")), // 2 lines tall
			).Align(tt.align)

			_, h := g.size(100, 100)
			assert.Equal(t, 2, h) // height of tallest child
			assert.Equal(t, tt.align, g.alignment)
		})
	}
}

func TestGroup_Render(t *testing.T) {
	var buf strings.Builder
	g := Group(
		Text("A"),
		Text("B"),
		Text("C"),
	)

	err := Print(g, WithWidth(80), WithHeight(10), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "A"), "should contain A")
	assert.True(t, strings.Contains(output, "B"), "should contain B")
	assert.True(t, strings.Contains(output, "C"), "should contain C")
}

func TestGroup_Chaining(t *testing.T) {
	g := Group(
		Text("A"),
		Text("B"),
	).Gap(3).Align(AlignCenter)

	assert.Equal(t, 3, g.gap)
	assert.Equal(t, AlignCenter, g.alignment)
}

func TestGroup_VaryingHeights(t *testing.T) {
	g := Group(
		Text("A"),
		Stack(Text("Line1"), Text("Line2")),
		Text("B"),
	)

	_, h := g.size(100, 100)
	assert.Equal(t, 2, h) // tallest is the stack with 2 lines
}

// ZStack tests

func TestZStack(t *testing.T) {
	z := ZStack(Text("A"))
	assert.NotNil(t, z)
}

func TestZStack_Size(t *testing.T) {
	z := ZStack(
		Text("Short"),
		Text("Longer text"),
	)

	w, h := z.size(100, 100)
	assert.Equal(t, 11, w) // "Longer text"
	assert.Equal(t, 1, h)
}

func TestZStack_Alignment(t *testing.T) {
	z := ZStack(Text("A")).Align(AlignCenter)
	assert.Equal(t, AlignCenter, z.alignment)
}

func TestZStack_Render(t *testing.T) {
	var buf strings.Builder
	z := ZStack(
		Text("Background"),
		Text("Top"),
	)

	err := Print(z, WithWidth(80), WithHeight(10), WithOutput(&buf))
	assert.NoError(t, err)
}
