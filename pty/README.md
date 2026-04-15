# pty

Pseudo-terminal allocation for running processes that expect a real TTY. Linux and macOS.

## When you need this

Some CLI tools behave differently when they detect they aren't on a terminal. `bash -i` refuses to act interactively, `python` drops its REPL prompt, `git` turns off color and pagination, `top` bails out. To drive one of these programmatically and still see the interactive behavior, you have to hand it a PTY.

`pty.Start` allocates a master/slave pair, wires the child's stdio to the slave, and gives you the master as an `io.ReadWriteCloser`. You read what the child prints, you write what it reads.

## Usage

```go
import (
    "io"
    "log"
    "os"
    "os/exec"

    "github.com/deepnoodle-ai/wonton/pty"
)

cmd := exec.Command("bash", "-i")
p, err := pty.Start(cmd, &pty.Size{Rows: 24, Cols: 80})
if err != nil {
    log.Fatal(err)
}
defer p.Close()

// Stream child output.
go io.Copy(os.Stdout, p)

// Send input to the child.
io.WriteString(p, "echo hello\n")

// Forward a resize.
p.Resize(pty.Size{Rows: 40, Cols: 120})

cmd.Wait()
```

Use `pty.Open` if you want the master/slave pair without starting a process.

## Platform support

| Platform | Status |
|----------|--------|
| Linux    | Supported (UNIX 98 ptmx). |
| macOS    | Supported. See caveat below. |
| Windows  | Not supported. Returns `ErrUnsupported`. |
| BSD      | Not supported. Returns `ErrUnsupported`. |

On macOS the Go runtime keeps `/dev/ptmx` in blocking mode, so `SetDeadline` on the master does not wake a blocked `Read`, and closing the master from another goroutine will not unblock one either. If you need those semantics on macOS, set the fd non-blocking yourself after `Open`:

```go
syscall.SetNonblock(int(p.Fd()), true)
```

Background: golang/go#22099, creack/pty#52.

## Concurrency

`PTY.Close` is safe to call from multiple goroutines. Only the first call closes the fd; the rest return nil. The master pointer is stored atomically so `Close` racing with `Read`, `Write`, `Resize`, `Fd`, or `File` on another goroutine is defined behavior.

## API

| Name | Purpose |
|------|---------|
| `Open() (*PTY, *os.File, error)` | Allocate a master/slave pair. Caller closes both sides. |
| `Start(cmd, size) (*PTY, error)` | Allocate and start `cmd` on the slave. `size` may be nil. |
| `StartWithAttrs(cmd, size, attrs)` | Like `Start`, but lets you pass a base `*syscall.SysProcAttr`. |
| `PTY.Read / Write / Close` | `io.ReadWriteCloser` on the master. |
| `PTY.Resize(Size)` | Set the window size. The child receives `SIGWINCH`. |
| `PTY.GetSize() (Size, error)` | Read the current window size. |
| `PTY.InheritSize(*os.File)` | Copy the window size from another TTY (typically your own stdin). |
| `PTY.Fd() / File()` | Access the underlying fd or `*os.File` for `SetDeadline`, `SyscallConn`, or direct syscalls. |
| `Size{Rows, Cols}` | Window dimensions in rows and columns. |
| `ErrUnsupported` | Returned on platforms with no PTY backend. |
