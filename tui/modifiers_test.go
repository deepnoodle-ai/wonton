package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestPadding_Equal(t *testing.T) {
	inner := Text("Hi")
	padded := Padding(2, inner)

	w, h := padded.size(100, 100)
	assert.Equal(t, 2+4, w) // 2 chars + 2 padding each side
	assert.Equal(t, 1+4, h) // 1 line + 2 padding each side
}

func TestPadding_Zero(t *testing.T) {
	inner := Text("Test")
	padded := Padding(0, inner)

	w, h := padded.size(100, 100)
	assert.Equal(t, 4, w)
	assert.Equal(t, 1, h)
}

func TestPaddingHV(t *testing.T) {
	inner := Text("X")
	padded := PaddingHV(3, 1, inner)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+6, w) // 1 char + 3 padding each side
	assert.Equal(t, 1+2, h) // 1 line + 1 padding each side
}

func TestPaddingHV_Asymmetric(t *testing.T) {
	inner := Text("AB")
	padded := PaddingHV(5, 2, inner)

	w, h := padded.size(100, 100)
	assert.Equal(t, 2+10, w) // 2 chars + 5 padding each side
	assert.Equal(t, 1+4, h)  // 1 line + 2 padding each side
}

func TestPaddingLTRB(t *testing.T) {
	inner := Text("A")
	padded := PaddingLTRB(1, 2, 3, 4, inner)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+1+3, w) // 1 char + left 1 + right 3
	assert.Equal(t, 1+2+4, h) // 1 line + top 2 + bottom 4
}

func TestPaddingLTRB_AllZero(t *testing.T) {
	inner := Text("Test")
	padded := PaddingLTRB(0, 0, 0, 0, inner)

	w, h := padded.size(100, 100)
	assert.Equal(t, 4, w)
	assert.Equal(t, 1, h)
}

func TestPadding_WithMaxConstraints(t *testing.T) {
	inner := Text("Hello World")
	padded := Padding(2, inner)

	// With narrow max width, padding is subtracted from available space
	w, h := padded.size(10, 100)
	assert.True(t, w <= 10, "width should respect max constraint")
	assert.True(t, h >= 1, "height should be at least 1")
}

func TestPadding_Render(t *testing.T) {
	var buf strings.Builder
	padded := Padding(1, Text("Hi"))

	err := Print(padded, WithWidth(20), WithHeight(5), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Hi"), "should contain text")
}

func TestPaddingHV_Render(t *testing.T) {
	var buf strings.Builder
	padded := PaddingHV(2, 1, Text("Test"))

	err := Print(padded, WithWidth(20), WithHeight(10), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Test"), "should contain text")
}

// Size modifiers tests

func TestWidth_Modifier(t *testing.T) {
	inner := Text("Hi")
	sized := Width(10, inner)

	w, h := sized.size(100, 100)
	assert.Equal(t, 10, w)
	assert.Equal(t, 1, h)
}

func TestHeight_Modifier(t *testing.T) {
	inner := Text("Hi")
	sized := Height(5, inner)

	w, h := sized.size(100, 100)
	assert.Equal(t, 2, w)
	assert.Equal(t, 5, h)
}

func TestSize_Modifier(t *testing.T) {
	inner := Text("A")
	sized := Size(15, 8, inner)

	w, h := sized.size(100, 100)
	assert.Equal(t, 15, w)
	assert.Equal(t, 8, h)
}

func TestMaxWidth_Modifier(t *testing.T) {
	inner := Text("A very long text that exceeds max width")
	sized := MaxWidth(10, inner)

	w, h := sized.size(100, 100)
	assert.True(t, w <= 10, "width should be at most 10")
	assert.True(t, h >= 1, "should have at least one line")
}

func TestMaxHeight_Modifier(t *testing.T) {
	// MaxHeight constrains the passed maxHeight to children
	// but Stack with fixed children doesn't shrink
	inner := Stack(
		Text("Line 1"),
		Text("Line 2"),
		Text("Line 3"),
	)
	sized := MaxHeight(5, inner)

	_, h := sized.size(100, 100)
	// With 3 text lines and maxHeight=5, the actual height is 3
	assert.Equal(t, 3, h)
}

func TestMaxHeight_WithFlexChild(t *testing.T) {
	// With a flexible child, MaxHeight affects expansion
	inner := Stack(
		Text("Top"),
		Spacer(),
		Text("Bottom"),
	)
	sized := MaxHeight(5, inner)

	_, h := sized.size(100, 10)
	// Spacer expands to fill, but MaxHeight limits to 5
	assert.Equal(t, 5, h)
}

func TestSize_WithConstraints(t *testing.T) {
	inner := Text("Test")
	sized := Size(50, 50, inner)

	// Max constraints should limit the size
	w, h := sized.size(20, 20)
	assert.Equal(t, 20, w)
	assert.Equal(t, 20, h)
}

func TestSize_Render(t *testing.T) {
	var buf strings.Builder
	sized := Size(10, 3, Text("Hi"))

	err := Print(sized, WithWidth(20), WithHeight(10), WithOutput(&buf))
	assert.NoError(t, err)
}

// Bordered view tests

func TestBordered_Basic(t *testing.T) {
	inner := Text("Content")
	bordered := Bordered(inner)

	assert.NotNil(t, bordered)
}

func TestBordered_WithBorder(t *testing.T) {
	inner := Text("Box")
	bordered := Bordered(inner).Border(&SingleBorder)

	w, h := bordered.size(100, 100)
	assert.Equal(t, 3+2, w) // 3 chars + 1 border each side
	assert.Equal(t, 1+2, h) // 1 line + 1 border each side
}

func TestBordered_WithTitle(t *testing.T) {
	inner := Text("Content")
	bordered := Bordered(inner).Border(&SingleBorder).Title("Title")

	assert.Equal(t, "Title", bordered.title)
}

func TestBordered_TitleStyle(t *testing.T) {
	inner := Text("Content")
	style := NewStyle().WithBold()
	bordered := Bordered(inner).Border(&SingleBorder).TitleStyle(style)

	assert.True(t, bordered.titleStyle.Bold)
}

func TestBordered_BorderFg(t *testing.T) {
	inner := Text("Content")
	bordered := Bordered(inner).Border(&SingleBorder).BorderFg(ColorRed)

	assert.Equal(t, ColorRed, bordered.borderStyle.Foreground)
}

func TestBordered_FocusID(t *testing.T) {
	inner := Text("Input")
	bordered := Bordered(inner).FocusID("my-input")

	assert.Equal(t, "my-input", bordered.focusID)
}

func TestBordered_FocusBorderFg(t *testing.T) {
	inner := Text("Input")
	bordered := Bordered(inner).FocusBorderFg(ColorBlue)

	assert.True(t, bordered.hasFocusBorder)
}

func TestBordered_FocusTitleStyle(t *testing.T) {
	inner := Text("Input")
	style := NewStyle().WithItalic()
	bordered := Bordered(inner).FocusTitleStyle(style)

	assert.NotNil(t, bordered.focusTitleStyle)
}

func TestBordered_Render(t *testing.T) {
	var buf strings.Builder
	bordered := Bordered(Text("Box")).Border(&SingleBorder)

	err := Print(bordered, WithWidth(20), WithHeight(10), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Box"), "should contain content")
	assert.True(t, strings.Contains(output, "─"), "should contain border")
}

func TestBordered_RenderWithTitle(t *testing.T) {
	var buf strings.Builder
	bordered := Bordered(Text("Content")).Border(&SingleBorder).Title("My Title")

	err := Print(bordered, WithWidth(30), WithHeight(10), WithOutput(&buf))
	assert.NoError(t, err)

	output := buf.String()
	assert.True(t, strings.Contains(output, "Content"), "should contain content")
	assert.True(t, strings.Contains(output, "My Title"), "should contain title")
}

func TestBordered_NoBorder(t *testing.T) {
	inner := Text("Plain")
	bordered := Bordered(inner)

	// Without border, size should match inner
	w, h := bordered.size(100, 100)
	assert.Equal(t, 5, w)
	assert.Equal(t, 1, h)
}

func TestBordered_Chaining(t *testing.T) {
	bordered := Bordered(Text("X")).
		Border(&DoubleBorder).
		Title("T").
		TitleStyle(NewStyle().WithBold()).
		BorderFg(ColorGreen).
		FocusID("input").
		FocusBorderFg(ColorYellow)

	assert.Equal(t, &DoubleBorder, bordered.border)
	assert.Equal(t, "T", bordered.title)
	assert.Equal(t, "input", bordered.focusID)
	assert.True(t, bordered.hasFocusBorder)
}

// Background tests

func TestBackground(t *testing.T) {
	inner := Text("Content")
	bg := Background(' ', NewStyle().WithBackground(ColorBlue), inner)

	assert.NotNil(t, bg)
}

func TestStack_Bg(t *testing.T) {
	s := Stack(Text("A"), Text("B"))
	bg := s.Bg(ColorRed)

	assert.NotNil(t, bg)
}

func TestGroup_Bg(t *testing.T) {
	g := Group(Text("A"), Text("B"))
	bg := g.Bg(ColorGreen)

	assert.NotNil(t, bg)
}

// Group padding modifiers

func TestGroup_Padding(t *testing.T) {
	g := Group(Text("A"), Text("B"))
	padded := g.Padding(2)

	w, h := padded.size(100, 100)
	assert.True(t, w > 2, "should have width for content plus padding")
	assert.Equal(t, 1+4, h) // 1 line + 2 padding each side
}

func TestGroup_PaddingHV(t *testing.T) {
	g := Group(Text("X"))
	padded := g.PaddingHV(3, 1)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+6, w) // 1 char + 3 padding each side
	assert.Equal(t, 1+2, h) // 1 line + 1 padding each side
}

func TestGroup_PaddingLTRB(t *testing.T) {
	g := Group(Text("A"))
	padded := g.PaddingLTRB(1, 2, 3, 4)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+1+3, w) // 1 char + left 1 + right 3
	assert.Equal(t, 1+2+4, h) // 1 line + top 2 + bottom 4
}

func TestGroup_Bordered(t *testing.T) {
	g := Group(Text("A"), Text("B"))
	bordered := g.Bordered()

	assert.NotNil(t, bordered)
}

// ZStack padding modifier

func TestZStack_Padding(t *testing.T) {
	z := ZStack(Text("A"))
	padded := z.Padding(2)

	w, h := padded.size(100, 100)
	assert.Equal(t, 1+4, w) // 1 char + 2 padding each side
	assert.Equal(t, 1+4, h) // 1 line + 2 padding each side
}

func TestZStack_Bordered(t *testing.T) {
	z := ZStack(Text("A"))
	bordered := z.Bordered()

	assert.NotNil(t, bordered)
}

// Text modifier tests

func TestTextView_Width(t *testing.T) {
	tv := Text("Hello")
	sized := tv.Width(20)

	w, h := sized.size(100, 100)
	assert.Equal(t, 20, w)
	assert.Equal(t, 1, h)
}

func TestTextView_Height(t *testing.T) {
	tv := Text("Hello")
	sized := tv.Height(3)

	w, h := sized.size(100, 100)
	assert.Equal(t, 5, w)
	assert.Equal(t, 3, h)
}

func TestTextView_MaxWidth(t *testing.T) {
	tv := Text("A very long text string")
	sized := tv.MaxWidth(10)

	w, _ := sized.size(100, 100)
	assert.True(t, w <= 10, "width should be at most 10")
}
