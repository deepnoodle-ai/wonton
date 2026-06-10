package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"syscall"
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

// Error represents a failed HTTP fetch. It preserves the HTTP status code
// and target URL so callers can decide how to handle the failure, and
// supports error unwrapping via [errors.Is] and [errors.As].
//
// Example:
//
//	var fetchErr *fetch.Error
//	if errors.As(err, &fetchErr) && fetchErr.StatusCode == 404 {
//	    // handle missing page
//	}
type Error struct {
	// StatusCode is the HTTP status code returned by the server, if the
	// request got far enough to receive a response. Zero otherwise.
	StatusCode int

	// URL is the URL that was being fetched.
	URL string

	// Err is the underlying error, if any.
	Err error
}

// Error returns a string representation of the fetch error, including the
// URL, status code, and underlying error when present.
func (e *Error) Error() string {
	msg := "fetch failed"
	if e.URL != "" {
		msg = "fetch " + e.URL + " failed"
	}
	if e.StatusCode != 0 {
		msg += fmt.Sprintf(": status %d %s", e.StatusCode, http.StatusText(e.StatusCode))
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	return msg
}

// Unwrap returns the underlying error, enabling use with [errors.Is] and
// [errors.As].
func (e *Error) Unwrap() error {
	return e.Err
}

// Retryable returns true if the error represents a temporary failure that
// might succeed on retry.
//
// The following status codes are considered retryable:
//   - 408 Request Timeout
//   - 429 Too Many Requests: rate limiting, retry after backoff
//   - 500 Internal Server Error: transient server issue
//   - 502 Bad Gateway: upstream server issue
//   - 503 Service Unavailable: temporary overload or maintenance
//   - 504 Gateway Timeout: upstream timeout
//
// Other client errors and permanent server errors are not retryable.
func (e *Error) Retryable() bool {
	switch e.StatusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}
	return false
}

// IsRetryable reports whether a fetch failure is worth retrying. It handles
// both HTTP-level failures ([Error] with a retryable status code) and
// transport-level failures (timeouts, connection resets, DNS timeouts),
// which arrive as errors from the HTTP client rather than as [Error] values.
//
// Pairs well with the retry package:
//
//	resp, err := retry.Do(ctx, func() (*fetch.Response, error) {
//	    return fetcher.Fetch(ctx, req)
//	}, retry.WithRetryIf(fetch.IsRetryable))
//
// Context cancellation is never retryable.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	var fetchErr *Error
	if errors.As(err, &fetchErr) {
		return fetchErr.Retryable()
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsTimeout || dnsErr.IsTemporary
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return false
}
