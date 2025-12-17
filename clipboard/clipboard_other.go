//go:build !darwin && !linux && !windows

package clipboard

import (
	"context"
)

// read is a stub implementation for unsupported platforms.
// It always returns ErrUnavailable.
func read(ctx context.Context) (string, error) {
	return "", ErrUnavailable
}

// write is a stub implementation for unsupported platforms.
// It always returns ErrUnavailable.
func write(ctx context.Context, text string) error {
	return ErrUnavailable
}
