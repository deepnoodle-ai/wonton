package cli

import (
	"fmt"
	"strings"
)

// Error types for CLI operations

// HelpRequested indicates help was shown (not an error).
type HelpRequested struct{}

func (e *HelpRequested) Error() string {
	return "help requested"
}

// ExitError indicates the command should exit with a specific code.
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	return e.Message
}

// Exit returns an error that causes the CLI to exit with the given code.
func Exit(code int) error {
	return &ExitError{Code: code}
}

// CommandError is a rich error with hints and codes.
type CommandError struct {
	message string
	hint    string
	code    string
	details []string
}

// Error creates a new command error.
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
func IsHelpRequested(err error) bool {
	_, ok := err.(*HelpRequested)
	return ok
}

// GetExitCode returns the exit code for an error.
func GetExitCode(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(*ExitError); ok {
		return e.Code
	}
	if IsHelpRequested(err) {
		return 0
	}
	return 1
}
