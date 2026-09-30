// Package terminalinput provides private, acknowledged ownership of terminal
// reads. It is shared by the terminal decoder and TUI runners.
package terminalinput

import "errors"

// Boundary ends a finite pre-release byte stream. It is not application EOF.
var Boundary = errors.New("terminal input ownership boundary")
var ErrUnsupported = errors.New("managed terminal input is unsupported")
