package terminalstate

import "testing"

func TestInsideTmux(t *testing.T) {
	for _, tc := range []struct {
		name, tmux, termProgram, term string
		want                          bool
	}{
		{"plain terminal", "", "iTerm.app", "xterm-256color", false},
		{"tmux pane", "/tmp/tmux-501/default,123,0", "tmux", "tmux-256color", true},
		{"tmux pane with screen TERM", "/tmp/tmux-501/default,123,0", "tmux", "screen-256color", true},
		{"tmux over ssh", "", "", "tmux-256color", true},
		{"TERM_PROGRAM only", "", "tmux", "xterm-256color", true},
		{"GNU screen", "", "", "screen-256color", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TMUX", tc.tmux)
			t.Setenv("TERM_PROGRAM", tc.termProgram)
			t.Setenv("TERM", tc.term)
			if got := InsideTmux(); got != tc.want {
				t.Fatalf("InsideTmux() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestKeyboardSequences(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("TERM", "xterm-256color")

	t.Setenv("TMUX", "")
	if got := KeyboardEnable(); got != "\x1b[>1u" {
		t.Fatalf("enable outside tmux = %q", got)
	}
	if got := KeyboardRelease(); got != "\x1b[<u" {
		t.Fatalf("release outside tmux = %q", got)
	}

	// tmux ignores the Kitty push and needs modifyOtherKeys to forward
	// Shift+Enter; the release resets it.
	t.Setenv("TMUX", "/tmp/tmux-501/default,123,0")
	if got := KeyboardEnable(); got != "\x1b[>1u\x1b[>4;1m" {
		t.Fatalf("enable inside tmux = %q", got)
	}
	if got := KeyboardRelease(); got != "\x1b[<u\x1b[>4m" {
		t.Fatalf("release inside tmux = %q", got)
	}
}
