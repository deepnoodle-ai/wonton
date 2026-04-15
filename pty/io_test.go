//go:build darwin || linux

package pty_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/deepnoodle-ai/wonton/pty"
)

// Tests in this file cover PTY I/O paths that are not exercised by the basic
// roundtrip tests: a Read blocked on the master being woken by an external
// event (deadline, concurrent Close), and the kernel line discipline echoing
// input written to the master back out with LF→CRLF translation.
//
// These scenarios are pinned regressions from the upstream creack/pty tests
// (issues #88, #114, #162) — worth keeping here since wonton's PTY wraps
// *os.File directly and any breakage in the runtime poller integration or
// termios defaults would first surface through paths like these.

const readUnblockBudget = time.Second

// TestOpen_MasterIsBlocking pins the invariant that Open returns a master fd
// in blocking mode. Go 1.26 flipped os.OpenFile to set O_NONBLOCK on
// /dev/ptmx on Darwin, where the runtime poller does not drive ptmx — that
// combination makes master Reads return EAGAIN immediately and every io.Copy
// loop in the package exits with zero bytes. pty.Open clears O_NONBLOCK to
// restore the historical semantics; this test fails fast if that clear is
// ever removed or a future runtime change sets the flag again after Open
// returns.
func TestOpen_MasterIsBlocking(t *testing.T) {
	skipIfUnsupported(t)

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	flags, err := unix.FcntlInt(master.Fd(), syscall.F_GETFL, 0)
	if err != nil {
		t.Fatalf("F_GETFL on master: %v", err)
	}
	if flags&syscall.O_NONBLOCK != 0 {
		t.Fatalf("master fd flags = 0x%x, O_NONBLOCK set; want blocking", flags)
	}
}

// skipIfBlockingPTY skips on platforms whose /dev/ptmx is intentionally
// blocking. On Darwin the Go runtime keeps ptmx in blocking mode (see
// creack/pty issues #52, #53 and golang/go#22099), so SetDeadline is not
// honored and a Close from another goroutine will not unblock a Read.
func skipIfBlockingPTY(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "darwin" {
		t.Skip("Darwin /dev/ptmx is intentionally blocking; deadline / close-interrupt semantics do not apply")
	}
}

// pollableMaster opens a PTY pair, puts the master in non-blocking mode so
// Read participates in the Go runtime poller, and arms a watchdog that writes
// a sentinel byte to the slave (unblocking a stuck Read so the test exits)
// and fails the test if it had to. Master/slave close and watchdog stop are
// all staged on t.
func pollableMaster(t *testing.T) *pty.PTY {
	t.Helper()

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatalf("pty.Open: %v", err)
	}
	t.Cleanup(func() { _ = master.Close() })
	t.Cleanup(func() { _ = slave.Close() })

	if err := syscall.SetNonblock(int(master.Fd()), true); err != nil {
		t.Fatalf("SetNonblock(master): %v", err)
	}

	watchdog := time.AfterFunc(readUnblockBudget, func() {
		_, _ = slave.Write([]byte{0xEE})
		t.Errorf("master.Read was not unblocked within %s", readUnblockBudget)
	})
	t.Cleanup(func() { watchdog.Stop() })

	return master
}

// TestMasterReadHonorsDeadline verifies that SetDeadline on the underlying
// *os.File wakes a blocked Read on the master with os.ErrDeadlineExceeded.
func TestMasterReadHonorsDeadline(t *testing.T) {
	skipIfBlockingPTY(t)

	master := pollableMaster(t)

	if err := master.File().SetDeadline(time.Now().Add(readUnblockBudget / 10)); err != nil {
		if errors.Is(err, os.ErrNoDeadline) {
			t.Skipf("SetDeadline unsupported on %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
		}
		t.Fatalf("SetDeadline: %v", err)
	}

	buf := make([]byte, 1)
	n, err := master.Read(buf)
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("Read: want os.ErrDeadlineExceeded, got n=%d buf=0x%X err=%v", n, buf[:n], err)
	}
}

// TestMasterReadUnblocksOnClose verifies that closing the master from another
// goroutine wakes a blocked Read with os.ErrClosed.
func TestMasterReadUnblocksOnClose(t *testing.T) {
	skipIfBlockingPTY(t)

	master := pollableMaster(t)

	go func() {
		time.Sleep(readUnblockBudget / 10)
		if err := master.Close(); err != nil {
			t.Errorf("master.Close: %v", err)
		}
	}()

	buf := make([]byte, 1)
	n, err := master.Read(buf)
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Read: want os.ErrClosed, got n=%d buf=0x%X err=%v", n, buf[:n], err)
	}
}

// TestLineDisciplineEchoesWithCRLF verifies that a freshly opened PTY pair
// has the kernel line discipline in canonical mode with ECHO enabled: bytes
// written to the master appear verbatim on the slave (raw input) and come
// back out of the master with LF translated to CRLF (echoed output).
//
// This is the observable fingerprint of default termios wiring. A regression
// in Open() that accidentally raw-modes the slave would break this test.
func TestLineDisciplineEchoesWithCRLF(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	const input = "ping\n"
	if _, err := io.WriteString(master, input); err != nil {
		t.Fatalf("master.Write: %v", err)
	}

	// Drain the raw input on the slave side so it doesn't backpressure the
	// line discipline before we can read the echo.
	raw := make([]byte, len(input))
	if _, err := io.ReadFull(slave, raw); err != nil {
		t.Fatalf("slave.Read (raw): %v", err)
	}
	if !bytes.Equal(raw, []byte(input)) {
		t.Errorf("slave raw read = %q, want %q", raw, input)
	}

	// The echo back from the master replaces the bare LF with CRLF.
	const wantEcho = "ping\r\n"
	echo := make([]byte, len(wantEcho))
	if err := master.File().SetReadDeadline(time.Now().Add(readUnblockBudget)); err != nil && !errors.Is(err, os.ErrNoDeadline) {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	if _, err := io.ReadFull(master, echo); err != nil {
		t.Fatalf("master.Read (echo): %v", err)
	}
	if !bytes.Equal(echo, []byte(wantEcho)) {
		t.Errorf("master echo = %q, want %q", echo, wantEcho)
	}
}
