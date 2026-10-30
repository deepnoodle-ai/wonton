//go:build darwin || linux

package terminalinput

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func TestIdleReadAcknowledgesPauseWithoutClosingTerminal(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	fd := int(slave.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(fd, state)
	original, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := New(slave)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	activeFlags, flagErr := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if flagErr != nil || activeFlags != original {
		t.Fatalf("active flags=%x original=%x error=%v", activeFlags, original, flagErr)
	}
	reading := make(chan struct{})
	result := make(chan error, 1)
	go func() { close(reading); var buf [16]byte; _, err := owner.Read(buf[:]); result <- err }()
	<-reading
	if err := owner.Pause(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, Boundary) {
			t.Fatalf("idle read=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("pause failed to settle idle read")
	}
	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil || flags != original {
		t.Fatalf("released flags=%x original=%x error=%v", flags, original, err)
	}
	if _, err := master.Write([]byte("child")); err != nil {
		t.Fatal(err)
	}
	var child [5]byte
	if _, err := io.ReadFull(slave, child[:]); err != nil || string(child[:]) != "child" {
		t.Fatalf("child bytes=%q err=%v", child, err)
	}
	if _, err := master.Write([]byte("residual")); err != nil {
		t.Fatal(err)
	}
	// Wait for the PTY to make the residual bytes available before flushing.
	ready := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	if _, err := unix.Poll(ready, 1000); err != nil || ready[0].Revents&unix.POLLIN == 0 {
		t.Fatalf("residual readiness=%v %v", ready, err)
	}
	if err := owner.DiscardResidual(); err != nil {
		t.Fatal(err)
	}
	if err := owner.Resume(); err != nil {
		t.Fatal(err)
	}
	if _, err := master.Write([]byte("new")); err != nil {
		t.Fatal(err)
	}
	var resumed [3]byte
	if _, err := io.ReadFull(owner, resumed[:]); err != nil || string(resumed[:]) != "new" {
		t.Fatalf("resumed bytes=%q error=%v", resumed, err)
	}
	if err := owner.Close(); err != nil {
		t.Fatal(err)
	}
	flags, _ = unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if flags != original {
		t.Fatal("shutdown left changed file flags")
	}
	if _, err := master.Write([]byte("after")); err != nil {
		t.Fatal(err)
	}
	var after [5]byte
	if _, err := io.ReadFull(slave, after[:]); err != nil || string(after[:]) != "after" {
		t.Fatalf("terminal closed: %q %v", after, err)
	}
}

func TestPauseCapturesFinitePreownedBytesAndStopWakesRead(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	fd := int(slave.Fd())
	state, err := term.MakeRaw(fd)
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(fd, state)
	owner, err := New(slave)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if _, err := master.Write([]byte("owned")); err != nil {
		t.Fatal(err)
	}
	ready := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	if _, err := unix.Poll(ready, 1000); err != nil || ready[0].Revents&unix.POLLIN == 0 {
		t.Fatal("preowned bytes not ready")
	}
	if err := owner.Pause(); err != nil {
		t.Fatal(err)
	}
	if _, err := master.Write([]byte("child")); err != nil {
		t.Fatal(err)
	}
	owned, err := io.ReadAll(owner)
	if string(owned) != "owned" || !errors.Is(err, Boundary) {
		t.Fatalf("captured=%q %v", owned, err)
	}
	var child [5]byte
	if _, err := io.ReadFull(slave, child[:]); err != nil || string(child[:]) != "child" {
		t.Fatalf("child=%q %v", child, err)
	}
	if err := owner.Resume(); err != nil {
		t.Fatal(err)
	}
	stopped := make(chan error, 1)
	go func() { var byte [1]byte; _, err := owner.Read(byte[:]); stopped <- err }()
	owner.Stop()
	select {
	case err := <-stopped:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("stop read=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stop did not wake idle read")
	}
}

func TestOwnerHangupWakesIdleRead(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer slave.Close()
	state, err := term.MakeRaw(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(slave.Fd()), state)
	owner, err := New(slave)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	result := make(chan error, 1)
	go func() { var data [1]byte; _, err := owner.Read(data[:]); result <- err }()
	master.Close()
	select {
	case err := <-result:
		if !errors.Is(err, io.EOF) {
			t.Fatalf("hangup=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("hung-up input did not settle")
	}
}

func TestWaitReadyWaitsForInputWithoutConsumingIt(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer slave.Close()
	state, err := term.MakeRaw(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(slave.Fd()), state)
	owner, err := New(slave)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	if owner.WaitReady(10 * time.Millisecond) {
		t.Fatal("idle terminal reported ready")
	}
	go func() { time.Sleep(10 * time.Millisecond); master.Write([]byte("[A")) }()
	if !owner.WaitReady(time.Second) {
		t.Fatal("input written during the wait was not reported")
	}
	var data [2]byte
	if _, err := io.ReadFull(owner, data[:]); err != nil || string(data[:]) != "[A" {
		t.Fatalf("read=%q %v", data, err)
	}
	result := make(chan bool, 1)
	go func() { result <- owner.WaitReady(time.Minute) }()
	time.Sleep(10 * time.Millisecond)
	if err := owner.Pause(); err != nil {
		t.Fatal(err)
	}
	select {
	case ready := <-result:
		if ready {
			t.Fatal("paused owner with nothing captured reported ready")
		}
	case <-time.After(time.Second):
		t.Fatal("pause did not wake the wait")
	}
}

// TestOwnerHangupKeepsInputCapturedByPause pauses and then hangs up while a
// Read is parked in poll. When the Read wakes to the hangup it must still
// return what Pause captured. The loss depends on scheduling, so repeat.
func TestOwnerHangupKeepsInputCapturedByPause(t *testing.T) {
	for i := 0; i < 50; i++ {
		master, slave, err := pty.Open()
		if err != nil {
			t.Fatal(err)
		}
		fd := int(slave.Fd())
		state, err := term.MakeRaw(fd)
		if err != nil {
			t.Fatal(err)
		}
		owner, err := New(slave)
		if err != nil {
			t.Fatal(err)
		}
		type readResult struct {
			data []byte
			err  error
		}
		result := make(chan readResult, 1)
		go func() { data, err := io.ReadAll(owner); result <- readResult{data, err} }()
		time.Sleep(time.Millisecond) // let Read reach poll
		if _, err := master.Write([]byte("saved")); err != nil {
			t.Fatal(err)
		}
		// Wait for the PTY to deliver the bytes, unless Read already took them.
		_, _ = unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}, 100)
		if err := owner.Pause(); err != nil {
			t.Fatal(err)
		}
		master.Close()
		var got readResult
		select {
		case got = <-result:
		case <-time.After(time.Second):
			t.Fatal("hung-up read did not settle")
		}
		owner.Close()
		term.Restore(fd, state)
		slave.Close()
		if string(got.data) != "saved" || (got.err != nil && !errors.Is(got.err, Boundary)) {
			t.Fatalf("iteration %d: read=%q %v, want %q", i, got.data, got.err, "saved")
		}
	}
}
