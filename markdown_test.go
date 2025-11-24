package gooey

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarkdownRenderer_BasicFormatting(t *testing.T) {
	renderer := NewMarkdownRenderer()

	tests := []struct {
		name     string
		markdown string
		contains []string // Strings that should appear in output
	}{
		{
			name:     "Bold text",
			markdown: "This is **bold** text",
			contains: []string{"This is ", "bold", " text"},
		},
		{
			name:     "Italic text",
			markdown: "This is *italic* text",
			contains: []string{"This is ", "italic", " text"},
		},
		{
			name:     "Inline code",
			markdown: "This is `code` text",
			contains: []string{"This is ", "code", " text"},
		},
		{
			name:     "Combined formatting",
			markdown: "**Bold** and *italic* and `code`",
			contains: []string{"Bold", "italic", "code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(tt.markdown)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Check that all expected strings appear in the output
			output := renderToPlainText(result)
			for _, expected := range tt.contains {
				require.Contains(t, output, expected)
			}
		})
	}
}

func TestMarkdownRenderer_Headings(t *testing.T) {
	renderer := NewMarkdownRenderer()

	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "H1",
			markdown: "# Heading 1",
			expected: "Heading 1",
		},
		{
			name:     "H2",
			markdown: "## Heading 2",
			expected: "Heading 2",
		},
		{
			name:     "H3",
			markdown: "### Heading 3",
			expected: "Heading 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(tt.markdown)
			require.NoError(t, err)
			require.NotNil(t, result)

			output := renderToPlainText(result)
			require.Contains(t, output, tt.expected)
		})
	}
}

func TestMarkdownRenderer_Lists(t *testing.T) {
	renderer := NewMarkdownRenderer()

	tests := []struct {
		name     string
		markdown string
		contains []string
	}{
		{
			name: "Unordered list",
			markdown: `- Item 1
- Item 2
- Item 3`,
			contains: []string{"Item 1", "Item 2", "Item 3"},
		},
		{
			name: "Ordered list",
			markdown: `1. First
2. Second
3. Third`,
			contains: []string{"First", "Second", "Third"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(tt.markdown)
			require.NoError(t, err)
			require.NotNil(t, result)

			output := renderToPlainText(result)
			for _, expected := range tt.contains {
				require.Contains(t, output, expected)
			}
		})
	}
}

func TestMarkdownRenderer_CodeBlocks(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"

	result, err := renderer.Render(markdown)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Code blocks may contain ANSI codes from syntax highlighting
	// Just check that we have some lines with code content
	require.Greater(t, len(result.Lines), 0)

	// Check that code text appears somewhere in the segments
	hasFunc := false
	hasPrintln := false
	for _, line := range result.Lines {
		for _, seg := range line.Segments {
			if strings.Contains(seg.Text, "func") {
				hasFunc = true
			}
			if strings.Contains(seg.Text, "Println") {
				hasPrintln = true
			}
		}
	}
	require.True(t, hasFunc, "Expected to find 'func' in code block")
	require.True(t, hasPrintln, "Expected to find 'Println' in code block")
}

func TestMarkdownRenderer_Links(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := "Check out [this link](https://example.com)"

	result, err := renderer.Render(markdown)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify hyperlink was created
	found := false
	for _, line := range result.Lines {
		for _, seg := range line.Segments {
			if seg.Hyperlink != nil {
				require.Equal(t, "https://example.com", seg.Hyperlink.URL)
				require.Equal(t, "this link", seg.Hyperlink.Text)
				found = true
			}
		}
	}
	require.True(t, found, "Expected to find hyperlink in rendered output")
}

func TestMarkdownRenderer_HorizontalRule(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := "Before\n\n---\n\nAfter"

	result, err := renderer.Render(markdown)
	require.NoError(t, err)
	require.NotNil(t, result)

	output := renderToPlainText(result)
	require.Contains(t, output, "Before")
	require.Contains(t, output, "After")
	// Should contain horizontal rule characters
	require.Contains(t, output, renderer.Theme.HorizontalRuleChar)
}

func TestMarkdownRenderer_Paragraph(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := "This is a paragraph.\n\nThis is another paragraph."

	result, err := renderer.Render(markdown)
	require.NoError(t, err)
	require.NotNil(t, result)

	output := renderToPlainText(result)
	require.Contains(t, output, "This is a paragraph.")
	require.Contains(t, output, "This is another paragraph.")
}

func TestMarkdownRenderer_ComplexDocument(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := `# My Document

This is a **bold** statement with *italic* emphasis.

## Features

- Feature 1
- Feature 2
- Feature 3

### Code Example

` + "```go\nfunc hello() {\n    return \"world\"\n}\n```" + `

Check out [the docs](https://example.com) for more info.

---

End of document.
`

	result, err := renderer.Render(markdown)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Lines)

	output := renderToPlainText(result)
	require.Contains(t, output, "My Document")
	require.Contains(t, output, "bold")
	require.Contains(t, output, "italic")
	require.Contains(t, output, "Features")
	require.Contains(t, output, "Feature 1")
	// Check for code content in segments (may have ANSI codes)
	hasHello := false
	for _, line := range result.Lines {
		for _, seg := range line.Segments {
			if strings.Contains(seg.Text, "hello") {
				hasHello = true
			}
		}
	}
	require.True(t, hasHello, "Expected to find 'hello' in code block")
	require.Contains(t, output, "the docs")
	require.Contains(t, output, "End of document")
}

func TestMarkdownRenderer_CustomTheme(t *testing.T) {
	renderer := NewMarkdownRenderer()

	// Create a custom theme
	customTheme := DefaultMarkdownTheme()
	customTheme.H1Style = NewStyle().WithForeground(ColorRed).WithBold()
	customTheme.BulletChar = "*"

	renderer.WithTheme(customTheme)

	markdown := "# Red Heading\n\n- Item"

	result, err := renderer.Render(markdown)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify custom bullet character is used
	output := renderToPlainText(result)
	require.Contains(t, output, "*") // Custom bullet char
}

func TestMarkdownRenderer_MaxWidth(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(20)

	// Long line that should wrap
	markdown := "This is a very long line of text that should wrap at the maximum width"

	result, err := renderer.Render(markdown)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have multiple lines due to wrapping
	require.Greater(t, len(result.Lines), 2, "Expected text to wrap into multiple lines")
}

func TestMarkdownWidget_BasicRendering(t *testing.T) {
	widget := NewMarkdownWidget("# Test\n\nParagraph")
	require.NotNil(t, widget)

	// Set bounds
	widget.SetBounds(image.Rect(0, 0, 80, 25))

	// Initialize
	widget.Init()

	// Render to string for testing
	output := widget.RenderToString()
	require.Contains(t, output, "Test")
	require.Contains(t, output, "Paragraph")
}

func TestMarkdownWidget_Scrolling(t *testing.T) {
	content := strings.Repeat("Line\n\n", 50) // Create many lines
	widget := NewMarkdownWidget(content)
	widget.SetBounds(image.Rect(0, 0, 80, 10)) // Small viewport

	widget.Init()

	require.Equal(t, 0, widget.GetScrollPosition())

	// Scroll down
	widget.ScrollBy(5)
	require.Equal(t, 5, widget.GetScrollPosition())

	// Scroll up
	widget.ScrollBy(-2)
	require.Equal(t, 3, widget.GetScrollPosition())

	// Scroll to position
	widget.ScrollTo(10)
	require.Equal(t, 10, widget.GetScrollPosition())
}

func TestMarkdownWidget_ContentUpdate(t *testing.T) {
	widget := NewMarkdownWidget("Initial content")
	widget.SetBounds(image.Rect(0, 0, 80, 25))
	widget.Init()

	output := widget.RenderToString()
	require.Contains(t, output, "Initial content")

	// Update content
	widget.SetContent("Updated content")
	output = widget.RenderToString()
	require.Contains(t, output, "Updated content")
	require.NotContains(t, output, "Initial content")
}

func TestMarkdownViewer_KeyHandling(t *testing.T) {
	// Create content with many lines to enable scrolling
	var contentBuilder strings.Builder
	for i := 0; i < 50; i++ {
		contentBuilder.WriteString(fmt.Sprintf("Line %d\n\n", i))
	}
	content := contentBuilder.String()

	viewer := NewMarkdownViewer(content)
	viewer.SetBounds(image.Rect(0, 0, 80, 10))
	viewer.Init()

	// Force rendering by calling RenderToString
	_ = viewer.content.RenderToString()

	// Verify we have enough lines
	lineCount := viewer.content.GetLineCount()
	require.Greater(t, lineCount, 10, "Should have more lines than viewport height")

	// Verify we can scroll (have enough content)
	require.True(t, viewer.content.CanScrollDown(), "Viewer should have enough content to scroll")

	// Test arrow down
	handled := viewer.HandleKey(KeyEvent{Key: KeyArrowDown})
	require.True(t, handled)
	require.Equal(t, 1, viewer.GetScrollPosition())

	// Test arrow up
	handled = viewer.HandleKey(KeyEvent{Key: KeyArrowUp})
	require.True(t, handled)
	require.Equal(t, 0, viewer.GetScrollPosition())

	// Test page down
	handled = viewer.HandleKey(KeyEvent{Key: KeyPageDown})
	require.True(t, handled)
	require.Greater(t, viewer.GetScrollPosition(), 0)

	// Test home
	viewer.ScrollTo(20)
	handled = viewer.HandleKey(KeyEvent{Key: KeyHome})
	require.True(t, handled)
	require.Equal(t, 0, viewer.GetScrollPosition())

	// Test end
	handled = viewer.HandleKey(KeyEvent{Key: KeyEnd})
	require.True(t, handled)
	require.Greater(t, viewer.GetScrollPosition(), 0)
}

// Helper function to convert rendered markdown to plain text for testing
func renderToPlainText(rendered *RenderedMarkdown) string {
	var result strings.Builder

	for _, line := range rendered.Lines {
		if line.Indent > 0 {
			result.WriteString(strings.Repeat(" ", line.Indent))
		}

		for _, seg := range line.Segments {
			result.WriteString(seg.Text)
		}

		result.WriteString("\n")
	}

	return result.String()
}
