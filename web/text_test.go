package web

import (
	"fmt"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestRemoveNonPrintableChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text with tabs and newlines",
			input:    "Hello\tWorld\n",
			expected: "Hello\tWorld\n",
		},
		{
			name:     "text with non-printable chars",
			input:    "Hello\x00\x01World",
			expected: "Hello  World",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only non-printable chars",
			input:    "\x00\x01\x02",
			expected: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeNonPrintableChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEndsWithPunctuation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "ends with period",
			input:    "Hello.",
			expected: true,
		},
		{
			name:     "ends with comma",
			input:    "Hello,",
			expected: true,
		},
		{
			name:     "ends with question mark",
			input:    "Hello?",
			expected: true,
		},
		{
			name:     "ends with exclamation",
			input:    "Hello!",
			expected: true,
		},
		{
			name:     "ends with quote",
			input:    "Hello\"",
			expected: true,
		},
		{
			name:     "ends with apostrophe",
			input:    "Hello'",
			expected: true,
		},
		{
			name:     "ends with letter",
			input:    "Hello",
			expected: false,
		},
		{
			name:     "ends with number",
			input:    "Hello123",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "single punctuation",
			input:    ".",
			expected: true,
		},
		{
			name:     "unicode characters",
			input:    "Hello世界.",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EndsWithPunctuation(tt.input)
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

// Example demonstrates checking for punctuation at the end of strings.
func ExampleEndsWithPunctuation() {
	fmt.Println(EndsWithPunctuation("Hello."))
	fmt.Println(EndsWithPunctuation("Hello?"))
	fmt.Println(EndsWithPunctuation("Hello"))
	fmt.Println(EndsWithPunctuation(""))

	// Output:
	// true
	// true
	// false
	// false
}
