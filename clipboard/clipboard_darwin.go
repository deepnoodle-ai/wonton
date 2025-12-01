//go:build darwin

package clipboard

import (
	"context"
	"strings"
)

func read(ctx context.Context) (string, error) {
	out, err := runCommand(ctx, "pbpaste")
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(out), "\n"), nil
}

func write(ctx context.Context, text string) error {
	return runCommandWithStdin(ctx, text, "pbcopy")
}
