//go:build !darwin && !linux

package tui

import (
	"io"
	"os"
	"time"
)

// queryBackground is not supported on this platform; DetectTerminalBackground falls
// back to COLORFGBG.
func queryBackground(*os.File, io.Writer, time.Duration) (RGB, bool, error) {
	return RGB{}, false, nil
}
