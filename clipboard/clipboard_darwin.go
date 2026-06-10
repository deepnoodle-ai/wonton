//go:build darwin

package clipboard

import (
	"context"
)

// read implements clipboard reading for macOS using pbpaste.
// pbpaste outputs the clipboard contents verbatim, so no trimming is needed;
// trimming would corrupt content that legitimately ends with a newline.
func read(ctx context.Context) (string, error) {
	out, err := runCommand(ctx, "pbpaste")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// write implements clipboard writing for macOS using pbcopy.
func write(ctx context.Context, text string) error {
	return runCommandWithStdin(ctx, text, "pbcopy")
}
