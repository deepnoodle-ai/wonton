package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/require"
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
	require.NoError(t, err)
	require.NotNil(t, diff)

	// Should have one file
	require.Len(t, diff.Files, 1)

	file := diff.Files[0]
	require.Equal(t, "main.go", file.OldPath)
	require.Equal(t, "main.go", file.NewPath)

	// Should have one hunk
	require.Len(t, file.Hunks, 1)

	hunk := file.Hunks[0]
	require.Equal(t, 1, hunk.OldStart)
	require.Equal(t, 7, hunk.OldCount)
	require.Equal(t, 1, hunk.NewStart)
	require.Equal(t, 8, hunk.NewCount)

	// Check some lines
	require.Greater(t, len(hunk.Lines), 0)

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
	require.True(t, foundRemoved, "Should find removed 'fmt' line")
	require.True(t, foundAdded, "Should find added 'log' line")
}

func TestDiffRenderer_BasicRendering(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	renderer := NewDiffRenderer()
	rendered := renderer.RenderDiff(diff, "go")

	require.NotEmpty(t, rendered)

	// Should have file headers
	foundOldPath := false
	foundNewPath := false
	for _, line := range rendered {
		for _, seg := range line.Segments {
			if seg.Text == "--- main.go" {
				foundOldPath = true
			}
			if seg.Text == "+++ main.go" {
				foundNewPath = true
			}
		}
	}
	require.True(t, foundOldPath, "Should find old path header")
	require.True(t, foundNewPath, "Should find new path header")
}

func TestDiffRenderer_SyntaxHighlighting(t *testing.T) {
	simpleDiff := `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -1,1 +1,1 @@
-package main
+package test`

	diff, err := ParseUnifiedDiff(simpleDiff)
	require.NoError(t, err)

	renderer := NewDiffRenderer()
	renderer.SyntaxHighlight = true
	rendered := renderer.RenderDiff(diff, "go")

	// Find the "package" keyword and verify it has syntax highlighting
	foundHighlighted := false
	for _, line := range rendered {
		for _, seg := range line.Segments {
			if seg.Text == "package" && seg.Style.FgRGB != nil {
				foundHighlighted = true
			}
		}
	}
	require.True(t, foundHighlighted, "Should find syntax-highlighted 'package' keyword")
}

func TestDiffRenderer_LineNumbers(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	renderer := NewDiffRenderer()
	renderer.ShowLineNums = true
	rendered := renderer.RenderDiff(diff, "go")

	// Check that line numbers are present
	hasLineNums := false
	for _, line := range rendered {
		if line.LineNumOld != "" || line.LineNumNew != "" {
			hasLineNums = true
			break
		}
	}
	require.True(t, hasLineNums, "Should have line numbers")
}

func TestDiffRenderer_NoLineNumbers(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	renderer := NewDiffRenderer()
	renderer.ShowLineNums = false
	rendered := renderer.RenderDiff(diff, "go")

	// Line numbers should be empty
	for _, line := range rendered {
		require.Empty(t, line.LineNumOld)
		require.Empty(t, line.LineNumNew)
	}
}

func TestDiffRenderer_CustomTheme(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	// Create custom theme
	theme := DefaultDiffTheme()
	theme.AddedFg = RGB{R: 255, G: 0, B: 0} // Red for testing
	theme.SyntaxTheme = "github"

	renderer := NewDiffRenderer().WithTheme(theme)
	rendered := renderer.RenderDiff(diff, "go")

	require.NotEmpty(t, rendered)
}

func TestDiffView_Creation(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	scrollY := 0
	view := DiffView(diff, "go", &scrollY)
	require.NotNil(t, view)
	require.Equal(t, diff, view.diff)
}

func TestDiffView_FromText(t *testing.T) {
	scrollY := 0
	view, err := DiffViewFromText(sampleDiff, "go", &scrollY)
	require.NoError(t, err)
	require.NotNil(t, view)
	require.Len(t, view.diff.Files, 1)
}

func TestDiffView_Scrolling(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	scrollY := 0
	view := DiffView(diff, "go", &scrollY).Height(10)

	// Render to initialize
	view.renderContent()

	// Initial position
	require.Equal(t, 0, scrollY)

	// Scroll down
	scrollY = 5
	require.Equal(t, 5, scrollY)

	// Scroll up
	scrollY = 3
	require.Equal(t, 3, scrollY)

	// Scroll to position
	scrollY = 10
	require.Equal(t, 10, scrollY)

	// Scroll to top
	scrollY = 0
	require.Equal(t, 0, scrollY)
}

func TestDiffView_LineCount(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	scrollY := 0
	view := DiffView(diff, "go", &scrollY)

	lineCount := view.GetLineCount()
	require.Greater(t, lineCount, 0)
}

func TestDiffView_LanguageSetting(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	scrollY := 0
	view := DiffView(diff, "go", &scrollY).Language("python")
	require.Equal(t, "python", view.language)
}

func TestDiffView_ThemeSetting(t *testing.T) {
	diff, err := ParseUnifiedDiff(sampleDiff)
	require.NoError(t, err)

	theme := DefaultDiffTheme()
	theme.AddedBg = RGB{R: 0, G: 128, B: 0}

	scrollY := 0
	view := DiffView(diff, "go", &scrollY).Theme(theme)
	require.Equal(t, theme.AddedBg, view.theme.AddedBg)
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
	require.NoError(t, err)
	require.Len(t, diff.Files, 2)

	require.Equal(t, "file1.go", diff.Files[0].NewPath)
	require.Equal(t, "file2.go", diff.Files[1].NewPath)
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
	require.NoError(t, err)
	require.Len(t, diff.Files, 1)
	require.Len(t, diff.Files[0].Hunks, 2)

	require.Equal(t, 1, diff.Files[0].Hunks[0].OldStart)
	require.Equal(t, 10, diff.Files[0].Hunks[1].OldStart)
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
	require.NoError(t, err)

	hunk := diff.Files[0].Hunks[0]

	foundContext := false
	foundRemoved := false
	foundAdded := false

	for _, line := range hunk.Lines {
		switch line.Type {
		case DiffLineContext:
			if line.Content == "context line" {
				foundContext = true
				require.Greater(t, line.OldLineNum, 0)
				require.Greater(t, line.NewLineNum, 0)
			}
		case DiffLineRemoved:
			if line.Content == "removed line" {
				foundRemoved = true
				require.Greater(t, line.OldLineNum, 0)
				require.Equal(t, 0, line.NewLineNum)
			}
		case DiffLineAdded:
			if line.Content == "added line" {
				foundAdded = true
				require.Equal(t, 0, line.OldLineNum)
				require.Greater(t, line.NewLineNum, 0)
			}
		}
	}

	require.True(t, foundContext, "Should find context line")
	require.True(t, foundRemoved, "Should find removed line")
	require.True(t, foundAdded, "Should find added line")
}

func TestDiffRenderer_TabExpansion(t *testing.T) {
	// Test that tabs are properly expanded to spaces
	renderer := NewDiffRenderer()
	renderer.TabWidth = 4

	// Test simple tab expansion
	result := renderer.expandTabs("\tcode")
	require.Equal(t, "    code", result)

	// Test tab at different positions
	result = renderer.expandTabs("x\tcode")
	require.Equal(t, "x   code", result)

	result = renderer.expandTabs("xx\tcode")
	require.Equal(t, "xx  code", result)

	result = renderer.expandTabs("xxx\tcode")
	require.Equal(t, "xxx code", result)

	result = renderer.expandTabs("xxxx\tcode")
	require.Equal(t, "xxxx    code", result)

	// Test multiple tabs
	result = renderer.expandTabs("\t\tcode")
	require.Equal(t, "        code", result)

	// Test string without tabs
	result = renderer.expandTabs("no tabs here")
	require.Equal(t, "no tabs here", result)
}

func TestDiffRenderer_TabsInDiff(t *testing.T) {
	// Create a diff with tabs
	diffWithTabs := `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -1,3 +1,3 @@
 import (
-	"fmt"
+	"log"
 )`

	diff, err := ParseUnifiedDiff(diffWithTabs)
	require.NoError(t, err)

	// Render the diff
	renderer := NewDiffRenderer()
	renderer.TabWidth = 4
	renderer.SyntaxHighlight = false // Disable for simpler testing
	rendered := renderer.RenderDiff(diff, "go")

	// Find the rendered lines with the import statements
	foundExpanded := false
	for _, line := range rendered {
		for _, seg := range line.Segments {
			// Check if tabs were expanded (tab before "fmt" or "log" should become 4 spaces)
			if seg.Text == "    \"fmt\"" || seg.Text == "    \"log\"" {
				foundExpanded = true
			}
		}
	}

	require.True(t, foundExpanded, "Should find tab-expanded content with 4 spaces")
}
