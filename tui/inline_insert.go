package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// This file lets InlineApp.Print add rows above the live region without
// redrawing it. The design and the rejected scroll-region approach are in
// docs/prds/scroll-regions.md.
//
// From the live region's last row, where the cursor rests, Print writes:
//
//  1. n line feeds. Where the live region sits at the screen bottom, they
//     scroll the whole screen up, and the top rows go into scrollback in
//     order. Where there are blank rows below it, the first line feeds only
//     move the cursor down.
//  2. A carriage return and "cursor up n+L-1" (L is the live region's
//     height). Either way this reaches the live region's top row.
//  3. "Insert n lines", which pushes the live region back down into the rows
//     the line feeds opened.
//  4. The n rows, then "cursor down L" back to the live region's last row.
//
// Every move is relative to the cursor, so the result does not depend on
// the screen height or on where the live region is. A height change that
// lands while the bytes are on their way cannot make it scroll the wrong
// rows.
//
// A width change can: a row wider than the screen would wrap onto the next
// row. So Print reads the real width first and redraws instead when the
// screen is narrower than the rows. Autowrap is also off while the rows are
// written (CSI ? 7 l ... CSI ? 7 h), so a narrowing that lands while the
// bytes travel cuts the end of a row and never writes over another row or
// the live region.

// insertChunk returns how many rows one insert may add above a live region
// of liveHeight rows on a screen of screenHeight rows, or 0 when Print must
// redraw instead. An insert of n rows is correct while n + liveHeight fits
// the screen; half the free rows leaves room for a screen that shrank and
// has not told the runtime yet.
func insertChunk(screenHeight, liveHeight int) int {
	free := screenHeight - liveHeight
	if screenHeight <= 0 || liveHeight <= 0 || free < 2 {
		return 0
	}
	return free / 2
}

// insertAboveLive builds the bytes that add rows above a live region of
// liveHeight rows, at most chunk rows per insert, inside one
// synchronized-output bracket. Attributes are reset before the line feeds
// and the insert, so the rows they open take the default background
// (background color erase). Each rendered row ends its own styles.
func insertAboveLive(rows []string, liveHeight, chunk int) string {
	var b strings.Builder
	b.WriteString("\x1b[?2026h\x1b[?7l")
	for len(rows) > 0 {
		n := min(chunk, len(rows))
		b.WriteString("\x1b[0m")
		b.WriteString(strings.Repeat("\n", n))
		b.WriteByte('\r')
		if up := n + liveHeight - 1; up > 0 {
			fmt.Fprintf(&b, "\x1b[%dA", up)
		}
		fmt.Fprintf(&b, "\x1b[%dL", n)
		b.WriteString(strings.Join(rows[:n], "\r\n"))
		fmt.Fprintf(&b, "\x1b[%dB", liveHeight)
		rows = rows[n:]
	}
	b.WriteString(insertEnd)
	return b.String()
}

// insertEnd turns autowrap back on and ends synchronized output.
const insertEnd = "\x1b[?7h\x1b[?2026l"

// printAboveLiveLocked adds rendered rows above the live region without
// redrawing it, and reports whether it did. It does not when there is no
// live region on screen, the live region leaves too little room, the size
// of the terminal is unknown, or the terminal is narrower than the rows
// (it narrowed and the event loop has not heard yet). The caller holds r.mu
// and the output gate.
func (r *InlineApp) printAboveLiveLocked(rendered string) bool {
	if !r.live.started {
		return false
	}
	// The live region must be drawn at the current width: after a width
	// change the terminal may have rewrapped its rows, so it can take more
	// rows than lastHeight until the next frame redraws it.
	if r.live.drawnWidth != r.config.Width {
		return false
	}
	width, height, ok := r.screenSize()
	if !ok || width < r.config.Width {
		return false
	}
	chunk := insertChunk(height, r.live.lastHeight)
	if chunk == 0 {
		return false
	}
	out := &checkedLiveOutput{destination: r.output}
	io.WriteString(out, insertAboveLive(strings.Split(rendered, "\r\n"), r.live.lastHeight, chunk))
	if out.err != nil {
		// Do not leave autowrap off, or the terminal holding its screen,
		// for a write that will never end.
		io.WriteString(r.output, insertEnd)
	}
	return true
}

// terminalSize reads the size of the terminal that w writes to. ok is false
// when w is not a terminal. Width and height always come from the same
// terminal: the one that shows the output.
func terminalSize(w io.Writer) (width, height int, ok bool) {
	f, isFile := w.(*os.File)
	if !isFile || !term.IsTerminal(int(f.Fd())) {
		return 0, 0, false
	}
	width, height, err := term.GetSize(int(f.Fd()))
	if err != nil || width <= 0 || height <= 0 {
		return 0, 0, false
	}
	return width, height, true
}

// refreshScreenSize reads the terminal size again when Run starts, so the
// live region is capped and Print sizes its inserts by the current height.
// NewInlineApp read it once, and the window may have changed since. A
// width that came from the terminal follows it too.
func (r *InlineApp) refreshScreenSize() {
	width, height, ok := terminalSize(r.output)
	if !ok {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.live.SetMaxHeight(height)
	if r.config.widthFromTerminal && width != r.config.Width {
		r.config.Width = width
		r.live.SetWidth(width)
	}
}
