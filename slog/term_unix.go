//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package slog

import "golang.org/x/sys/unix"

const ioctlReadTermios = unix.TIOCGETA
