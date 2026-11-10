package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// isTerminal reports whether f is a terminal. It is a variable so tests can
// simulate a terminal.
var isTerminal = func(f *os.File) bool {
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
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// The parse helpers below convert a command-line, environment or default
// value to a typed value. Their errors are plain sentences about the value,
// such as `"abc" is not a whole number`, so Go's internal error text never
// reaches the user.

// parseInt parses s, ignoring surrounding space, as an int.
func parseInt(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	return n, valueError(s, err)
}

// parseInt64 parses s, ignoring surrounding space, as an int64.
func parseInt64(s string) (int64, error) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n, valueError(s, err)
}

// parseFloat parses s, ignoring surrounding space, as a float64.
func parseFloat(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f, valueError(s, err)
}

// parseDuration parses s, ignoring surrounding space, as a time.Duration.
func parseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("%q is not a duration (e.g. 30s, 5m)", s)
	}
	return d, nil
}

// valueError rewrites a strconv error for the value s as a plain sentence.
// It returns nil for a nil error and leaves other errors unchanged.
func valueError(s string, err error) error {
	if err == nil {
		return nil
	}
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		return err
	}
	if errors.Is(numErr.Err, strconv.ErrRange) {
		return fmt.Errorf("%q is out of range", s)
	}
	if numErr.Func == "ParseFloat" {
		return fmt.Errorf("%q is not a number", s)
	}
	return fmt.Errorf("%q is not a whole number", s)
}
