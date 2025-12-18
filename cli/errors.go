package cli

import (
	"errors"
	"fmt"
	"strings"
)

// This file defines error types for CLI operations.

// HelpRequested indicates help was shown to the user (not a true error).
//
// This is returned when the user passes --help or -h, or when help is
// explicitly displayed. Check for it with IsHelpRequested:
//
//	if cli.IsHelpRequested(err) {
//	    os.Exit(0)
//	}
type HelpRequested struct{}

func (e *HelpRequested) Error() string {
	return "help requested"
}

// ExitError indicates the command should exit with a specific exit code.
//
// Use Exit to create an ExitError:
//
//	return cli.Exit(2)  // Exit with code 2
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

// Exit returns an error that causes the CLI to exit with the given code.
func Exit(code int) error {
	return &ExitError{Code: code, Message: fmt.Sprintf("exit status %d", code)}
}

// CommandError is a rich error with hints, details, and error codes.
//
// Use Error or Errorf to create a CommandError, then add hints and details:
//
//	return cli.Errorf("failed to connect to %s", host).
//	    Hint("Check your network connection and firewall settings").
//	    Detail("Timeout: %s", timeout).
//	    Code("ERR_CONNECTION")
type CommandError struct {
	message string
	hint    string
	code    string
	details []string
}

// Error creates a new command error with the given message.
//
//	return cli.Error("configuration file not found")
func Error(message string) *CommandError {
	return &CommandError{message: message}
}

// Errorf creates a new command error with formatting.
func Errorf(format string, args ...any) *CommandError {
	return &CommandError{message: fmt.Sprintf(format, args...)}
}

func (e *CommandError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.message)
	if e.hint != "" {
		sb.WriteString("\n\nHint: ")
		sb.WriteString(e.hint)
	}
	if len(e.details) > 0 {
		sb.WriteString("\n\nDetails:\n")
		for _, d := range e.details {
			sb.WriteString("  - ")
			sb.WriteString(d)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// Hint adds a hint to the error.
func (e *CommandError) Hint(h string) *CommandError {
	e.hint = h
	return e
}

// Code sets the error code.
func (e *CommandError) Code(c string) *CommandError {
	e.code = c
	return e
}

// Detail adds a detail line.
func (e *CommandError) Detail(format string, args ...any) *CommandError {
	e.details = append(e.details, fmt.Sprintf(format, args...))
	return e
}

// ErrorCode returns the error code if set.
func (e *CommandError) ErrorCode() string {
	return e.code
}

// IsHelpRequested checks if the error is a help request.
// This function supports wrapped errors via errors.As.
func IsHelpRequested(err error) bool {
	var helpErr *HelpRequested
	return errors.As(err, &helpErr)
}

// GetExitCode returns the appropriate exit code for an error.
// This function supports wrapped errors via errors.As.
//
// Returns:
//   - 0 if err is nil or HelpRequested
//   - The code from ExitError if err is an ExitError
//   - 1 for all other errors
func GetExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	if IsHelpRequested(err) {
		return 0
	}
	return 1
}
