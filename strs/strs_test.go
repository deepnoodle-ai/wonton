package strs

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestFirstNonEmpty(t *testing.T) {
	assert.Equal(t, FirstNonEmpty("", "", "third"), "third")
	assert.Equal(t, FirstNonEmpty("first", "second"), "first")
	assert.Equal(t, FirstNonEmpty(" ", "value"), " ", "whitespace counts as non-empty")
	assert.Equal(t, FirstNonEmpty("", ""), "")
	assert.Equal(t, FirstNonEmpty(), "")
}

func TestFirstNonBlank(t *testing.T) {
	assert.Equal(t, FirstNonBlank("", "  \t\n ", "  value  "), "  value  ")
	assert.Equal(t, FirstNonBlank("  ", ""), "")
	assert.Equal(t, FirstNonBlank(), "")
}

func TestFirstNonBlankTrim(t *testing.T) {
	assert.Equal(t, FirstNonBlankTrim("", "  \t ", "  value  "), "value")
	assert.Equal(t, FirstNonBlankTrim("\n"), "")
}

func TestDedupe(t *testing.T) {
	assert.Equal(t, Dedupe([]string{"b", "a", "b", "c", "a"}), []string{"b", "a", "c"})
	assert.Equal(t, Dedupe([]string{"", "x", ""}), []string{"", "x"}, "empty strings are preserved")
	assert.Nil(t, Dedupe(nil))
	assert.Nil(t, Dedupe([]string{}))
}

func TestDedupeNonBlank(t *testing.T) {
	got := DedupeNonBlank([]string{" a ", "a", "", "  ", "b", "\tb\t"})
	assert.Equal(t, got, []string{"a", "b"}, "values are trimmed before dedup")
	assert.Nil(t, DedupeNonBlank([]string{"", "   "}), "an all-blank input yields nil")
	assert.Nil(t, DedupeNonBlank(nil))
}

func TestDedupeDoesNotAliasInput(t *testing.T) {
	in := []string{"a", "b", "a"}
	out := Dedupe(in)
	out[0] = "mutated"
	assert.Equal(t, in[0], "a")
}
