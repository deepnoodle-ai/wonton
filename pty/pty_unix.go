//go:build darwin || linux

package pty

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// Resize sets the terminal size of the PTY. Any process on the slave side
// receives a SIGWINCH signal.
func (p *PTY) Resize(size Size) error {
	if p == nil || p.master == nil {
		return os.ErrClosed
	}
	return unix.IoctlSetWinsize(int(p.master.Fd()), unix.TIOCSWINSZ, &unix.Winsize{
		Row: size.Rows,
		Col: size.Cols,
	})
}

// GetSize returns the current terminal size of the PTY.
func (p *PTY) GetSize() (Size, error) {
	if p == nil || p.master == nil {
		return Size{}, os.ErrClosed
	}
	ws, err := unix.IoctlGetWinsize(int(p.master.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return Size{}, err
	}
	return Size{Rows: ws.Row, Cols: ws.Col}, nil
}

// InheritSize copies the terminal size from tty to the PTY. This is typically
// used to synchronize the PTY size with the controlling terminal.
func (p *PTY) InheritSize(tty *os.File) error {
	if p == nil || p.master == nil {
		return os.ErrClosed
	}
	if tty == nil {
		return os.ErrClosed
	}
	ws, err := unix.IoctlGetWinsize(int(tty.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return fmt.Errorf("pty: get size from source: %w", err)
	}
	return unix.IoctlSetWinsize(int(p.master.Fd()), unix.TIOCSWINSZ, ws)
}

// Open allocates a new PTY pair without starting a command. Returns the master
// side as a [*PTY] and the slave as an [*os.File]. The caller is responsible
// for closing both.
//
// Most callers should use [Start] instead.
func Open() (*PTY, *os.File, error) {
	master, slave, err := open()
	if err != nil {
		return nil, nil, err
	}
	// Go 1.26's os.OpenFile sets O_NONBLOCK on /dev/ptmx on Darwin, but the
	// Darwin runtime poller does not drive ptmx — so a non-blocking Read
	// returns EAGAIN directly to the caller and any io.Copy loop exits with
	// zero bytes. Force the master back to blocking so Reads wait for data
	// as the rest of the package assumes. Tests that want deadline or
	// close-interrupt semantics opt back into non-blocking explicitly via
	// syscall.SetNonblock.
	if err := syscall.SetNonblock(int(master.Fd()), false); err != nil {
		master.Close()
		slave.Close()
		return nil, nil, fmt.Errorf("pty: clear O_NONBLOCK on master: %w", err)
	}
	return &PTY{master: master}, slave, nil
}

// Start allocates a PTY and starts the command with its stdin, stdout, and
// stderr connected to the slave side. The command runs in a new session with
// a controlling terminal.
//
// If size is non-nil, the PTY is resized before the command starts. The
// returned PTY must be closed when no longer needed. The command should be
// waited on separately via cmd.Wait().
//
// If cmd.SysProcAttr is nil, Start creates one. If it is already set, Start
// adds Setsid and Setctty but preserves all other fields.
//
// Example:
//
//	cmd := exec.Command("bash", "-i")
//	p, err := pty.Start(cmd, &pty.Size{Rows: 24, Cols: 80})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer p.Close()
//	go io.Copy(os.Stdout, p)
//	io.Copy(p, os.Stdin)
//	cmd.Wait()
func Start(cmd *exec.Cmd, size *Size) (*PTY, error) {
	return StartWithAttrs(cmd, size, nil)
}

// StartWithAttrs is like [Start] but accepts a custom [*syscall.SysProcAttr].
// If attrs is non-nil, it is used as the base; Setsid and Setctty are added
// on top. If attrs is nil and cmd.SysProcAttr is already set, the existing
// value is used as the base.
//
// This is useful when you need to set additional process attributes such as
// Setpgid, Credential, or Cloneflags.
func StartWithAttrs(cmd *exec.Cmd, size *Size, attrs *syscall.SysProcAttr) (*PTY, error) {
	p, slave, err := Open()
	if err != nil {
		return nil, err
	}
	defer slave.Close()

	if size != nil {
		if err := p.Resize(*size); err != nil {
			p.Close()
			return nil, err
		}
	}

	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave

	// Determine the base SysProcAttr.
	spa := attrs
	if spa == nil {
		spa = cmd.SysProcAttr
	}
	if spa == nil {
		spa = &syscall.SysProcAttr{}
	}
	spa.Setsid = true
	spa.Setctty = true
	cmd.SysProcAttr = spa

	if err := cmd.Start(); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}
