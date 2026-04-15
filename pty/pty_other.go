//go:build !darwin && !linux && !windows

package pty

import (
	"os"
	"os/exec"
	"syscall"
)

func (p *PTY) Resize(size Size) error     { return ErrUnsupported }
func (p *PTY) GetSize() (Size, error)     { return Size{}, ErrUnsupported }
func (p *PTY) InheritSize(*os.File) error { return ErrUnsupported }

func Open() (*PTY, *os.File, error)        { return nil, nil, ErrUnsupported }
func Start(*exec.Cmd, *Size) (*PTY, error) { return nil, ErrUnsupported }
func StartWithAttrs(*exec.Cmd, *Size, *syscall.SysProcAttr) (*PTY, error) {
	return nil, ErrUnsupported
}
