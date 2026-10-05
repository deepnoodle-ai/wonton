package terminal

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// stripControl removes C0, DEL, and C1 control characters from s, so a value
// written inside an escape sequence cannot end it early or start another.
// Invalid UTF-8 bytes, such as a raw 0x9c (the 8-bit string terminator),
// become U+FFFD.
func stripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// stripTextControl is stripControl for display text: it keeps newlines and
// tabs, which print as layout rather than as commands to the terminal.
func stripTextControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r != '\n' && r != '\t' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// sanitizeLinkID removes the characters an OSC 8 id cannot contain: control
// characters, and ';' and ':', which separate the sequence's parameters.
func sanitizeLinkID(id string) string {
	return strings.Map(func(r rune) rune {
		if r == ';' || r == ':' || unicode.IsControl(r) {
			return -1
		}
		return r
	}, id)
}

// hasControl reports whether s contains a control character or invalid UTF-8.
func hasControl(s string) bool {
	if !utf8.ValidString(s) {
		return true
	}
	return strings.IndexFunc(s, unicode.IsControl) >= 0
}
