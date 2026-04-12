//go:build darwin || linux

package pty_test

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/pty"
)

// TestChildSeesTTY asserts that a process started via pty.Start has a real
// controlling TTY on all three standard descriptors. If Setsid/Setctty wiring
// in pty_unix.go ever regresses, `test -t` will fail and so will this.
func TestChildSeesTTY(t *testing.T) {
	skipIfUnsupported(t)

	cmd := exec.Command("sh", "-c", "test -t 0 && test -t 1 && test -t 2 && echo TTY-OK")
	p, err := pty.Start(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	var buf bytes.Buffer
	io.Copy(&buf, p)
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child exit: %v (output=%q)", err, buf.String())
	}
	if !strings.Contains(buf.String(), "TTY-OK") {
		t.Errorf("child did not see a TTY; output=%q", buf.String())
	}
}

// TestChildSeesRequestedSize asserts that the size passed to pty.Start is the
// size the child's controlling TTY reports via `stty size`. This is the
// end-to-end check for TIOCSWINSZ wiring before exec.
func TestChildSeesRequestedSize(t *testing.T) {
	skipIfUnsupported(t)

	cmd := exec.Command("sh", "-c", "stty size")
	p, err := pty.Start(cmd, &pty.Size{Rows: 30, Cols: 100})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	var buf bytes.Buffer
	io.Copy(&buf, p)
	if err := cmd.Wait(); err != nil {
		t.Fatalf("child exit: %v", err)
	}
	got := strings.TrimSpace(buf.String())
	// `stty size` prints "rows cols".
	if !strings.Contains(got, "30 100") {
		t.Errorf("stty size = %q, want to contain %q", got, "30 100")
	}
}

// TestResizeDeliversSIGWINCH asserts that calling Resize on the master
// delivers SIGWINCH to the child process. The child installs a trap and
// echoes a sentinel when the signal fires.
func TestResizeDeliversSIGWINCH(t *testing.T) {
	skipIfUnsupported(t)

	// Shell loops briefly waiting for SIGWINCH. When the trap fires, it
	// prints WINCH:<rows>x<cols> and exits.
	script := `
trap 'set -- $(stty size); echo "WINCH:${1}x${2}"; exit 0' WINCH
i=0
while [ $i -lt 40 ]; do
	sleep 0.1
	i=$((i+1))
done
echo "TIMEOUT"
exit 1
`
	cmd := exec.Command("sh", "-c", script)
	p, err := pty.Start(cmd, &pty.Size{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Read all output in the background.
	var mu sync.Mutex
	var buf bytes.Buffer
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		b := make([]byte, 1024)
		for {
			n, err := p.Read(b)
			if n > 0 {
				mu.Lock()
				buf.Write(b[:n])
				mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	// Give the child a moment to install the trap before firing SIGWINCH.
	time.Sleep(200 * time.Millisecond)
	if err := p.Resize(pty.Size{Rows: 42, Cols: 132}); err != nil {
		t.Fatal("Resize:", err)
	}

	// Wait for the child to exit, then close the master so the reader drains.
	waitErr := cmd.Wait()
	p.Close()
	<-readDone

	mu.Lock()
	out := buf.String()
	mu.Unlock()

	if waitErr != nil {
		t.Fatalf("child exit: %v (output=%q)", waitErr, out)
	}
	if !strings.Contains(out, "WINCH:42x132") {
		t.Errorf("SIGWINCH not delivered with new size; output=%q", out)
	}
}

// TestLargeSlaveToMasterRoundtrip writes a ~64KB payload to the slave side
// and drains it from the master, asserting byte-identity. Line-discipline
// echo is avoided by going slave->master directly on a non-controlled PTY.
// The reader runs concurrently with the writer to avoid the pty buffer
// back-pressuring us into deadlock.
func TestLargeSlaveToMasterRoundtrip(t *testing.T) {
	skipIfUnsupported(t)

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	// ~64KB of non-special bytes (printable ASCII). Staying under typical
	// 128KB pts buffers keeps the test simple without a goroutine dance.
	const n = 64 * 1024
	payload := bytes.Repeat([]byte("abcdefghijklmnop"), n/16)

	var readErr error
	var got []byte
	done := make(chan struct{})
	go func() {
		defer close(done)
		var out bytes.Buffer
		b := make([]byte, 4096)
		for out.Len() < len(payload) {
			m, err := master.Read(b)
			if m > 0 {
				out.Write(b[:m])
			}
			if err != nil {
				readErr = err
				return
			}
		}
		got = out.Bytes()
	}()

	if _, err := slave.Write(payload); err != nil {
		t.Fatal("slave write:", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out reading large payload from master")
	}

	if readErr != nil && !errors.Is(readErr, io.EOF) {
		t.Fatalf("master read: %v", readErr)
	}
	if len(got) < len(payload) {
		t.Fatalf("short read: got %d bytes, want %d", len(got), len(payload))
	}
	if !bytes.Equal(got[:len(payload)], payload) {
		// Find first divergence for a useful error.
		for i := 0; i < len(payload); i++ {
			if got[i] != payload[i] {
				t.Fatalf("byte mismatch at offset %d: got %#x, want %#x", i, got[i], payload[i])
			}
		}
	}
}
