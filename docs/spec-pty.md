# Spec: `pty` Package

Replace `github.com/creack/pty` with a local implementation. Removes 1
third-party dependency.

## Motivation

The `creack/pty` library provides pseudo-terminal allocation. Wonton uses it in
a single file (`termsession/session.go`) with only 3 call sites. A local
replacement can:

- Provide a structured `PTY` type instead of raw `*os.File`
- Offer nil-safe, idempotent `Close()` for reliable cleanup
- Use method-based API (`p.Resize()`) instead of free functions with fd arguments
- Leverage `x/sys/unix` safe helpers instead of raw `unsafe.Pointer` + syscall
- Support initial sizing in `Start()` without a separate call

## Current Usage

All usage is in `termsession/session.go` — 3 call sites:

| Call Site | Function | Purpose |
|-----------|----------|---------|
| `Session.Start()` | `pty.Start(s.cmd)` | Allocate PTY, start command |
| `Session.Resize()` | `pty.Setsize(ptmx, &pty.Winsize{Rows, Cols})` | Resize PTY |
| `Session.syncSize()` | `pty.Setsize(ptmx, &pty.Winsize{Rows, Cols})` | Sync PTY size on startup/SIGWINCH |

Only `Rows` and `Cols` fields of `Winsize` are used (never `X` or `Y` pixel
fields).

## Package Location

`github.com/deepnoodle-ai/wonton/pty`

Rationale: PTY allocation is a general-purpose primitive, not specific to
`termsession` or `tty`. The `tty` package is about terminal detection; PTY
creation is a different concern. Follows the pattern of other standalone Wonton
packages.

## Public API

```go
// Package pty provides pseudo-terminal (PTY) allocation and management.
//
// A PTY is a pair of virtual terminal devices: a master and a slave. The master
// side is used by the controlling process to read output and write input. The
// slave side is connected to a child process's stdin/stdout/stderr, making the
// child believe it's running in a real terminal.
//
// Example:
//
//	cmd := exec.Command("bash")
//	p, err := pty.Start(cmd, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer p.Close()
//	io.Copy(os.Stdout, p)
package pty

// ErrUnsupported is returned when PTY operations are not supported on the
// current platform.
var ErrUnsupported = errors.New("pty: unsupported platform")

// Size represents terminal dimensions.
type Size struct {
	Rows uint16 // Number of rows (lines).
	Cols uint16 // Number of columns (characters per line).
}

// PTY represents an allocated pseudo-terminal. It wraps the master file
// descriptor and provides methods for reading, writing, resizing, and cleanup.
// Implements io.ReadWriteCloser.
//
// Always call Close when finished to release the underlying file descriptor.
type PTY struct {
	master *os.File
}

// Read reads from the PTY master. Returns output produced by the child process.
func (p *PTY) Read(b []byte) (int, error)

// Write writes to the PTY master. Sends input to the child process.
func (p *PTY) Write(b []byte) (int, error)

// Close closes the PTY master file descriptor. Signals EOF to any process
// reading from the slave side. Safe to call multiple times.
func (p *PTY) Close() error

// Fd returns the file descriptor of the master side.
func (p *PTY) Fd() uintptr

// Resize sets the terminal size of the PTY. Any process on the slave side
// receives a SIGWINCH signal.
func (p *PTY) Resize(size Size) error

// GetSize returns the current terminal size of the PTY.
func (p *PTY) GetSize() (Size, error)

// Open allocates a new PTY pair without starting a command. Returns the master
// side as a PTY and the slave as an *os.File. The caller is responsible for
// closing both. Most callers should use Start instead.
func Open() (master *PTY, slave *os.File, err error)

// Start allocates a PTY and starts the command with its stdin, stdout, and
// stderr connected to the slave side. The command runs in a new session with
// a controlling terminal.
//
// If size is non-nil, the PTY is resized before the command starts. The
// returned PTY must be closed when no longer needed. The command should be
// waited on separately via cmd.Wait().
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
func Start(cmd *exec.Cmd, size *Size) (*PTY, error)
```

## Platform Implementations

### Darwin (`pty_darwin.go`)

```
Open:
  1. syscall.Open("/dev/ptmx", O_RDWR|O_CLOEXEC, 0)
  2. ioctl(fd, TIOCPTYGNAME, &buf)  → slave name
  3. ioctl(fd, TIOCPTYGRANT, 0)     → grantpt
  4. ioctl(fd, TIOCPTYUNLK, 0)      → unlockpt
  5. os.OpenFile(slaveName, O_RDWR|O_NOCTTY, 0)
```

Constants available in `golang.org/x/sys/unix`.

### Linux (`pty_linux.go`)

```
Open:
  1. os.OpenFile("/dev/ptmx", O_RDWR, 0)
  2. ioctl(fd, TIOCGPTN, &n)     → PTS number
  3. ioctl(fd, TIOCSPTLCK, &zero) → unlock
  4. os.OpenFile("/dev/pts/"+n, O_RDWR|O_NOCTTY, 0)
```

### FreeBSD (`pty_freebsd.go`)

```
Open:
  1. syscall.Syscall(SYS_POSIX_OPENPT, O_RDWR|O_CLOEXEC, 0, 0)
  2. ioctl(fd, FIODGNAME, &arg) → slave name
  3. os.OpenFile("/dev/"+name, O_RDWR, 0)
```

Requires a locally defined `fiodgnameArg` struct (`{Len int32, Buf *byte}`).

### OpenBSD (`pty_openbsd.go`)

```
Open:
  1. os.OpenFile("/dev/ptm", O_RDWR|O_CLOEXEC, 0)
  2. ioctl(fd, PTMGET, &ptmget) → {Cfd, Sfd, Cn, Sn}
  3. os.NewFile(Cfd), os.NewFile(Sfd)
```

Requires a locally defined `ptmget` struct and `PTMGET` constant.

### NetBSD / Dragonfly

Similar to Linux: open `/dev/ptmx`, platform-appropriate ioctls.

### Windows (`pty_windows.go`)

All functions return `ErrUnsupported`. Matches creack/pty and Wonton's
termsession Windows stubs.

### Other (`pty_other.go`)

Fallback for unsupported platforms. All functions return `ErrUnsupported`.

## Resize Implementation (shared Unix)

Uses the safe `x/sys/unix` helpers already depended on by Wonton's `tty`
package:

```go
func (p *PTY) Resize(size Size) error {
    return unix.IoctlSetWinsize(int(p.master.Fd()), unix.TIOCSWINSZ, &unix.Winsize{
        Row: size.Rows,
        Col: size.Cols,
    })
}

func (p *PTY) GetSize() (Size, error) {
    ws, err := unix.IoctlGetWinsize(int(p.master.Fd()), unix.TIOCGWINSZ)
    if err != nil {
        return Size{}, err
    }
    return Size{Rows: ws.Row, Cols: ws.Col}, nil
}
```

## Start Implementation (shared Unix)

```go
func Start(cmd *exec.Cmd, size *Size) (*PTY, error) {
    master, slave, err := Open()
    if err != nil {
        return nil, err
    }
    defer slave.Close()

    if size != nil {
        if err := master.Resize(*size); err != nil {
            master.Close()
            return nil, err
        }
    }

    cmd.Stdin = slave
    cmd.Stdout = slave
    cmd.Stderr = slave

    if cmd.SysProcAttr == nil {
        cmd.SysProcAttr = &syscall.SysProcAttr{}
    }
    cmd.SysProcAttr.Setsid = true
    cmd.SysProcAttr.Setctty = true

    if err := cmd.Start(); err != nil {
        master.Close()
        return nil, err
    }
    return master, nil
}
```

## Resource Management

- `Close()` is nil-safe and idempotent (sets `master` to nil after close)
- `PTY` implements `io.ReadWriteCloser` — works with `defer`, `io.Copy`, etc.
- `Open()` returns the slave as caller's responsibility; `Start()` handles it
  automatically (closes after cmd.Start)

## Testing Strategy

1. **`TestOpen`** — allocate PTY, write to master, read from slave (and vice
   versa), close both
2. **`TestStart`** — start `echo hello`, read output, verify contents, wait for
   clean exit
3. **`TestResize`** — start PTY, resize, verify `GetSize()` matches
4. **`TestClose_Idempotent`** — call `Close()` twice, no panic or error
5. **`TestClose_Nil`** — methods on nil `*PTY`, no panic
6. **`TestStart_InvalidCommand`** — nonexistent command, verify error, no
   resource leak
7. **`TestReadWriteCloser`** — compile-time interface check:
   `var _ io.ReadWriteCloser = (*PTY)(nil)`
8. **`TestOpen_SlaveClose`** — closing slave after `Start()` doesn't break
   master
9. **Platform skip** — check for `/dev/ptmx` or `/dev/ptm` availability, skip
   in restricted containers

## Migration

### Changes to `termsession/session.go`

**Import:**
```go
// Before
"github.com/creack/pty"

// After
"github.com/deepnoodle-ai/wonton/pty"
```

**Session struct field:**
```go
// Before
pty *os.File

// After
pty *pty.PTY
```

**Start:**
```go
// Before
ptmx, err := pty.Start(s.cmd)

// After
p, err := pty.Start(s.cmd, nil)
```

**Resize / syncSize:**
```go
// Before
pty.Setsize(ptmx, &pty.Winsize{
    Rows: uint16(height),
    Cols: uint16(width),
})

// After
ptmx.Resize(pty.Size{
    Rows: uint16(height),
    Cols: uint16(width),
})
```

**Read/Write** — no changes needed; `*pty.PTY` implements `io.ReadWriter` just
like `*os.File`.

**Dependency removal**: Remove `github.com/creack/pty v1.1.24` from `go.mod`.

## File Structure

```
pty/
    pty.go             # Package doc, types (PTY, Size), shared methods, Open/Start signatures
    pty_darwin.go      # open() for darwin
    pty_linux.go       # open() for linux
    pty_freebsd.go     # open() for freebsd
    pty_openbsd.go     # open() for openbsd
    pty_netbsd.go      # open() for netbsd
    pty_dragonfly.go   # open() for dragonfly
    pty_windows.go     # stubs returning ErrUnsupported
    pty_other.go       # stubs returning ErrUnsupported
    start.go           # Start() implementation (build tag: !windows)
    start_windows.go   # Start() stub
    pty_test.go        # tests
```
