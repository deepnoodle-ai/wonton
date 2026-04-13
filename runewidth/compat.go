package runewidth

import "sync/atomic"

// terminalCompat flips a small set of width answers to match what terminals
// actually draw, for clusters where the Unicode spec and the terminal
// ecosystem disagree. See [SetTerminalCompatMode] for the full list.
//
// Read on the hot paths of [RuneWidth], [StringWidth], and [Graphemes]. The
// atomic load adds ~1ns and only runs for non-ASCII input (ASCII fast paths
// never touch the relevant code). Callers should set the flag once at
// startup — it is not intended to be toggled per-call.
var terminalCompat atomic.Bool

// SetTerminalCompatMode toggles a small set of width answers so that
// [RuneWidth], [StringWidth], [Graphemes], and everything built on top of
// them agrees with what a real terminal will draw, even when Unicode says
// otherwise.
//
// When enabled, the following cluster widths change:
//
//   - Emoji keycap sequences (`#⃣`, `*⃣`, `0⃣` … `9⃣`): width 1 instead of 2.
//     Unicode UTS #51 defines these as emoji presentation (width 2 in most
//     cluster-aware libraries), but the overwhelming majority of terminal
//     emulators do not composite base + VS16 + U+20E3 into a keycap glyph
//     and instead render just the base character at width 1.
//   - Two-em dash (U+2E3A): width 1 instead of 3.
//   - Three-em dash (U+2E3B): width 1 instead of 4.
//     Unicode assigns these visual widths of 3 and 4 for typographic
//     correctness, but every terminal emulator draws them as a single cell.
//
// Grapheme segmentation is unaffected: a keycap is still one cluster, not
// three. Only the reported display width changes.
//
// Call this once at startup if you are building a TUI whose layout math
// must match what the user sees on screen. The [examples/termprobe]
// command will tell you whether your target terminal needs it.
//
// Defaults to disabled (Unicode-strict).
func SetTerminalCompatMode(enabled bool) {
	terminalCompat.Store(enabled)
}

// TerminalCompatMode reports whether terminal-compat width mode is active.
// See [SetTerminalCompatMode] for what the mode changes.
func TerminalCompatMode() bool {
	return terminalCompat.Load()
}
