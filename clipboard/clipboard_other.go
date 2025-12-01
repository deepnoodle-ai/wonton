//go:build !darwin && !linux && !windows

package clipboard

import (
	"context"
)

func read(ctx context.Context) (string, error) {
	return "", ErrUnavailable
}

func write(ctx context.Context, text string) error {
	return ErrUnavailable
}
