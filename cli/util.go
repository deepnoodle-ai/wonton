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
//
// Use this to detect if your program is running in an interactive environment:
//
//	if cli.IsTTY() {
//	    // Show interactive prompts
//	} else {
//	    // Use non-interactive mode
//	}
func IsTTY() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

// IsPiped returns true if stdin is being piped from another command.
//
// Use this to detect piped input:
//
//	if cli.IsPiped() {
//	    // Read from stdin
//	    scanner := bufio.NewScanner(os.Stdin)
//	}
func IsPiped() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
