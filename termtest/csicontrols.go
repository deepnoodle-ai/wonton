package termtest

import "strings"

// runCSIControls runs the C0 controls found inside the CSI bytes seq, in
// order, and returns seq with them removed. controls is how many control
// bytes (C0 or DEL) seq holds, so the common case of none costs nothing.
// CR, LF, VT, FF, BS, and HT act as they do outside a sequence; other C0
// controls and DEL are ignored, as in xterm.
func (s *Screen) runCSIControls(seq string, controls int) string {
	if controls == 0 {
		return seq
	}
	var params strings.Builder
	params.Grow(len(seq) - controls)
	for i := 0; i < len(seq); i++ {
		c := seq[i]
		switch {
		case c == 0x7F:
		case c < 0x20:
			s.writeCluster(string(c), 0)
		default:
			params.WriteByte(c)
		}
	}
	return params.String()
}
