// Package pty provides pseudo-terminal (PTY) allocation and management.
//
// A PTY is a pair of virtual terminal devices: a master and a slave. The master
// side is used by the controlling process to read output and write input. The
// slave side is connected to a child process's stdin/stdout/stderr, making the
// child believe it's running in a real terminal.
//
// # Platform Support
//
// Tier 1 (tested): Linux, Darwin (macOS).
// Tier 2 (compiles, best-effort): FreeBSD, OpenBSD, NetBSD, DragonFly BSD.
// Unsupported: Windows and other platforms return [ErrUnsupported].
//
// # Example
//
//	cmd := exec.Command("bash")
//	p, err := pty.Start(cmd, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer p.Close()
//	io.Copy(os.Stdout, p)
package pty

import (
	"errors"
	"os"
)

// ErrUnsupported is returned when PTY operations are not supported on the
// current platform.
var ErrUnsupported = errors.New("pty: unsupported platform")

// Size represents terminal dimensions in rows and columns.
type Size struct {
	Rows uint16 // Number of rows (lines).
	Cols uint16 // Number of columns (characters per line).
}

// PTY represents an allocated pseudo-terminal. It wraps the master file
// descriptor and provides methods for reading, writing, resizing, and cleanup.
//
// PTY implements [io.ReadWriteCloser].
//
// Always call [PTY.Close] when finished to release the underlying file
// descriptor.
type PTY struct {
	master *os.File
}

// Read reads from the PTY master. Returns output produced by the child process.
func (p *PTY) Read(b []byte) (int, error) {
	if p == nil || p.master == nil {
		return 0, os.ErrClosed
	}
	return p.master.Read(b)
}

// Write writes to the PTY master. Sends input to the child process.
func (p *PTY) Write(b []byte) (int, error) {
	if p == nil || p.master == nil {
		return 0, os.ErrClosed
	}
	return p.master.Write(b)
}

// Close closes the PTY master file descriptor. Signals EOF to any process
// reading from the slave side. Safe to call multiple times; subsequent calls
// return nil.
func (p *PTY) Close() error {
	if p == nil || p.master == nil {
		return nil
	}
	err := p.master.Close()
	p.master = nil
	return err
}

// Fd returns the file descriptor of the master side. Returns ^0 if the PTY
// is nil or closed.
func (p *PTY) Fd() uintptr {
	if p == nil || p.master == nil {
		return ^uintptr(0)
	}
	return p.master.Fd()
}

// File returns the underlying [*os.File] for the master side. This is useful
// for callers that need the file for poll/select integration or
// [os.File.SyscallConn]. Returns nil if the PTY is closed.
func (p *PTY) File() *os.File {
	if p == nil {
		return nil
	}
	return p.master
}
