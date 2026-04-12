package pty_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/pty"
)

func skipIfUnsupported(t *testing.T) {
	t.Helper()
	switch runtime.GOOS {
	case "linux", "darwin":
		// Tier 1: always run.
	case "freebsd", "netbsd", "openbsd", "dragonfly":
		// Tier 2: run if available.
		if _, err := os.Stat("/dev/ptmx"); err != nil {
			if _, err := os.Stat("/dev/ptm"); err != nil {
				t.Skip("no PTY device available")
			}
		}
	default:
		t.Skip("PTY not supported on " + runtime.GOOS)
	}
}

// Compile-time interface check.
var _ io.ReadWriteCloser = (*pty.PTY)(nil)

func TestOpen(t *testing.T) {
	skipIfUnsupported(t)

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	// Write to the slave side and read from the master. This direction works
	// reliably without terminal line discipline interfering.
	msg := []byte("hello from slave")
	if _, err := slave.Write(msg); err != nil {
		t.Fatal("write to slave:", err)
	}

	buf := make([]byte, 256)
	n, err := master.Read(buf)
	if err != nil {
		t.Fatal("read from master:", err)
	}
	got := string(buf[:n])
	if !strings.Contains(got, "hello from slave") {
		t.Errorf("master read = %q, want to contain %q", got, "hello from slave")
	}
}

func TestStart(t *testing.T) {
	skipIfUnsupported(t)

	cmd := exec.Command("echo", "hello world")
	p, err := pty.Start(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	var buf bytes.Buffer
	io.Copy(&buf, p)
	cmd.Wait()

	got := buf.String()
	if !strings.Contains(got, "hello world") {
		t.Errorf("output = %q, want to contain %q", got, "hello world")
	}
}

func TestStart_WithSize(t *testing.T) {
	skipIfUnsupported(t)

	cmd := exec.Command("true")
	size := &pty.Size{Rows: 40, Cols: 120}
	p, err := pty.Start(cmd, size)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	got, err := p.GetSize()
	if err != nil {
		t.Fatal("GetSize:", err)
	}
	if got.Rows != 40 || got.Cols != 120 {
		t.Errorf("size = %+v, want {Rows:40 Cols:120}", got)
	}
	cmd.Wait()
}

func TestResize(t *testing.T) {
	skipIfUnsupported(t)

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	if err := master.Resize(pty.Size{Rows: 50, Cols: 200}); err != nil {
		t.Fatal("Resize:", err)
	}

	got, err := master.GetSize()
	if err != nil {
		t.Fatal("GetSize:", err)
	}
	if got.Rows != 50 || got.Cols != 200 {
		t.Errorf("size = %+v, want {Rows:50 Cols:200}", got)
	}
}

func TestClose_Idempotent(t *testing.T) {
	skipIfUnsupported(t)

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	slave.Close()

	if err := master.Close(); err != nil {
		t.Fatal("first Close:", err)
	}
	// Second close should not error or panic.
	if err := master.Close(); err != nil {
		t.Errorf("second Close returned error: %v", err)
	}
}

func TestClose_Nil(t *testing.T) {
	var p *pty.PTY

	// These should not panic.
	if err := p.Close(); err != nil {
		t.Errorf("nil Close: %v", err)
	}
	if _, err := p.Read(make([]byte, 1)); err != os.ErrClosed {
		t.Errorf("nil Read err = %v, want os.ErrClosed", err)
	}
	if _, err := p.Write([]byte("x")); err != os.ErrClosed {
		t.Errorf("nil Write err = %v, want os.ErrClosed", err)
	}
}

func TestStart_InvalidCommand(t *testing.T) {
	skipIfUnsupported(t)

	cmd := exec.Command("/nonexistent/command/that/does/not/exist")
	p, err := pty.Start(cmd, nil)
	if err == nil {
		p.Close()
		t.Fatal("expected error for invalid command")
	}
}

func TestFd(t *testing.T) {
	skipIfUnsupported(t)

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	fd := master.Fd()
	if fd == ^uintptr(0) {
		t.Error("Fd() returned invalid sentinel on open PTY")
	}

	// After close, Fd should return sentinel.
	master.Close()
	if master.Fd() != ^uintptr(0) {
		t.Error("Fd() should return sentinel after Close")
	}
}

func TestFile(t *testing.T) {
	skipIfUnsupported(t)

	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()

	f := master.File()
	if f == nil {
		t.Error("File() returned nil on open PTY")
	}

	// nil PTY returns nil File.
	var p *pty.PTY
	if p.File() != nil {
		t.Error("nil PTY File() should return nil")
	}
}

func TestStartWithAttrs(t *testing.T) {
	skipIfUnsupported(t)

	cmd := exec.Command("echo", "attrs test")
	p, err := pty.StartWithAttrs(cmd, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	var buf bytes.Buffer
	io.Copy(&buf, p)
	cmd.Wait()

	got := buf.String()
	if !strings.Contains(got, "attrs test") {
		t.Errorf("output = %q, want to contain %q", got, "attrs test")
	}
}

func TestInheritSize(t *testing.T) {
	skipIfUnsupported(t)

	// Create two PTYs; set a known size on one and inherit to the other.
	src, srcSlave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	defer srcSlave.Close()

	dst, dstSlave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	defer dstSlave.Close()

	if err := src.Resize(pty.Size{Rows: 33, Cols: 111}); err != nil {
		t.Fatal("Resize src:", err)
	}

	if err := dst.InheritSize(src.File()); err != nil {
		t.Fatal("InheritSize:", err)
	}

	got, err := dst.GetSize()
	if err != nil {
		t.Fatal("GetSize:", err)
	}
	if got.Rows != 33 || got.Cols != 111 {
		t.Errorf("inherited size = %+v, want {Rows:33 Cols:111}", got)
	}
}

func TestOpen_SlaveCloseDoesNotBreakMaster(t *testing.T) {
	skipIfUnsupported(t)

	cmd := exec.Command("echo", "slave close test")
	p, err := pty.Start(cmd, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	// Start closes the slave internally. The master should still be readable.
	var buf bytes.Buffer
	io.Copy(&buf, p)
	cmd.Wait()

	if !strings.Contains(buf.String(), "slave close test") {
		t.Errorf("output = %q, want to contain %q", buf.String(), "slave close test")
	}
}
