//go:build !darwin && !linux

package terminalinput

import "os"

type Owner struct{}

func New(*os.File) (*Owner, error)      { return nil, ErrUnsupported }
func (*Owner) Read([]byte) (int, error) { return 0, ErrUnsupported }
func (*Owner) Pause() error             { return ErrUnsupported }
func (*Owner) DiscardResidual() error   { return ErrUnsupported }
func (*Owner) Resume() error            { return ErrUnsupported }
func (*Owner) Stop()                    {}
func (*Owner) Close() error             { return nil }
