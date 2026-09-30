// Package terminalstate connects terminal's private state to its TUI runner.
// The terminal package installs one immutable accessor during initialization;
// no terminal instances or lifecycle registrations are retained globally.
package terminalstate

import "io"

type Access struct {
	FD             int
	Output         io.Writer
	Snapshot       func() (Transition, error)
	CleanupRuntime func(raw, kitty bool) error
}
type Transition struct {
	Release func() error
	Restore func(stopping bool) error
}

// Inspect is installed only by the terminal package during package init.
var Inspect func(any) (Access, bool)
