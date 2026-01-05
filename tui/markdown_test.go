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

	// Should have 2 content lines and 1 blank line between them (no trailing blank)
	assert.Equal(t, 2, contentLines, "Should have 2 content lines for 2 paragraphs")
	assert.Equal(t, 1, blankLines, "Should have 1 blank line between paragraphs")
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

	// Verify that soft break becomes a space (not concatenated without space)
	assert.Contains(t, output, "one Line", "Soft line break should become a space")
	assert.NotContains(t, output, "oneLine", "Text should not be concatenated without space")
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

func TestMarkdownRenderer_SnakeCase(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(80)

	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "snake_case word",
			markdown: "The function read_file reads files",
			expected: "The function read_file reads files",
		},
		{
			name:     "multiple underscores",
			markdown: "Use the get_user_data function",
			expected: "Use the get_user_data function",
		},
		{
			name:     "snake_case at start",
			markdown: "read_file is a function",
			expected: "read_file is a function",
		},
		{
			name:     "snake_case at end",
			markdown: "Call the read_file",
			expected: "Call the read_file",
		},
		{
			name:     "underscore in list",
			markdown: "• read_file - reads files",
			expected: "• read_file - reads files",
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

func TestMarkdownRenderer_SimpleTable(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := `| Name | Status |
|------|--------|
| Alice | Active |
| Bob | Pending |`

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	output := renderToPlainText(result)

	// Check that table content is present
	assert.Contains(t, output, "Name")
	assert.Contains(t, output, "Status")
	assert.Contains(t, output, "Alice")
	assert.Contains(t, output, "Active")
	assert.Contains(t, output, "Bob")
	assert.Contains(t, output, "Pending")

	// Check for box-drawing characters
	assert.Contains(t, output, "┌")
	assert.Contains(t, output, "┐")
	assert.Contains(t, output, "└")
	assert.Contains(t, output, "┘")
	assert.Contains(t, output, "│")
	assert.Contains(t, output, "─")
}

func TestMarkdownRenderer_TableAlignment(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := `| Left | Center | Right |
|:-----|:------:|------:|
| L | C | R |`

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	output := renderToPlainText(result)
	assert.Contains(t, output, "Left")
	assert.Contains(t, output, "Center")
	assert.Contains(t, output, "Right")
}

func TestMarkdownRenderer_TableWithFormatting(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := `| Feature | Description |
|---------|-------------|
| **Bold** | Some text |
| *Italic* | More text |
| ` + "`code`" + ` | Code text |`

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	output := renderToPlainText(result)
	assert.Contains(t, output, "Bold")
	assert.Contains(t, output, "Italic")
	assert.Contains(t, output, "code")
}

func TestMarkdownRenderer_TableWithOtherContent(t *testing.T) {
	renderer := NewMarkdownRenderer()

	markdown := `# Heading

Some paragraph text.

| Col1 | Col2 |
|------|------|
| A | B |

More text after the table.`

	result, err := renderer.Render(markdown)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	output := renderToPlainText(result)
	assert.Contains(t, output, "Heading")
	assert.Contains(t, output, "Some paragraph text")
	assert.Contains(t, output, "Col1")
	assert.Contains(t, output, "Col2")
	assert.Contains(t, output, "More text after the table")
}

func TestMarkdownRenderer_LinksWithUnderscores(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(80)

	tests := []struct {
		name     string
		markdown string
		expected string
		linkText string
		linkURL  string
	}{
		{
			name:     "link text with underscore",
			markdown: "Check [my_link](https://example.com)",
			expected: "Check my_link",
			linkText: "my_link",
			linkURL:  "https://example.com",
		},
		{
			name:     "link text with multiple underscores",
			markdown: "See [get_user_data](https://api.com/get_user_data)",
			expected: "See get_user_data",
			linkText: "get_user_data",
			linkURL:  "https://api.com/get_user_data",
		},
		{
			name:     "URL with underscores",
			markdown: "Check [docs](https://example.com/path_to_file)",
			expected: "Check docs",
			linkText: "docs",
			linkURL:  "https://example.com/path_to_file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(tt.markdown)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			output := strings.TrimSpace(renderToPlainText(result))
			assert.Equal(t, tt.expected, output)

			// Verify hyperlink was created correctly
			found := false
			for _, line := range result.Lines {
				for _, seg := range line.Segments {
					if seg.Hyperlink != nil {
						assert.Equal(t, tt.linkText, seg.Hyperlink.Text)
						assert.Equal(t, tt.linkURL, seg.Hyperlink.URL)
						found = true
					}
				}
			}
			assert.True(t, found, "Expected to find hyperlink")
		})
	}
}

func TestMarkdownRenderer_InlineCodeWithParentheses(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(80)

	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "function call in code",
			markdown: "Use `foo(bar)` to call",
			expected: "Use foo(bar) to call",
		},
		{
			name:     "function with args in code",
			markdown: "Call `doSomething(arg1, arg2)` here",
			expected: "Call doSomething(arg1, arg2) here",
		},
		{
			name:     "multiple parens in code",
			markdown: "Run `fn(a)(b)` now",
			expected: "Run fn(a)(b) now",
		},
		{
			name:     "method call in code",
			markdown: "Use `obj.method(x)` for this",
			expected: "Use obj.method(x) for this",
		},
		{
			name:     "nested parens in code",
			markdown: "Try `foo(bar(baz))` here",
			expected: "Try foo(bar(baz)) here",
		},
		{
			name:     "parens in regular text",
			markdown: "This is (parenthetical) text",
			expected: "This is (parenthetical) text",
		},
		{
			name:     "function before code",
			markdown: "The `foo()` function",
			expected: "The foo() function",
		},
		{
			name:     "bold with parens",
			markdown: "This is **foo(bar)** text",
			expected: "This is foo(bar) text",
		},
		{
			name:     "italic with parens",
			markdown: "This is *foo(bar)* text",
			expected: "This is foo(bar) text",
		},
		{
			name:     "link with parens in text",
			markdown: "Check [foo(bar)](https://example.com)",
			expected: "Check foo(bar)",
		},
		{
			name:     "parenthetical after code",
			markdown: "`cmd` (optional)",
			expected: "cmd (optional)",
		},
		{
			name:     "code after open paren",
			markdown: "Call (`foo`)",
			expected: "Call (foo)",
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

func TestMarkdownRenderer_LinkDisplaySpacing(t *testing.T) {
	renderer := NewMarkdownRenderer()
	renderer.WithMaxWidth(80)

	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "link at start of text",
			markdown: "[Example](https://example.com) is a site",
			expected: "Example is a site",
		},
		{
			name:     "link at end of text",
			markdown: "Visit [Example](https://example.com)",
			expected: "Visit Example",
		},
		{
			name:     "link in middle of text",
			markdown: "Check out [Example](https://example.com) for more",
			expected: "Check out Example for more",
		},
		{
			name:     "multiple links",
			markdown: "See [one](url1) and [two](url2) here",
			expected: "See one and two here",
		},
		{
			name:     "link with punctuation after",
			markdown: "Read [the docs](https://docs.com).",
			expected: "Read the docs.",
		},
		{
			name:     "link with punctuation before",
			markdown: "Is this [correct](url)?",
			expected: "Is this correct?",
		},
		{
			name:     "link after colon",
			markdown: "See: [link](url)",
			expected: "See: link",
		},
		{
			name:     "link in parens",
			markdown: "Check ([link](url)) here",
			expected: "Check (link) here",
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

func TestMarkdownRenderer_NoTrailingEmptyLines(t *testing.T) {
	renderer := NewMarkdownRenderer()

	tests := []struct {
		name     string
		markdown string
	}{
		{
			name:     "Single paragraph",
			markdown: "This is a simple paragraph.",
		},
		{
			name:     "Multiple paragraphs",
			markdown: "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:     "Heading",
			markdown: "# Heading",
		},
		{
			name:     "Heading with paragraph",
			markdown: "# Heading\n\nSome text after.",
		},
		{
			name:     "Unordered list",
			markdown: "- Item 1\n- Item 2\n- Item 3",
		},
		{
			name:     "Ordered list",
			markdown: "1. First\n2. Second\n3. Third",
		},
		{
			name:     "Code block",
			markdown: "```\ncode here\n```",
		},
		{
			name:     "Blockquote",
			markdown: "> This is a quote",
		},
		{
			name:     "Mixed content",
			markdown: "# Title\n\nSome text.\n\n- List item\n\n```\ncode\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(tt.markdown)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, len(result.Lines) > 0, "Expected at least one line")

			// Last line should not be empty
			lastLine := result.Lines[len(result.Lines)-1]
			assert.True(t, len(lastLine.Segments) > 0,
				"Last line should not be empty (have segments), got %d lines total", len(result.Lines))
		})
	}
}

func TestIsBlankLine(t *testing.T) {
	tests := []struct {
		name     string
		line     StyledLine
		expected bool
	}{
		{
			name:     "nil segments",
			line:     StyledLine{Segments: nil},
			expected: true,
		},
		{
			name:     "empty segments slice",
			line:     StyledLine{Segments: []StyledSegment{}},
			expected: true,
		},
		{
			name:     "single space segment",
			line:     StyledLine{Segments: []StyledSegment{{Text: " "}}},
			expected: true,
		},
		{
			name:     "multiple whitespace segments",
			line:     StyledLine{Segments: []StyledSegment{{Text: " "}, {Text: "  "}, {Text: "\t"}}},
			expected: true,
		},
		{
			name:     "empty string segment",
			line:     StyledLine{Segments: []StyledSegment{{Text: ""}}},
			expected: true,
		},
		{
			name:     "content segment",
			line:     StyledLine{Segments: []StyledSegment{{Text: "hello"}}},
			expected: false,
		},
		{
			name:     "content with surrounding whitespace",
			line:     StyledLine{Segments: []StyledSegment{{Text: " hello "}}},
			expected: false,
		},
		{
			name:     "mixed whitespace and content",
			line:     StyledLine{Segments: []StyledSegment{{Text: " "}, {Text: "world"}}},
			expected: false,
		},
		{
			name:     "line with only indent (no segments)",
			line:     StyledLine{Indent: 4, Segments: nil},
			expected: true,
		},
		{
			name:     "line with indent and content",
			line:     StyledLine{Indent: 4, Segments: []StyledSegment{{Text: "indented"}}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBlankLine(tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMarkdownRenderer_TrailingBlankLineTrimming(t *testing.T) {
	renderer := NewMarkdownRenderer()

	t.Run("paragraph followed by list has no trailing blank", func(t *testing.T) {
		// This is a common pattern that was causing extra blank lines
		markdown := "Here is the content:\n\n- Item 1\n- Item 2\n- Item 3"

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify last line has content (not blank)
		lastLine := result.Lines[len(result.Lines)-1]
		assert.False(t, isBlankLine(lastLine), "Last line should not be blank")

		// Verify internal blank line exists (between paragraph and list)
		hasInternalBlank := false
		for i := 0; i < len(result.Lines)-1; i++ {
			if isBlankLine(result.Lines[i]) {
				hasInternalBlank = true
				break
			}
		}
		assert.True(t, hasInternalBlank, "Should have internal blank line for paragraph spacing")
	})

	t.Run("list followed by paragraph has no trailing blank", func(t *testing.T) {
		markdown := "- Item 1\n- Item 2\n\nFollowing paragraph."

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		lastLine := result.Lines[len(result.Lines)-1]
		assert.False(t, isBlankLine(lastLine), "Last line should not be blank")
	})

	t.Run("multiple paragraphs preserve internal blanks but trim trailing", func(t *testing.T) {
		markdown := "First paragraph.\n\nSecond paragraph.\n\nThird paragraph."

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Count blank lines
		blankCount := 0
		contentCount := 0
		for _, line := range result.Lines {
			if isBlankLine(line) {
				blankCount++
			} else {
				contentCount++
			}
		}

		// Should have 3 content lines and 2 internal blank lines
		assert.Equal(t, 3, contentCount, "Should have 3 content lines")
		assert.Equal(t, 2, blankCount, "Should have 2 internal blank lines (between paragraphs)")

		// Last line should not be blank
		lastLine := result.Lines[len(result.Lines)-1]
		assert.False(t, isBlankLine(lastLine), "Last line should not be blank")
	})

	t.Run("complex document with list ending has no trailing blank", func(t *testing.T) {
		// Simulates typical AI assistant output ending with a list
		markdown := `Here is what I found:

The file contains:

- Configuration settings
- API endpoints
- Database schemas

Would you like more details?`

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		lastLine := result.Lines[len(result.Lines)-1]
		assert.False(t, isBlankLine(lastLine), "Last line should not be blank")

		// Last line should contain the question
		output := renderToPlainText(result)
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
		lastTextLine := lines[len(lines)-1]
		assert.Contains(t, lastTextLine, "Would you like more details")
	})

	t.Run("code block at end has no trailing blank", func(t *testing.T) {
		markdown := "Here is some code:\n\n```go\nfunc main() {}\n```"

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		lastLine := result.Lines[len(result.Lines)-1]
		assert.False(t, isBlankLine(lastLine), "Last line should not be blank")
	})

	t.Run("blockquote at end has no trailing blank", func(t *testing.T) {
		markdown := "Someone said:\n\n> This is a famous quote"

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		lastLine := result.Lines[len(result.Lines)-1]
		assert.False(t, isBlankLine(lastLine), "Last line should not be blank")
	})

	t.Run("horizontal rule at end has no trailing blank", func(t *testing.T) {
		markdown := "Some content\n\n---"

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		lastLine := result.Lines[len(result.Lines)-1]
		assert.False(t, isBlankLine(lastLine), "Last line should not be blank")
	})
}

func TestMarkdownRenderer_InternalSpacingPreserved(t *testing.T) {
	renderer := NewMarkdownRenderer()

	t.Run("blank line between paragraphs is preserved", func(t *testing.T) {
		markdown := "First paragraph.\n\nSecond paragraph."

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)

		// Should have exactly: content, blank, content (3 lines)
		// The blank line between paragraphs should be preserved
		assert.Equal(t, 3, len(result.Lines), "Should have 3 lines: para, blank, para")

		assert.False(t, isBlankLine(result.Lines[0]), "First line should have content")
		assert.True(t, isBlankLine(result.Lines[1]), "Second line should be blank (spacing)")
		assert.False(t, isBlankLine(result.Lines[2]), "Third line should have content")
	})

	t.Run("blank line between paragraph and list is preserved", func(t *testing.T) {
		markdown := "Introduction:\n\n- Item 1\n- Item 2"

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)

		// Find the blank line
		foundBlank := false
		for i := 0; i < len(result.Lines)-1; i++ {
			if isBlankLine(result.Lines[i]) {
				foundBlank = true
				// Verify next line has content (list item)
				assert.False(t, isBlankLine(result.Lines[i+1]),
					"Line after internal blank should have content")
				break
			}
		}
		assert.True(t, foundBlank, "Should have blank line between paragraph and list")
	})

	t.Run("blank line between list and paragraph is preserved", func(t *testing.T) {
		markdown := "- Item 1\n- Item 2\n\nConclusion."

		result, err := renderer.Render(markdown)
		assert.NoError(t, err)

		// Count blanks - should be exactly 1 (between list and conclusion)
		blankCount := 0
		for _, line := range result.Lines {
			if isBlankLine(line) {
				blankCount++
			}
		}
		assert.Equal(t, 1, blankCount, "Should have exactly 1 blank line between list and paragraph")
	})
}
