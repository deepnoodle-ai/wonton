package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

const sampleDiff = `diff --git a/main.go b/main.go
--- a/main.go
+++ b/main.go
@@ -1,7 +1,8 @@
 package main

 import (
-	"fmt"
+	"log"
+	"os"
 )

 func main() {
-	fmt.Println("Hello")
+	log.Println("Hello, World!")
 }`

func TestParseUnifiedDiff(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)
	assert.NotNil(t, diff)

	// Should have one file
	assert.Len(t, diff.Files, 1)

	file := diff.Files[0]
	assert.Equal(t, "main.go", file.OldPath)
	assert.Equal(t, "main.go", file.NewPath)

	// Should have one hunk
	assert.Len(t, file.Hunks, 1)

	hunk := file.Hunks[0]
	assert.Equal(t, 1, hunk.OldStart)
	assert.Equal(t, 7, hunk.OldCount)
	assert.Equal(t, 1, hunk.NewStart)
	assert.Equal(t, 8, hunk.NewCount)

	// Check some lines
	assert.Greater(t, len(hunk.Lines), 0)

	// Find the removed "fmt" line
	foundRemoved := false
	foundAdded := false
	for _, line := range hunk.Lines {
		if line.Type == DiffLineRemoved && line.Content == "\t\"fmt\"" {
			foundRemoved = true
		}
		if line.Type == DiffLineAdded && line.Content == "\t\"log\"" {
			foundAdded = true
		}
	}
	assert.True(t, foundRemoved, "Should find removed 'fmt' line")
	assert.True(t, foundAdded, "Should find added 'log' line")
}

func TestParseDiff_MultipleFiles(t *testing.T) {
	multiFileDiff := `diff --git a/file1.go b/file1.go
--- a/file1.go
+++ b/file1.go
@@ -1,1 +1,1 @@
-old content
+new content
diff --git a/file2.go b/file2.go
--- a/file2.go
+++ b/file2.go
@@ -1,1 +1,1 @@
-old content 2
+new content 2`

	diff, err := ParseUnifiedDiff(multiFileDiff)
	assert.NoError(t, err)
	assert.Len(t, diff.Files, 2)

	assert.Equal(t, "file1.go", diff.Files[0].NewPath)
	assert.Equal(t, "file2.go", diff.Files[1].NewPath)
}

func TestParseDiff_MultipleHunks(t *testing.T) {
	multiHunkDiff := `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,2 +1,2 @@
 line1
-line2
+line2 modified
@@ -10,2 +10,2 @@
 line10
-line11
+line11 modified`

	diff, err := ParseUnifiedDiff(multiHunkDiff)
	assert.NoError(t, err)
	assert.Len(t, diff.Files, 1)
	assert.Len(t, diff.Files[0].Hunks, 2)

	assert.Equal(t, 1, diff.Files[0].Hunks[0].OldStart)
	assert.Equal(t, 10, diff.Files[0].Hunks[1].OldStart)
}

func TestDiffLineTypes(t *testing.T) {
	testDiff := `diff --git a/test.txt b/test.txt
--- a/test.txt
+++ b/test.txt
@@ -1,3 +1,3 @@
 context line
-removed line
+added line`

	diff, err := ParseUnifiedDiff(testDiff)
	assert.NoError(t, err)

	hunk := diff.Files[0].Hunks[0]

	foundContext := false
	foundRemoved := false
	foundAdded := false

	for _, line := range hunk.Lines {
		switch line.Type {
		case DiffLineContext:
			if line.Content == "context line" {
				foundContext = true
				assert.Greater(t, line.OldLineNum, 0)
				assert.Greater(t, line.NewLineNum, 0)
			}
		case DiffLineRemoved:
			if line.Content == "removed line" {
				foundRemoved = true
				assert.Greater(t, line.OldLineNum, 0)
				assert.Equal(t, 0, line.NewLineNum)
			}
		case DiffLineAdded:
			if line.Content == "added line" {
				foundAdded = true
				assert.Equal(t, 0, line.OldLineNum)
				assert.Greater(t, line.NewLineNum, 0)
			}
		}
	}

	assert.True(t, foundContext, "Should find context line")
	assert.True(t, foundRemoved, "Should find removed line")
	assert.True(t, foundAdded, "Should find added line")
}
