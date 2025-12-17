package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
		match   bool
	}{
		{"empty pattern", "", "hello", true},
		{"empty text", "a", "", false},
		{"both empty", "", "", true},
		{"exact match", "hello", "hello", true},
		{"case insensitive", "HELLO", "hello", true},
		{"subsequence match", "hlo", "hello", true},
		{"subsequence with gaps", "hw", "hello world", true},
		{"beginning match", "hel", "hello", true},
		{"end match", "rld", "world", true},
		{"no match", "xyz", "hello", false},
		{"pattern longer than text", "helloworld", "hello", false},
		{"mixed case pattern", "HeLLo", "hello", true},
		{"pattern with space", "h w", "hello world", true},
		{"partial subsequence", "hwx", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FuzzyMatch(tt.pattern, tt.text)
			assert.Equal(t, tt.match, result)
		})
	}
}

func TestFuzzySearch(t *testing.T) {
	candidates := []string{
		"hello",
		"world",
		"hello world",
		"help",
		"helicopter",
		"howdy",
	}

	t.Run("empty pattern returns all", func(t *testing.T) {
		results := FuzzySearch("", candidates)
		assert.Len(t, results, len(candidates))
	})

	t.Run("filters matches", func(t *testing.T) {
		results := FuzzySearch("hel", candidates)
		assert.Contains(t, results, "hello")
		assert.Contains(t, results, "help")
		assert.Contains(t, results, "helicopter")
		assert.Contains(t, results, "hello world")
		assert.NotContains(t, results, "world")
		assert.NotContains(t, results, "howdy")
	})

	t.Run("sorted by length", func(t *testing.T) {
		results := FuzzySearch("hel", candidates)
		// Shorter matches should come first
		for i := 0; i < len(results)-1; i++ {
			assert.True(t, len(results[i]) <= len(results[i+1]))
		}
	})

	t.Run("no matches", func(t *testing.T) {
		results := FuzzySearch("xyz", candidates)
		assert.Empty(t, results)
	})

	t.Run("case insensitive", func(t *testing.T) {
		results := FuzzySearch("HELLO", candidates)
		assert.Contains(t, results, "hello")
		assert.Contains(t, results, "hello world")
	})
}
