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

	raw, err := os.ReadFile(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("snapshot not found: %s\nRun with -update to create it.\n\nActual output:\n%s", snapshotPath, actual)
		}
		t.Fatalf("failed to read snapshot: %v", err)
	}

	// Normalize CRLF → LF so snapshots survive Windows git checkouts
	// (core.autocrlf converts LF to CRLF on checkout by default).
	expected := strings.ReplaceAll(string(raw), "\r\n", "\n")

	if actual != expected {
		diff := Diff(expected, actual)
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
	if expected == actual {
		return ""
	}

	expectedLines := diffSplit(expected)
	actualLines := diffSplit(actual)
	ops := diffLines(expectedLines, actualLines)

	// Line offsets: ePos[k]/aPos[k] = expected/actual lines consumed before ops[k]
	ePos := make([]int, len(ops)+1)
	aPos := make([]int, len(ops)+1)
	for k, op := range ops {
		ePos[k+1] = ePos[k]
		aPos[k+1] = aPos[k]
		switch op.kind {
		case ' ':
			ePos[k+1]++
			aPos[k+1]++
		case '-':
			ePos[k+1]++
		case '+':
			aPos[k+1]++
		}
	}

	// Group changed ops into hunks padded with context lines
	const contextLines = 2
	var hunks [][2]int
	for k, op := range ops {
		if op.kind == ' ' {
			continue
		}
		start := max(0, k-contextLines)
		end := min(len(ops), k+contextLines+1)
		if len(hunks) > 0 && start <= hunks[len(hunks)-1][1] {
			hunks[len(hunks)-1][1] = end
		} else {
			hunks = append(hunks, [2]int{start, end})
		}
	}

	var diff bytes.Buffer
	diff.WriteString("--- Expected\n")
	diff.WriteString("+++ Actual\n")
	for _, h := range hunks {
		start, end := h[0], h[1]
		diff.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			ePos[start]+1, ePos[end]-ePos[start],
			aPos[start]+1, aPos[end]-aPos[start]))
		for k := start; k < end; k++ {
			diff.WriteByte(ops[k].kind)
			diff.WriteByte(' ')
			diff.WriteString(ops[k].text)
			diff.WriteByte('\n')
		}
	}

	return diff.String()
}

// diffSplit splits s into lines for diffing. A missing trailing newline is
// represented as an explicit marker line so that "a\n" and "a" differ.
func diffSplit(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if lines[len(lines)-1] == "" {
		return lines[:len(lines)-1]
	}
	return append(lines, `\ No newline at end of file`)
}

// diffOp is a single line of a diff: kind is ' ' (unchanged), '-' (only in
// expected), or '+' (only in actual).
type diffOp struct {
	kind byte
	text string
}

// diffLines computes a line-level diff of a and b using a longest-common-
// subsequence alignment, so an inserted or deleted line doesn't cascade into
// marking every following line as changed.
func diffLines(a, b []string) []diffOp {
	n, m := len(a), len(b)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}

	ops := make([]diffOp, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{' ', a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, diffOp{'-', a[i]})
			i++
		default:
			ops = append(ops, diffOp{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, diffOp{'-', a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffOp{'+', b[j]})
	}
	return ops
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
