package runewidth

import "testing"

// withCompat runs fn with terminal-compat mode enabled and restores the
// previous setting before returning, even on panic. Restoration happens
// synchronously (not via t.Cleanup) so that callers can assert on the
// post-restore behavior.
func withCompat(t *testing.T, fn func()) {
	t.Helper()
	prev := TerminalCompatMode()
	SetTerminalCompatMode(true)
	defer SetTerminalCompatMode(prev)
	fn()
}

func TestTerminalCompatMode_Default(t *testing.T) {
	if TerminalCompatMode() {
		t.Fatalf("compat mode should default to false")
	}
}

func TestTerminalCompatMode_Keycaps(t *testing.T) {
	cases := []struct {
		name    string
		cluster string
	}{
		{"hash keycap", "#\uFE0F\u20E3"},
		{"star keycap", "*\uFE0F\u20E3"},
		{"zero keycap", "0\uFE0F\u20E3"},
		{"nine keycap", "9\uFE0F\u20E3"},
	}

	// Default (Unicode-strict): keycap clusters report width 2.
	for _, tc := range cases {
		if got := StringWidth(tc.cluster); got != 2 {
			t.Errorf("default: StringWidth(%s) = %d, want 2", tc.name, got)
		}
	}

	// Compat: same clusters report width 1.
	withCompat(t, func() {
		for _, tc := range cases {
			if got := StringWidth(tc.cluster); got != 1 {
				t.Errorf("compat: StringWidth(%s) = %d, want 1", tc.name, got)
			}
		}
	})

	// And after restoration, we're back to width 2.
	for _, tc := range cases {
		if got := StringWidth(tc.cluster); got != 2 {
			t.Errorf("after restore: StringWidth(%s) = %d, want 2", tc.name, got)
		}
	}
}

func TestTerminalCompatMode_KeycapEmbeddedInString(t *testing.T) {
	// The override must apply to keycaps that aren't at the end of the
	// input — exercises the in-loop boundary branch in firstGraphemeCluster.
	s := "a #\uFE0F\u20E3 b"
	if got := StringWidth(s); got != 6 { // a(1) + space(1) + keycap(2) + space(1) + b(1) = 6
		t.Errorf("default: StringWidth(%q) = %d, want 6", s, got)
	}
	withCompat(t, func() {
		if got := StringWidth(s); got != 5 { // keycap collapses to 1
			t.Errorf("compat: StringWidth(%q) = %d, want 5", s, got)
		}
	})
}

func TestTerminalCompatMode_EmDashes(t *testing.T) {
	// Default: U+2E3A is width 3, U+2E3B is width 4.
	if got := RuneWidth(0x2E3A); got != 3 {
		t.Errorf("default: RuneWidth(U+2E3A) = %d, want 3", got)
	}
	if got := RuneWidth(0x2E3B); got != 4 {
		t.Errorf("default: RuneWidth(U+2E3B) = %d, want 4", got)
	}
	if got := StringWidth("\u2E3A"); got != 3 {
		t.Errorf("default: StringWidth(U+2E3A) = %d, want 3", got)
	}
	if got := StringWidth("\u2E3B"); got != 4 {
		t.Errorf("default: StringWidth(U+2E3B) = %d, want 4", got)
	}

	withCompat(t, func() {
		if got := RuneWidth(0x2E3A); got != 1 {
			t.Errorf("compat: RuneWidth(U+2E3A) = %d, want 1", got)
		}
		if got := RuneWidth(0x2E3B); got != 1 {
			t.Errorf("compat: RuneWidth(U+2E3B) = %d, want 1", got)
		}
		if got := StringWidth("\u2E3A"); got != 1 {
			t.Errorf("compat: StringWidth(U+2E3A) = %d, want 1", got)
		}
		if got := StringWidth("\u2E3B"); got != 1 {
			t.Errorf("compat: StringWidth(U+2E3B) = %d, want 1", got)
		}
	})
}

func TestTerminalCompatMode_LeavesOtherClustersAlone(t *testing.T) {
	// Compat mode is targeted: it must not affect VS16 emoji, flags,
	// ZWJ sequences, skin tones, CJK, or ASCII.
	cases := []struct {
		name    string
		cluster string
		want    int
	}{
		{"ASCII", "A", 1},
		{"CJK", "\u4E2D", 2},
		{"VS16 heart", "\u2764\uFE0F", 2},
		{"text heart", "\u2764", 1},
		{"JP flag", "\U0001F1EF\U0001F1F5", 2},
		{"family of four", "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466", 2},
		{"waving hand + skin", "\U0001F44B\U0001F3FD", 2},
	}
	withCompat(t, func() {
		for _, tc := range cases {
			if got := StringWidth(tc.cluster); got != tc.want {
				t.Errorf("compat: StringWidth(%s) = %d, want %d", tc.name, got, tc.want)
			}
		}
	})
}
