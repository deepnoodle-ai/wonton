package termtest

import (
	"strings"
	"testing"
)

// AssertContains asserts that the screen contains the given text anywhere in its content.
// The search is case-sensitive and can match across line boundaries.
//
// Example:
//
//	screen.Write([]byte("Hello, World!"))
//	termtest.AssertContains(t, screen, "World")  // passes
func AssertContains(t *testing.T, screen *Screen, text string) {
	t.Helper()
	if !screen.Contains(text) {
		t.Errorf("screen does not contain %q\n\nScreen content:\n%s", text, screen.Text())
	}
}

// AssertNotContains asserts that the screen does not contain the given text.
func AssertNotContains(t *testing.T, screen *Screen, text string) {
	t.Helper()
	if screen.Contains(text) {
		t.Errorf("screen unexpectedly contains %q\n\nScreen content:\n%s", text, screen.Text())
	}
}

// AssertRow asserts that a specific row exactly matches the expected text.
// Row indices are 0-based. Trailing spaces in the row are trimmed before comparison.
//
// Example:
//
//	screen.Write([]byte("Line1\nLine2"))
//	termtest.AssertRow(t, screen, 0, "Line1")  // passes
//	termtest.AssertRow(t, screen, 1, "Line2")  // passes
func AssertRow(t *testing.T, screen *Screen, row int, expected string) {
	t.Helper()
	actual := screen.Row(row)
	if actual != expected {
		t.Errorf("row %d mismatch:\nexpected: %q\nactual:   %q", row, expected, actual)
	}
}

// AssertRowContains asserts that a specific row contains the given text substring.
// Unlike AssertRow, this does a substring match instead of exact equality.
func AssertRowContains(t *testing.T, screen *Screen, row int, text string) {
	t.Helper()
	actual := screen.Row(row)
	if !strings.Contains(actual, text) {
		t.Errorf("row %d does not contain %q\nactual: %q", row, text, actual)
	}
}

// AssertRowPrefix asserts that a specific row starts with the given prefix.
// Useful for checking prompts or indentation in terminal output.
func AssertRowPrefix(t *testing.T, screen *Screen, row int, prefix string) {
	t.Helper()
	actual := screen.Row(row)
	if !strings.HasPrefix(actual, prefix) {
		t.Errorf("row %d does not start with %q\nactual: %q", row, prefix, actual)
	}
}

// AssertCursor asserts that the cursor is at the expected (x, y) position.
// Coordinates are 0-based, with (0, 0) at the top-left corner.
//
// Example:
//
//	screen.Write([]byte("Hello"))
//	termtest.AssertCursor(t, screen, 5, 0)  // After "Hello"
func AssertCursor(t *testing.T, screen *Screen, x, y int) {
	t.Helper()
	actualX, actualY := screen.Cursor()
	if actualX != x || actualY != y {
		t.Errorf("cursor position mismatch:\nexpected: (%d, %d)\nactual:   (%d, %d)", x, y, actualX, actualY)
	}
}

// AssertTextEqual asserts that the screen text matches exactly.
// The comparison includes newlines and considers all rows.
// If there's a mismatch, a unified diff is displayed.
func AssertTextEqual(t *testing.T, screen *Screen, expected string) {
	t.Helper()
	actual := screen.Text()
	if actual != expected {
		diff := Diff(expected, actual)
		t.Errorf("screen text mismatch:\n%s", diff)
	}
}

// AssertCell asserts that the cell at (x, y) contains the expected character.
// Coordinates are 0-based.
func AssertCell(t *testing.T, screen *Screen, x, y int, expected rune) {
	t.Helper()
	cell := screen.Cell(x, y)
	if cell.Char != expected {
		t.Errorf("cell (%d, %d) mismatch:\nexpected: %q\nactual:   %q", x, y, expected, cell.Char)
	}
}

// AssertCellStyle asserts that the cell at (x, y) has exactly the expected style.
// All style fields must match (foreground, background, bold, italic, etc.).
func AssertCellStyle(t *testing.T, screen *Screen, x, y int, style Style) {
	t.Helper()
	cell := screen.Cell(x, y)
	if cell.Style != style {
		t.Errorf("cell (%d, %d) style mismatch:\nexpected: %+v\nactual:   %+v", x, y, style, cell.Style)
	}
}

// AssertCellBold asserts that the cell at (x, y) has the expected bold state.
// This is a convenience function for checking just the bold attribute.
func AssertCellBold(t *testing.T, screen *Screen, x, y int, bold bool) {
	t.Helper()
	cell := screen.Cell(x, y)
	if cell.Style.Bold != bold {
		t.Errorf("cell (%d, %d) bold mismatch: expected %v, got %v", x, y, bold, cell.Style.Bold)
	}
}

// AssertEmpty asserts that the screen contains only whitespace.
// This checks that all cells are spaces or the screen is blank.
func AssertEmpty(t *testing.T, screen *Screen) {
	t.Helper()
	text := screen.Text()
	trimmed := strings.TrimSpace(strings.ReplaceAll(text, "\n", ""))
	if trimmed != "" {
		t.Errorf("screen is not empty:\n%s", text)
	}
}

// AssertEqual asserts that two screens have identical text content.
// Styles are not compared, only the visible text. For style comparison, use EqualStyled.
func AssertEqual(t *testing.T, expected, actual *Screen) {
	t.Helper()
	if !Equal(expected, actual) {
		diff := Diff(expected.Text(), actual.Text())
		t.Errorf("screens do not match:\n%s", diff)
	}
}

// RequireContains is like AssertContains but calls t.Fatal instead of t.Error.
// Use this when the test cannot continue if the assertion fails.
func RequireContains(t *testing.T, screen *Screen, text string) {
	t.Helper()
	if !screen.Contains(text) {
		t.Fatalf("screen does not contain %q\n\nScreen content:\n%s", text, screen.Text())
	}
}

// RequireRow is like AssertRow but calls t.Fatal instead of t.Error.
// Use this when the test cannot continue if the assertion fails.
func RequireRow(t *testing.T, screen *Screen, row int, expected string) {
	t.Helper()
	actual := screen.Row(row)
	if actual != expected {
		t.Fatalf("row %d mismatch:\nexpected: %q\nactual:   %q", row, expected, actual)
	}
}
