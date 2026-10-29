package terminalinput

import "testing"

func TestHasReply(t *testing.T) {
	for _, tc := range []struct {
		in    string
		final byte
		want  bool
	}{
		{"\x1b[?62;22c", 'c', true},
		{"\x1b]11;rgb:0/0/0\x07\x1b[?1;2c", 'c', true},
		{"\x1b[?0u\x1b[?6c", 'u', true},
		{"\x1b[?0u\x1b[?6c", 'c', true},
		{"\x1b[?62;22", 'c', false},
		{"\x1b[62c", 'c', false},
		{"\x1b[?c", 'c', false},
		{"u\x1b[?62;22c", 'u', false},
		{"", 'c', false},
	} {
		if got := HasReply([]byte(tc.in), tc.final); got != tc.want {
			t.Errorf("HasReply(%q, %q) = %v, want %v", tc.in, tc.final, got, tc.want)
		}
	}
}
