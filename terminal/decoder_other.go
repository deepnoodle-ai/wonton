//go:build !(darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || aix)

package terminal

import (
	"os"
	"time"
)

// fileWaiter returns nil: polling a console handle is not supported here, so
// the decoder does not wait after an ESC byte.
func fileWaiter(*os.File) func(time.Duration) bool { return nil }
