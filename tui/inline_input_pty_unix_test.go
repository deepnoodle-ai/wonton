//go:build darwin || linux

package tui

import (
	"bytes"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/deepnoodle-ai/wonton/assert"
)

// The prompt takes its width from the output terminal first, then the
// input terminal.
func TestPromptWidthFromTerminal(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer master.Close()
	defer slave.Close()
	assert.NoError(t, pty.Setsize(slave, &pty.Winsize{Cols: 132, Rows: 40}))

	width, fromTerminal := promptWidth(slave, strings.NewReader(""))
	assert.Equal(t, 132, width)
	assert.True(t, fromTerminal)

	width, fromTerminal = promptWidth(&bytes.Buffer{}, slave)
	assert.Equal(t, 132, width)
	assert.True(t, fromTerminal)

	// A resize shows in the next measure.
	assert.NoError(t, pty.Setsize(slave, &pty.Winsize{Cols: 100, Rows: 40}))
	width, _ = promptWidth(slave, nil)
	assert.Equal(t, 100, width)
}

// The resize watcher calls back on SIGWINCH and its goroutine exits on stop.
func TestWatchPromptResize(t *testing.T) {
	resized := make(chan struct{}, 1)
	stop := watchPromptResize(func() {
		select {
		case resized <- struct{}{}:
		default:
		}
	})
	assert.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGWINCH))
	select {
	case <-resized:
	case <-time.After(2 * time.Second):
		t.Fatal("no callback after SIGWINCH")
	}
	stop()
	stop() // safe to call again
}

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

// In raw mode "\n" does not return the cursor to column 0, so the line
// printed after submit must end with "\r\n". Otherwise the next output
// starts in the column where the input ended.
func TestPromptFinalLineReturnsToColumnZeroOnPTY(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	oldStdin := os.Stdin
	os.Stdin = slave
	defer func() { os.Stdin = oldStdin }()

	if _, err := master.WriteString("3 + 7\r"); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	got, err := Prompt("> ", WithPromptFrame(false), WithPromptOutput(&out))
	assert.NoError(t, err)
	assert.Equal(t, "3 + 7", got)
	assert.True(t, strings.Contains(out.String(), "3 + 7\r\n"), "output: %q", out.String())
}
