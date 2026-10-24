package tui

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// TerminalBackground is whether the terminal's background color is dark or
// light. DetectTerminalBackground returns it.
type TerminalBackground int

const (
	// TerminalBackgroundUnknown means the background could not be determined.
	TerminalBackgroundUnknown TerminalBackground = iota
	// TerminalBackgroundDark is a dark background, such as black or dark grey.
	TerminalBackgroundDark
	// TerminalBackgroundLight is a light background, such as white or cream.
	TerminalBackgroundLight
)

// String returns "unknown", "dark", or "light".
func (b TerminalBackground) String() string {
	switch b {
	case TerminalBackgroundDark:
		return "dark"
	case TerminalBackgroundLight:
		return "light"
	default:
		return "unknown"
	}
}

// defaultBackgroundTimeout is how long DetectTerminalBackground waits for the
// terminal to answer when the caller passes no timeout.
const defaultBackgroundTimeout = 100 * time.Millisecond

// DetectTerminalBackground reports whether the terminal's background is dark
// or light, so an app can pick colors that read well on it. It asks the terminal for its background color (an OSC 11 query) and
// waits up to timeout for the answer; a timeout of 0 or less waits 100ms.
// When the terminal does not answer, it falls back to the COLORFGBG
// environment variable, and then to TerminalBackgroundUnknown.
//
// Call it once at startup, before a Runtime, InlineApp, or anything else
// reads standard input: it puts the terminal in raw mode for the query and
// reads the answer from standard input. Keys typed while it waits are
// dropped. When the terminal answers in time, it reads only up to the end of
// the answer, so keys typed after that stay unread. When it times out, it
// flushes all pending input, so a slow answer already on its way is not read
// as keys; an answer that arrives after the flush still would be.
//
// It returns TerminalBackgroundUnknown at once, with no error, when standard
// output is not a terminal. When standard input is not a terminal, or TERM is
// empty or "dumb", it skips the query and uses only COLORFGBG. The error
// reports a query that failed, such as a failed write; the returned value is
// still the best answer available. On platforms other than Linux and macOS
// it uses only COLORFGBG.
//
// Example:
//
//	bg, _ := tui.DetectTerminalBackground(0)
//	theme := darkTheme
//	if bg == tui.TerminalBackgroundLight {
//		theme = lightTheme
//	}
func DetectTerminalBackground(timeout time.Duration) (TerminalBackground, error) {
	if timeout <= 0 {
		timeout = defaultBackgroundTimeout
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return TerminalBackgroundUnknown, nil
	}
	var err error
	if term.IsTerminal(int(os.Stdin.Fd())) && !skipBackgroundQuery(os.Getenv("TERM")) {
		var color RGB
		var ok bool
		color, ok, err = queryBackground(os.Stdin, os.Stdout, timeout)
		if ok {
			return backgroundOf(color), nil
		}
	}
	return colorFGBGBackground(os.Getenv("COLORFGBG")), err
}

func skipBackgroundQuery(termVar string) bool {
	return termVar == "" || termVar == "dumb"
}

// backgroundOf classifies a color by its luminance. 0.18 is CIE lightness
// 50, the perceived midpoint between black and white.
func backgroundOf(c RGB) TerminalBackground {
	if c.Luminance() < 0.18 {
		return TerminalBackgroundDark
	}
	return TerminalBackgroundLight
}

// colorFGBGBackground reads COLORFGBG, which some terminals (rxvt, Konsole,
// iTerm2) set to "fg;bg" or "fg;default;bg" using ANSI color numbers.
func colorFGBGBackground(value string) TerminalBackground {
	if value == "" {
		return TerminalBackgroundUnknown
	}
	fields := strings.Split(value, ";")
	bg, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil || bg < 0 || bg > 15 {
		return TerminalBackgroundUnknown
	}
	// 0-6 and 8 are black, dark colors, and dark grey; 7 and 9-15 are
	// light grey, bright colors, and white.
	if bg <= 6 || bg == 8 {
		return TerminalBackgroundDark
	}
	return TerminalBackgroundLight
}

// backgroundQuery asks for the background color, then for the primary device
// attributes (DA1). Every terminal answers DA1, and answers in order, so the
// DA1 reply marks the end of whatever the terminal will say.
const backgroundQuery = "\x1b]11;?\x07\x1b[c"

// backgroundReplies is what a terminal has said so far in reply to
// backgroundQuery.
type backgroundReplies struct {
	color RGB
	ok    bool // color holds the terminal's background color
	done  bool // the DA1 reply arrived, so no more replies will come
}

func scanBackgroundReplies(b []byte) backgroundReplies {
	var r backgroundReplies
	r.color, r.ok = parseBackgroundReply(b)
	r.done = hasDeviceAttributesReply(b)
	return r
}

// hasDeviceAttributesReply reports whether b holds a DA1 reply, ESC [ ? ... c.
func hasDeviceAttributesReply(b []byte) bool {
	for {
		i := bytes.Index(b, []byte("\x1b[?"))
		if i < 0 {
			return false
		}
		b = b[i+3:]
		j := 0
		for j < len(b) && (b[j] >= '0' && b[j] <= '9' || b[j] == ';') {
			j++
		}
		if j < len(b) && b[j] == 'c' {
			return true
		}
	}
}

// parseBackgroundReply finds an OSC 11 reply in b, ESC ] 11 ; <color>
// followed by BEL or ST, and returns its color.
func parseBackgroundReply(b []byte) (RGB, bool) {
	for {
		i := bytes.Index(b, []byte("\x1b]11;"))
		if i < 0 {
			return RGB{}, false
		}
		b = b[i+5:]
		end := bytes.IndexAny(b, "\x07\x1b\x9c")
		if end < 0 {
			return RGB{}, false
		}
		if b[end] == 0x1b && (end+1 >= len(b) || b[end+1] != '\\') {
			continue // an ESC that does not start ST: not a well-formed reply
		}
		if c, ok := parseXColor(string(b[:end])); ok {
			return c, true
		}
	}
}

// parseXColor parses the X11 color forms terminals use in OSC replies:
// rgb:R/G/B and rgba:R/G/B/A, where each component is 1 to 4 hex digits.
func parseXColor(s string) (RGB, bool) {
	var parts []string
	switch {
	case strings.HasPrefix(s, "rgb:"):
		parts = strings.Split(s[4:], "/")
		if len(parts) != 3 {
			return RGB{}, false
		}
	case strings.HasPrefix(s, "rgba:"):
		parts = strings.Split(s[5:], "/")
		if len(parts) != 4 {
			return RGB{}, false
		}
	default:
		return RGB{}, false
	}
	var channels [3]uint8
	for i := range channels {
		p := parts[i]
		if len(p) < 1 || len(p) > 4 {
			return RGB{}, false
		}
		v, err := strconv.ParseUint(p, 16, 16)
		if err != nil {
			return RGB{}, false
		}
		// Scale n hex digits to 8 bits: "f", "ff", "fff", "ffff" are all 255.
		top := uint64(1)<<(4*len(p)) - 1
		channels[i] = uint8((v*255 + top/2) / top)
	}
	return RGB{R: channels[0], G: channels[1], B: channels[2]}, true
}
