//go:build windows

package pty_test

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/deepnoodle-ai/wonton/pty"
)

// TestWindowsStubContract asserts that the Windows build pins every public
// entry point to ErrUnsupported. This is the contract callers rely on when
// feature-detecting PTY support at runtime.
func TestWindowsStubContract(t *testing.T) {
	if _, _, err := pty.Open(); !errors.Is(err, pty.ErrUnsupported) {
		t.Errorf("Open() err = %v, want ErrUnsupported", err)
	}
	if _, err := pty.Start(exec.Command("cmd", "/c", "echo hi"), nil); !errors.Is(err, pty.ErrUnsupported) {
		t.Errorf("Start() err = %v, want ErrUnsupported", err)
	}
	if _, err := pty.StartWithAttrs(exec.Command("cmd", "/c", "echo hi"), nil, nil); !errors.Is(err, pty.ErrUnsupported) {
		t.Errorf("StartWithAttrs() err = %v, want ErrUnsupported", err)
	}
}
