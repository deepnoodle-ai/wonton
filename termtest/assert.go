package termtest

import (
	"strings"
	"testing"
)

// AssertContains asserts that the screen contains the given text.
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

// AssertRow asserts that a specific row contains the expected text.
func AssertRow(t *testing.T, screen *Screen, row int, expected string) {
	t.Helper()
	actual := screen.Row(row)
	if actual != expected {
		t.Errorf("row %d mismatch:\nexpected: %q\nactual:   %q", row, expected, actual)
	}
}

// AssertRowContains asserts that a specific row contains the given text.
func AssertRowContains(t *testing.T, screen *Screen, row int, text string) {
	t.Helper()
	actual := screen.Row(row)
	if !strings.Contains(actual, text) {
		t.Errorf("row %d does not contain %q\nactual: %q", row, text, actual)
	}
}

// AssertRowPrefix asserts that a specific row starts with the given text.
func AssertRowPrefix(t *testing.T, screen *Screen, row int, prefix string) {
	t.Helper()
	actual := screen.Row(row)
	if !strings.HasPrefix(actual, prefix) {
		t.Errorf("row %d does not start with %q\nactual: %q", row, prefix, actual)
	}
}

// AssertCursor asserts that the cursor is at the expected position.
func AssertCursor(t *testing.T, screen *Screen, x, y int) {
	t.Helper()
	actualX, actualY := screen.Cursor()
	if actualX != x || actualY != y {
		t.Errorf("cursor position mismatch:\nexpected: (%d, %d)\nactual:   (%d, %d)", x, y, actualX, actualY)
	}
}

// AssertText asserts that the screen text matches exactly.
func AssertTextEqual(t *testing.T, screen *Screen, expected string) {
	t.Helper()
	actual := screen.Text()
	if actual != expected {
		diff := Diff(expected, actual)
		t.Errorf("screen text mismatch:\n%s", diff)
	}
}

// AssertCell asserts that a specific cell has the expected character.
func AssertCell(t *testing.T, screen *Screen, x, y int, expected rune) {
	t.Helper()
	cell := screen.Cell(x, y)
	if cell.Char != expected {
		t.Errorf("cell (%d, %d) mismatch:\nexpected: %q\nactual:   %q", x, y, expected, cell.Char)
	}
}

// AssertCellStyle asserts that a specific cell has the expected style properties.
func AssertCellStyle(t *testing.T, screen *Screen, x, y int, style Style) {
	t.Helper()
	cell := screen.Cell(x, y)
	if cell.Style != style {
		t.Errorf("cell (%d, %d) style mismatch:\nexpected: %+v\nactual:   %+v", x, y, style, cell.Style)
	}
}

// AssertCellBold asserts that a specific cell is bold (or not).
func AssertCellBold(t *testing.T, screen *Screen, x, y int, bold bool) {
	t.Helper()
	cell := screen.Cell(x, y)
	if cell.Style.Bold != bold {
		t.Errorf("cell (%d, %d) bold mismatch: expected %v, got %v", x, y, bold, cell.Style.Bold)
	}
}

// AssertEmpty asserts that the screen is empty (all spaces).
func AssertEmpty(t *testing.T, screen *Screen) {
	t.Helper()
	text := screen.Text()
	trimmed := strings.TrimSpace(strings.ReplaceAll(text, "\n", ""))
	if trimmed != "" {
		t.Errorf("screen is not empty:\n%s", text)
	}
}

// AssertEqual asserts that two screens have the same text content.
func AssertEqual(t *testing.T, expected, actual *Screen) {
	t.Helper()
	if !Equal(expected, actual) {
		diff := Diff(expected.Text(), actual.Text())
		t.Errorf("screens do not match:\n%s", diff)
	}
}

// RequireContains is like AssertContains but fails immediately.
func RequireContains(t *testing.T, screen *Screen, text string) {
	t.Helper()
	if !screen.Contains(text) {
		t.Fatalf("screen does not contain %q\n\nScreen content:\n%s", text, screen.Text())
	}
}

// RequireRow is like AssertRow but fails immediately.
func RequireRow(t *testing.T, screen *Screen, row int, expected string) {
	t.Helper()
	actual := screen.Row(row)
	if actual != expected {
		t.Fatalf("row %d mismatch:\nexpected: %q\nactual:   %q", row, expected, actual)
	}
}
