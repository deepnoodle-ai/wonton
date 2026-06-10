package web

import (
	"fmt"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text needing trimming",
			input:    "  Hello World  ",
			expected: "Hello World",
		},
		{
			name:     "text with HTML entities",
			input:    "Hello &amp; World &lt;test&gt;",
			expected: "Hello & World <test>",
		},
		{
			name:     "text with special quotes",
			input:    `"Hello" 'World'`,
			expected: "\"Hello\" 'World'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   \t\n  ",
			expected: "",
		},
		{
			name:     "text with non-printable chars",
			input:    "Hello\x00World",
			expected: "Hello World",
		},
		{
			name:     "interior whitespace preserved",
			input:    "Hello\tWorld\nAgain",
			expected: "Hello\tWorld\nAgain",
		},
		{
			name:     "non-breaking space becomes regular space",
			input:    "Hello\u00a0World",
			expected: "Hello World",
		},
		{
			name:     "leading entity-encoded space is trimmed",
			input:    "&#32;Hello",
			expected: "Hello",
		},
		{
			name:     "leading entity-encoded nbsp is trimmed",
			input:    "&nbsp;Hello&nbsp;",
			expected: "Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Example demonstrates text normalization for web content.
func ExampleNormalizeText() {
	// Trim whitespace
	fmt.Println(NormalizeText("  Hello  "))

	// Unescape HTML entities
	fmt.Println(NormalizeText("Hello &amp; goodbye"))

	// Convert HTML tags (entities)
	fmt.Println(NormalizeText("&lt;div&gt;"))

	// Remove non-printable characters
	fmt.Println(NormalizeText("Hello\x00World"))

	// Output:
	// Hello
	// Hello & goodbye
	// <div>
	// Hello World
}
