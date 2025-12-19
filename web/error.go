package web

import "fmt"

// FetchError represents an HTTP fetch failure with status code information.
//
// This error type wraps the underlying error while preserving the HTTP status
// code, enabling callers to make decisions based on the type of failure.
// It implements the standard error interface and supports error unwrapping
// via [errors.Unwrap].
//
// Example:
//
//	resp, err := http.Get(url)
//	if err != nil {
//	    return nil, err
//	}
//	if resp.StatusCode >= 400 {
//	    return nil, web.NewFetchError(resp.StatusCode, fmt.Errorf("request failed"))
//	}
type FetchError struct {
	// StatusCode is the HTTP status code returned by the server.
	StatusCode int

	// Err is the underlying error describing the failure.
	Err error
}

// NewFetchError creates a new FetchError with the given HTTP status code and
// underlying error.
func NewFetchError(statusCode int, err error) *FetchError {
	return &FetchError{StatusCode: statusCode, Err: err}
}

// Error returns a string representation of the fetch error, including the
// status code and underlying error message.
func (e *FetchError) Error() string {
	return fmt.Sprintf("fetch failed with status code %d: %s", e.StatusCode, e.Err)
}

// Unwrap returns the underlying error, enabling use with [errors.Is] and
// [errors.As].
func (e *FetchError) Unwrap() error {
	return e.Err
}

// IsRecoverable returns true if the error represents a temporary failure that
// might succeed on retry.
//
// The following status codes are considered recoverable:
//   - 429 Too Many Requests: Rate limiting, retry after backoff
//   - 500 Internal Server Error: Transient server issue
//   - 502 Bad Gateway: Upstream server issue
//   - 503 Service Unavailable: Temporary overload or maintenance
//   - 504 Gateway Timeout: Upstream timeout
//
// Client errors (4xx except 429) and permanent server errors are not
// considered recoverable.
func (e *FetchError) IsRecoverable() bool {
	return e.StatusCode == 429 || // Too Many Requests
		e.StatusCode == 500 || // Internal Server Error
		e.StatusCode == 502 || // Bad Gateway
		e.StatusCode == 503 || // Service Unavailable
		e.StatusCode == 504 // Gateway Timeout
}
