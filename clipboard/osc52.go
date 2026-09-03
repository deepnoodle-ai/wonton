package clipboard

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// OSC52Limit is the most text WriteOSC52 will send. Terminals impose their own
// caps on an OSC 52 payload and a sequence over the limit is usually dropped
// whole rather than truncated, so a very large selection is refused here
// instead of appearing to work.
const OSC52Limit = 100 * 1024

// ErrTooLarge is returned by WriteOSC52 for text over OSC52Limit bytes.
var ErrTooLarge = errors.New("clipboard: text exceeds the OSC 52 limit")

// WriteOSC52 asks the terminal to put text on the system clipboard, by writing
// an OSC 52 sequence to w.
//
// This is the rung of the ladder that works when nothing local does: over SSH
// the native clipboard tools address the wrong machine, and OSC 52 addresses
// the terminal the user is actually sitting at. It has no reply, so a
// successful write here means the bytes were sent and nothing more — not every
// terminal implements OSC 52, and some require it to be turned on. Tell the
// user what was attempted, never that it landed.
//
// w must be the terminal itself. Write to it from the goroutine that owns the
// terminal — for an application on a Runtime, that is the render goroutine —
// or the sequence can be cut in half by a concurrent frame.
//
// Under tmux, prefer handing the text to tmux (`tmux load-buffer -w -`): the
// DCS passthrough this would otherwise need depends on tmux's own
// `set-clipboard` setting and fails silently when it is off.
func WriteOSC52(w io.Writer, text string) error {
	if len(text) > OSC52Limit {
		return fmt.Errorf("%w: %d bytes, limit %d", ErrTooLarge, len(text), OSC52Limit)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	// c is the clipboard selection; BEL terminates. ST (ESC \) is the other
	// legal terminator, but BEL is accepted more widely.
	_, err := fmt.Fprintf(w, "\033]52;c;%s\a", encoded)
	return err
}
