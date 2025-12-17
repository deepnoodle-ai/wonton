package termtest

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestDiff(t *testing.T) {
	expected := "line1\nline2\nline3"
	actual := "line1\nmodified\nline3"

	diff := Diff(expected, actual)

	assert.Contains(t, diff, "--- Expected")
	assert.Contains(t, diff, "+++ Actual")
	assert.Contains(t, diff, "line2")
	assert.Contains(t, diff, "modified")
}

func TestDiffIdentical(t *testing.T) {
	text := "line1\nline2"
	diff := Diff(text, text)

	// Should be empty for identical content
	assert.Equal(t, "", diff)
}

func TestDiffAddedLines(t *testing.T) {
	expected := "line1\nline2"
	actual := "line1\nline2\nline3"

	diff := Diff(expected, actual)

	assert.Contains(t, diff, "line3")
}

func TestDiffRemovedLines(t *testing.T) {
	expected := "line1\nline2\nline3"
	actual := "line1\nline2"

	diff := Diff(expected, actual)

	assert.Contains(t, diff, "line3")
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
		{"Test*Star", "Test_Star"},
		{"Test?Question", "Test_Question"},
		{"Test\"Quote", "Test_Quote"},
		{"Test<Less", "Test_Less"},
		{"Test>Greater", "Test_Greater"},
		{"Test|Pipe", "Test_Pipe"},
	}

	for _, tt := range tests {
		result := sanitizeName(tt.input)
		assert.Equal(t, tt.expected, result, "sanitizeName(%q)", tt.input)
	}
}

func TestEqualFunc(t *testing.T) {
	s1 := NewScreen(20, 5)
	s1.WriteString("Hello")

	s2 := NewScreen(20, 5)
	s2.WriteString("Hello")

	s3 := NewScreen(20, 5)
	s3.WriteString("World")

	assert.True(t, Equal(s1, s2))
	assert.False(t, Equal(s1, s3))
}

func TestEqualStyledFunc(t *testing.T) {
	s1 := NewScreen(20, 5)
	s1.Write([]byte("\x1b[1mHello"))

	s2 := NewScreen(20, 5)
	s2.Write([]byte("\x1b[1mHello"))

	s3 := NewScreen(20, 5)
	s3.WriteString("Hello")

	// Same text with same style
	assert.True(t, EqualStyled(s1, s2))

	// Same text with different style
	assert.False(t, EqualStyled(s1, s3))
}

func TestEqualStyledDifferentSizes(t *testing.T) {
	s1 := NewScreen(20, 5)
	s2 := NewScreen(30, 10)

	// Different sizes should not be equal
	assert.False(t, EqualStyled(s1, s2))
}

func TestDiffMultipleChanges(t *testing.T) {
	expected := strings.Join([]string{
		"line1",
		"line2",
		"line3",
		"line4",
		"line5",
		"line6",
		"line7",
		"line8",
	}, "\n")

	actual := strings.Join([]string{
		"line1",
		"modified2",
		"line3",
		"line4",
		"line5",
		"modified6",
		"line7",
		"line8",
	}, "\n")

	diff := Diff(expected, actual)

	// Should contain both changes
	assert.Contains(t, diff, "line2")
	assert.Contains(t, diff, "modified2")
	assert.Contains(t, diff, "line6")
	assert.Contains(t, diff, "modified6")
}

// Additional snapshot tests

func TestDiffEmptyStrings(t *testing.T) {
	diff := Diff("", "")
	assert.Equal(t, "", diff)
}

func TestDiffOneEmpty(t *testing.T) {
	diff := Diff("content", "")
	assert.Contains(t, diff, "content")

	diff = Diff("", "content")
	assert.Contains(t, diff, "content")
}

func TestDiffSingleLineChange(t *testing.T) {
	diff := Diff("old", "new")
	assert.Contains(t, diff, "old")
	assert.Contains(t, diff, "new")
}

func TestDiffOnlyAdditions(t *testing.T) {
	expected := "line1\nline2"
	actual := "line1\nline2\nline3\nline4"

	diff := Diff(expected, actual)
	assert.Contains(t, diff, "line3")
	assert.Contains(t, diff, "line4")
}

func TestDiffOnlyDeletions(t *testing.T) {
	expected := "line1\nline2\nline3\nline4"
	actual := "line1\nline2"

	diff := Diff(expected, actual)
	assert.Contains(t, diff, "line3")
	assert.Contains(t, diff, "line4")
}

func TestDiffLongLines(t *testing.T) {
	longLine := strings.Repeat("x", 1000)
	expected := longLine
	actual := longLine + "y"

	diff := Diff(expected, actual)
	assert.NotEqual(t, "", diff)
}

func TestDiffSpecialCharacters(t *testing.T) {
	expected := "line with\ttab\nline with 日本語"
	actual := "line with\ttab\nline with 中文"

	diff := Diff(expected, actual)
	assert.Contains(t, diff, "日本語")
	assert.Contains(t, diff, "中文")
}

func TestSanitizeNameComplex(t *testing.T) {
	// Test multiple special chars in sequence
	result := sanitizeName("Test/With:Many*Special?Chars")
	assert.Equal(t, "Test_With_Many_Special_Chars", result)

	// Test already sanitized name
	result = sanitizeName("AlreadyClean")
	assert.Equal(t, "AlreadyClean", result)
}

func TestEqualEmptyScreens(t *testing.T) {
	s1 := NewScreen(10, 5)
	s2 := NewScreen(10, 5)

	assert.True(t, Equal(s1, s2))
	assert.True(t, EqualStyled(s1, s2))
}

func TestEqualDifferentDimensions(t *testing.T) {
	s1 := NewScreen(10, 5)
	s1.WriteString("Test")

	s2 := NewScreen(20, 10)
	s2.WriteString("Test")

	// Equal compares Text() which includes newlines for each row
	// Different height means different Text() output
	assert.False(t, Equal(s1, s2))

	// EqualStyled also compares dimensions (different)
	assert.False(t, EqualStyled(s1, s2))
}

func TestEqualWithStyles(t *testing.T) {
	s1 := NewScreen(20, 5)
	s1.Write([]byte("\x1b[1;31mRed Bold\x1b[0m"))

	s2 := NewScreen(20, 5)
	s2.Write([]byte("\x1b[1;31mRed Bold\x1b[0m"))

	s3 := NewScreen(20, 5)
	s3.Write([]byte("\x1b[32mGreen\x1b[0m"))
	s3.WriteString(" Bold") // Different content

	// Same content and style
	assert.True(t, Equal(s1, s2))
	assert.True(t, EqualStyled(s1, s2))

	// Different content
	assert.False(t, Equal(s1, s3))
	assert.False(t, EqualStyled(s1, s3))
}
