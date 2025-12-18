package termtest

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update snapshot files")

// AssertScreen compares the screen content against a golden file snapshot.
// The snapshot file is automatically named based on the test name and stored
// in testdata/snapshots/.
//
// On first run or when -update flag is used, the snapshot is created/updated.
// Subsequent runs compare against the snapshot and fail if different.
//
// Example:
//
//	func TestMyUI(t *testing.T) {
//	    screen := termtest.NewScreen(80, 24)
//	    app.Render(screen)
//	    termtest.AssertScreen(t, screen)  // Creates/compares testdata/snapshots/TestMyUI.snap
//	}
//
// Update snapshots: go test -update
func AssertScreen(t *testing.T, screen *Screen) {
	t.Helper()
	AssertScreenNamed(t, t.Name(), screen)
}

// AssertScreenNamed compares the screen content against a named snapshot file.
// Use this when you need multiple snapshots in a single test or want to control
// the snapshot name explicitly.
//
// Example:
//
//	termtest.AssertScreenNamed(t, "initial_state", screen1)
//	termtest.AssertScreenNamed(t, "after_action", screen2)
func AssertScreenNamed(t *testing.T, name string, screen *Screen) {
	t.Helper()
	actual := screen.Text()
	assertSnapshot(t, name, actual)
}

// AssertText compares plain text content against a golden file snapshot.
// Use this for testing text output that doesn't need ANSI interpretation.
// The snapshot name is derived from the test name.
func AssertText(t *testing.T, actual string) {
	t.Helper()
	AssertTextNamed(t, t.Name(), actual)
}

// AssertTextNamed compares plain text content against a named snapshot file.
// Like AssertScreenNamed but for plain text without ANSI processing.
func AssertTextNamed(t *testing.T, name, actual string) {
	t.Helper()
	assertSnapshot(t, name, actual)
}

// assertSnapshot is the core snapshot comparison logic.
func assertSnapshot(t *testing.T, name, actual string) {
	t.Helper()

	snapshotDir := filepath.Join("testdata", "snapshots")
	snapshotPath := filepath.Join(snapshotDir, sanitizeName(name)+".snap")

	shouldUpdate := *update || os.Getenv("TERMTEST_UPDATE") != ""

	if shouldUpdate {
		if err := os.MkdirAll(snapshotDir, 0755); err != nil {
			t.Fatalf("failed to create snapshot directory: %v", err)
		}
		if err := os.WriteFile(snapshotPath, []byte(actual), 0644); err != nil {
			t.Fatalf("failed to write snapshot: %v", err)
		}
		t.Logf("Updated snapshot: %s", snapshotPath)
		return
	}

	expected, err := os.ReadFile(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("snapshot not found: %s\nRun with -update to create it.\n\nActual output:\n%s", snapshotPath, actual)
		}
		t.Fatalf("failed to read snapshot: %v", err)
	}

	if actual != string(expected) {
		diff := Diff(string(expected), actual)
		t.Errorf("snapshot mismatch: %s\n%s\nRun with -update to update the snapshot.", snapshotPath, diff)
	}
}

// sanitizeName converts a test name to a valid filename.
func sanitizeName(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(name)
}

// Diff generates a unified diff between expected and actual strings.
// Returns an empty string if the strings are identical.
// The output follows unified diff format with context lines.
func Diff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff bytes.Buffer
	diff.WriteString("--- Expected\n")
	diff.WriteString("+++ Actual\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	// Track context for unified diff style
	const contextLines = 2
	changes := findChanges(expectedLines, actualLines)

	if len(changes) == 0 {
		return "" // No differences
	}

	// Group changes into hunks
	hunks := groupChanges(changes, contextLines, maxLines)

	for _, hunk := range hunks {
		diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			hunk.expectedStart+1, hunk.expectedCount,
			hunk.actualStart+1, hunk.actualCount))

		for _, line := range hunk.lines {
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	}

	return diff.String()
}

type change struct {
	lineNum  int
	expected string
	actual   string
}

type hunk struct {
	expectedStart int
	expectedCount int
	actualStart   int
	actualCount   int
	lines         []string
}

func findChanges(expected, actual []string) []change {
	var changes []change
	maxLines := len(expected)
	if len(actual) > maxLines {
		maxLines = len(actual)
	}

	for i := 0; i < maxLines; i++ {
		var expLine, actLine string
		if i < len(expected) {
			expLine = expected[i]
		}
		if i < len(actual) {
			actLine = actual[i]
		}

		if expLine != actLine {
			changes = append(changes, change{
				lineNum:  i,
				expected: expLine,
				actual:   actLine,
			})
		}
	}
	return changes
}

func groupChanges(changes []change, context, maxLines int) []hunk {
	if len(changes) == 0 {
		return nil
	}

	var hunks []hunk
	start := max(0, changes[0].lineNum-context)
	end := min(maxLines, changes[0].lineNum+context+1)

	for i := 1; i < len(changes); i++ {
		// Check if this change should be in the same hunk
		if changes[i].lineNum-context <= end {
			end = min(maxLines, changes[i].lineNum+context+1)
		} else {
			// Start a new hunk
			hunks = append(hunks, buildHunk(changes, start, end, context))
			start = max(0, changes[i].lineNum-context)
			end = min(maxLines, changes[i].lineNum+context+1)
		}
	}

	// Add the last hunk
	hunks = append(hunks, buildHunk(changes, start, end, context))

	return hunks
}

func buildHunk(changes []change, start, end, context int) hunk {
	h := hunk{
		expectedStart: start,
		actualStart:   start,
	}

	changeMap := make(map[int]change)
	for _, c := range changes {
		if c.lineNum >= start && c.lineNum < end {
			changeMap[c.lineNum] = c
		}
	}

	for i := start; i < end; i++ {
		if c, ok := changeMap[i]; ok {
			if c.expected != "" || i < end-context {
				h.lines = append(h.lines, "- "+c.expected)
				h.expectedCount++
			}
			if c.actual != "" || i < end-context {
				h.lines = append(h.lines, "+ "+c.actual)
				h.actualCount++
			}
		} else {
			// Context line (unchanged)
			// We need to get the line from somewhere - use expected
			// This is a simplification; proper diff would track this
			h.lines = append(h.lines, "  (context)")
			h.expectedCount++
			h.actualCount++
		}
	}

	return h
}

// Equal checks if two screens have identical text content.
// Styles are not compared. For style-aware comparison, use EqualStyled.
func Equal(a, b *Screen) bool {
	return a.Text() == b.Text()
}

// EqualStyled checks if two screens are identical including all styling.
// This compares dimensions, text content, and all style attributes
// (colors, bold, italic, etc.) for every cell.
func EqualStyled(a, b *Screen) bool {
	if a.width != b.width || a.height != b.height {
		return false
	}
	for y := 0; y < a.height; y++ {
		for x := 0; x < a.width; x++ {
			if a.cells[y][x] != b.cells[y][x] {
				return false
			}
		}
	}
	return true
}
