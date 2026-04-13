// Package runewidth provides functions for determining the display width of
// Unicode characters and strings in terminal emulators, plus Unicode Standard
// Annex #29 (UAX#29) grapheme cluster segmentation.
//
// It correctly handles East Asian wide characters, emoji (including ZWJ
// sequences, skin tones, and flags), combining marks, and other zero-width
// characters. The grapheme segmenter implements UAX#29 in full and is
// conformant against the Unicode GraphemeBreakTest at the version given by
// [UnicodeVersion].
//
// The ASCII fast path in [StringWidth] returns len(s) for pure-ASCII strings
// with zero allocations and no table lookups. [StringWidth], [FitLeft], and
// [FitRight] allocate zero bytes on any input (ASCII or Unicode), and
// [Graphemes] is an allocation-free iterator. [Truncate] and [Fit] may
// allocate one string when they must append the tail to a truncated input.
// [WidthIndex] allocates one []int the length of its input.
package runewidth

// RuneWidth returns the number of terminal cells needed to display rune r.
//
// Returns 0 for non-printable and combining characters, 1 for most characters,
// 2 for wide characters (CJK ideographs, fullwidth forms, regional indicators,
// most emoji), and 3 or 4 for the two-em and three-em dashes.
//
// This function operates on individual runes. For strings that may contain
// multi-rune grapheme clusters (emoji sequences, combining marks), use
// [StringWidth] instead — RuneWidth cannot account for a following VS16, a
// ZWJ sequence, or a keycap combining mark.
func RuneWidth(r rune) int {
	switch {
	case r < 0x20:
		return 0 // C0 controls
	case r < 0x7F:
		return 1 // Printable ASCII
	case r < 0xA0:
		return 0 // DEL + C1 controls
	case r == 0xAD:
		return 0 // Soft hyphen
	}
	return runeWidthForRune(r)
}

// StringWidth returns the number of terminal cells needed to display string s.
//
// It correctly handles multi-rune grapheme clusters such as:
//   - Emoji ZWJ sequences (e.g., family emoji 👨‍👩‍👧‍👦)
//   - Skin tone modifiers (e.g., 👋🏽)
//   - Flag sequences (e.g., 🇺🇸)
//   - Combining characters (e.g., é composed as e + ◌́)
//   - Variation selectors (VS15 for text, VS16 for emoji presentation)
//
// For pure-ASCII strings (bytes >= 0x20 and < 0x80), this returns len(s)
// with zero allocations.
func StringWidth(s string) int {
	// ASCII fast path: if every byte is printable ASCII, width == len.
	if isASCII(s) {
		return len(s)
	}

	w := 0
	graphemeIter(s, func(_ string, gw int) {
		w += gw
	})
	return w
}

// Truncate returns s truncated to at most w terminal cells, with tail appended
// if truncation occurred.
//
// The result never exceeds w cells in display width. Truncation respects
// grapheme cluster boundaries: it will not split a multi-rune cluster.
//
// Common usage:
//
//	Truncate("Hello, 世界!", 8, "…")  // "Hello, …"
//	Truncate("short", 10, "…")        // "short" (no truncation)
func Truncate(s string, w int, tail string) string {
	if w < 0 {
		return ""
	}
	sw := StringWidth(s)
	if sw <= w {
		return s
	}

	tailWidth := StringWidth(tail)
	target := w - tailWidth
	if target <= 0 {
		// Even the tail won't fit or just barely. Return as much of tail as fits.
		if tailWidth <= w {
			return tail
		}
		result, _ := FitLeft(tail, w)
		return result
	}

	result, _ := FitLeft(s, target)
	return result + tail
}

// Fit is [Truncate] that also returns the display width of the result. It
// returns s unchanged (and its actual width) when it already fits in w cells,
// otherwise the truncated-plus-tail form and the actual width of that.
//
// This is the single-call replacement for the "Truncate-then-StringWidth"
// pattern common in TUI layout code — one pass of the grapheme iterator
// instead of three. On pure-ASCII input it hits the fast path and returns
// without any Unicode work.
func Fit(s string, w int, tail string) (string, int) {
	if w < 0 {
		return "", 0
	}
	if isASCII(s) {
		if len(s) <= w {
			return s, len(s)
		}
		// Need to truncate with tail.
		tailW := StringWidth(tail)
		if tailW >= w {
			if tailW == w {
				return tail, tailW
			}
			r, rw := FitLeft(tail, w)
			return r, rw
		}
		return s[:w-tailW] + tail, w
	}
	sw := StringWidth(s)
	if sw <= w {
		return s, sw
	}
	tailW := StringWidth(tail)
	target := w - tailW
	if target <= 0 {
		if tailW <= w {
			return tail, tailW
		}
		r, rw := FitLeft(tail, w)
		return r, rw
	}
	head, headW := FitLeft(s, target)
	return head + tail, headW + tailW
}

// FitLeft returns the longest prefix of s that fits in w terminal cells, along
// with the actual display width of the returned string. Truncation respects
// grapheme cluster boundaries.
//
// If w <= 0, it returns ("", 0). If the entire string fits, it returns (s, StringWidth(s)).
func FitLeft(s string, w int) (result string, width int) {
	if w <= 0 {
		return "", 0
	}

	// ASCII fast path.
	if isASCII(s) {
		if len(s) <= w {
			return s, len(s)
		}
		return s[:w], w
	}

	accumulated := 0
	lastEnd := 0

	rest := s
	for len(rest) > 0 {
		cluster, next, gw := firstGraphemeCluster(rest)
		if accumulated+gw > w {
			break
		}
		accumulated += gw
		lastEnd += len(cluster)
		rest = next
	}

	return s[:lastEnd], accumulated
}

// FitRight returns the longest suffix of s that fits in w terminal cells, along
// with the actual display width of the returned string. Grapheme cluster
// boundaries are respected.
//
// If w <= 0, it returns ("", 0). If the entire string fits, it returns
// (s, StringWidth(s)). This function allocates zero bytes on any input.
func FitRight(s string, w int) (result string, width int) {
	if w <= 0 {
		return "", 0
	}

	// ASCII fast path.
	if isASCII(s) {
		if len(s) <= w {
			return s, len(s)
		}
		return s[len(s)-w:], w
	}

	// Two-pass forward scan. Pass 1: total width. Pass 2: drop clusters from
	// the front until what remains fits. This is O(n) in bytes, 0 allocations,
	// and avoids the complexity of running UAX#29 in reverse.
	total := StringWidth(s)
	if total <= w {
		return s, total
	}

	needDrop := total - w
	dropped := 0
	i := 0
	for i < len(s) {
		var cluster string
		var cw int
		cluster, _, cw = firstGraphemeCluster(s[i:])
		dropped += cw
		i += len(cluster)
		if dropped >= needDrop {
			break
		}
	}

	return s[i:], total - dropped
}

// WidthIndex returns a slice of length len(s) mapping each byte offset in s
// to the display column at which that byte renders. Bytes that are part of
// the same grapheme cluster all map to the column of the cluster's first
// byte — for example, in "e\u0301" the combining acute's two bytes both map
// to the same column as 'e'.
//
// This is the function a text editor needs to translate a byte-level cursor
// position into a visual column, or vice versa (via search). The returned
// slice has len(s)+0 entries and is exclusively the caller's; WidthIndex
// itself performs exactly one allocation (the slice).
//
// If s is empty, the result is nil (no allocation).
func WidthIndex(s string) []int {
	if len(s) == 0 {
		return nil
	}
	result := make([]int, len(s))
	if isASCII(s) {
		for i := range result {
			result[i] = i
		}
		return result
	}
	i := 0
	col := 0
	for i < len(s) {
		var cluster string
		var cw int
		cluster, _, cw = firstGraphemeCluster(s[i:])
		for j := 0; j < len(cluster); j++ {
			result[i+j] = col
		}
		col += cw
		i += len(cluster)
	}
	return result
}

// isASCII reports whether every byte in s is printable ASCII (0x20..0x7E).
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < 0x20 || b >= 0x7F {
			return false
		}
	}
	return true
}
