package unidiff

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestLineType_String(t *testing.T) {
	tests := []struct {
		name     string
		lineType LineType
		want     string
	}{
		{"context", LineContext, "context"},
		{"added", LineAdded, "added"},
		{"removed", LineRemoved, "removed"},
		{"header", LineHeader, "header"},
		{"hunk header", LineHunkHeader, "hunk"},
		{"unknown", LineType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.lineType.String())
		})
	}
}

func TestParse_SingleFile(t *testing.T) {
	diffText := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 package main

+import "fmt"
 func main() {
`

	diff, err := Parse(diffText)
	assert.NoError(t, err)
	assert.Len(t, diff.Files, 1)

	file := diff.Files[0]
	assert.Equal(t, "file.go", file.OldPath)
	assert.Equal(t, "file.go", file.NewPath)
	assert.Len(t, file.Hunks, 1)

	hunk := file.Hunks[0]
	assert.Equal(t, 1, hunk.OldStart)
	assert.Equal(t, 3, hunk.OldCount)
	assert.Equal(t, 1, hunk.NewStart)
	assert.Equal(t, 4, hunk.NewCount)
	assert.Equal(t, "@@ -1,3 +1,4 @@", hunk.Header)
}

func TestParse_MultipleFiles(t *testing.T) {
	diffText := `diff --git a/file1.go b/file1.go
--- a/file1.go
+++ b/file1.go
@@ -1,2 +1,2 @@
-old line
+new line
diff --git a/file2.go b/file2.go
--- a/file2.go
+++ b/file2.go
@@ -1 +1 @@
-another old
+another new
`

	diff, err := Parse(diffText)
	assert.NoError(t, err)
	assert.Len(t, diff.Files, 2)

	assert.Equal(t, "file1.go", diff.Files[0].OldPath)
	assert.Equal(t, "file2.go", diff.Files[1].OldPath)
}

func TestParse_LineTypes(t *testing.T) {
	diffText := "diff --git a/test.txt b/test.txt\n" +
		"--- a/test.txt\n" +
		"+++ b/test.txt\n" +
		"@@ -1,4 +1,4 @@\n" +
		" context line\n" +
		"-removed line\n" +
		"+added line\n" +
		" another context"

	diff, err := Parse(diffText)
	assert.NoError(t, err)
	assert.Len(t, diff.Files, 1)

	lines := diff.Files[0].Hunks[0].Lines
	assert.Len(t, lines, 4)

	// Context line
	assert.Equal(t, LineContext, lines[0].Type)
	assert.Equal(t, "context line", lines[0].Content)
	assert.Equal(t, 1, lines[0].OldLineNum)
	assert.Equal(t, 1, lines[0].NewLineNum)

	// Removed line
	assert.Equal(t, LineRemoved, lines[1].Type)
	assert.Equal(t, "removed line", lines[1].Content)
	assert.Equal(t, 2, lines[1].OldLineNum)
	assert.Equal(t, 0, lines[1].NewLineNum)

	// Added line
	assert.Equal(t, LineAdded, lines[2].Type)
	assert.Equal(t, "added line", lines[2].Content)
	assert.Equal(t, 0, lines[2].OldLineNum)
	assert.Equal(t, 2, lines[2].NewLineNum)

	// Another context line
	assert.Equal(t, LineContext, lines[3].Type)
	assert.Equal(t, "another context", lines[3].Content)
}

func TestParse_MultipleHunks(t *testing.T) {
	diffText := `diff --git a/test.txt b/test.txt
--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
 first
-old1
+new1
 end1
@@ -10,3 +10,3 @@
 second
-old2
+new2
 end2
`

	diff, err := Parse(diffText)
	assert.NoError(t, err)
	assert.Len(t, diff.Files, 1)
	assert.Len(t, diff.Files[0].Hunks, 2)

	hunk1 := diff.Files[0].Hunks[0]
	assert.Equal(t, 1, hunk1.OldStart)
	assert.Equal(t, 1, hunk1.NewStart)

	hunk2 := diff.Files[0].Hunks[1]
	assert.Equal(t, 10, hunk2.OldStart)
	assert.Equal(t, 10, hunk2.NewStart)
}

func TestParse_EmptyDiff(t *testing.T) {
	diff, err := Parse("")
	assert.NoError(t, err)
	assert.Empty(t, diff.Files)
}

func TestParse_RawLine(t *testing.T) {
	diffText := `diff --git a/test.txt b/test.txt
--- a/test.txt
+++ b/test.txt
@@ -1,2 +1,2 @@
 context
+added
`

	diff, err := Parse(diffText)
	assert.NoError(t, err)

	lines := diff.Files[0].Hunks[0].Lines
	assert.Equal(t, " context", lines[0].RawLine)
	assert.Equal(t, "+added", lines[1].RawLine)
}

func TestParse_PathWithoutPrefix(t *testing.T) {
	// Test when paths don't have a/ or b/ prefix
	diffText := `diff --git a/file.go b/file.go
--- file.go
+++ file.go
@@ -1 +1 @@
-old
+new
`

	diff, err := Parse(diffText)
	assert.NoError(t, err)
	assert.Equal(t, "file.go", diff.Files[0].OldPath)
	assert.Equal(t, "file.go", diff.Files[0].NewPath)
}

func TestDiff_Stats(t *testing.T) {
	tests := []struct {
		name      string
		diffText  string
		additions int
		deletions int
		files     int
	}{
		{
			name: "single file with changes",
			diffText: `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 context
-removed
+added1
+added2
`,
			additions: 2,
			deletions: 1,
			files:     1,
		},
		{
			name: "multiple files",
			diffText: `diff --git a/file1.go b/file1.go
--- a/file1.go
+++ b/file1.go
@@ -1 +1 @@
-old
+new
diff --git a/file2.go b/file2.go
--- a/file2.go
+++ b/file2.go
@@ -1 +1,2 @@
+added
 context
`,
			additions: 2,
			deletions: 1,
			files:     2,
		},
		{
			name:      "empty diff",
			diffText:  "",
			additions: 0,
			deletions: 0,
			files:     0,
		},
		{
			name: "only context lines",
			diffText: `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,2 +1,2 @@
 context1
 context2
`,
			additions: 0,
			deletions: 0,
			files:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := Parse(tt.diffText)
			assert.NoError(t, err)

			stats := diff.Stats()
			assert.Equal(t, tt.files, stats.FilesChanged)
			assert.Equal(t, tt.additions, stats.Additions)
			assert.Equal(t, tt.deletions, stats.Deletions)
		})
	}
}

func TestParse_EmptyLine(t *testing.T) {
	// Test handling of empty context lines
	diffText := "diff --git a/test.txt b/test.txt\n" +
		"--- a/test.txt\n" +
		"+++ b/test.txt\n" +
		"@@ -1,3 +1,3 @@\n" +
		" line1\n" +
		"\n" +
		" line3"

	diff, err := Parse(diffText)
	assert.NoError(t, err)

	lines := diff.Files[0].Hunks[0].Lines
	assert.Len(t, lines, 3)

	// Empty line should be treated as context
	assert.Equal(t, LineContext, lines[1].Type)
	assert.Equal(t, "", lines[1].Content)
}

func TestParse_HunkWithoutCounts(t *testing.T) {
	// When counts aren't provided (e.g., @@ -1 +1 @@), the count defaults to 1
	diffText := `diff --git a/test.txt b/test.txt
--- a/test.txt
+++ b/test.txt
@@ -5 +5 @@
-old
+new
`

	diff, err := Parse(diffText)
	assert.NoError(t, err)

	hunk := diff.Files[0].Hunks[0]
	assert.Equal(t, 5, hunk.OldStart)
	assert.Equal(t, 5, hunk.NewStart)
}
