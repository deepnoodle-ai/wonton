package terminalinput

// HasReply reports whether b holds a terminal reply of the form
// ESC [ ? <digits and semicolons> final, such as a primary device attributes
// (DA1) reply (final 'c') or a Kitty keyboard flags reply (final 'u'). At
// least one parameter byte is required, so a typed 'c' or 'u' after an
// unrelated sequence does not count.
func HasReply(b []byte, final byte) bool {
	for i := 0; i+3 < len(b); i++ {
		if b[i] != 0x1b || b[i+1] != '[' || b[i+2] != '?' {
			continue
		}
		j := i + 3
		for j < len(b) && (b[j] >= '0' && b[j] <= '9' || b[j] == ';') {
			j++
		}
		if j > i+3 && j < len(b) && b[j] == final {
			return true
		}
	}
	return false
}
