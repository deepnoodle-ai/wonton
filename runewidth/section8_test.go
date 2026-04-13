package runewidth

import (
	"testing"
)

// The tests in this file pin the correctness corpus from
// docs/runewidth-improvements.md section 8. That section documents 14 cases
// from runewidth/internal/bench/compat_test.go where wonton/runewidth returns
// a different width than the other two libraries in the Go ecosystem, and
// every one of those 14 is a case where wonton is correct per current
// Unicode + what real terminal emulators render.
//
// These are tests, not benchmarks, so they run in `go test ./...` on every
// platform and catch any table regeneration or grapheme-state-machine change
// that would silently drop a correctness win.

// TestSection8_WontonUniqueKeycaps pins the 4 keycap sequences where wonton
// is alone among the three libraries in returning width 2. This is the
// "base ∈ {#,*,0-9} + VS16 + U+20E3" postprocess pass in
// grapheme.firstGraphemeCluster.
func TestSection8_WontonUniqueKeycaps(t *testing.T) {
	cases := []struct {
		name string
		s    string
	}{
		{"hash keycap", "#\uFE0F\u20E3"},     // #️⃣
		{"asterisk keycap", "*\uFE0F\u20E3"}, // *️⃣
		{"zero keycap", "0\uFE0F\u20E3"},     // 0️⃣
		{"nine keycap", "9\uFE0F\u20E3"},     // 9️⃣
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w := StringWidth(tc.s); w != 2 {
				t.Errorf("StringWidth(%+q) = %d, want 2", tc.s, w)
			}
		})
	}
}

// TestSection8_DisagreementsWithGoRunewidth pins the 10 corpus entries where
// wonton matches uniseg v0.4.7 and go-runewidth v0.0.21 is the outlier. See
// docs/runewidth-improvements.md section 8 and
// runewidth/testdata/compat-report.md for the source of these numbers.
func TestSection8_DisagreementsWithGoRunewidth(t *testing.T) {
	cases := []struct {
		name string
		s    string
		want int
	}{
		// Indic cluster widths — wonton and uniseg both compute cluster
		// width by summing the first rune's width rather than every rune,
		// matching how terminals render these clusters.
		{"indic tamil", "\u0BA4\u0BAE\u0BBF\u0BB4\u0BCD", 4}, // தமிழ்
		{"indic bengali", "\u09AC\u09BE\u0982\u09B2\u09BE", 3}, // বাংলা

		// Emoji with VS16 — the variation selector promotes the
		// text-default glyph to emoji width.
		{"heart VS16", "\u2764\uFE0F", 2},    // ❤️
		{"warning VS16", "\u26A0\uFE0F", 2},  // ⚠️

		// ZWJ rainbow flag: base + VS16 + ZWJ + rainbow.
		{"rainbow flag", "\U0001F3F3\uFE0F\u200D\U0001F308", 2}, // 🏳️‍🌈

		// Regional indicator flags: each pair is one grapheme cluster of
		// width 2.
		{"US flag", "\U0001F1FA\U0001F1F8", 2},                                 // 🇺🇸
		{"JP flag", "\U0001F1EF\U0001F1F5", 2},                                 // 🇯🇵
		{"two flags", "\U0001F1FA\U0001F1F8\U0001F1EF\U0001F1F5", 4},           // 🇺🇸🇯🇵

		// Wide punctuation — U+2E3A and U+2E3B are handled as explicit rune
		// special cases in runeWidthFromPacked (U+2E3A → 3, U+2E3B → 4) to
		// match terminal rendering, not via East Asian Width alone.
		{"a two-em-dash b", "a\u2E3Ab", 5}, // a⸺b = 1 + 3 + 1
		{"three-em dash", "\u2E3B", 4},      // ⸻
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if w := StringWidth(tc.s); w != tc.want {
				t.Errorf("StringWidth(%+q) = %d, want %d", tc.s, w, tc.want)
			}
		})
	}
}

// TestGraphemes_PublicIter asserts that the exported Graphemes iterator
// agrees with the internal graphemeIter callback form on every corpus case.
// This is the only direct test of the iter.Seq2[string, int] API; the older
// TestGraphemeClusters exercises the callback form only.
func TestGraphemes_PublicIter(t *testing.T) {
	corpus := []string{
		"",
		"Hello",
		"中文",
		"e\u0301",
		"\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466",
		"\U0001F44B\U0001F3FD",
		"\U0001F1FA\U0001F1F8",
		"#\uFE0F\u20E3",
		"Hi 世界 👋🏽!",
	}
	for _, s := range corpus {
		t.Run(s, func(t *testing.T) {
			type pair struct {
				c string
				w int
			}
			var viaIter []pair
			for c, w := range Graphemes(s) {
				viaIter = append(viaIter, pair{c, w})
			}
			var viaCallback []pair
			graphemeIter(s, func(c string, w int) {
				viaCallback = append(viaCallback, pair{c, w})
			})

			if len(viaIter) != len(viaCallback) {
				t.Fatalf("iter emitted %d clusters, callback emitted %d for %+q",
					len(viaIter), len(viaCallback), s)
			}
			for i := range viaIter {
				if viaIter[i] != viaCallback[i] {
					t.Errorf("cluster %d diverges: iter=%+v callback=%+v (input %+q)",
						i, viaIter[i], viaCallback[i], s)
				}
			}

			// Summed width must match StringWidth.
			total := 0
			for _, p := range viaIter {
				total += p.w
			}
			if total != StringWidth(s) {
				t.Errorf("sum of cluster widths = %d, StringWidth = %d for %+q",
					total, StringWidth(s), s)
			}
		})
	}

	// Early-stop invariant: breaking out of the range loop after the first
	// yielded pair must not panic and must not walk the rest of the input.
	t.Run("early stop", func(t *testing.T) {
		s := "abc"
		count := 0
		for range Graphemes(s) {
			count++
			break
		}
		if count != 1 {
			t.Errorf("early-stop yielded %d clusters, want 1", count)
		}
	})
}
