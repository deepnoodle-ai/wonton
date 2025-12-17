package fetch

import (
	"errors"
	"fmt"
)

// ErrUnsupported is returned when an unsupported option is requested.
var ErrUnsupported = errors.New("unsupported option")

// ErrUnsupportedOption returns an error indicating the given option is not supported.
func ErrUnsupportedOption(option string) error {
	return fmt.Errorf("%w: %s", ErrUnsupported, option)
}

// RequestError represents an error that occurred during a fetch request.
type RequestError struct {
	err        error
	statusCode int
	rawURL     string
}

// StatusCode returns the HTTP status code, if available.
func (r *RequestError) StatusCode() int {
	return r.statusCode
}

// RawURL returns the URL that was being fetched.
func (r *RequestError) RawURL() string {
	return r.rawURL
}

// Error implements the error interface.
func (r *RequestError) Error() string {
	return r.err.Error()
}

// Unwrap returns the underlying error.
func (r *RequestError) Unwrap() error {
	return r.err
}

// NewRequestError creates a new RequestError wrapping the given error.
func NewRequestError(err error) *RequestError {
	return &RequestError{err: err}
}

// NewRequestErrorf creates a new RequestError with a formatted message.
func NewRequestErrorf(format string, args ...any) *RequestError {
	return &RequestError{err: fmt.Errorf(format, args...)}
}

// WithStatusCode sets the HTTP status code on the error.
func (r *RequestError) WithStatusCode(statusCode int) *RequestError {
	r.statusCode = statusCode
	return r
}

// WithRawURL sets the URL on the error.
func (r *RequestError) WithRawURL(rawURL string) *RequestError {
	r.rawURL = rawURL
	return r
}

// IsRequestError checks if the given error is a RequestError.
func IsRequestError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*RequestError)
	return ok
}
