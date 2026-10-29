package tmux

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/terminal"
)

// DefaultCaptureInterval is the shortest time between two captures of a busy
// pane: about one frame at 60 frames per second.
const DefaultCaptureInterval = 16 * time.Millisecond

// TerminalOptions configures Server.Open.
type TerminalOptions struct {
	// Session is the name of the tmux session behind the terminal. Open
	// attaches to it if it exists and creates it if not.
	Session string

	// Dir, Command, and Env configure a session Open creates; see
	// SessionOptions. They are ignored when the session already exists.
	Dir     string
	Command string
	Env     []string

	// Width and Height are the pane's size in cells. An existing session is
	// resized to match. Zero keeps tmux's size.
	Width, Height int

	// OnUpdate is called from a background goroutine each time Screen has a
	// new snapshot, and once more when the terminal ends. A TUI typically
	// wakes its event loop here (tui.Runtime.SendEvent). It must not block.
	OnUpdate func()

	// CaptureInterval is the shortest time between two captures of a busy
	// pane. Zero means DefaultCaptureInterval.
	CaptureInterval time.Duration
}

// Terminal is a live tmux pane mirrored into the program: the session's
// current pane, captured into a Screen whenever its program writes, and fed
// keys, pastes, and mouse events. It is safe for concurrent use.
//
// The tmux session is independent of the Terminal. Detach leaves it running
// for the next Open; Kill ends it.
type Terminal struct {
	server   *Server
	ctl      *Control
	pane     string // pane ID, a stable target that survives renames
	sessID   string // session ID ("$1"), which also survives renames
	onUpdate func()
	interval time.Duration
	buffer   string // paste-buffer name private to this terminal
	verbatim string // "-S " where paste-buffer can skip sanitizing, else ""
	done     chan struct{}

	mu         sync.Mutex
	session    string
	screen     *Screen
	width      int // the size last asked of tmux
	height     int
	retryAt    time.Time // when a size tmux refused may be asked again
	capturing  bool
	dirty      bool // output arrived while a capture was running
	timer      *time.Timer
	last       time.Time // when the last capture started
	lastOutput time.Time
	detaching  bool
	started    bool // the first capture is done; notifications count from here
}

// Open attaches a Terminal to a session, creating the session first if it
// does not exist. When Open returns, Screen already holds the pane's
// contents.
func (s *Server) Open(ctx context.Context, opts TerminalOptions) (*Terminal, error) {
	if opts.Session == "" {
		return nil, errors.New("tmux: TerminalOptions.Session is required")
	}
	if !s.Available() {
		return nil, ErrNotInstalled
	}
	if !s.HasSession(ctx, opts.Session) {
		err := s.NewSession(ctx, SessionOptions{
			Name:    opts.Session,
			Dir:     opts.Dir,
			Command: opts.Command,
			Width:   opts.Width,
			Height:  opts.Height,
			Env:     opts.Env,
		})
		if err != nil {
			return nil, err
		}
	}

	t := &Terminal{
		server:   s,
		session:  opts.Session,
		onUpdate: opts.OnUpdate,
		interval: opts.CaptureInterval,
		done:     make(chan struct{}),
	}
	if t.interval <= 0 {
		t.interval = DefaultCaptureInterval
	}

	// The control connection is not bound to ctx: the terminal lives until
	// Detach, Kill, or the session ends, not until Open's caller returns.
	ctl, err := s.Attach(context.WithoutCancel(ctx), opts.Session, t.notified)
	if err != nil {
		return nil, err
	}
	t.ctl = ctl

	out, err := ctl.Command("display-message -p -t " + Quote("="+opts.Session+":") + " '#{pane_id} #{session_id}'")
	if err != nil || len(out) == 0 {
		_ = ctl.Close()
		if err == nil {
			err = errors.New("tmux: session has no pane")
		}
		return nil, err
	}
	t.pane, t.sessID, _ = strings.Cut(strings.TrimSpace(out[0]), " ")
	t.buffer = "wonton-" + strings.TrimPrefix(t.pane, "%")

	// paste-buffer turns control characters into visible ones unless given
	// -S, which tmux 3.5 added. Ask for it with a buffer that does not exist:
	// only an older tmux complains about the flag.
	if _, err := ctl.Command("paste-buffer -S -b " + Quote(t.buffer+"-probe") + " -t " + t.pane); err == nil ||
		!strings.Contains(err.Error(), "unknown flag") {
		t.verbatim = "-S "
	}

	if opts.Width > 0 && opts.Height > 0 {
		t.width, t.height = opts.Width, opts.Height
		if _, err := ctl.Command(t.resizeCommand(opts.Width, opts.Height)); err != nil {
			t.retryAt = time.Now().Add(resizeRetryDelay)
		}
	}
	if err := t.capture(); err != nil {
		_ = ctl.Close()
		return nil, err
	}
	t.start()

	go func() {
		<-ctl.Done()
		t.mu.Lock()
		if t.timer != nil {
			t.timer.Stop()
		}
		t.mu.Unlock()
		close(t.done)
		t.update()
	}()
	return t, nil
}

// Session returns the name of the terminal's tmux session.
func (t *Terminal) Session() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.session
}

// PaneID returns tmux's ID for the terminal's pane ("%3").
func (t *Terminal) PaneID() string { return t.pane }

// Screen returns the latest snapshot of the pane. It never returns nil.
func (t *Terminal) Screen() *Screen {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.screen
}

// LastOutput returns when the pane's program last wrote anything.
func (t *Terminal) LastOutput() time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastOutput
}

// Done is closed when the terminal ends: its session exited or was killed,
// the server stopped, or Detach was called.
func (t *Terminal) Done() <-chan struct{} { return t.done }

// Exited reports whether the terminal ended on its own — its program exited
// or its session was killed — rather than by Detach.
func (t *Terminal) Exited() bool {
	select {
	case <-t.done:
	default:
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return !t.detaching
}

// Detach disconnects from the session and leaves it running.
func (t *Terminal) Detach() error {
	t.mu.Lock()
	t.detaching = true
	t.mu.Unlock()
	return t.ctl.Close()
}

// Kill ends the session and every process in it.
func (t *Terminal) Kill() error {
	_, err := t.ctl.Command("kill-session -t " + Quote(t.sessID))
	if err != nil && !errors.Is(err, ErrClosed) {
		return err
	}
	<-t.done
	return nil
}

// Rename renames the terminal's session.
func (t *Terminal) Rename(name string) error {
	if name == "" || strings.ContainsAny(name, ":.") {
		return fmt.Errorf("tmux: invalid session name %q", name)
	}
	_, err := t.ctl.Command("rename-session -t " + Quote(t.sessID) + " " + Quote(name))
	if err == nil {
		t.mu.Lock()
		t.session = name
		t.mu.Unlock()
	}
	return err
}

// resizeRetryDelay is how long Resize waits before asking again for a size
// tmux refused, so a caller that resizes on every frame does not send it on
// every frame.
const resizeRetryDelay = time.Second

// Resize sets the pane's size in cells. It returns at once; the screen is
// recaptured once tmux has resized the pane. Asking for the size already
// asked for does nothing, unless tmux refused it more than a second ago.
func (t *Terminal) Resize(width, height int) {
	if width < 1 || height < 1 {
		return
	}
	t.mu.Lock()
	if width == t.width && height == t.height && (t.retryAt.IsZero() || time.Now().Before(t.retryAt)) {
		t.mu.Unlock()
		return
	}
	t.width, t.height = width, height
	t.retryAt = time.Time{}
	t.mu.Unlock()
	select {
	case <-t.done:
		return
	default:
	}
	errc := t.ctl.Go(t.resizeCommand(width, height))
	go func() {
		if err, ok := <-errc; ok && err != nil {
			t.mu.Lock()
			if t.width == width && t.height == height {
				t.retryAt = time.Now().Add(resizeRetryDelay)
			}
			t.mu.Unlock()
		}
	}()
}

func (t *Terminal) resizeCommand(width, height int) string {
	return fmt.Sprintf("resize-window -t %s -x %d -y %d", t.pane, width, height)
}

// SendKey types a key into the pane. A key typed while the pane is in copy
// mode scrolls the mode for the navigation keys and leaves the mode
// otherwise, the way a terminal's scrollback behaves.
func (t *Terminal) SendKey(ev terminal.KeyEvent) error {
	if ev.Paste != "" {
		return t.Paste(ev.Paste)
	}
	name, literal, ok := KeyName(ev)
	if !ok {
		return nil
	}
	if t.Screen().InMode {
		if cmd, ok := modeKey(ev); ok {
			return t.async(fmt.Sprintf("send-keys -t %s -X %s", t.pane, cmd))
		}
		// Its own command, not a list: the screen may be stale and the pane
		// already out of the mode, and a failed cancel would skip the key.
		cancel := t.async(fmt.Sprintf("send-keys -t %s -X cancel", t.pane))
		if ev.Key == terminal.KeyEscape {
			return cancel
		}
	}
	if literal {
		return t.async(fmt.Sprintf("send-keys -t %s -l %s", t.pane, Quote(name)))
	}
	return t.async(fmt.Sprintf("send-keys -t %s %s", t.pane, Quote(name)))
}

// modeKey maps the keys that navigate scrollback to copy-mode commands.
func modeKey(ev terminal.KeyEvent) (string, bool) {
	switch ev.Key {
	case terminal.KeyArrowUp:
		return "scroll-up", true
	case terminal.KeyArrowDown:
		return "scroll-down", true
	case terminal.KeyPageUp:
		return "page-up", true
	case terminal.KeyPageDown:
		return "page-down", true
	case terminal.KeyHome:
		return "history-top", true
	case terminal.KeyEnd:
		return "history-bottom", true
	}
	return "", false
}

// SendText types text into the pane as if each character were a key.
func (t *Terminal) SendText(text string) error {
	if text == "" {
		return nil
	}
	return t.async(fmt.Sprintf("send-keys -t %s -l %s", t.pane, Quote(text)))
}

// Paste pastes text into the pane. If the pane's program has turned on
// bracketed paste, the text arrives bracketed, so a shell or editor can tell
// it from typing. Newlines arrive as carriage returns, as from a terminal.
func (t *Terminal) Paste(text string) error {
	if text == "" {
		return nil
	}
	text = stripPasteEnd(text)
	return t.async(fmt.Sprintf("set-buffer -b %s %s ; paste-buffer %s-p -d -b %s -t %s",
		t.buffer, Quote(text), t.verbatim, t.buffer, t.pane))
}

// stripPasteEnd removes every bracketed paste end marker from text, which
// could otherwise end the paste early and run the rest as typed commands.
// Removing one can join the pieces around it into another, so it repeats.
func stripPasteEnd(text string) string {
	for strings.Contains(text, "\x1b[201~") {
		text = strings.ReplaceAll(text, "\x1b[201~", "")
	}
	return text
}

// write sends bytes to the pane's program unchanged.
func (t *Terminal) write(data string) error {
	return t.async(fmt.Sprintf("set-buffer -b %s %s ; paste-buffer %s-r -d -b %s -t %s",
		t.buffer, Quote(data), t.verbatim, t.buffer, t.pane))
}

// SendMouse forwards a mouse event at the pane-local cell (x, y) to the
// pane's program, if the program has turned on mouse reporting. It reports
// whether the event was sent; when it was not, the caller is free to give
// the event its own meaning.
func (t *Terminal) SendMouse(ev terminal.MouseEvent, x, y int) bool {
	screen := t.Screen()
	if !screen.MouseReporting {
		return false
	}
	// Motion goes only to a program that asked for it.
	switch {
	case ev.Type == terminal.MouseDrag && !screen.MouseDrags,
		ev.Type == terminal.MouseMove && !screen.MouseMotion:
		return false
	}
	seq, ok := MouseSequence(ev, x, y, screen.MouseSGR)
	if !ok {
		return false
	}
	_ = t.write(seq)
	return true
}

// Scroll scrolls the pane by lines: negative up into the history, positive
// back down. A full-screen program on the alternate screen gets cursor keys
// instead, the way terminals translate the wheel there. Otherwise scrolling
// up enters copy mode, which leaves again on its own when scrolled back to
// the bottom.
func (t *Terminal) Scroll(lines int) {
	if lines == 0 {
		return
	}
	screen := t.Screen()
	n := lines
	dir := "down"
	if lines < 0 {
		n = -lines
		dir = "up"
	}
	switch {
	case screen.InMode:
		_ = t.async(fmt.Sprintf("send-keys -t %s -X -N %d scroll-%s", t.pane, n, dir))
	case screen.AltScreen:
		key := "Down"
		if dir == "up" {
			key = "Up"
		}
		_ = t.async(fmt.Sprintf("send-keys -t %s -N %d %s", t.pane, n, key))
	case dir == "up":
		_ = t.async(fmt.Sprintf("copy-mode -e -t %s ; send-keys -t %s -X -N %d scroll-up", t.pane, t.pane, n))
	}
}

// Command runs a tmux command over the terminal's connection; see
// Control.Command. Use PaneID to target the terminal's pane.
func (t *Terminal) Command(command string) ([]string, error) {
	return t.ctl.Command(command)
}

// async sends a command without waiting for tmux to answer, so a keystroke
// never waits on a round trip. The reply is consumed in the background.
func (t *Terminal) async(command string) error {
	select {
	case <-t.done:
		return ErrClosed
	default:
	}
	errc := t.ctl.Go(command)
	select {
	case err, ok := <-errc:
		// Go fails synchronously when the command could not be sent.
		if ok && err != nil && !isReplyError(err) {
			return err
		}
	default:
	}
	return nil
}

func isReplyError(err error) bool {
	var ce *CommandError
	return errors.As(err, &ce)
}

// notified runs on the control connection's reader goroutine, so it only
// schedules work.
func (t *Terminal) notified(n Notification) {
	t.mu.Lock()
	started := t.started
	if !started && n.Name == "output" {
		// Open is still taking its first capture, which may have missed
		// this. start captures again once it is done.
		t.dirty = true
	}
	t.mu.Unlock()
	if !started {
		return
	}
	switch n.Name {
	case "session-changed":
		// With detach-on-destroy off, tmux moves the client to another
		// session when ours ends rather than detaching it. Either way the
		// terminal is over. Close waits for this goroutine, so not here.
		if id, _, _ := strings.Cut(n.Args, " "); id != t.sessID {
			go t.ctl.Close()
		}
	case "output":
		if n.Pane != t.pane {
			return
		}
		t.mu.Lock()
		t.lastOutput = time.Now()
		t.mu.Unlock()
		t.requestCapture()
	case "layout-change", "pane-mode-changed", "window-pane-changed", "session-window-changed":
		t.requestCapture()
	}
}

// start lets notifications schedule captures, once Open has the pane and
// its first capture. Output that arrived before then gets a capture now.
func (t *Terminal) start() {
	t.mu.Lock()
	t.started = true
	missed := t.dirty
	t.mu.Unlock()
	if missed {
		t.requestCapture()
	}
}

// requestCapture captures the pane soon. At most one capture runs at a time,
// and captures of a busy pane are spaced by the capture interval; output
// that arrives meanwhile is picked up by the next one.
func (t *Terminal) requestCapture() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.capturing || t.timer != nil {
		t.dirty = true
		return
	}
	if wait := t.interval - time.Since(t.last); wait > 0 {
		t.timer = time.AfterFunc(wait, func() {
			t.mu.Lock()
			t.timer = nil
			t.mu.Unlock()
			t.requestCapture()
		})
		return
	}
	t.capturing = true
	t.dirty = false
	go func() {
		_ = t.capture()
		t.mu.Lock()
		t.capturing = false
		again := t.dirty
		t.mu.Unlock()
		if again {
			t.requestCapture()
		}
	}()
}

// captureFormat asks for everything Screen reports besides the cells. The
// fields are separated by the unit separator, which no title or path holds.
var captureFormat = strings.Join([]string{
	"#{pane_id}", "#{pane_width}", "#{pane_height}",
	"#{cursor_x}", "#{cursor_y}", "#{cursor_flag}",
	"#{alternate_on}", "#{mouse_any_flag}", "#{mouse_sgr_flag}",
	"#{pane_in_mode}", "#{scroll_position}", "#{history_size}",
	"#{copy_cursor_x}", "#{copy_cursor_y}",
	"#{mouse_button_flag}", "#{mouse_all_flag}",
	"#{pane_current_command}", "#{pane_current_path}", "#{pane_title}",
}, "\x1f")

const captureFields = 19

// WrappedRows reports, for each row of the latest Screen, whether the row
// continues on the next one: part of a line too long for the pane, which the
// terminal wrapped. Text copied across such rows joins them without a line
// break, the way a terminal copies; see tui.GridSelection.Text.
//
// It asks tmux each time, so call it when copying, not every frame.
func (t *Terminal) WrappedRows() ([]bool, error) {
	screen := t.Screen()
	rng := ""
	if screen.InMode && screen.ScrollPosition > 0 {
		top := -screen.ScrollPosition
		rng = fmt.Sprintf(" -S %d -E %d", top, top+screen.Height-1)
	}
	// tmux has no per-row wrap flag to ask for. Capture the rows as they
	// are and again with wrapped lines joined, and compare.
	out, err := t.ctl.Command(fmt.Sprintf("capture-pane -p -N%s -t %s ; capture-pane -p -J%s -t %s",
		rng, t.pane, rng, t.pane))
	if err != nil {
		return nil, err
	}
	n := min(screen.Height, len(out))
	return wrappedRows(out[:n], out[n:]), nil
}

// wrappedRows matches rows to the joined lines they make up. A row that is
// not a whole line on its own is the start of its line, so it wraps.
func wrappedRows(rows, joined []string) []bool {
	wrapped := make([]bool, len(rows))
	i := 0
	for _, line := range joined {
		if i >= len(rows) {
			break
		}
		want := strings.TrimRight(line, " ")
		acc := rows[i]
		for strings.TrimRight(acc, " ") != want && i+1 < len(rows) &&
			strings.HasPrefix(line, strings.TrimRight(acc, " ")) {
			wrapped[i] = true
			i++
			acc += rows[i]
		}
		i++
	}
	return wrapped
}

// capture takes one snapshot of the pane. The format line and the capture
// travel as one command, so they describe the same instant.
func (t *Terminal) capture() error {
	t.mu.Lock()
	t.last = time.Now()
	t.mu.Unlock()

	out, err := t.ctl.Command(fmt.Sprintf(
		"display-message -p -t %s %s ; capture-pane -p -e -N -t %s",
		t.pane, Quote(captureFormat), t.pane))
	if err != nil {
		return err
	}
	screen, err := parseSnapshot(out)
	if err != nil {
		return err
	}
	if screen.InMode && screen.ScrollPosition > 0 {
		// Copy mode shows history, which a plain capture does not: ask for
		// the lines the mode is showing.
		top := -screen.ScrollPosition
		lines, err := t.ctl.Command(fmt.Sprintf("capture-pane -p -e -N -S %d -E %d -t %s",
			top, top+screen.Height-1, t.pane))
		if err != nil {
			return err
		}
		screen.Cells = ParseCapture(lines, screen.Width, screen.Height)
	}
	t.mu.Lock()
	t.screen = screen
	t.mu.Unlock()
	t.update()
	return nil
}

func parseSnapshot(out []string) (*Screen, error) {
	if len(out) == 0 {
		return nil, errors.New("tmux: empty capture")
	}
	f := formatFields(out[0])
	if len(f) < captureFields {
		return nil, fmt.Errorf("tmux: unexpected capture header %q", out[0])
	}
	num := func(i int) int {
		n, _ := strconv.Atoi(f[i])
		return n
	}
	s := &Screen{
		PaneID:         f[0],
		Width:          num(1),
		Height:         num(2),
		CursorX:        num(3),
		CursorY:        num(4),
		CursorVisible:  f[5] == "1",
		AltScreen:      f[6] == "1",
		MouseReporting: f[7] == "1",
		MouseSGR:       f[8] == "1",
		InMode:         f[9] == "1",
		ScrollPosition: num(10),
		HistorySize:    num(11),
		MouseDrags:     f[14] == "1" || f[15] == "1",
		MouseMotion:    f[15] == "1",
		Command:        f[16],
		Path:           f[17],
		Title:          strings.Join(f[18:], "\x1f"),
		Captured:       time.Now(),
	}
	if s.InMode {
		s.CursorX, s.CursorY = num(12), num(13)
		s.CursorVisible = true
	}
	s.Cells = ParseCapture(out[1:], s.Width, s.Height)
	return s, nil
}

func (t *Terminal) update() {
	if t.onUpdate != nil {
		t.onUpdate()
	}
}
