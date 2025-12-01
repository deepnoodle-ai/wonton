package cli

import (
	"os"

	"golang.org/x/term"
)

// isTerminal checks if a file is a terminal.
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// IsTTY returns true if both stdin and stdout are terminals.
func IsTTY() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

// IsPiped returns true if stdin is being piped.
func IsPiped() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
