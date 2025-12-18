// Package clipboard provides cross-platform system clipboard access for reading
// and writing text content. It works seamlessly on macOS, Linux (both X11 and
// Wayland), and Windows by leveraging native clipboard utilities.
//
// # Basic Usage
//
// The simplest way to interact with the clipboard is using the Read and Write functions:
//
//	// Write text to clipboard
//	err := clipboard.Write("Hello, World!")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Read text from clipboard
//	text, err := clipboard.Read()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(text)
//
// # Context Support
//
// For fine-grained control over timeouts and cancellation, use the context-aware functions:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
//	defer cancel()
//
//	err := clipboard.WriteContext(ctx, "data")
//	if err == clipboard.ErrTimeout {
//		fmt.Println("operation timed out")
//	}
//
// # Platform Support
//
// The package automatically selects the appropriate clipboard implementation based
// on the operating system:
//
//   - macOS: Uses pbcopy and pbpaste commands
//   - Linux X11: Uses xclip or xsel (tries both in order of preference)
//   - Linux Wayland: Uses wl-copy and wl-paste commands
//   - Windows: Uses PowerShell Get-Clipboard and clip.exe commands
//   - Other platforms: Returns ErrUnavailable
//
// Use the Available function to check if clipboard functionality is supported:
//
//	if !clipboard.Available() {
//		log.Fatal("clipboard not available on this system")
//	}
//
// # Error Handling
//
// The package defines two sentinel errors for common failure cases:
//
//   - ErrUnavailable: Returned when clipboard tools are not found on the system
//   - ErrTimeout: Returned when an operation exceeds its timeout duration
//
// Default operations timeout after 5 seconds, but this can be customized using
// the WithTimeout or Context variants of each function.
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

// ErrUnavailable is returned when clipboard access is not available on the current
// system. This typically occurs when required clipboard utilities are not installed
// (e.g., xclip/xsel on Linux) or the platform is unsupported.
var ErrUnavailable = errors.New("clipboard: not available on this system")

// ErrTimeout is returned when a clipboard operation exceeds its timeout duration.
// The default timeout is 5 seconds, but can be customized using ReadWithTimeout,
// WriteWithTimeout, or the Context variants.
var ErrTimeout = errors.New("clipboard: operation timed out")

// defaultTimeout is the default timeout for clipboard operations.
const defaultTimeout = 5 * time.Second

// Read reads text from the system clipboard with the default timeout (5 seconds).
// Returns the clipboard contents as a string, or an error if the read fails.
//
// Common errors include ErrUnavailable (clipboard tools not found) and ErrTimeout
// (operation took too long). For custom timeouts, use ReadWithTimeout or ReadContext.
func Read() (string, error) {
	return ReadWithTimeout(defaultTimeout)
}

// ReadWithTimeout reads from the clipboard with a custom timeout duration.
// This is useful when you need to control how long the operation can take.
//
// Example:
//
//	text, err := clipboard.ReadWithTimeout(2 * time.Second)
//	if err == clipboard.ErrTimeout {
//		fmt.Println("clipboard read timed out")
//	}
func ReadWithTimeout(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return ReadContext(ctx)
}

// ReadContext reads from the clipboard with full context support for cancellation
// and deadline management. This is the most flexible read function, allowing
// integration with existing context hierarchies.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
//	defer cancel()
//
//	text, err := clipboard.ReadContext(ctx)
//	if err != nil {
//		return err
//	}
func ReadContext(ctx context.Context) (string, error) {
	return read(ctx)
}

// Write writes text to the system clipboard with the default timeout (5 seconds).
// Returns an error if the write operation fails.
//
// Common errors include ErrUnavailable (clipboard tools not found) and ErrTimeout
// (operation took too long). For custom timeouts, use WriteWithTimeout or WriteContext.
func Write(text string) error {
	return WriteWithTimeout(text, defaultTimeout)
}

// WriteWithTimeout writes to the clipboard with a custom timeout duration.
// This is useful when you need to control how long the operation can take.
//
// Example:
//
//	err := clipboard.WriteWithTimeout("data", 2*time.Second)
//	if err == clipboard.ErrTimeout {
//		fmt.Println("clipboard write timed out")
//	}
func WriteWithTimeout(text string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return WriteContext(ctx, text)
}

// WriteContext writes to the clipboard with full context support for cancellation
// and deadline management. This is the most flexible write function, allowing
// integration with existing context hierarchies.
//
// Example:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	err := clipboard.WriteContext(ctx, "important data")
//	if err != nil {
//		return err
//	}
func WriteContext(ctx context.Context, text string) error {
	return write(ctx, text)
}

// Available returns true if clipboard functionality is available on the current system.
// It checks for the presence of required clipboard utilities based on the operating system.
//
// On Linux, it checks for xclip, xsel (X11), or wl-copy/wl-paste (Wayland).
// On macOS, it checks for pbcopy/pbpaste.
// On Windows, it always returns true (PowerShell is assumed to be available).
// On other platforms, it returns false.
//
// Use this function before attempting clipboard operations to provide better error
// messages or fallback behavior:
//
//	if !clipboard.Available() {
//		fmt.Println("Please install xclip or xsel for clipboard support")
//		return
//	}
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

// Clear clears the clipboard by writing an empty string to it.
// Returns an error if the clear operation fails.
//
// This is equivalent to calling Write("") and is provided as a convenience function.
func Clear() error {
	return Write("")
}

// runCommand executes a command with context and returns stdout.
func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

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
