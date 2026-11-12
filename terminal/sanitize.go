package terminal

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// stripAllControl removes C0, DEL, and C1 control characters from s, so a value
// written inside an escape sequence cannot end it early or start another.
// Invalid UTF-8 bytes, such as a raw 0x9c (the 8-bit string terminator),
// become U+FFFD.
func stripAllControl(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// StripControl removes control characters from text that will be written to
// the terminal, so the text cannot carry escape sequences: C0 controls
// (including ESC and BEL), DEL, and C1 controls are dropped, and invalid
// UTF-8 bytes become U+FFFD. Newlines and tabs are kept, since they print as
// layout rather than as commands.
//
// Use it on untrusted text, such as model output or text from a web page,
// before passing it to [Format], [FormatWithID], or [Fallback], which write
// their text as given.
func StripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r != '\n' && r != '\t' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// cellRune returns r, or a space if r is a control character. A cell is
// drawn, so a control rune in it would reach the terminal as a command.
func cellRune(r rune) rune {
	if unicode.IsControl(r) {
		return ' '
	}
	return r
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
