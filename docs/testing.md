# Testing

This document covers how wonton's test suite is organized, how to run it
locally (including for a different OS than the one you're developing on),
and what is and isn't covered by CI.

## Running tests locally

The baseline is a single command:

```bash
go test ./...
```

That runs every package against whatever OS and architecture you're on. On
an Apple M4, that's `darwin/arm64`; on a cloud Linux box it's
`linux/amd64` or `linux/arm64`. There are no build flags to set — every
test that is gated to a specific OS uses `//go:build` tags, so the Go
toolchain picks the right subset automatically.

Useful variants:

```bash
go test -race ./...                 # Race detector. Slower but catches real bugs.
go test -run TestSection8 ./...     # Filter by test name.
go test -cover ./...                # Per-package coverage summary.
go vet ./...                        # Catches a lot without running anything.
```

## Reproducing the Linux CI job from macOS

CI (`.github/workflows/go-test.yml`) runs the suite on `ubuntu-latest`,
`macos-latest`, and `windows-latest`. When a test passes locally on your
Mac but goes red in CI on Ubuntu, `scripts/test-linux.sh` reproduces the
Linux job inside Docker without leaving your dev box:

```bash
./scripts/test-linux.sh                           # full suite
./scripts/test-linux.sh -race ./pty/...           # race detector, pty only
./scripts/test-linux.sh -run TestSession_Resize ./termsession/
```

The script builds `scripts/Dockerfile.linux` on first run (a `golang:1.25-alpine`
base with `git` and the module cache pre-warmed), then mounts the working
copy at `/src` and runs `go test`. Any arguments you pass are forwarded
to `go test` inside the container.

### Why the `-t` flag matters

The wonton `pty/` and `termsession/` packages allocate real pseudo-terminals
and expect the test process to have a real controlling TTY (`test -t 0`
must succeed inside the child process). Docker containers get a TTY only
when you pass `-t` at `docker run` time, so `scripts/test-linux.sh` always
sets it. If you ever run the container directly without `-t`, the PTY tests
will fail or flake.

## Cross-compilation checks

You don't need a Linux box to catch many Linux-only build errors. From
macOS or anywhere else, you can cross-compile:

```bash
GOOS=linux   GOARCH=386   go build ./...   # 32-bit pointer sanity
GOOS=linux   GOARCH=arm64 go build ./...
GOOS=windows GOARCH=amd64 go vet ./...
```

These three entries match the `cross-compile` job in
`.github/workflows/go-test.yml`. The native builds (`linux/amd64`,
`darwin/amd64`) are covered by the `test` matrix on their respective
runners, so they aren't repeated here. The `linux/386` entry exists
specifically to catch any `int`/`uintptr` size assumptions that would
only surface on a 32-bit build — in particular, anything in `pty/` that
touches raw ioctl argument layouts or anything in `runewidth/` that
indexes the 2-stage BMP+SMP lookup tables.

## What's OS-gated and why

| Package | Gate | Reason |
|---|---|---|
| `pty/pty_unix.go` | `darwin \|\| linux` | `/dev/ptmx` + ioctl layout is POSIX-specific. |
| `pty/pty_darwin.go` | filename tag | Uses TIOCPTYGNAME/TIOCPTYGRANT/TIOCPTYUNLK. |
| `pty/pty_linux.go` | filename tag | Uses TIOCGPTN/TIOCSPTLCK + `/dev/pts/N` lookup. |
| `pty/pty_windows.go` | filename tag | Returns `ErrUnsupported` — no ConPTY yet. |
| `pty/pty_other.go` | `!darwin && !linux && !windows` | Fallback: `ErrUnsupported`. |
| `pty/pty_{linux,darwin,windows,unix}_test.go` | matching build tags | Per-OS behavioral tests. |
| `termsession/session_{unix,windows}.go` | OS-specific | `SIGWINCH` handling vs. no-op. |
| `termsession/session_e2e_test.go` | `darwin \|\| linux` | Spawns real `sh` with SIGWINCH traps. |
| `terminal/terminal_{unix,windows}.go` | OS-specific | Same SIGWINCH split. |

Two deliberate non-gates worth calling out:

- **`runewidth/`** has no OS-gated code at all. The 2-stage lookup tables are
  pure Go byte strings; there is no endianness or pointer-size concern. The
  Section 8 correctness pins in `runewidth/section8_test.go` run on every
  platform.
- **`pty/pty_test.go`** (the original test file) is not build-tagged — it
  uses a runtime `skipIfUnsupported` switch instead. This lets it compile
  on every OS (including Windows, where it becomes a no-op that the
  Windows-specific contract test supplements).

## Platform support explicitly not in scope

**BSDs (FreeBSD, NetBSD, OpenBSD, DragonFly).** Earlier best-effort BSD
shims existed in the `pty/` package but never compiled against the current
`golang.org/x/sys/unix` release — three of the four used symbols that
don't exist for their target OS. Rather than keep code in the tree that
lies about working, they were removed when the test suite was expanded.
If BSD support matters to you, the path forward is to add it back under
a CI job that actually runs on a BSD VM (via `cross-platform-actions/action`
or similar), not as a compile-only shim.

**ConPTY on Windows.** `pty_windows.go` returns `ErrUnsupported` for every
entry point. Adding real Windows PTY support would mean wiring up
`CreatePseudoConsole` and its attendant handle dance, which is a real
project rather than a stub. Until then, the Windows contract test in
`pty_windows_test.go` pins the `ErrUnsupported` behavior so callers can
feature-detect reliably.

## Runtime gates inside tests

A handful of tests do runtime rather than compile-time skipping. The ones
worth knowing about:

- **`pty_test.go:skipIfUnsupported`** — runtime skip on non-darwin/linux.
- **`termsession/session_e2e_test.go:devNull`** — skips on Windows because
  `/dev/null` isn't the right path and the test expects POSIX file
  semantics.

If you add a new platform-specific test, prefer a `//go:build` tag over a
runtime skip unless you have a specific reason (usually: you want a single
test file to document a cross-platform contract).

## Coverage snapshot

The weakest package by line coverage is `tui/` (~57%) — that's UI code
and not the cross-platform correctness story this document covers. Every
package in the focus list for cross-platform confidence (`pty/`,
`termsession/`, `terminal/`, `runewidth/`) sits in the 70–90% range and
is exercised by both CI and the Docker Linux runner.

Regenerate the snapshot any time with:

```bash
go test -cover ./...
make cover-text       # per-function drill-down
make cover-html       # HTML report
```
