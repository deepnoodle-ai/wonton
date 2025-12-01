// Package clipboard provides cross-platform system clipboard access.
// It supports reading from and writing to the system clipboard on macOS, Linux, and Windows.
//
// Basic usage:
//
//	// Copy to clipboard
//	err := clipboard.Write("Hello, World!")
//
//	// Read from clipboard
//	text, err := clipboard.Read()
//
// The package uses native clipboard utilities:
//   - macOS: pbcopy/pbpaste
//   - Linux: xclip or xsel (with X11), wl-copy/wl-paste (Wayland)
//   - Windows: PowerShell clip.exe / Get-Clipboard
package clipboard

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ErrUnavailable is returned when clipboard access is not available.
var ErrUnavailable = errors.New("clipboard: not available on this system")

// ErrTimeout is returned when a clipboard operation times out.
var ErrTimeout = errors.New("clipboard: operation timed out")

// defaultTimeout is the default timeout for clipboard operations.
const defaultTimeout = 5 * time.Second

// Read reads text from the system clipboard.
func Read() (string, error) {
	return ReadWithTimeout(defaultTimeout)
}

// ReadWithTimeout reads from the clipboard with a custom timeout.
func ReadWithTimeout(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return ReadContext(ctx)
}

// ReadContext reads from the clipboard with context support.
func ReadContext(ctx context.Context) (string, error) {
	return read(ctx)
}

// Write writes text to the system clipboard.
func Write(text string) error {
	return WriteWithTimeout(text, defaultTimeout)
}

// WriteWithTimeout writes to the clipboard with a custom timeout.
func WriteWithTimeout(text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return WriteContext(ctx, text)
}

// WriteContext writes to the clipboard with context support.
func WriteContext(ctx context.Context, text string) error {
	return write(ctx, text)
}

// Available returns true if clipboard functionality is available on this system.
func Available() bool {
	switch runtime.GOOS {
	case "darwin":
		_, err := exec.LookPath("pbcopy")
		return err == nil
	case "linux":
		// Check for X11 clipboard tools
		if _, err := exec.LookPath("xclip"); err == nil {
			return true
		}
		if _, err := exec.LookPath("xsel"); err == nil {
			return true
		}
		// Check for Wayland clipboard tools
		if _, err := exec.LookPath("wl-copy"); err == nil {
			return true
		}
		return false
	case "windows":
		return true // PowerShell is always available on Windows
	default:
		return false
	}
}

// Clear clears the clipboard contents.
func Clear() error {
	return Write("")
}

// runCommand executes a command with context and returns combined output.
func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return nil, ErrTimeout
	}
	if err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

// runCommandWithStdin executes a command with stdin input.
func runCommandWithStdin(ctx context.Context, input string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(input)

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return ErrTimeout
	}
	return err
}
