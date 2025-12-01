package env

import (
	"errors"
	"fmt"
	"strings"
)

// AggregateError collects multiple parsing errors.
type AggregateError struct {
	Errors []error
}

func (e *AggregateError) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%d errors:\n", len(e.Errors))
	for _, err := range e.Errors {
		fmt.Fprintf(&b, "  - %s\n", err.Error())
	}
	return b.String()
}

// Unwrap returns the list of errors for errors.Is/As support.
func (e *AggregateError) Unwrap() []error {
	return e.Errors
}

// Is implements errors.Is support.
func (e *AggregateError) Is(target error) bool {
	for _, err := range e.Errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

// ParseError represents a general parsing error.
type ParseError struct {
	Err error
}

func (e *ParseError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("parse error: %v", e.Err)
	}
	return "parse error"
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// FieldError represents an error parsing a specific field.
type FieldError struct {
	Field  string
	EnvVar string
	Value  string
	Err    error
}

func (e *FieldError) Error() string {
	if e.EnvVar != "" && e.Value != "" {
		return fmt.Sprintf("field %s (%s): cannot parse %q: %v", e.Field, e.EnvVar, e.Value, e.Err)
	}
	if e.EnvVar != "" {
		return fmt.Sprintf("field %s (%s): %v", e.Field, e.EnvVar, e.Err)
	}
	return fmt.Sprintf("field %s: %v", e.Field, e.Err)
}

func (e *FieldError) Unwrap() error {
	return e.Err
}

// VarNotSetError indicates a required environment variable is not set.
type VarNotSetError struct {
	Field  string
	EnvVar string
}

func (e *VarNotSetError) Error() string {
	return fmt.Sprintf("required variable %s not set (field: %s)", e.EnvVar, e.Field)
}

// EmptyVarError indicates an environment variable is set but empty.
type EmptyVarError struct {
	Field  string
	EnvVar string
}

func (e *EmptyVarError) Error() string {
	return fmt.Sprintf("variable %s is empty (field: %s)", e.EnvVar, e.Field)
}

// FileLoadError indicates an error loading content from a file.
type FileLoadError struct {
	Field    string
	EnvVar   string
	Filename string
	Err      error
}

func (e *FileLoadError) Error() string {
	return fmt.Sprintf("field %s (%s): cannot load file %q: %v", e.Field, e.EnvVar, e.Filename, e.Err)
}

func (e *FileLoadError) Unwrap() error {
	return e.Err
}

// HasError checks if an error contains a specific error type.
func HasError[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}

// GetErrors extracts all errors of a specific type from an AggregateError.
func GetErrors[T error](err error) []T {
	var result []T
	var agg *AggregateError
	if errors.As(err, &agg) {
		for _, e := range agg.Errors {
			var target T
			if errors.As(e, &target) {
				result = append(result, target)
			}
		}
	}
	return result
}
