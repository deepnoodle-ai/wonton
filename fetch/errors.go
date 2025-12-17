package fetch

import (
	"errors"
	"fmt"
)

// ErrUnsupported is returned when an unsupported fetch option is requested.
//
// This sentinel error can be checked using errors.Is to determine if a failure
// was due to an unsupported feature. Use ErrUnsupportedOption to create errors
// that wrap this sentinel.
var ErrUnsupported = errors.New("unsupported option")

// ErrUnsupportedOption returns an error indicating the given option is not supported.
//
// The returned error wraps ErrUnsupported, so it can be checked with errors.Is.
//
// Example:
//
//	err := fetch.ErrUnsupportedOption("Mobile")
//	if errors.Is(err, fetch.ErrUnsupported) {
//		// Handle unsupported option
//	}
func ErrUnsupportedOption(option string) error {
	return fmt.Errorf("%w: %s", ErrUnsupported, option)
}

// RequestError represents an error that occurred during a fetch request.
//
// RequestError enriches a base error with additional context like HTTP status
// code and URL. It supports method chaining for setting optional fields and
// implements the error unwrapping interface.
//
// Example:
//
//	err := fetch.NewRequestErrorf("failed to parse HTML: %w", parseErr).
//		WithStatusCode(200).
//		WithRawURL("https://example.com")
type RequestError struct {
	err        error
	statusCode int
	rawURL     string
}

// StatusCode returns the HTTP status code associated with this error.
//
// Returns 0 if no status code was set.
func (r *RequestError) StatusCode() int {
	return r.statusCode
}

// RawURL returns the URL that was being fetched when the error occurred.
//
// Returns an empty string if no URL was set.
func (r *RequestError) RawURL() string {
	return r.rawURL
}

// Error implements the error interface, returning the underlying error message.
func (r *RequestError) Error() string {
	return r.err.Error()
}

// Unwrap returns the underlying error, supporting error unwrapping with errors.Is/As.
func (r *RequestError) Unwrap() error {
	return r.err
}

// NewRequestError creates a new RequestError wrapping the given error.
//
// Use the WithStatusCode and WithRawURL methods to add additional context.
func NewRequestError(err error) *RequestError {
	return &RequestError{err: err}
}

// NewRequestErrorf creates a new RequestError with a formatted error message.
//
// This is a convenience function that combines fmt.Errorf with NewRequestError.
// Use the WithStatusCode and WithRawURL methods to add additional context.
func NewRequestErrorf(format string, args ...any) *RequestError {
	return &RequestError{err: fmt.Errorf(format, args...)}
}

// WithStatusCode sets the HTTP status code on the error and returns the error.
//
// This method supports method chaining.
func (r *RequestError) WithStatusCode(statusCode int) *RequestError {
	r.statusCode = statusCode
	return r
}

// WithRawURL sets the URL on the error and returns the error.
//
// This method supports method chaining.
func (r *RequestError) WithRawURL(rawURL string) *RequestError {
	r.rawURL = rawURL
	return r
}

// IsRequestError checks if the given error is a RequestError.
//
// Returns true if err is a *RequestError, false otherwise (including when err is nil).
func IsRequestError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*RequestError)
	return ok
}
