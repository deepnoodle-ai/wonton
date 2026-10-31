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
