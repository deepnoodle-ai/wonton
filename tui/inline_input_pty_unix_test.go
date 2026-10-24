//go:build darwin || linux

package tui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/creack/pty"
)

// TestPromptMaskedSubmitOnPTY runs Prompt against a real PTY. After submit,
// the secret must not appear in the output and the cursor must be visible.
func TestPromptMaskedSubmitOnPTY(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	oldStdin := os.Stdin
	os.Stdin = slave
	defer func() { os.Stdin = oldStdin }()

	if _, err := master.WriteString("my-secret\r"); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	got, err := Prompt("Password: ", WithMask("*"), WithPromptOutput(&out))
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if got != "my-secret" {
		t.Fatalf("Prompt returned %q", got)
	}
	s := out.String()
	if strings.Contains(s, "my-secret") {
		t.Fatalf("output leaks the secret: %q", s)
	}
	if hide := strings.LastIndex(s, "\x1b[?25l"); hide >= 0 && strings.LastIndex(s, "\x1b[?25h") < hide {
		t.Fatalf("cursor left hidden: %q", s)
	}
}
