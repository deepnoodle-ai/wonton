//go:build darwin || linux

package tui

import (
	"io"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/internal/terminalinput"
)

// queryBackground writes backgroundQuery to out and reads the replies from
// in, a terminal, for up to timeout. It stops at the end of the DA1 reply,
// so it never consumes input that follows. On a timeout it flushes pending
// input instead. See terminalinput.Query.
func queryBackground(in *os.File, out io.Writer, timeout time.Duration) (RGB, bool, error) {
	reply, err := terminalinput.Query(int(in.Fd()), out, backgroundQuery, timeout)
	color, ok := parseBackgroundReply(reply)
	return color, ok, err
}
