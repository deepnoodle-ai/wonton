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

// redactTailSize is how much settled text a Redactor keeps from earlier
// chunks: room for a key and its delimiter ("Authorization: Bearer ").
const redactTailSize = 256

// redactHoldLimit caps how much text a Redactor holds back. A run of text
// longer than this is recorded before it ends.
const redactHoldLimit = 8 << 10

// Redactor redacts secrets in a stream of terminal output or input, one chunk
// at a time. Redacting each chunk alone misses a secret that two writes or
// reads split: a program that prints "Token: " and then the value records the
// value with no key in front of it, and the first part of a token such as
// "AKIAIOSFODNN" matches no pattern until the rest arrives.
//
// So a Redactor matches each chunk together with the settled text before it
// (the tail), and holds back the run of text at the end of the chunk, since
// the next chunk can continue it. A secret is always a run of text with no
// whitespace, so text that ends in whitespace or a break is settled. The held
// text is returned with the next chunk, or by Flush.
//
// Matching runs on the text, with escape sequences taken out, so a style change
// between a key and its value does not hide the value, and a redaction never
// removes an escape sequence. Any other escape sequence, such as a cursor move,
// separates the text on either side like a line break: a key still matches a
// value drawn after a cursor move, but a value does not run on into the next
// text drawn elsewhere. The payload of a string sequence, such as an OSC 8
// hyperlink's URL, is redacted on its own.
//
// Use one Redactor per stream, such as one for output and one for input. The
// zero value is ready to use. A Redactor is not safe for concurrent use.
type Redactor struct {
	tail string // settled text before held, before redaction
	held string // the end of the stream, which the next chunk can continue
}

// Redact returns the stream up to the end of data, with each secret replaced
// by [REDACTED], less the text it holds back. The result can be empty.
func (r *Redactor) Redact(data string) string {
	return r.settle(r.held+data, false)
}

// Flush returns the text r holds back, redacted, and resets r, so the next
// call to Redact starts a new stream. Call it when nothing more will continue
// the stream, such as when recording stops or pauses, or the held text is
// lost.
func (r *Redactor) Flush() string {
	out := r.settle(r.held, true)
	*r = Redactor{}
	return out
}

// settle redacts chunk, the held text and the new data. Unless final, it holds
// back the run of text at the end of chunk, and an escape sequence that the
// next chunk finishes.
func (r *Redactor) settle(chunk string, final bool) string {
	// Split chunk into text and escape sequences, and join the text.
	type part struct {
		s   string
		raw int // offset of s in chunk
		at  int // offset of the text in text, or -1 for an escape sequence
	}
	var parts []part
	var text strings.Builder
	end := len(chunk) // the start of an unfinished escape sequence to hold
	for i := 0; i < len(chunk); {
		if chunk[i] == '\x1b' {
			n, done := escapeLen(chunk[i:])
			if !done && !final && len(chunk)-i <= redactHoldLimit {
				end = i
				break
			}
			parts = append(parts, part{chunk[i : i+n], i, -1})
			if breaksText(chunk[i : i+n]) {
				text.WriteByte('\n')
			}
			i += n
			continue
		}
		n := strings.IndexByte(chunk[i:], '\x1b')
		if n < 0 {
			n = len(chunk) - i
		}
		parts = append(parts, part{chunk[i : i+n], i, text.Len()})
		text.WriteString(chunk[i : i+n])
		i += n
	}

	// Find the run of text at the end, which the next write can continue.
	// The whitespace is what \s matches in the patterns.
	joined := r.tail + text.String()
	from := len(r.tail)
	holdAt, hold := text.Len(), end
	if !final {
		run := strings.LastIndexAny(joined, " \t\n\f\r") + 1
		run = max(run, from)
		if run < len(joined) {
			for _, p := range parts {
				if p.at >= 0 && p.at+len(p.s) > run-from {
					holdAt, hold = run-from, p.raw+run-from-p.at
					break
				}
			}
			if len(chunk)-hold > redactHoldLimit {
				holdAt, hold = text.Len(), end
			}
		}
	}

	spans := secretSpans(joined, from)
	var b strings.Builder
	for _, p := range parts {
		if p.raw >= hold {
			break
		}
		if p.at < 0 {
			b.WriteString(redactEscape(p.s))
			continue
		}
		writeRedacted(&b, p.s[:min(len(p.s), hold-p.raw)], p.at, spans)
	}
	r.tail = lastBytes(joined[:from+holdAt], redactTailSize)
	r.held = chunk[hold:]
	return b.String()
}

// span is a byte range [start, end).
type span struct{ start, end int }

// secretSpans returns the byte ranges of the secrets in s that end after from,
// sorted, merged, and relative to from. A secret that starts before from has a
// negative start. A pattern's capture group is the key and delimiter in
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
			spans = append(spans, span{start - from, end - from})
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
// which begins with ESC, and whether s holds all of it. An unterminated
// sequence runs to the end of s.
func escapeLen(s string) (n int, done bool) {
	if len(s) < 2 {
		return len(s), false
	}
	switch c := s[1]; {
	case c == '[':
		// CSI: parameter and intermediate bytes, then a final byte.
		for i := 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1, true
			}
			if s[i] == '\x1b' {
				return i, true // an ESC cancels the CSI and starts anew
			}
		}
		return len(s), false
	case isStringSequence(c):
		// The payload ends with BEL or ST (ESC \). Any other ESC ends it too.
		for i := 2; i < len(s); i++ {
			if s[i] == '\a' {
				return i + 1, true
			}
			if s[i] == '\x1b' {
				if i+1 == len(s) {
					return len(s), false // the ESC can start ST
				}
				if s[i+1] == '\\' {
					return i + 2, true
				}
				return i, true
			}
		}
		return len(s), false
	default:
		// Intermediate bytes, then a final byte.
		i := 1
		for i < len(s) && s[i] >= 0x20 && s[i] <= 0x2f {
			i++
		}
		if i == len(s) {
			return len(s), false
		}
		if s[i] < 0x30 || s[i] > 0x7e {
			// Not a final byte, such as a second ESC: the sequence ends
			// before it, and the byte is read on its own.
			return i, true
		}
		return i + 1, true
	}
}

// breaksText reports whether the escape sequence seq separates the text on
// either side. Only SGR sequences, which set the style, and string sequences
// do not. The rest move the cursor or change the screen or its modes, and no
// key and value are written with one of them in between.
func breaksText(seq string) bool {
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
	var r Redactor
	return r.Redact(data) + r.Flush()
}

// RedactCredentials returns input with each secret replaced by [REDACTED].
// It redacts input on its own; to redact a stream that arrives in chunks, use
// a Redactor.
func RedactCredentials(input string) string {
	return redactSecretPatterns(input)
}
