//go:build darwin

package clipboard

import (
	"context"
	"strings"
)

// read implements clipboard reading for macOS using pbpaste.
func read(ctx context.Context) (string, error) {
	out, err := runCommand(ctx, "pbpaste")
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

// write implements clipboard writing for macOS using pbcopy.
func write(ctx context.Context, text string) error {
	return runCommandWithStdin(ctx, text, "pbcopy")
}
