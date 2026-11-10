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
	cause   error
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

// Wrap attaches an underlying error to this CommandError.
func (e *CommandError) Wrap(err error) *CommandError {
	e.cause = err
	return e
}

// Unwrap returns the underlying error, implementing the errors.Unwrap interface.
func (e *CommandError) Unwrap() error { return e.cause }

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

// UsageError reports that the program was invoked incorrectly: an unknown
// flag, command or subcommand, a missing or invalid value, a missing or
// unexpected argument, or a missing required flag.
//
// The package returns a UsageError for every error it finds while it parses
// and validates the command line. Errors returned by handlers are not usage
// errors. GetExitCode returns 2 for a UsageError, which is the Unix
// convention for "called incorrectly". Check for one with IsUsageError.
type UsageError struct {
	// Message is the complete error text, including any hint that follows
	// a blank line. When it is empty, Error returns Err's text.
	Message string

	// Err is the underlying error, if any. errors.Is and errors.As see it
	// through Unwrap.
	Err error
}

func (e *UsageError) Error() string {
	if e.Message == "" && e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error, if any.
func (e *UsageError) Unwrap() error {
	return e.Err
}

// asUsageError marks an error returned by a validator as a usage error.
// Errors that already carry an exit status (ExitError, UsageError) or ask
// for help are returned unchanged.
func asUsageError(err error) error {
	if err == nil || IsUsageError(err) || IsHelpRequested(err) {
		return err
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return err
	}
	return &UsageError{Err: err}
}

// usageErrorf returns a UsageError with a formatted message.
func usageErrorf(format string, args ...any) error {
	return &UsageError{Message: fmt.Sprintf(format, args...)}
}

// IsUsageError reports whether err is, or wraps, a UsageError.
func IsUsageError(err error) bool {
	var usageErr *UsageError
	return errors.As(err, &usageErr)
}

// GetExitCode returns the appropriate exit code for an error.
// This function supports wrapped errors via errors.As.
//
// Returns:
//   - 0 if err is nil or HelpRequested
//   - The code from ExitError if err is an ExitError
//   - 2 if err is a UsageError (the command line was invalid)
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
	if IsUsageError(err) {
		return 2
	}
	return 1
}
