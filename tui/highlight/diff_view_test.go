package highlight

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/tui"
	"github.com/deepnoodle-ai/wonton/tui/tuitest"
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

func TestDiffRenderer_BasicRendering(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

	renderer := NewDiffRenderer()
	rendered := renderer.RenderDiff(diff, "go")

	assert.NotEmpty(t, rendered)

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
	assert.True(t, foundOldPath, "Should find old path header")
	assert.True(t, foundNewPath, "Should find new path header")
}

func TestDiffRenderer_SyntaxHighlighting(t *testing.T) {
	simpleDiff := `diff --git a/test.go b/test.go
--- a/test.go
+++ b/test.go
@@ -1,1 +1,1 @@
-package main
+package test`

	diff, err := tui.ParseUnifiedDiff(simpleDiff)
	assert.NoError(t, err)

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
	assert.True(t, foundHighlighted, "Should find syntax-highlighted 'package' keyword")
}

func TestDiffRenderer_LineNumbers(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

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
	assert.True(t, hasLineNums, "Should have line numbers")
}

func TestDiffRenderer_NoLineNumbers(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

	renderer := NewDiffRenderer()
	renderer.ShowLineNums = false
	rendered := renderer.RenderDiff(diff, "go")

	// Line numbers should be empty
	for _, line := range rendered {
		assert.Empty(t, line.LineNumOld)
		assert.Empty(t, line.LineNumNew)
	}
}

func TestDiffRenderer_CustomTheme(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

	// Create custom theme
	theme := DefaultDiffTheme()
	theme.AddedFg = tui.RGB{R: 255, G: 0, B: 0} // Red for testing
	theme.SyntaxTheme = "github"

	renderer := NewDiffRenderer().WithTheme(theme)
	rendered := renderer.RenderDiff(diff, "go")

	assert.NotEmpty(t, rendered)
}

func TestDiffView_Creation(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

	scrollY := 0
	view := DiffView(diff, "go", &scrollY)
	assert.NotNil(t, view)
	assert.Equal(t, diff, view.diff)
}

func TestDiffView_FromText(t *testing.T) {
	scrollY := 0
	view, err := DiffViewFromText(sampleDiff, "go", &scrollY)
	assert.NoError(t, err)
	assert.NotNil(t, view)
	assert.Len(t, view.diff.Files, 1)
}

func TestDiffView_Scrolling(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

	scrollY := 0
	view := DiffView(diff, "go", &scrollY).Height(10)

	// Render to initialize
	view.renderContent()

	// Initial position
	assert.Equal(t, 0, scrollY)

	// Scroll down
	scrollY = 5
	assert.Equal(t, 5, scrollY)

	// Scroll up
	scrollY = 3
	assert.Equal(t, 3, scrollY)

	// Scroll to position
	scrollY = 10
	assert.Equal(t, 10, scrollY)

	// Scroll to top
	scrollY = 0
	assert.Equal(t, 0, scrollY)
}

func TestDiffView_LineCount(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

	scrollY := 0
	view := DiffView(diff, "go", &scrollY)

	lineCount := view.GetLineCount()
	assert.Greater(t, lineCount, 0)
}

func TestDiffView_LanguageSetting(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

	scrollY := 0
	view := DiffView(diff, "go", &scrollY).Language("python")
	assert.Equal(t, "python", view.language)
}

func TestDiffView_ThemeSetting(t *testing.T) {
	diff, err := tui.ParseUnifiedDiff(sampleDiff)
	assert.NoError(t, err)

	theme := DefaultDiffTheme()
	theme.AddedBg = tui.RGB{R: 0, G: 128, B: 0}

	scrollY := 0
	view := DiffView(diff, "go", &scrollY).Theme(theme)
	assert.Equal(t, theme.AddedBg, view.theme.AddedBg)
}

func TestDiffRenderer_TabExpansion(t *testing.T) {
	// Test that tabs are properly expanded to spaces
	renderer := NewDiffRenderer()
	renderer.TabWidth = 4

	// Test simple tab expansion
	result := renderer.expandTabs("\tcode")
	assert.Equal(t, "    code", result)

	// Test tab at different positions
	result = renderer.expandTabs("x\tcode")
	assert.Equal(t, "x   code", result)

	result = renderer.expandTabs("xx\tcode")
	assert.Equal(t, "xx  code", result)

	result = renderer.expandTabs("xxx\tcode")
	assert.Equal(t, "xxx code", result)

	result = renderer.expandTabs("xxxx\tcode")
	assert.Equal(t, "xxxx    code", result)

	// Test multiple tabs
	result = renderer.expandTabs("\t\tcode")
	assert.Equal(t, "        code", result)

	// Test string without tabs
	result = renderer.expandTabs("no tabs here")
	assert.Equal(t, "no tabs here", result)

	// Wide CJK character (width 2) before tab should expand to 2 spaces to reach tab stop 4
	result = renderer.expandTabs("中\tcode")
	assert.Equal(t, "中  code", result)

	// Combining mark (2 runes, display width 1) before tab should expand to 3 spaces to reach tab stop 4
	result = renderer.expandTabs("e\u0301\tcode")
	assert.Equal(t, "e\u0301   code", result)

	// Zero or negative TabWidth falls back to 4 instead of panicking with divide-by-zero
	zeroTabRenderer := &DiffRenderer{TabWidth: 0}
	assert.Equal(t, "    code", zeroTabRenderer.expandTabs("\tcode"))

	// Nil diff returns nil without panicking
	assert.Nil(t, renderer.RenderDiff(nil, "go"))
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

	diff, err := tui.ParseUnifiedDiff(diffWithTabs)
	assert.NoError(t, err)

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

	assert.True(t, foundExpanded, "Should find tab-expanded content with 4 spaces")
}

func TestDiffView_LongLineClips(t *testing.T) {
	diffText := "--- a/f.txt\n+++ b/f.txt\n@@ -1,2 +1,2 @@\n-" + strings.Repeat("x", 60) + "\n \n"
	scrollY := 0
	view, err := DiffViewFromText(diffText, "", &scrollY)
	assert.NoError(t, err)
	screen := tuitest.Screen(view, tui.WithWidth(40), tui.WithHeight(10))
	for y := range 9 {
		if strings.Contains(screen.Row(y), "xxx") {
			assert.False(t, strings.Contains(screen.Row(y+1), "x"), "wrapped onto next row: %q", screen.Row(y+1))
			return
		}
	}
	t.Fatal("diff line not rendered")
}
