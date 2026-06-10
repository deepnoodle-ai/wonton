package color

import (
	"os"

	"github.com/deepnoodle-ai/wonton/tty"
)

// Enabled controls whether the high-level helpers (Apply, ApplyBg,
// ApplyDim, ApplyBold, Sprint, Sprintf, and ApplyGradient) emit ANSI escape
// sequences. It is initialized automatically at startup, following the
// common precedence:
//
//  1. FORCE_COLOR / CLICOLOR_FORCE (non-empty, non-"0"): force on
//  2. NO_COLOR (non-empty) or CLICOLOR=0: force off
//  3. Auto-detect: on iff os.Stdout is a terminal
//
// Override it at startup to take manual control, for example from a
// command-line flag, or to base detection on a different stream:
//
//	color.Enabled = color.ShouldColorize(os.Stderr)
//
// Enabled is not synchronized; set it during program initialization, before
// any goroutines that render output are started.
//
// The low-level sequence functions (ForegroundSeq, BackgroundSeq, and the
// SGR code functions) ignore Enabled and always produce escape sequences.
var Enabled = ShouldColorize(os.Stdout)

// ShouldColorize reports whether colors should be used for the given output
// file, following the common precedence:
//
//  1. FORCE_COLOR / CLICOLOR_FORCE (non-empty, non-"0"): force on
//  2. NO_COLOR (non-empty) or CLICOLOR=0: force off
//  3. Auto-detect: true iff f is a terminal
//
// The Enabled variable is initialized with ShouldColorize(os.Stdout). Call
// this directly when making per-stream decisions, e.g. coloring stderr
// while stdout is piped.
//
// Example:
//
//	color.Enabled = color.ShouldColorize(os.Stderr)
func ShouldColorize(f *os.File) bool {
	if envForceColor() {
		return true
	}
	if envNoColor() {
		return false
	}
	return tty.IsTerminal(f)
}

// envNoColor reports whether NO_COLOR or CLICOLOR=0 is set in a way that
// disables colored output.
//
// NO_COLOR spec (https://no-color.org/): any non-empty value disables color.
// CLICOLOR spec (https://bixense.com/clicolors/): CLICOLOR=0 disables color.
func envNoColor() bool {
	if v, ok := os.LookupEnv("NO_COLOR"); ok && v != "" {
		return true
	}
	if v, ok := os.LookupEnv("CLICOLOR"); ok && v == "0" {
		return true
	}
	return false
}

// envForceColor reports whether FORCE_COLOR or CLICOLOR_FORCE is set in a
// way that forces colored output, per common cross-tool conventions.
// An empty value or "0" does not force.
func envForceColor() bool {
	if v, ok := os.LookupEnv("FORCE_COLOR"); ok && v != "" && v != "0" {
		return true
	}
	if v, ok := os.LookupEnv("CLICOLOR_FORCE"); ok && v != "" && v != "0" {
		return true
	}
	return false
}
