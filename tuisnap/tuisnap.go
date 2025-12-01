// Package tuisnap provides snapshot testing utilities for TUI applications.
// It captures the rendered output of TUI views and compares them against golden files.
//
// Basic usage:
//
//	func TestMyView(t *testing.T) {
//	    view := MyView()
//	    tuisnap.Assert(t, view, 80, 24)
//	}
//
// To update snapshots, run tests with -update flag:
//
//	go test -update ./...
//
// Or set the TUISNAP_UPDATE environment variable:
//
//	TUISNAP_UPDATE=1 go test ./...
package tuisnap

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/terminal"
)

var update = flag.Bool("update", false, "update snapshot files")

// View is the interface for TUI views that can be rendered.
type View interface {
	// These are the internal render methods from tui.View
	render(frame terminal.RenderFrame, bounds image.Rectangle)
	size(maxWidth, maxHeight int) (width, height int)
}

// Assert renders a view and compares it to a golden file.
// The snapshot name is derived from the test name.
func Assert(t *testing.T, view View, width, height int) {
	t.Helper()
	AssertNamed(t, t.Name(), view, width, height)
}

// AssertNamed renders a view and compares it to a named golden file.
func AssertNamed(t *testing.T, name string, view View, width, height int) {
	t.Helper()

	// Render the view
	actual := Render(view, width, height)

	// Get snapshot path
	snapshotDir := filepath.Join("testdata", "snapshots")
	snapshotPath := filepath.Join(snapshotDir, sanitizeName(name)+".snap")

	// Check if we should update
	shouldUpdate := *update || os.Getenv("TUISNAP_UPDATE") != ""

	if shouldUpdate {
		// Update the snapshot
		if err := os.MkdirAll(snapshotDir, 0755); err != nil {
			t.Fatalf("failed to create snapshot directory: %v", err)
		}
		if err := os.WriteFile(snapshotPath, []byte(actual), 0644); err != nil {
			t.Fatalf("failed to write snapshot: %v", err)
		}
		t.Logf("Updated snapshot: %s", snapshotPath)
		return
	}

	// Compare to existing snapshot
	expected, err := os.ReadFile(snapshotPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("snapshot not found: %s\nRun with -update to create it.\n\nActual output:\n%s", snapshotPath, actual)
		}
		t.Fatalf("failed to read snapshot: %v", err)
	}

	if actual != string(expected) {
		diff := generateDiff(string(expected), actual)
		t.Errorf("snapshot mismatch: %s\n%s\nRun with -update to update the snapshot.", snapshotPath, diff)
	}
}

// Render renders a view to a string representation.
func Render(view View, width, height int) string {
	// Create a test terminal buffer
	var buf bytes.Buffer
	term := terminal.NewTestTerminal(width, height, &buf)

	// Get a render frame
	frame, err := term.BeginFrame()
	if err != nil {
		return fmt.Sprintf("error starting frame: %v", err)
	}

	// Render the view
	bounds := image.Rect(0, 0, width, height)
	view.render(frame, bounds)

	// End the frame
	term.EndFrame(frame)

	// Convert buffer to string
	return bufferToString(term, width, height)
}

// bufferToString converts the terminal buffer to a string representation.
func bufferToString(term *terminal.Terminal, width, height int) string {
	var buf bytes.Buffer

	for y := 0; y < height; y++ {
		line := ""
		trailingSpaces := 0

		for x := 0; x < width; x++ {
			cell := term.GetCell(x, y)
			r := cell.Char
			if r == 0 {
				r = ' '
			}

			if r == ' ' {
				trailingSpaces++
			} else {
				// Add accumulated spaces
				line += strings.Repeat(" ", trailingSpaces)
				trailingSpaces = 0
				line += string(r)
			}
		}

		// Trim trailing spaces
		buf.WriteString(line)
		buf.WriteRune('\n')
	}

	return buf.String()
}

// sanitizeName converts a test name to a valid filename.
func sanitizeName(name string) string {
	// Replace problematic characters
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

// generateDiff creates a simple diff between expected and actual strings.
func generateDiff(expected, actual string) string {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	var diff bytes.Buffer
	diff.WriteString("--- Expected\n")
	diff.WriteString("+++ Actual\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		var expLine, actLine string
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actLine = actualLines[i]
		}

		if expLine != actLine {
			diff.WriteString(fmt.Sprintf("@@ line %d @@\n", i+1))
			diff.WriteString(fmt.Sprintf("- %q\n", expLine))
			diff.WriteString(fmt.Sprintf("+ %q\n", actLine))
		}
	}

	return diff.String()
}

// RenderString is a simpler version that renders without styles.
// Useful for quick visual inspection.
func RenderString(view View, width, height int) string {
	return Render(view, width, height)
}

// AssertString compares the rendered output to an expected string.
// This is useful for inline test assertions without golden files.
func AssertString(t *testing.T, view View, width, height int, expected string) {
	t.Helper()

	actual := Render(view, width, height)

	if actual != expected {
		diff := generateDiff(expected, actual)
		t.Errorf("output mismatch:\n%s", diff)
	}
}

// StyledCell represents a cell with its character and style information.
type StyledCell struct {
	Char       rune
	Foreground terminal.Color
	Background terminal.Color
	Bold       bool
	Italic     bool
	Underline  bool
}

// RenderStyled renders a view and returns the full styled output.
func RenderStyled(view View, width, height int) [][]StyledCell {
	var buf bytes.Buffer
	term := terminal.NewTestTerminal(width, height, &buf)

	frame, err := term.BeginFrame()
	if err != nil {
		return nil
	}

	bounds := image.Rect(0, 0, width, height)
	view.render(frame, bounds)
	term.EndFrame(frame)

	result := make([][]StyledCell, height)
	for y := 0; y < height; y++ {
		result[y] = make([]StyledCell, width)
		for x := 0; x < width; x++ {
			cell := term.GetCell(x, y)
			result[y][x] = StyledCell{
				Char:       cell.Char,
				Foreground: cell.Style.Foreground,
				Background: cell.Style.Background,
				Bold:       cell.Style.Bold,
				Italic:     cell.Style.Italic,
				Underline:  cell.Style.Underline,
			}
		}
	}

	return result
}

// HasText checks if the rendered output contains the given text.
func HasText(view View, width, height int, text string) bool {
	output := Render(view, width, height)
	return strings.Contains(output, text)
}

// TextAt returns the text at a specific position.
func TextAt(view View, width, height, x, y, length int) string {
	var buf bytes.Buffer
	term := terminal.NewTestTerminal(width, height, &buf)

	frame, err := term.BeginFrame()
	if err != nil {
		return ""
	}

	bounds := image.Rect(0, 0, width, height)
	view.render(frame, bounds)
	term.EndFrame(frame)

	var result strings.Builder
	for i := 0; i < length && x+i < width; i++ {
		cell := term.GetCell(x+i, y)
		r := cell.Char
		if r == 0 {
			r = ' '
		}
		result.WriteRune(r)
	}

	return result.String()
}

// Convenience type alias for cleaner imports
type (
	// RenderFrame is a render frame from the terminal package
	RenderFrame = terminal.RenderFrame
)

var _ io.Writer // ensure io import is used
