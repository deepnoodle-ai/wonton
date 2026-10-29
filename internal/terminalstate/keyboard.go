package terminalstate

import (
	"os"
	"strings"
)

// Enhanced key reporting lets a terminal report keys such as Shift+Enter
// distinctly from Enter. Terminals that implement the Kitty keyboard protocol
// turn it on with a push and off with a pop.
//
// tmux (3.2 through at least 3.8) does not implement that protocol and ignores
// both sequences. It forwards modified keys only to a pane that asks for
// xterm's modifyOtherKeys, and only when its extended-keys option is on. So
// inside tmux the enable sequence also asks for modifyOtherKeys mode 1, which
// changes only keys that have no legacy encoding (Ctrl+letter still sends a
// control byte), and the release sequence resets it. tmux then sends
// Shift+Enter as ESC [ 27 ; 2 ; 13 ~, or as ESC [ 13 ; 2 u, both of which the
// terminal package decodes.
const (
	kittyKeyboardPush  = "\x1b[>1u"
	kittyKeyboardPop   = "\x1b[<u"
	modifyOtherKeysOn  = "\x1b[>4;1m"
	modifyOtherKeysOff = "\x1b[>4m"
)

// KeyboardEnable returns the sequence that turns on enhanced key reporting.
func KeyboardEnable() string {
	if InsideTmux() {
		return kittyKeyboardPush + modifyOtherKeysOn
	}
	return kittyKeyboardPush
}

// KeyboardRelease returns the sequence that undoes KeyboardEnable.
func KeyboardRelease() string {
	if InsideTmux() {
		return kittyKeyboardPop + modifyOtherKeysOff
	}
	return kittyKeyboardPop
}

// InsideTmux reports whether the program's terminal is a tmux pane. tmux sets
// TMUX and TERM_PROGRAM in its panes; TERM=tmux* also survives ssh. A pane
// whose TERM is screen* and that has neither variable is indistinguishable
// from GNU screen, so it counts as not tmux.
func InsideTmux() bool {
	return os.Getenv("TMUX") != "" ||
		os.Getenv("TERM_PROGRAM") == "tmux" ||
		strings.HasPrefix(os.Getenv("TERM"), "tmux")
}
