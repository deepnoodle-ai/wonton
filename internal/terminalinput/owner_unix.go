//go:build darwin || linux

package terminalinput

import (
	"errors"
	"io"
	"os"
	"sync"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// Owner keeps terminal reads nonblocking while the application owns input.
// It restores the original file flags before acknowledging release. Poll uses
// a separate wake pipe; neither pause nor shutdown closes the terminal file.
type Owner struct {
	mu                      sync.Mutex
	readMu                  sync.Mutex
	fd, originalFlags       int
	wakeRead, wakeWrite     int
	paused, stopped, closed bool
	captured                []byte
}

func New(file *os.File) (*Owner, error) {
	if file == nil || !term.IsTerminal(int(file.Fd())) {
		return nil, ErrUnsupported
	}
	fd := int(file.Fd())
	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		return nil, err
	}
	pipe := make([]int, 2)
	if err := unix.Pipe(pipe); err != nil {
		return nil, err
	}
	for _, control := range pipe {
		unix.CloseOnExec(control)
		if err := unix.SetNonblock(control, true); err != nil {
			unix.Close(pipe[0])
			unix.Close(pipe[1])
			return nil, err
		}
	}
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_SETFL, flags|unix.O_NONBLOCK); err != nil {
		unix.Close(pipe[0])
		unix.Close(pipe[1])
		return nil, err
	}
	return &Owner{fd: fd, originalFlags: flags, wakeRead: pipe[0], wakeWrite: pipe[1]}, nil
}

// Read never leaves an uninterruptible OS read outstanding. The mutex protects
// the check-and-read operation; poll waits without it and wakes on controls.
func (o *Owner) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}
	o.readMu.Lock()
	defer o.readMu.Unlock()
	for {
		o.mu.Lock()
		if o.stopped {
			o.mu.Unlock()
			return 0, io.EOF
		}
		if o.paused {
			if len(o.captured) > 0 {
				n := copy(dst, o.captured)
				o.captured = o.captured[n:]
				o.mu.Unlock()
				return n, nil
			}
			o.mu.Unlock()
			return 0, Boundary
		}
		n, err := unix.Read(o.fd, dst)
		o.mu.Unlock()
		if n > 0 {
			return n, nil
		}
		if err == nil {
			return 0, io.EOF
		}
		if !errors.Is(err, unix.EAGAIN) && !errors.Is(err, unix.EINTR) {
			return 0, err
		}
		descriptors := []unix.PollFd{{Fd: int32(o.fd), Events: unix.POLLIN}, {Fd: int32(o.wakeRead), Events: unix.POLLIN}}
		if _, err := unix.Poll(descriptors, -1); err != nil {
			if errors.Is(err, unix.EINTR) {
				continue
			}
			return 0, err
		}
		if descriptors[1].Revents != 0 {
			var bytes [64]byte
			_, _ = unix.Read(o.wakeRead, bytes[:])
		}
		if descriptors[0].Revents&unix.POLLNVAL != 0 {
			return 0, unix.EBADF
		}
	}
}

func (o *Owner) wakeLocked() { _, _ = unix.Write(o.wakeWrite, []byte{1}) }

// Pause captures only the bytes readable at its cutoff, restores file flags,
// and acknowledges that no OS read can run until Resume. The decoder can still
// consume this captured finite stream before it observes Boundary.
func (o *Owner) Pause() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.stopped {
		return io.EOF
	}
	if o.paused {
		return nil
	}
	o.paused = true
	defer o.wakeLocked()
	count, err := availableBytes(o.fd)
	if err == nil {
		var buf [4096]byte
		for count > 0 {
			n, readErr := unix.Read(o.fd, buf[:min(count, len(buf))])
			if n > 0 {
				o.captured = append(o.captured, buf[:n]...)
				count -= n
			}
			if errors.Is(readErr, unix.EINTR) {
				continue
			}
			if readErr != nil {
				if !errors.Is(readErr, unix.EAGAIN) {
					err = readErr
				}
				break
			}
			if n == 0 {
				break
			}
		}
	}
	_, flagErr := unix.FcntlInt(uintptr(o.fd), unix.F_SETFL, o.originalFlags)
	return errors.Join(err, flagErr)
}

func (o *Owner) DiscardResidual() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.paused {
		return errors.New("terminal input is not paused")
	}
	return discardInput(o.fd)
}

// Resume is valid after the same decoder acknowledges neutral framing.
func (o *Owner) Resume() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.stopped {
		return io.EOF
	}
	if !o.paused {
		return nil
	}
	if len(o.captured) != 0 {
		return errors.New("pre-release input has not been drained")
	}
	if _, err := unix.FcntlInt(uintptr(o.fd), unix.F_SETFL, o.originalFlags|unix.O_NONBLOCK); err != nil {
		return err
	}
	o.paused = false
	o.wakeLocked()
	return nil
}

// Stop records shutdown and wakes an idle poll without waiting for its caller.
func (o *Owner) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.stopped {
		o.stopped = true
		o.wakeLocked()
	}
}

// Close waits for a low-level read to settle, restores flags, and closes only
// private wake descriptors. The terminal remains open for its original owner.
func (o *Owner) Close() error {
	o.Stop()
	o.readMu.Lock()
	defer o.readMu.Unlock()
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return nil
	}
	o.closed = true
	_, err := unix.FcntlInt(uintptr(o.fd), unix.F_SETFL, o.originalFlags)
	return errors.Join(err, unix.Close(o.wakeRead), unix.Close(o.wakeWrite))
}
