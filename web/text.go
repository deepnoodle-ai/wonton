package web

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"
)

// NormalizeText applies transformations to clean up text extracted from web pages.
//
// The following transformations are applied in order:
//   - Trim leading and trailing whitespace
//   - Unescape HTML entities (e.g., "&amp;" becomes "&", "&lt;" becomes "<")
//   - Replace non-printable characters with spaces
//
// Non-printable characters are any Unicode characters that are not printable
// according to unicode.IsPrint() and are not whitespace. These are replaced
// with spaces rather than removed to preserve word boundaries.
//
// Returns the original text unchanged if it's empty after trimming.
//
// Example:
//
//	text := web.NormalizeText("  Hello &amp; goodbye  ")
//	fmt.Println(text) // "Hello & goodbye"
//
//	text = web.NormalizeText("&lt;div&gt;content&lt;/div&gt;")
//	fmt.Println(text) // "<div>content</div>"
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

// EndsWithPunctuation checks if a string ends with a common punctuation mark.
//
// The following punctuation characters are recognized: . , : ; ? ! " '
//
// This function correctly handles Unicode strings by checking the last rune
// rather than the last byte. Returns false for empty strings.
//
// Example:
//
//	web.EndsWithPunctuation("Hello.")  // true
//	web.EndsWithPunctuation("Hello?")  // true
//	web.EndsWithPunctuation("Hello")   // false
//	web.EndsWithPunctuation("")        // false
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
