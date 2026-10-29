package terminal

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// Common patterns for secrets that should be redacted. When a pattern has a
// capture group, the group is the key and delimiter that precede the secret
// (such as "password=" or "Bearer "). It is kept for context and only the rest
// of the match is redacted.
var secretPatterns = []*regexp.Regexp{
	// Password fields
	regexp.MustCompile(`(?i)(password[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(passwd[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(pwd[:\s=]+)\S+`),

	// API keys and tokens
	regexp.MustCompile(`(?i)(api[_-]?key[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(api[_-]?secret[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(token[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(access[_-]?token[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(auth[_-]?token[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(bearer\s+)\S+`),

	// Common secret environment variables
	regexp.MustCompile(`(?i)(secret[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(private[_-]?key[:\s=]+)\S+`),

	// Long hexadecimal strings (likely tokens/keys)
	regexp.MustCompile(`\b[a-fA-F0-9]{32,}\b`),

	// JWT tokens (xxx.yyy.zzz format)
	regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\b`),

	// AWS keys
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),

	// GitHub tokens
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{36,}`),

	// Generic secret-like strings (base64-ish)
	regexp.MustCompile(`(?i)((?:secret|key|token|password)[:\s=]+)[a-zA-Z0-9+/]{20,}={0,2}`),
}

// redactTailSize is how much text a redactor keeps from earlier events: room
// for a key and its delimiter ("Authorization: Bearer ") and the start of a
// secret that continues into the next event.
const redactTailSize = 256

// redactor redacts secrets in a stream of terminal output or input, one event
// at a time. Redacting each event alone misses a value whose key an earlier
// event wrote: a program that prints "Token: " and then the value in a second
// write records the value with no key in front of it. A redactor matches each
// event together with the end of the text before it.
//
// Matching runs on the text, with escape sequences taken out, so a style change
// between a key and its value does not hide the value, and a redaction never
// removes an escape sequence. A sequence that can move the cursor separates the
// text on either side like a line break: a key still matches a value drawn
// after a cursor move, but a value does not run on into the next text drawn
// elsewhere. The payload of a string sequence, such as an OSC 8 hyperlink's
// URL, is redacted on its own.
type redactor struct {
	tail string // text at the end of earlier events, before redaction
}

// redact returns data with each secret replaced by [REDACTED].
func (r *redactor) redact(data string) string {
	// Split data into text and escape sequences, and join the text.
	type part struct {
		s  string
		at int // offset of the text in text, or -1 for an escape sequence
	}
	var parts []part
	var text strings.Builder
	for i := 0; i < len(data); {
		if data[i] == '\x1b' {
			n := escapeLen(data[i:])
			parts = append(parts, part{data[i : i+n], -1})
			if movesCursor(data[i : i+n]) {
				text.WriteByte('\n')
			}
			i += n
			continue
		}
		n := strings.IndexByte(data[i:], '\x1b')
		if n < 0 {
			n = len(data) - i
		}
		parts = append(parts, part{data[i : i+n], text.Len()})
		text.WriteString(data[i : i+n])
		i += n
	}

	joined := r.tail + text.String()
	spans := secretSpans(joined, len(r.tail))
	r.tail = lastBytes(joined, redactTailSize)

	var b strings.Builder
	for _, p := range parts {
		if p.at < 0 {
			b.WriteString(redactEscape(p.s))
		} else {
			writeRedacted(&b, p.s, p.at, spans)
		}
	}
	return b.String()
}

// span is a byte range [start, end).
type span struct{ start, end int }

// secretSpans returns the byte ranges of the secrets in s that end after from,
// sorted, merged, and relative to from. A secret that starts before from is
// cut to start at from. A pattern's capture group is the key and delimiter in
// front of the secret, which are kept. The key is found by the capture group,
// not by splitting the match on ':' or '=': the secret itself can contain
// those characters (base64 padding, URLs with credentials), and splitting
// there would leak part of it.
func secretSpans(s string, from int) []span {
	var spans []span
	for _, pattern := range secretPatterns {
		for _, m := range pattern.FindAllStringSubmatchIndex(s, -1) {
			start, end := m[0], m[1]
			if len(m) > 2 && m[3] >= 0 {
				start = m[3]
			}
			if end <= from || start >= end {
				continue
			}
			spans = append(spans, span{max(start, from) - from, end - from})
		}
	}
	slices.SortFunc(spans, func(a, b span) int { return a.start - b.start })
	merged := spans[:0]
	for _, sp := range spans {
		if n := len(merged); n > 0 && sp.start <= merged[n-1].end {
			merged[n-1].end = max(merged[n-1].end, sp.end)
			continue
		}
		merged = append(merged, sp)
	}
	return merged
}

// writeRedacted writes s, the text at offset at, with the parts that spans
// cover replaced. A span that started in earlier text was marked there, so its
// rest is dropped.
func writeRedacted(b *strings.Builder, s string, at int, spans []span) {
	pos := 0
	for _, sp := range spans {
		start, end := sp.start-at, sp.end-at
		if end <= pos || start >= len(s) {
			continue
		}
		if start >= pos {
			b.WriteString(s[pos:start])
			b.WriteString("[REDACTED]")
		}
		pos = min(end, len(s))
	}
	b.WriteString(s[pos:])
}

// isStringSequence reports whether ESC c starts a sequence that carries a
// string payload: OSC, DCS, SOS, PM, or APC.
func isStringSequence(c byte) bool {
	return c == ']' || c == 'P' || c == 'X' || c == '^' || c == '_'
}

// escapeLen returns the length of the escape sequence at the start of s,
// which begins with ESC. An unterminated sequence runs to the end of s.
func escapeLen(s string) int {
	if len(s) < 2 {
		return len(s)
	}
	switch c := s[1]; {
	case c == '[':
		// CSI: parameter and intermediate bytes, then a final byte.
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1
			}
		}
		return len(s)
	case isStringSequence(c):
		// The payload ends with BEL or ST (ESC \). Any other ESC ends it too.
		for i := 2; i < len(s); i++ {
			if s[i] == '\a' {
				return i + 1
			}
			if s[i] == '\x1b' {
				if i+1 < len(s) && s[i+1] == '\\' {
					return i + 2
				}
				return i
			}
		}
		return len(s)
	default:
		// Intermediate bytes, then a final byte.
		i := 1
		for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2f {
			i++
		}
		return min(i+1, len(s))
	}
}

// movesCursor reports whether the escape sequence seq can move the cursor. SGR
// sequences, which set the style, and string sequences do not.
func movesCursor(seq string) bool {
	if len(seq) < 2 || isStringSequence(seq[1]) {
		return false
	}
	return seq[1] != '[' || seq[len(seq)-1] != 'm'
}

// redactEscape redacts the payload of a string sequence, which can carry text
// such as a hyperlink's URL. Other escape sequences are returned unchanged.
func redactEscape(seq string) string {
	if len(seq) < 3 || !isStringSequence(seq[1]) {
		return seq
	}
	end := len(seq)
	if strings.HasSuffix(seq, "\a") {
		end--
	} else if strings.HasSuffix(seq[2:], "\x1b\\") {
		end -= 2
	}
	payload := seq[2:end]
	var b strings.Builder
	b.WriteString(seq[:2])
	writeRedacted(&b, payload, 0, secretSpans(payload, 0))
	b.WriteString(seq[end:])
	return b.String()
}

// lastBytes returns at most the last n bytes of s, starting on a rune boundary.
func lastBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[len(s)-n:]
	for len(s) > 0 && !utf8.RuneStart(s[0]) {
		s = s[1:]
	}
	return s
}

// redactSecretPatterns redacts the secrets in data on its own, with no text
// from earlier events.
func redactSecretPatterns(data string) string {
	var r redactor
	return r.redact(data)
}

// RedactCredentials can be called manually to test redaction
func RedactCredentials(input string) string {
	return redactSecretPatterns(input)
}
