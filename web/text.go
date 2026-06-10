package web

import (
	"html"
	"strings"
	"unicode"
)

// NormalizeText applies transformations to clean up text extracted from web
// pages.
//
// The following transformations are applied in order:
//   - Unescape HTML entities (e.g., "&amp;" becomes "&", "&lt;" becomes "<")
//   - Replace non-breaking spaces with regular spaces
//   - Replace non-printable characters with spaces
//   - Trim leading and trailing whitespace
//
// Non-printable characters are any Unicode characters that are not printable
// according to unicode.IsPrint and are not whitespace. These are replaced
// with spaces rather than removed to preserve word boundaries.
//
// Example:
//
//	text := web.NormalizeText("  Hello &amp; goodbye  ")
//	fmt.Println(text) // "Hello & goodbye"
//
//	text = web.NormalizeText("&lt;div&gt;content&lt;/div&gt;")
//	fmt.Println(text) // "<div>content</div>"
func NormalizeText(text string) string {
	text = html.UnescapeString(text)

	var builder strings.Builder
	builder.Grow(len(text))
	for _, r := range text {
		switch {
		case r == '\u00a0': // non-breaking space, common in web text
			builder.WriteRune(' ')
		case unicode.IsPrint(r) || unicode.IsSpace(r):
			builder.WriteRune(r)
		default:
			builder.WriteRune(' ')
		}
	}
	return strings.TrimSpace(builder.String())
}
