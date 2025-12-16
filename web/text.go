package web

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"
)

// NormalizeText applies transformations to the given text that are commonly
// helpful for cleaning up text read from a webpage.
// - Trim whitespace
// - Unescape HTML entities
// - Remove non-printable characters
func NormalizeText(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return text
	}
	text = html.UnescapeString(text)
	text = removeNonPrintableChars(text)
	return text
}

func removeNonPrintableChars(input string) string {
	var builder strings.Builder
	for _, r := range input {
		if unicode.IsPrint(r) || unicode.IsSpace(r) {
			builder.WriteRune(r)
		} else {
			builder.WriteRune(' ')
		}
	}
	return builder.String()
}

var punctuation = map[rune]bool{
	'.':  true,
	',':  true,
	':':  true,
	';':  true,
	'?':  true,
	'!':  true,
	'"':  true,
	'\'': true,
}

// EndsWithPunctuation checks if a string ends with a punctuation mark.
func EndsWithPunctuation(s string) bool {
	if len(s) == 0 {
		return false
	}
	// Get the last rune efficiently without converting the entire string
	lastRune, size := utf8.DecodeLastRuneInString(s)
	if size == 0 {
		return false
	}
	return punctuation[lastRune]
}
