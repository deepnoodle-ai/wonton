//go:build linux

package slog

import "golang.org/x/sys/unix"

const ioctlReadTermios = unix.TCGETS
