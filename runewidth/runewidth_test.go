package runewidth

import (
	"strings"
	"testing"
)

// --- RuneWidth ---

func TestRuneWidth_ASCII(t *testing.T) {
	for r := rune(0x20); r < 0x7F; r++ {
		if w := RuneWidth(r); w != 1 {
			t.Errorf("RuneWidth(%U) = %d, want 1", r, w)
		}
	}
}

func TestRuneWidth_Controls(t *testing.T) {
	// C0 controls.
	for r := rune(0); r < 0x20; r++ {
		if w := RuneWidth(r); w != 0 {
			t.Errorf("RuneWidth(%U) = %d, want 0", r, w)
		}
	}
	// DEL.
	if w := RuneWidth(0x7F); w != 0 {
		t.Errorf("RuneWidth(DEL) = %d, want 0", w)
	}
	// C1 controls.
	for r := rune(0x80); r < 0xA0; r++ {
		if w := RuneWidth(r); w != 0 {
			t.Errorf("RuneWidth(%U) = %d, want 0", r, w)
		}
	}
}

func TestRuneWidth_SoftHyphen(t *testing.T) {
	if w := RuneWidth(0xAD); w != 0 {
		t.Errorf("RuneWidth(SOFT HYPHEN) = %d, want 0", w)
	}
}

func TestRuneWidth_Latin(t *testing.T) {
	tests := []rune{'é', 'ñ', 'ü', 'ö', 'ß', 'à', 'ø'}
	for _, r := range tests {
		if w := RuneWidth(r); w != 1 {
			t.Errorf("RuneWidth(%U %c) = %d, want 1", r, r, w)
		}
	}
}

func TestRuneWidth_CJK(t *testing.T) {
	tests := []rune{
		'中', '文', '字', // CJK Unified Ideographs
		'가', '나', // Hangul Syllables
		'ア', 'カ', // Katakana (fullwidth)
		'あ', 'い', // Hiragana
		'Ａ', 'Ｚ', // Fullwidth Latin
		'０', '９', // Fullwidth Digits
	}
	for _, r := range tests {
		if w := RuneWidth(r); w != 2 {
			t.Errorf("RuneWidth(%U %c) = %d, want 2", r, r, w)
		}
	}
}

func TestRuneWidth_Emoji(t *testing.T) {
	tests := []struct {
		r    rune
		want int
	}{
		{'😀', 2},
		{'🎉', 2},
		{'❤', 1}, // U+2764 text presentation by default
		{'⭐', 2}, // U+2B50 has Emoji_Presentation=Yes
		{'👍', 2},
		{'🔥', 2},
	}
	for _, tt := range tests {
		if w := RuneWidth(tt.r); w != tt.want {
			t.Errorf("RuneWidth(%U %c) = %d, want %d", tt.r, tt.r, w, tt.want)
		}
	}
}

func TestRuneWidth_Combining(t *testing.T) {
	tests := []rune{
		0x0300, // COMBINING GRAVE ACCENT
		0x0301, // COMBINING ACUTE ACCENT
		0x0302, // COMBINING CIRCUMFLEX ACCENT
		0x0308, // COMBINING DIAERESIS
		0x20E3, // COMBINING ENCLOSING KEYCAP
	}
	for _, r := range tests {
		if w := RuneWidth(r); w != 0 {
			t.Errorf("RuneWidth(%U) = %d, want 0", r, w)
		}
	}
}

func TestRuneWidth_WideDashes(t *testing.T) {
	// uniseg v0.4.7 assigns these two punctuation characters widths > 2.
	if w := RuneWidth(0x2E3A); w != 3 {
		t.Errorf("RuneWidth(U+2E3A TWO-EM DASH) = %d, want 3", w)
	}
	if w := RuneWidth(0x2E3B); w != 4 {
		t.Errorf("RuneWidth(U+2E3B THREE-EM DASH) = %d, want 4", w)
	}
	// And StringWidth agrees.
	if w := StringWidth("a\u2E3Ab"); w != 1+3+1 {
		t.Errorf("StringWidth(a⸺b) = %d, want 5", w)
	}
	if w := StringWidth("\u2E3B"); w != 4 {
		t.Errorf("StringWidth(⸻) = %d, want 4", w)
	}
}

func TestUnicodeVersion(t *testing.T) {
	if UnicodeVersion != "17.0.0" {
		t.Errorf("UnicodeVersion = %q, want 17.0.0", UnicodeVersion)
	}
}

func TestRuneWidth_Unicode17(t *testing.T) {
	// Code points newly assigned in Unicode 17.0 per DerivedAge-17.0.0.txt.
	// Using non-emoji additions so the test isolates the table-regeneration
	// aspect from any emoji-data churn.
	//
	// U+20C1 SAUDI RIYAL SIGN — currency symbol, new-in-17.0. It should be
	// a plain printable width-1 character with no grapheme break property.
	if w := RuneWidth(0x20C1); w != 1 {
		t.Errorf("RuneWidth(U+20C1 SAUDI RIYAL SIGN, new in 17.0) = %d, want 1", w)
	}
	if p := lookupGB(0x20C1); p != gbOther {
		t.Errorf("lookupGB(U+20C1) = %d, want gbOther", p)
	}

	// U+1ACF..U+1ADD — new combining marks added in 17.0. These must be
	// classified as gbExtend and have width 0.
	for r := rune(0x1ACF); r <= 0x1ADD; r++ {
		if p := lookupGB(r); p != gbExtend {
			t.Errorf("lookupGB(U+%X new 17.0 combining mark) = %d, want gbExtend", r, p)
		}
		if w := RuneWidth(r); w != 0 {
			t.Errorf("RuneWidth(U+%X new 17.0 combining mark) = %d, want 0", r, w)
		}
	}
}

func TestIncbInitialBounds(t *testing.T) {
	// incbInitial uses a 4-band fast reject to skip the lookupInCB binary
	// search on non-Indic runes. This test guards against drift: every
	// InCB=Consonant code point in the generated table must pass the fast
	// reject and land on incbAfterConsonant, otherwise GB9c would silently
	// stop firing for that script.
	for _, iv := range incbProperty {
		if iv.prop != incbConsonant {
			continue
		}
		for _, r := range []rune{iv.lo, iv.hi} {
			if s := incbInitial(r); s != incbAfterConsonant {
				t.Errorf("incbInitial(U+%04X) = %d, want incbAfterConsonant (consonant interval [U+%04X..U+%04X] — update the bands in grapheme.go)",
					r, s, iv.lo, iv.hi)
			}
		}
	}
	// Runes that should always reject cheaply without touching lookupInCB.
	for _, r := range []rune{'a', 0x4E00, 0x1F600, 0x200D, 0x0300} {
		if s := incbInitial(r); s != incbStateNone {
			t.Errorf("incbInitial(U+%04X) = %d, want incbStateNone", r, s)
		}
	}
}

func TestRuneWidth_ZeroWidth(t *testing.T) {
	tests := []rune{
		0x200B, // ZERO WIDTH SPACE
		0x200C, // ZERO WIDTH NON-JOINER
		0x200D, // ZERO WIDTH JOINER
		0xFEFF, // BOM / ZERO WIDTH NO-BREAK SPACE
	}
	for _, r := range tests {
		if w := RuneWidth(r); w != 0 {
			t.Errorf("RuneWidth(%U) = %d, want 0", r, w)
		}
	}
}

// --- StringWidth ---

func TestStringWidth_Empty(t *testing.T) {
	if w := StringWidth(""); w != 0 {
		t.Errorf("StringWidth(\"\") = %d, want 0", w)
	}
}

func TestStringWidth_ASCII(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"Hello", 5},
		{"Hello, World!", 13},
		{"abcdef", 6},
		{" ", 1},
		{"  ", 2},
	}
	for _, tt := range tests {
		if w := StringWidth(tt.s); w != tt.want {
			t.Errorf("StringWidth(%q) = %d, want %d", tt.s, w, tt.want)
		}
	}
}

func TestStringWidth_CJK(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"中文", 4},
		{"Hello中文", 9},
		{"中Hello文", 9},
	}
	for _, tt := range tests {
		if w := StringWidth(tt.s); w != tt.want {
			t.Errorf("StringWidth(%q) = %d, want %d", tt.s, w, tt.want)
		}
	}
}

func TestStringWidth_Combining(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"e\u0301", 1},             // é (e + combining acute accent)
		{"e\u0301e\u0301", 2},      // éé
		{"noe\u0308l", 4},          // noël
		{"a\u0300\u0301\u0302", 1}, // a with three combining marks
	}
	for _, tt := range tests {
		if w := StringWidth(tt.s); w != tt.want {
			t.Errorf("StringWidth(%q) = %d, want %d", tt.s, w, tt.want)
		}
	}
}

func TestStringWidth_Emoji(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"simple emoji", "😀", 2},
		{"two emoji", "😀😃", 4},
		{"emoji in text", "Hi 😀!", 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := StringWidth(tt.s); w != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, w, tt.want)
			}
		})
	}
}

func TestStringWidth_ZWJSequences(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		// 👨‍👩‍👧‍👦 = U+1F468 ZWJ U+1F469 ZWJ U+1F467 ZWJ U+1F466
		{"family", "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466", 2},
		// 👩‍💻 = U+1F469 ZWJ U+1F4BB
		{"woman technologist", "\U0001F469\u200D\U0001F4BB", 2},
		// 🏳️‍🌈 = U+1F3F3 VS16 ZWJ U+1F308
		{"rainbow flag", "\U0001F3F3\uFE0F\u200D\U0001F308", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := StringWidth(tt.s); w != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, w, tt.want)
			}
		})
	}
}

func TestStringWidth_SkinTone(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		// 👋🏽 = U+1F44B U+1F3FD
		{"wave medium skin", "\U0001F44B\U0001F3FD", 2},
		// 👍🏿 = U+1F44D U+1F3FF
		{"thumbsup dark skin", "\U0001F44D\U0001F3FF", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := StringWidth(tt.s); w != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, w, tt.want)
			}
		})
	}
}

func TestStringWidth_Flags(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		// 🇺🇸 = U+1F1FA U+1F1F8
		{"US flag", "\U0001F1FA\U0001F1F8", 2},
		// 🇯🇵 = U+1F1EF U+1F1F5
		{"JP flag", "\U0001F1EF\U0001F1F5", 2},
		// Two flags
		{"two flags", "\U0001F1FA\U0001F1F8\U0001F1EF\U0001F1F5", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := StringWidth(tt.s); w != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, w, tt.want)
			}
		})
	}
}

func TestStringWidth_Keycap(t *testing.T) {
	// #️⃣ = # + VS16 + U+20E3
	s := "#\uFE0F\u20E3"
	if w := StringWidth(s); w != 2 {
		t.Errorf("StringWidth(keycap #) = %d, want 2", w)
	}
}

func TestStringWidth_Mixed(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want int
	}{
		{"ascii+cjk+emoji", "Hi 世界 👋🏽!", 11},
		{"combining+wide", "e\u0301中", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := StringWidth(tt.s); w != tt.want {
				t.Errorf("StringWidth(%q) = %d, want %d", tt.s, w, tt.want)
			}
		})
	}
}

func TestStringWidth_WithControls(t *testing.T) {
	// Tab and newline should be width 0.
	if w := StringWidth("a\tb"); w != 2 {
		t.Errorf("StringWidth(\"a\\tb\") = %d, want 2", w)
	}
}

// --- Truncate ---

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		w    int
		tail string
		want string
	}{
		{"no truncation", "Hello", 10, "…", "Hello"},
		{"exact fit", "Hello", 5, "…", "Hello"},
		{"truncate ascii", "Hello, World!", 8, "…", "Hello, …"},
		{"truncate cjk", "中文字", 4, "…", "中…"},
		{"truncate to zero", "Hello", 0, "", ""},
		{"empty string", "", 5, "…", ""},
		{"tail only", "Hello, World!", 1, "…", "…"},
		{"no tail", "Hello, World!", 5, "", "Hello"},
		{"wide char boundary", "a中b", 2, "…", "a…"},
		{"negative width", "Hello", -1, "…", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.w, tt.tail)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d, %q) = %q, want %q", tt.s, tt.w, tt.tail, got, tt.want)
			}
		})
	}
}

func TestTruncate_GraphemeBoundary(t *testing.T) {
	// Should not split a ZWJ sequence.
	family := "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"
	got := Truncate("X"+family+"Y", 3, "")
	// The family emoji is width 2. "X" is 1. Total "X"+family = 3. "Y" would be 4.
	if w := StringWidth(got); w > 3 {
		t.Errorf("Truncate result %q has width %d, want <= 3", got, w)
	}
}

func TestTruncate_Invariant(t *testing.T) {
	// StringWidth of truncated result never exceeds w.
	tests := []string{
		"Hello, World!",
		"中文字テスト",
		"Hello 👨‍👩‍👧‍👦 World",
		"e\u0301e\u0301e\u0301",
		"\U0001F1FA\U0001F1F8 flag",
	}
	for _, s := range tests {
		for w := 0; w <= StringWidth(s)+2; w++ {
			result := Truncate(s, w, "…")
			rw := StringWidth(result)
			if rw > w {
				t.Errorf("Truncate(%q, %d, \"…\") = %q (width %d), exceeds limit", s, w, result, rw)
			}
		}
	}
}

// --- Fit ---

func TestFit(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		w         int
		tail      string
		wantStr   string
		wantWidth int
	}{
		{"empty", "", 5, "…", "", 0},
		{"fits", "Hello", 10, "…", "Hello", 5},
		{"exact", "Hello", 5, "…", "Hello", 5},
		{"truncate ascii", "Hello, World!", 8, "…", "Hello, …", 8},
		{"truncate cjk", "中文字", 4, "…", "中…", 3},
		{"tail only", "Hello, World!", 1, "…", "…", 1},
		{"no tail", "Hello, World!", 5, "", "Hello", 5},
		{"negative width", "Hello", -1, "…", "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotW := Fit(tt.s, tt.w, tt.tail)
			if got != tt.wantStr || gotW != tt.wantWidth {
				t.Errorf("Fit(%q, %d, %q) = (%q, %d), want (%q, %d)",
					tt.s, tt.w, tt.tail, got, gotW, tt.wantStr, tt.wantWidth)
			}
		})
	}
}

// --- FitLeft ---

func TestFitLeft(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		w         int
		wantStr   string
		wantWidth int
	}{
		{"empty", "", 5, "", 0},
		{"zero width", "Hello", 0, "", 0},
		{"negative width", "Hello", -1, "", 0},
		{"fits completely", "Hello", 10, "Hello", 5},
		{"exact fit", "Hello", 5, "Hello", 5},
		{"truncate ascii", "Hello", 3, "Hel", 3},
		{"truncate cjk", "中文字", 3, "中", 2},
		{"wide char won't fit", "中", 1, "", 0},
		{"mixed", "a中b", 3, "a中", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStr, gotWidth := FitLeft(tt.s, tt.w)
			if gotStr != tt.wantStr || gotWidth != tt.wantWidth {
				t.Errorf("FitLeft(%q, %d) = (%q, %d), want (%q, %d)",
					tt.s, tt.w, gotStr, gotWidth, tt.wantStr, tt.wantWidth)
			}
		})
	}
}

// --- FitRight ---

func TestFitRight(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		w         int
		wantStr   string
		wantWidth int
	}{
		{"empty", "", 5, "", 0},
		{"zero width", "Hello", 0, "", 0},
		{"negative width", "Hello", -1, "", 0},
		{"fits completely", "Hello", 10, "Hello", 5},
		{"exact fit", "Hello", 5, "Hello", 5},
		{"truncate ascii", "Hello", 3, "llo", 3},
		{"truncate cjk", "中文字", 3, "字", 2},
		{"wide char won't fit", "中", 1, "", 0},
		{"mixed", "a中b", 3, "中b", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStr, gotWidth := FitRight(tt.s, tt.w)
			if gotStr != tt.wantStr || gotWidth != tt.wantWidth {
				t.Errorf("FitRight(%q, %d) = (%q, %d), want (%q, %d)",
					tt.s, tt.w, gotStr, gotWidth, tt.wantStr, tt.wantWidth)
			}
		})
	}
}

// --- WidthIndex ---

func TestWidthIndex(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want []int
	}{
		{"empty", "", nil},
		{"ascii", "abc", []int{0, 1, 2}},
		{"ascii long", "Hi!", []int{0, 1, 2}},
		// "a中b" = a (1 byte, col 0) + 中 (3 bytes, col 1,1,1; width 2) + b (1 byte, col 3)
		{"cjk", "a中b", []int{0, 1, 1, 1, 3}},
		// "e\u0301" = e (1 byte, col 0) + combining acute (2 bytes, col 0)
		{"combining", "e\u0301", []int{0, 0, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WidthIndex(tt.s)
			if !intSliceEqual(got, tt.want) {
				t.Errorf("WidthIndex(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestWidthIndex_ClusterBytesShareColumn(t *testing.T) {
	// Every byte inside a multi-byte cluster maps to the cluster's starting column.
	s := "X" + "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466" + "Y"
	idx := WidthIndex(s)
	if idx[0] != 0 {
		t.Errorf("idx[0] = %d, want 0 for 'X'", idx[0])
	}
	// 'X' is 1 byte, then the family emoji is one cluster starting at byte 1
	// with column 1 and width 2. The Y should start at column 3.
	familyStart := 1
	familyLen := len("\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466")
	for i := familyStart; i < familyStart+familyLen; i++ {
		if idx[i] != 1 {
			t.Errorf("idx[%d] = %d, want 1 (inside family cluster)", i, idx[i])
		}
	}
	yStart := familyStart + familyLen
	if idx[yStart] != 3 {
		t.Errorf("idx[Y] = %d, want 3", idx[yStart])
	}
}

// --- ASCII fast path ---

func TestASCIIFastPath(t *testing.T) {
	// Verify the fast path produces the same result as the slow path.
	s := "Hello, World! This is a test of ASCII-only content."
	if !isASCII(s) {
		t.Fatal("expected isASCII to be true for ASCII string")
	}
	if w := StringWidth(s); w != len(s) {
		t.Errorf("StringWidth ASCII = %d, want %d", w, len(s))
	}
}

func TestIsASCII(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"Hello", true},
		{"Hello 世界", false},
		{"abc\ndef", false},       // newline < 0x20
		{"abc\tdef", false},       // tab < 0x20
		{"", true},                // empty is trivially ASCII
		{"\x7F", false},           // DEL
		{"Hello\x80World", false}, // 0x80
	}
	for _, tt := range tests {
		if got := isASCII(tt.s); got != tt.want {
			t.Errorf("isASCII(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

// --- Grapheme iterator ---

func TestGraphemeClusters(t *testing.T) {
	tests := []struct {
		name           string
		s              string
		wantClusters   int
		wantTotalWidth int
	}{
		{"ascii", "Hello", 5, 5},
		{"cjk", "中文", 2, 4},
		{"combining", "e\u0301", 1, 1},
		{"two combining", "e\u0301a\u0308", 2, 2},
		{"zwj family", "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466", 1, 2},
		{"skin tone", "\U0001F44B\U0001F3FD", 1, 2},
		{"flag", "\U0001F1FA\U0001F1F8", 1, 2},
		{"keycap", "#\uFE0F\u20E3", 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusters := 0
			totalWidth := 0
			graphemeIter(tt.s, func(_ string, w int) {
				clusters++
				totalWidth += w
			})
			if clusters != tt.wantClusters {
				t.Errorf("grapheme clusters = %d, want %d", clusters, tt.wantClusters)
			}
			if totalWidth != tt.wantTotalWidth {
				t.Errorf("total width = %d, want %d", totalWidth, tt.wantTotalWidth)
			}
		})
	}
}

// --- Fuzz ---

func FuzzStringWidth(f *testing.F) {
	f.Add("Hello, World!")
	f.Add("中文")
	f.Add("😀")
	f.Add("\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466")
	f.Add("e\u0301")
	f.Add("\U0001F1FA\U0001F1F8")
	f.Add("#\uFE0F\u20E3")
	f.Add("")

	f.Fuzz(func(t *testing.T, s string) {
		w := StringWidth(s)
		if w < 0 {
			t.Errorf("StringWidth(%q) = %d, want >= 0", s, w)
		}
	})
}

func FuzzTruncate(f *testing.F) {
	f.Add("Hello, World!", 5)
	f.Add("中文字", 3)
	f.Add("😀", 1)
	f.Add("", 0)

	f.Fuzz(func(t *testing.T, s string, w int) {
		if w < 0 {
			w = 0
		}
		if w > 1000 {
			w = 1000
		}
		result := Truncate(s, w, "")
		rw := StringWidth(result)
		if rw > w {
			t.Errorf("Truncate(%q, %d, \"\") = %q (width %d), exceeds limit", s, w, result, rw)
		}
	})
}

// --- Benchmarks ---

func BenchmarkRuneWidth_ASCII(b *testing.B) {
	for b.Loop() {
		RuneWidth('A')
	}
}

func BenchmarkRuneWidth_CJK(b *testing.B) {
	for b.Loop() {
		RuneWidth('中')
	}
}

func BenchmarkRuneWidth_Emoji(b *testing.B) {
	for b.Loop() {
		RuneWidth('😀')
	}
}

func BenchmarkStringWidth_ASCIIShort(b *testing.B) {
	s := "Hello, World!"
	for b.Loop() {
		StringWidth(s)
	}
}

func BenchmarkStringWidth_ASCIILong(b *testing.B) {
	s := strings.Repeat("Hello, World! ", 100)
	for b.Loop() {
		StringWidth(s)
	}
}

func BenchmarkStringWidth_CJK(b *testing.B) {
	s := "中文字テスト日本語"
	for b.Loop() {
		StringWidth(s)
	}
}

func BenchmarkStringWidth_Emoji(b *testing.B) {
	s := "Hello 😀🎉👍🔥 World"
	for b.Loop() {
		StringWidth(s)
	}
}

func BenchmarkStringWidth_ZWJ(b *testing.B) {
	s := "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466"
	for b.Loop() {
		StringWidth(s)
	}
}

func BenchmarkStringWidth_Mixed(b *testing.B) {
	s := "Hello 世界 👋🏽! noe\u0308l"
	for b.Loop() {
		StringWidth(s)
	}
}

func BenchmarkTruncate_ASCII(b *testing.B) {
	s := strings.Repeat("Hello, World! ", 10)
	for b.Loop() {
		Truncate(s, 50, "…")
	}
}

func BenchmarkTruncate_Mixed(b *testing.B) {
	s := "Hello 世界 👋🏽! This is a longer mixed string"
	for b.Loop() {
		Truncate(s, 20, "…")
	}
}

func BenchmarkFitLeft_ASCII(b *testing.B) {
	s := strings.Repeat("Hello, World! ", 10)
	for b.Loop() {
		FitLeft(s, 50)
	}
}

func BenchmarkFitRight_ASCII(b *testing.B) {
	s := strings.Repeat("Hello, World! ", 10)
	for b.Loop() {
		FitRight(s, 50)
	}
}

func BenchmarkFitRight_CJK(b *testing.B) {
	s := strings.Repeat("中文字テスト日本語 ", 10)
	for b.Loop() {
		FitRight(s, 20)
	}
}

func BenchmarkFitRight_Emoji(b *testing.B) {
	s := strings.Repeat("Hello 😀🎉👍🔥 ", 10)
	for b.Loop() {
		FitRight(s, 20)
	}
}

func BenchmarkFitRight_Mixed(b *testing.B) {
	s := "Hello 世界 👋🏽 family \U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466 and some text"
	for b.Loop() {
		FitRight(s, 20)
	}
}
