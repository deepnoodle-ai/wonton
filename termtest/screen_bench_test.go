package termtest

import (
	"fmt"
	"strings"
	"testing"
)

// benchmarkFrame builds one full-screen frame the way a diffing renderer
// sends it: a cursor move per row, colors, and text in every cell.
func benchmarkFrame(width, height int) []byte {
	var b strings.Builder
	line := strings.Repeat("abcdefghij", width/10+1)[:width-8]
	for y := 0; y < height; y++ {
		fmt.Fprintf(&b, "\x1b[%d;1H\x1b[1;3%dm%4d \x1b[0m%s\x1b[K", y+1, y%8, y, line)
	}
	return []byte(b.String())
}

// BenchmarkScreenWriteFrame measures Write on a screen that never switches
// to the alternate screen, the common case for every termtest user.
func BenchmarkScreenWriteFrame(b *testing.B) {
	frame := benchmarkFrame(80, 24)
	s := NewScreen(80, 24)
	b.SetBytes(int64(len(frame)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s.Write(frame)
	}
}

// scrolledScreen returns a width x height screen after the given number of
// output lines. With wrapped set, each line is twice the width, so it
// soft-wraps and a width change has to reflow it.
func scrolledScreen(width, height, lines int, wrapped bool) *Screen {
	s := NewScreen(width, height)
	text := "short line"
	if wrapped {
		text = strings.Repeat("x", 2*width)
	}
	var b strings.Builder
	for i := 0; i < lines; i++ {
		fmt.Fprintf(&b, "%d %s\r\n", i, text)
	}
	_, _ = s.Write([]byte(b.String()))
	return s
}

// BenchmarkScreenResize measures Resize on a screen with a long scrollback,
// as a test that drags the window after a lot of output does. Each case
// alternates between two sizes, so every call changes the size.
func BenchmarkScreenResize(b *testing.B) {
	cases := []struct {
		name          string
		wrapped       bool
		width, height int
	}{
		{"height/short-lines", false, 100, 40},
		{"width/short-lines", false, 90, 30},
		{"height/wrapped-lines", true, 100, 40},
		{"width/wrapped-lines", true, 90, 30},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			s := scrolledScreen(100, 30, 10000, tc.wrapped)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if i%2 == 0 {
					s.Resize(tc.width, tc.height)
				} else {
					s.Resize(100, 30)
				}
			}
		})
	}
}
