//go:build windows

package clipboard

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// The Windows backend calls the Win32 clipboard API directly. Starting
// PowerShell for every read took a second or more on a busy machine, which
// used up the timeout before the clipboard was even touched.

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

var (
	user32                     = windows.NewLazySystemDLL("user32.dll")
	kernel32                   = windows.NewLazySystemDLL("kernel32.dll")
	procOpenClipboard          = user32.NewProc("OpenClipboard")
	procCloseClipboard         = user32.NewProc("CloseClipboard")
	procEmptyClipboard         = user32.NewProc("EmptyClipboard")
	procGetClipboardData       = user32.NewProc("GetClipboardData")
	procSetClipboardData       = user32.NewProc("SetClipboardData")
	procIsClipboardFormatAvail = user32.NewProc("IsClipboardFormatAvailable")
	procGlobalAlloc            = kernel32.NewProc("GlobalAlloc")
	procGlobalFree             = kernel32.NewProc("GlobalFree")
	procGlobalLock             = kernel32.NewProc("GlobalLock")
	procGlobalUnlock           = kernel32.NewProc("GlobalUnlock")
	procGlobalSize             = kernel32.NewProc("GlobalSize")
)

// openRetryInterval is how long to wait between attempts to open a
// clipboard that another process holds.
const openRetryInterval = 10 * time.Millisecond

// openClipboard opens the clipboard for the calling OS thread. Only one
// process can hold the clipboard, and others hold it briefly all the time
// (clipboard managers, remote desktop, the shell). So it retries until
// the clipboard is free or ctx is done. The caller must lock the OS thread.
func openClipboard(ctx context.Context) error {
	for {
		if r, _, _ := procOpenClipboard.Call(0); r != 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return ErrTimeout
			}
			return ctx.Err()
		case <-time.After(openRetryInterval):
		}
	}
}

// globalPtr converts a locked global memory handle's address to a pointer.
// The memory is outside the Go heap, so the garbage collector never moves
// it. The indirection keeps go vet from flagging the uintptr conversion.
func globalPtr(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

// read implements clipboard reading for Windows. It returns "" when the
// clipboard holds no text.
func read(ctx context.Context) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := openClipboard(ctx); err != nil {
		return "", err
	}
	defer procCloseClipboard.Call()

	if r, _, _ := procIsClipboardFormatAvail.Call(cfUnicodeText); r == 0 {
		return "", nil
	}
	h, _, err := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", fmt.Errorf("clipboard: GetClipboardData: %w", err)
	}
	p, _, err := procGlobalLock.Call(h)
	if p == 0 {
		return "", fmt.Errorf("clipboard: GlobalLock: %w", err)
	}
	defer procGlobalUnlock.Call(h)

	// Bound the read by the allocation size in case the text is not
	// NUL-terminated.
	size, _, _ := procGlobalSize.Call(h)
	n := int(size / 2)
	if n == 0 {
		return "", nil
	}
	buf := unsafe.Slice((*uint16)(globalPtr(p)), n)
	return windows.UTF16ToString(buf), nil
}

// write implements clipboard writing for Windows.
func write(ctx context.Context, text string) error {
	data, err := windows.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("clipboard: %w", err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := openClipboard(ctx); err != nil {
		return err
	}
	defer procCloseClipboard.Call()

	if r, _, err := procEmptyClipboard.Call(); r == 0 {
		return fmt.Errorf("clipboard: EmptyClipboard: %w", err)
	}

	size := uintptr(len(data)) * 2
	h, _, err := procGlobalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return fmt.Errorf("clipboard: GlobalAlloc: %w", err)
	}
	p, _, err := procGlobalLock.Call(h)
	if p == 0 {
		procGlobalFree.Call(h)
		return fmt.Errorf("clipboard: GlobalLock: %w", err)
	}
	copy(unsafe.Slice((*uint16)(globalPtr(p)), len(data)), data)
	procGlobalUnlock.Call(h)

	// On success the system owns the memory. On failure we still own it.
	if r, _, err := procSetClipboardData.Call(cfUnicodeText, h); r == 0 {
		procGlobalFree.Call(h)
		return fmt.Errorf("clipboard: SetClipboardData: %w", err)
	}
	return nil
}
