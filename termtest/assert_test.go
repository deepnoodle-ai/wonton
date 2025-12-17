package termtest

import (
	"fmt"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestAssertContains(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello World")

	// This should pass
	AssertContains(t, s, "Hello")
	AssertContains(t, s, "World")
}

func TestAssertNotContains(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello World")

	// This should pass
	AssertNotContains(t, s, "Goodbye")
}

func TestAssertRow(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Line1\nLine2\nLine3")

	AssertRow(t, s, 0, "Line1")
	AssertRow(t, s, 1, "Line2")
	AssertRow(t, s, 2, "Line3")
}

func TestAssertRowContains(t *testing.T) {
	s := NewScreen(30, 5)
	s.WriteString("The quick brown fox")

	AssertRowContains(t, s, 0, "quick")
	AssertRowContains(t, s, 0, "brown")
}

func TestAssertRowPrefix(t *testing.T) {
	s := NewScreen(30, 5)
	s.WriteString("> Command input")

	AssertRowPrefix(t, s, 0, "> ")
	AssertRowPrefix(t, s, 0, "> Command")
}

func TestAssertCursor(t *testing.T) {
	s := NewScreen(20, 10)
	s.SetCursor(5, 7)

	AssertCursor(t, s, 5, 7)
}

func TestAssertCell(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("ABC")

	AssertCell(t, s, 0, 0, 'A')
	AssertCell(t, s, 1, 0, 'B')
	AssertCell(t, s, 2, 0, 'C')
}

func TestAssertCellBold(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1mBold\x1b[0mNormal"))

	AssertCellBold(t, s, 0, 0, true)  // 'B' is bold
	AssertCellBold(t, s, 4, 0, false) // 'N' is not bold
}

func TestAssertEmpty(t *testing.T) {
	s := NewScreen(20, 5)
	AssertEmpty(t, s)
}

func TestAssertEqual(t *testing.T) {
	s1 := NewScreen(20, 5)
	s1.WriteString("Hello")

	s2 := NewScreen(20, 5)
	s2.WriteString("Hello")

	AssertEqual(t, s1, s2)
}

func TestEqual(t *testing.T) {
	s1 := NewScreen(20, 5)
	s1.WriteString("Hello")

	s2 := NewScreen(20, 5)
	s2.WriteString("Hello")

	s3 := NewScreen(20, 5)
	s3.WriteString("World")

	assert.True(t, Equal(s1, s2))
	assert.False(t, Equal(s1, s3))
}

func TestEqualStyled(t *testing.T) {
	s1 := NewScreen(20, 5)
	s1.Write([]byte("\x1b[1mHello"))

	s2 := NewScreen(20, 5)
	s2.Write([]byte("\x1b[1mHello"))

	s3 := NewScreen(20, 5)
	s3.WriteString("Hello") // Same text, no bold

	assert.True(t, EqualStyled(s1, s2))
	assert.False(t, EqualStyled(s1, s3))
}

// Additional assertion tests

func TestAssertTextEqual(t *testing.T) {
	s := NewScreen(10, 3)
	s.WriteString("ABC")

	AssertTextEqual(t, s, "ABC\n\n\n")
}

func TestAssertCellStyle(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1;3;31mX")) // Bold, italic, red

	expected := Style{
		Bold:       true,
		Italic:     true,
		Foreground: Color{Type: ColorBasic, Value: 1},
	}
	AssertCellStyle(t, s, 0, 0, expected)
}

func TestAssertRowEmptyScreen(t *testing.T) {
	s := NewScreen(20, 5)
	AssertRow(t, s, 0, "")
	AssertRow(t, s, 4, "")
}

func TestAssertCursorAtOrigin(t *testing.T) {
	s := NewScreen(20, 10)
	AssertCursor(t, s, 0, 0) // New screen starts at origin
}

func TestAssertCursorAfterWrite(t *testing.T) {
	s := NewScreen(20, 10)
	s.WriteString("Hello")
	AssertCursor(t, s, 5, 0)
}

func TestAssertCursorAfterNewline(t *testing.T) {
	s := NewScreen(20, 10)
	s.WriteString("Hello\nWorld")
	AssertCursor(t, s, 5, 1)
}

func TestAssertCellAtEndOfRow(t *testing.T) {
	s := NewScreen(5, 3)
	s.WriteString("ABCDE")

	AssertCell(t, s, 4, 0, 'E')
}

func TestAssertRowWithANSI(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1mBold\x1b[0m Normal"))

	AssertRow(t, s, 0, "Bold Normal")
}

func TestAssertRowContainsMultiple(t *testing.T) {
	s := NewScreen(50, 5)
	s.WriteString("The quick brown fox jumps over the lazy dog")

	AssertRowContains(t, s, 0, "quick")
	AssertRowContains(t, s, 0, "fox")
	AssertRowContains(t, s, 0, "lazy")
}

func TestAssertRowPrefixVariants(t *testing.T) {
	tests := []struct {
		content string
		prefix  string
	}{
		{"> prompt", "> "},
		{"$ command", "$ "},
		{">>> python", ">>>"},
		{"  indented", "  "},
	}

	for _, tt := range tests {
		s := NewScreen(30, 5)
		s.WriteString(tt.content)
		AssertRowPrefix(t, s, 0, tt.prefix)
	}
}

func TestAssertEmptyAfterClear(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Content")
	s.Clear()

	AssertEmpty(t, s)
}

func TestAssertEmptyNewScreen(t *testing.T) {
	s := NewScreen(20, 5)
	AssertEmpty(t, s)
}

func TestAssertContainsAcrossLines(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Line1\nLine2\nLine3")

	// Contains searches full text including newlines
	AssertContains(t, s, "Line1")
	AssertContains(t, s, "Line2")
	AssertContains(t, s, "Line3")
}

func TestAssertNotContainsAfterClear(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Secret")
	s.Clear()

	AssertNotContains(t, s, "Secret")
}

func TestAssertCellBoldToggle(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1mA\x1b[22mB\x1b[1mC"))

	AssertCellBold(t, s, 0, 0, true)  // A is bold
	AssertCellBold(t, s, 1, 0, false) // B is not bold
	AssertCellBold(t, s, 2, 0, true)  // C is bold
}

func TestAssertEqualIdenticalScreens(t *testing.T) {
	s1 := NewScreen(20, 5)
	s1.WriteString("Same content")

	s2 := NewScreen(20, 5)
	s2.WriteString("Same content")

	AssertEqual(t, s1, s2)
}

func TestRequireContainsSuccess(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello World")

	// Should not panic
	RequireContains(t, s, "Hello")
	RequireContains(t, s, "World")
}

func TestRequireRowSuccess(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Expected")

	// Should not panic
	RequireRow(t, s, 0, "Expected")
}

// Example of using assertion helpers in tests.
func ExampleAssertContains() {
	// This would be in a real test function
	screen := NewScreen(40, 5)
	screen.Write([]byte("Error: file not found"))
	
	// In a test, you would use a real *testing.T
	// AssertContains(t, screen, "Error:")
	
	// For demonstration:
	fmt.Println(screen.Contains("Error:"))
	
	// Output:
	// true
}

// Example of checking a specific row.
func ExampleAssertRow() {
	screen := NewScreen(40, 5)
	screen.Write([]byte("Menu\n> Option 1\n> Option 2"))
	
	// In a test:
	// AssertRow(t, screen, 0, "Menu")
	// AssertRow(t, screen, 1, "> Option 1")
	
	fmt.Println(screen.Row(0))
	fmt.Println(screen.Row(1))
	
	// Output:
	// Menu
	// > Option 1
}

// Example of checking cursor position.
func ExampleAssertCursor() {
	screen := NewScreen(40, 5)
	screen.Write([]byte("Hello"))
	
	// In a test:
	// AssertCursor(t, screen, 5, 0)
	
	x, y := screen.Cursor()
	fmt.Printf("Cursor: (%d, %d)\n", x, y)
	
	// Output:
	// Cursor: (5, 0)
}
