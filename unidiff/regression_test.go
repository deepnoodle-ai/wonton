package unidiff

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// BUG-071: lines after a complete hunk, such as blank separators and the
// git format-patch signature, are not part of the hunk.
func TestParseIgnoresLinesAfterCompleteHunk(t *testing.T) {
	diff, err := Parse("diff --git a/f.txt b/f.txt\n--- a/f.txt\n+++ b/f.txt\n@@ -1 +1 @@\n-old\n+new\n\n-- \n2.40.0\n")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(diff.Files))
	assert.Equal(t, 1, len(diff.Files[0].Hunks))
	lines := diff.Files[0].Hunks[0].Lines
	assert.Equal(t, 2, len(lines))
	assert.Equal(t, LineRemoved, lines[0].Type)
	assert.Equal(t, LineAdded, lines[1].Type)
}

func TestParseBlankLineBetweenFiles(t *testing.T) {
	diff, err := Parse("--- a/a.txt\n+++ b/a.txt\n@@ -1 +1 @@\n-a\n+b\n\n--- a/b.txt\n+++ b/b.txt\n@@ -1 +1 @@\n-c\n+d\n")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(diff.Files))
	assert.Equal(t, 2, len(diff.Files[0].Hunks[0].Lines))
}

// BUG-071: once one side of a hunk is used up, a line for that side ends the
// hunk instead of being consumed.
func TestParseStopsAtExhaustedSide(t *testing.T) {
	// The hunk declares 1 old line and 2 new lines, but only has one "+"
	// line before the next file header.
	diff, err := Parse("--- a/a.txt\n+++ b/a.txt\n@@ -1,1 +1,2 @@\n-a\n+b\n--- a/next.go\n+++ b/next.go\n@@ -1 +1 @@\n-x\n+y\n")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(diff.Files))
	assert.Equal(t, "next.go", diff.Files[1].OldPath)
	assert.Equal(t, 2, len(diff.Files[0].Hunks[0].Lines))
}

// A hunk before any file header belongs to no file and must not attach to
// the next one.
func TestParseDropsHunkBeforeFileHeader(t *testing.T) {
	diff, err := Parse("@@ -1 +1 @@\n-orphan\n+orphan\ndiff --git a/f.txt b/f.txt\n--- a/f.txt\n+++ b/f.txt\n@@ -1 +1 @@\n-old\n+new\n")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(diff.Files))
	assert.Equal(t, "f.txt", diff.Files[0].NewPath)
	assert.Equal(t, 1, len(diff.Files[0].Hunks))
	assert.Equal(t, "old", diff.Files[0].Hunks[0].Lines[0].Content)
}
