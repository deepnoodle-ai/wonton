package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
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
			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Check that all expected strings appear in the output
			output := renderToPlainText(result)
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
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
			assert.NoError(t, err)
			assert.NotNil(t, result)

			output := renderToPlainText(result)
			assert.Contains(t, output, tt.expected)
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
			assert.NoError(t, err)
			assert.NotNil(t, result)

			output := renderToPlainText(result)
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestMarkdownRenderer_CodeBlocks(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Code blocks may contain ANSI codes from syntax highlighting
	// Just check that we have some lines with code content
	assert.Greater(t, len(result.Lines), 0)

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
	assert.True(t, hasFunc, "Expected to find 'func' in code block")
	assert.True(t, hasPrintln, "Expected to find 'Println' in code block")
}

func TestMarkdownRenderer_Links(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := "Check out [this link](https://example.com)"

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify hyperlink was created
	found := false
	for _, line := range result.Lines {
		for _, seg := range line.Segments {
			if seg.Hyperlink != nil {
				assert.Equal(t, "https://example.com", seg.Hyperlink.URL)
				assert.Equal(t, "this link", seg.Hyperlink.Text)
				found = true
			}
		}
	}
	assert.True(t, found, "Expected to find hyperlink in rendered output")
}

func TestMarkdownRenderer_HorizontalRule(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := "Before\n\n---\n\nAfter"

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	output := renderToPlainText(result)
	assert.Contains(t, output, "Before")
	assert.Contains(t, output, "After")
	// Should contain horizontal rule characters
	assert.Contains(t, output, renderer.Theme.HorizontalRuleChar)
}

func TestMarkdownRenderer_Paragraph(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := "This is a paragraph.\n\nThis is another paragraph."

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	output := renderToPlainText(result)
	assert.Contains(t, output, "This is a paragraph.")
	assert.Contains(t, output, "This is another paragraph.")
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
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Lines)

	output := renderToPlainText(result)
	assert.Contains(t, output, "My Document")
	assert.Contains(t, output, "bold")
	assert.Contains(t, output, "italic")
	assert.Contains(t, output, "Features")
	assert.Contains(t, output, "Feature 1")
	// Check for code content in segments (may have ANSI codes)
	hasHello := false
	for _, line := range result.Lines {
		for _, seg := range line.Segments {
			if strings.Contains(seg.Text, "hello") {
				hasHello = true
			}
		}
	}
	assert.True(t, hasHello, "Expected to find 'hello' in code block")
	assert.Contains(t, output, "the docs")
	assert.Contains(t, output, "End of document")
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
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify custom bullet character is used
	output := renderToPlainText(result)
	assert.Contains(t, output, "*") // Custom bullet char
}

func TestMarkdownRenderer_MaxWidth(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(20)

	// Long line that should wrap
	markdown := "This is a very long line of text that should wrap at the maximum width"

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should have multiple lines due to wrapping
	assert.Greater(t, len(result.Lines), 2, "Expected text to wrap into multiple lines")
}

func TestMarkdownView_BasicRendering(t *testing.T) {
	scrollY := 0
	view := Markdown("# Test\n\nParagraph", &scrollY)
	assert.NotNil(t, view)

	// Force rendering
	view.renderContent(80)

	// Check that content was rendered
	assert.NotNil(t, view.rendered)
	assert.Greater(t, len(view.rendered.Lines), 0)

	// Check content is present
	output := renderToPlainText(view.rendered)
	assert.Contains(t, output, "Test")
	assert.Contains(t, output, "Paragraph")
}

func TestMarkdownView_Scrolling(t *testing.T) {
	content := strings.Repeat("Line\n\n", 50) // Create many lines
	scrollY := 0
	view := Markdown(content, &scrollY).Height(10)

	// Force rendering
	view.renderContent(80)

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
}

func TestMarkdownView_ContentRendering(t *testing.T) {
	scrollY := 0
	view := Markdown("Initial content", &scrollY)

	// Render
	view.renderContent(80)
	output := renderToPlainText(view.rendered)
	assert.Contains(t, output, "Initial content")

	// Create new view with updated content
	view2 := Markdown("Updated content", &scrollY)
	view2.renderContent(80)
	output2 := renderToPlainText(view2.rendered)
	assert.Contains(t, output2, "Updated content")
	assert.NotContains(t, output2, "Initial content")
}

func TestMarkdownView_LineCount(t *testing.T) {
	// Create content with many lines
	var contentBuilder strings.Builder
	for i := 0; i < 50; i++ {
		contentBuilder.WriteString(fmt.Sprintf("Line %d\n\n", i))
	}
	content := contentBuilder.String()

	scrollY := 0
	view := Markdown(content, &scrollY)
	view.renderContent(80)

	// Verify we have enough lines
	lineCount := view.GetLineCount()
	assert.Greater(t, lineCount, 10, "Should have more lines than viewport height")
}

func TestMarkdownRenderer_WrapSegments_NoLeadingSpace(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(10)

	// Create a paragraph that will wrap - "hello world foo" at width 10
	// should become:
	// "hello"
	// "world foo"
	markdown := "hello world foo"

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Check that wrapped lines don't start with a space
	for i, line := range result.Lines {
		if len(line.Segments) > 0 {
			firstSeg := line.Segments[0]
			if len(firstSeg.Text) > 0 && firstSeg.Text[0] == ' ' {
				t.Errorf("Line %d starts with a space: %q", i, firstSeg.Text)
			}
		}
	}

	// Verify the content is still correct
	output := renderToPlainText(result)
	assert.Contains(t, output, "hello")
	assert.Contains(t, output, "world")
	assert.Contains(t, output, "foo")
}

func TestMarkdownRenderer_HardLineBreak(t *testing.T) {
	renderer := NewMarkdownRenderer()

	// In markdown, two trailing spaces create a hard line break
	markdown := "Line one  \nLine two"

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should have at least 2 content lines (plus the trailing blank line after paragraph)
	// The hard line break should create a new line
	contentLines := 0
	for _, line := range result.Lines {
		if len(line.Segments) > 0 {
			contentLines++
		}
	}
	assert.GreaterOrEqual(t, contentLines, 2, "Hard line break should create separate lines")

	// Verify both texts are present
	output := renderToPlainText(result)
	assert.Contains(t, output, "Line one")
	assert.Contains(t, output, "Line two")
}

func TestMarkdownRenderer_HardLineBreakWithWrapping(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(40)

	// Test that hard line breaks are preserved even when wrapping is enabled
	markdown := "This is line one  \nThis is line two"

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Should have at least 2 content lines due to the hard break
	contentLines := 0
	for _, line := range result.Lines {
		if len(line.Segments) > 0 {
			contentLines++
		}
	}
	assert.GreaterOrEqual(t, contentLines, 2, "Hard line break should create separate lines even with wrapping")

	output := renderToPlainText(result)
	assert.Contains(t, output, "This is line one")
	assert.Contains(t, output, "This is line two")
}

func TestMarkdownRenderer_WrapPreservesLineCount(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(40)

	// Two separate paragraphs
	markdown := "First paragraph here.\n\nSecond paragraph here."

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)

	// Count lines that have content
	contentLines := 0
	blankLines := 0
	for _, line := range result.Lines {
		if len(line.Segments) > 0 {
			contentLines++
		} else {
			blankLines++
		}
	}

	// Should have 2 content lines and at least 2 blank lines (one after each paragraph)
	assert.Equal(t, 2, contentLines, "Should have 2 content lines for 2 paragraphs")
	assert.GreaterOrEqual(t, blankLines, 2, "Should have blank lines after paragraphs")
}

func TestMarkdownRenderer_SoftLineBreak(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(80)

	// Soft line break (single newline without trailing spaces) should be treated as a space
	// Note: no trailing spaces before newline, so this is NOT a hard break
	markdown := "Line one\nLine two"

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Without hard break, both texts should appear on the same line (joined with space)
	// Count non-blank lines
	contentLines := 0
	for _, line := range result.Lines {
		if len(line.Segments) > 0 {
			contentLines++
		}
	}
	// Should be 1 content line since soft breaks are treated as spaces
	assert.Equal(t, 1, contentLines, "Soft line break should NOT create separate lines")

	// Verify the content is still present
	output := renderToPlainText(result)
	assert.Contains(t, output, "Line one")
	assert.Contains(t, output, "Line two")
}

func TestMarkdownRenderer_PunctuationAfterFormatting(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(80)

	tests := []struct {
		name     string
		markdown string
		expected string // Exact expected text (no extra space before punctuation)
	}{
		{
			name:     "Exclamation after link",
			markdown: "Check out [this link](https://example.com)!",
			expected: "Check out this link!",
		},
		{
			name:     "Exclamation after bold",
			markdown: "This is **bold**!",
			expected: "This is bold!",
		},
		{
			name:     "Exclamation after italic",
			markdown: "This is *italic*!",
			expected: "This is italic!",
		},
		{
			name:     "Exclamation after code",
			markdown: "Run `cmd`!",
			expected: "Run cmd!",
		},
		{
			name:     "Period after link",
			markdown: "See [docs](https://docs.com).",
			expected: "See docs.",
		},
		{
			name:     "Comma after bold",
			markdown: "**Important**, not optional",
			expected: "Important, not optional",
		},
		{
			name:     "Question mark after italic",
			markdown: "Is this *correct*?",
			expected: "Is this correct?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(tt.markdown)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			output := strings.TrimSpace(renderToPlainText(result))
			assert.Equal(t, tt.expected, output)
		})
	}
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
