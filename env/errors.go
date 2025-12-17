package env

import (
	"errors"
	"fmt"
	"strings"
)

// AggregateError collects multiple parsing errors that occur during configuration parsing.
// It implements the error interface and provides unwrapping support for errors.Is and errors.As.
// When multiple fields fail to parse, all errors are collected and reported together.
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

// ParseError represents a general parsing error that occurs during
// configuration loading. It wraps underlying errors for additional context.
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

// FieldError represents an error parsing a specific field in the configuration struct.
// It includes the field name, environment variable name, attempted value, and underlying error.
type FieldError struct {
	Field  string // Name of the struct field
	EnvVar string // Environment variable name
	Value  string // Value that failed to parse
	Err    error  // Underlying error
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
// This occurs when a field is marked with the "required" tag option or
// when WithRequiredIfNoDefault is used and no default value is provided.
type VarNotSetError struct {
	Field  string // Name of the struct field
	EnvVar string // Environment variable that was not set
}

func (e *VarNotSetError) Error() string {
	return fmt.Sprintf("required variable %s not set (field: %s)", e.EnvVar, e.Field)
}

// EmptyVarError indicates an environment variable is set but empty when a non-empty
// value is required. This occurs when a field is marked with the "notEmpty" tag option.
type EmptyVarError struct {
	Field  string // Name of the struct field
	EnvVar string // Environment variable that was empty
}

func (e *EmptyVarError) Error() string {
	return fmt.Sprintf("variable %s is empty (field: %s)", e.EnvVar, e.Field)
}

// FileLoadError indicates an error loading content from a file.
// This occurs when a field is marked with the "file" tag option and the
// file path cannot be read.
type FileLoadError struct {
	Field    string // Name of the struct field
	EnvVar   string // Environment variable containing the file path
	Filename string // Path to the file that failed to load
	Err      error  // Underlying error from file read operation
}

func (e *FileLoadError) Error() string {
	return fmt.Sprintf("field %s (%s): cannot load file %q: %v", e.Field, e.EnvVar, e.Filename, e.Err)
}

func (e *FileLoadError) Unwrap() error {
	return e.Err
}

// HasError checks if an error contains a specific error type using errors.As.
// It's useful for checking if a particular error type exists in an error chain
// or within an AggregateError.
//
// Example:
//
//	if env.HasError[*env.VarNotSetError](err) {
//	    fmt.Println("Missing required variables")
//	}
func HasError[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}

// GetErrors extracts all errors of a specific type from an AggregateError.
// Returns an empty slice if the error is not an AggregateError or if no
// matching errors are found.
//
// Example:
//
//	varErrors := env.GetErrors[*env.VarNotSetError](err)
//	for _, e := range varErrors {
//	    fmt.Printf("Missing: %s\n", e.EnvVar)
//	}
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
