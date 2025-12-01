//go:build linux

package clipboard

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

func read(ctx context.Context) (string, error) {
	// Check for Wayland
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-paste"); err == nil {
			out, err := runCommand(ctx, "wl-paste", "--no-newline")
			if err != nil {
				return "", err
			}
			return string(out), nil
		}
	}

	// Try xclip first
	if _, err := exec.LookPath("xclip"); err == nil {
		out, err := runCommand(ctx, "xclip", "-selection", "clipboard", "-o")
		if err != nil {
			return "", err
		}
		return strings.TrimSuffix(string(out), "\n"), nil
	}

	// Try xsel
	if _, err := exec.LookPath("xsel"); err == nil {
		out, err := runCommand(ctx, "xsel", "--clipboard", "--output")
		if err != nil {
			return "", err
		}
		return strings.TrimSuffix(string(out), "\n"), nil
	}

	return "", ErrUnavailable
}

func write(ctx context.Context, text string) error {
	// Check for Wayland
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return runCommandWithStdin(ctx, text, "wl-copy")
		}
	}

	// Try xclip first
	if _, err := exec.LookPath("xclip"); err == nil {
		return runCommandWithStdin(ctx, text, "xclip", "-selection", "clipboard")
	}

	// Try xsel
	if _, err := exec.LookPath("xsel"); err == nil {
		return runCommandWithStdin(ctx, text, "xsel", "--clipboard", "--input")
	}

	return ErrUnavailable
}
