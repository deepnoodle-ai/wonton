package gooey

import (
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// Common errors
var (
	ErrOutOfBounds   = errors.New("coordinates out of bounds")
	ErrNotInRawMode  = errors.New("operation requires raw mode")
	ErrClosed        = errors.New("terminal is closed")
	ErrInvalidFrame  = errors.New("invalid frame passed to EndFrame")
	ErrAlreadyActive = errors.New("component is already active")
)

// gooeyError is a helper for creating new error types within the gooey package.
// It helps maintain consistency in error messages.
type gooeyError string

func (e gooeyError) Error() string {
	return string(e)
}

// Cell represents a single character cell on the terminal
type Cell struct {
	Char         rune
	Style        Style
	Width        int  // Display width of the character (0 for continuation cells, 1-2 for actual chars)
	Continuation bool // True if this cell is a continuation of a wide character
}

// DirtyRegion tracks the rectangular area that has been modified
type DirtyRegion struct {
	MinX  int
	MinY  int
	MaxX  int
	MaxY  int
	dirty bool
}

// Empty returns true if the dirty region is empty
func (dr *DirtyRegion) Empty() bool {
	return !dr.dirty
}

// Clear resets the dirty region
func (dr *DirtyRegion) Clear() {
	dr.dirty = false
	dr.MinX = 0
	dr.MinY = 0
	dr.MaxX = 0
	dr.MaxY = 0
}

// Mark marks a cell as dirty, expanding the dirty region if necessary
func (dr *DirtyRegion) Mark(x, y int) {
	if !dr.dirty {
		dr.MinX = x
		dr.MinY = y
		dr.MaxX = x
		dr.MaxY = y
		dr.dirty = true
	} else {
		if x < dr.MinX {
			dr.MinX = x
		}
		if x > dr.MaxX {
			dr.MaxX = x
		}
		if y < dr.MinY {
			dr.MinY = y
		}
		if y > dr.MaxY {
			dr.MaxY = y
		}
	}
}

// MarkRect marks a rectangular region as dirty in O(1) time using bounding box expansion
func (dr *DirtyRegion) MarkRect(x, y, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}

	// Calculate the bounds of the rectangle
	minX := x
	minY := y
	maxX := x + width - 1
	maxY := y + height - 1

	if !dr.dirty {
		dr.MinX = minX
		dr.MinY = minY
		dr.MaxX = maxX
		dr.MaxY = maxY
		dr.dirty = true
	} else {
		// Expand bounding box to include this rectangle
		if minX < dr.MinX {
			dr.MinX = minX
		}
		if maxX > dr.MaxX {
			dr.MaxX = maxX
		}
		if minY < dr.MinY {
			dr.MinY = minY
		}
		if maxY > dr.MaxY {
			dr.MaxY = maxY
		}
	}
}

// RenderFrame represents a rendering surface for a single frame.
// All operations on a RenderFrame are atomic within the context of that frame.
//
// Thread Safety: RenderFrame is NOT thread-safe. Only one goroutine should use
// a RenderFrame at a time. The frame is obtained from BeginFrame() which locks
// the terminal, ensuring exclusive access.
type RenderFrame interface {
	// SetCell sets the character and style at the given position.
	//
	// Preconditions:
	//   - Frame must be active (between BeginFrame and EndFrame)
	//   - Coordinates must be within bounds: 0 <= x < width, 0 <= y < height
	//
	// Postconditions:
	//   - The cell at (x, y) will display 'char' with 'style' on EndFrame()
	//   - If the cell was previously set, old value is replaced
	//
	// Errors:
	//   - Returns ErrOutOfBounds if x or y are invalid
	//
	// Performance: O(1)
	SetCell(x, y int, char rune, style Style) error

	// PrintStyled outputs text at the specified position using the provided style.
	// Text automatically wraps to the next line when reaching the frame edge (default terminal behavior).
	//
	// Preconditions:
	//   - Frame must be active (between BeginFrame and EndFrame)
	//   - Starting coordinates must be within bounds
	//
	// Postconditions:
	//   - Text will be rendered starting at (x, y)
	//   - Text automatically wraps at frame edge (character-by-character)
	//   - Newlines (\n) advance to the next line
	//
	// Performance: O(len(text))
	PrintStyled(x, y int, text string, style Style) error

	// PrintTruncated outputs text at the specified position using the provided style.
	// Text is truncated (clipped) at the frame edge without wrapping to the next line.
	//
	// Use this when you want text to be cut off at the boundary rather than wrap.
	//
	// Preconditions:
	//   - Frame must be active (between BeginFrame and EndFrame)
	//   - Starting coordinates must be within bounds
	//
	// Postconditions:
	//   - Text will be rendered starting at (x, y)
	//   - Text is truncated at frame edge (no wrapping)
	//   - Newlines (\n) advance to the next line
	//
	// Performance: O(len(text))
	PrintTruncated(x, y int, text string, style Style) error

	// FillStyled fills a rectangular area with a character and a specific style.
	//
	// Preconditions:
	//   - Frame must be active (between BeginFrame and EndFrame)
	//   - Rectangle should be within bounds (out-of-bounds areas are clipped)
	//
	// Postconditions:
	//   - All cells in the rectangle will display 'char' with 'style'
	//   - Cells outside terminal boundaries are ignored
	//
	// Errors:
	//   - Returns ErrOutOfBounds if any part of the rectangle is invalid
	//
	// Performance: O(width * height)
	FillStyled(x, y, width, height int, char rune, style Style) error

	// Size returns the dimensions of the frame
	//
	// Returns: (width, height) of the terminal in character cells
	//
	// Performance: O(1)
	Size() (width, height int)

	// GetBounds returns the rectangular bounds of the frame.
	// This includes the starting (Min.X, Min.Y) and ending (Max.X, Max.Y) coordinates
	// relative to the terminal's top-left corner (0,0).
	//
	// Performance: O(1)
	GetBounds() image.Rectangle

	// SubFrame returns a new RenderFrame that is a sub-rectangle of the current frame.
	// All drawing operations on the sub-frame will be clipped to its bounds
	// and translated so that (0,0) of the sub-frame corresponds to the top-left
	// of the specified rectangle within the parent frame.
	//
	// IMPORTANT: When drawing to a SubFrame, always use coordinates relative to (0,0),
	// NOT the bounds from GetBounds().Min. The SubFrame automatically handles coordinate
	// translation.
	//
	// Example:
	//   subFrame := frame.SubFrame(image.Rect(10, 5, 30, 15))
	//   subFrame.PrintStyled(0, 0, "Hello", style)  // Correct: draws at top-left of subframe
	//   // NOT: subFrame.PrintStyled(10, 5, "Hello", style)  // Wrong: would draw outside subframe
	//
	// Preconditions:
	//   - The rectangle `rect` must be relative to the parent frame's coordinates.
	//
	// Postconditions:
	//   - A new RenderFrame is returned, clipped to the intersection of `rect`
	//     and the parent frame's bounds.
	//   - The returned frame uses local coordinates starting at (0, 0).
	//
	// Performance: O(1)
	SubFrame(rect image.Rectangle) RenderFrame

	// Fill fills the entire frame with a character and style.
	// This is a convenience method equivalent to FillStyled(0, 0, width, height, char, style).
	//
	// Use this when you want to fill the entire frame without worrying about coordinates.
	Fill(char rune, style Style) error

	// PrintHyperlink outputs a clickable hyperlink using OSC 8 protocol.
	// The hyperlink will be clickable in terminals that support OSC 8 (iTerm2, WezTerm, kitty, etc.).
	// In terminals that don't support OSC 8, the escape codes are ignored and only the text is shown.
	//
	// Preconditions:
	//   - Frame must be active (between BeginFrame and EndFrame)
	//   - Starting coordinates must be within bounds
	//   - Hyperlink must be valid (non-empty URL and text)
	//
	// Postconditions:
	//   - Hyperlink will be rendered starting at (x, y)
	//   - In supported terminals, clicking the link opens the URL
	//   - Text is styled according to the hyperlink's Style field
	//
	// Performance: O(len(text))
	PrintHyperlink(x, y int, link Hyperlink) error

	// PrintHyperlinkFallback outputs a hyperlink using fallback format: "Text (URL)".
	// Use this when you want to explicitly show the URL, or when you know the terminal
	// doesn't support OSC 8.
	//
	// Preconditions:
	//   - Frame must be active (between BeginFrame and EndFrame)
	//   - Starting coordinates must be within bounds
	//   - Hyperlink must be valid (non-empty URL and text)
	//
	// Postconditions:
	//   - Text and URL will be rendered starting at (x, y) in format "Text (URL)"
	//   - Text is styled according to the hyperlink's Style field
	//
	// Performance: O(len(text) + len(url))
	PrintHyperlinkFallback(x, y int, link Hyperlink) error
}

// terminalRenderFrame implements RenderFrame. It represents a view into the terminal's backbuffer.
type terminalRenderFrame struct {
	t      *Terminal
	bounds image.Rectangle // The actual drawing area of this frame, relative to terminal (0,0)
}

func (tf *terminalRenderFrame) GetBounds() image.Rectangle {
	return tf.bounds
}

func (tf *terminalRenderFrame) SubFrame(rect image.Rectangle) RenderFrame {
	// Calculate the intersection of the new rectangle with the current frame's bounds
	newBounds := tf.bounds.Intersect(rect.Add(tf.bounds.Min))

	// If the intersection is empty, return a frame with zero size
	if newBounds.Empty() {
		return &terminalRenderFrame{
			t:      tf.t,
			bounds: image.Rectangle{},
		}
	}

	return &terminalRenderFrame{
		t:      tf.t,
		bounds: newBounds,
	}
}

// Terminal represents a terminal interface with double-buffering capabilities.
//
// Terminal provides a high-level interface for terminal manipulation with features like:
//   - Double-buffered rendering to prevent flicker
//   - Atomic frame rendering via BeginFrame/EndFrame
//   - ANSI escape sequence generation
//   - Raw mode and alternate screen buffer support
//
// Thread Safety: All public methods are thread-safe and can be called from
// multiple goroutines. However, for rendering, use the BeginFrame/EndFrame
// pattern to ensure atomic updates.
//
// Lifecycle:
//  1. Create with NewTerminal()
//  2. Enable raw mode and alternate screen if needed
//  3. Use BeginFrame/EndFrame for rendering
//  4. Call Close() when done to restore terminal state
//
// Example:
//
//	term, _ := NewTerminal()
//	defer term.Close()
//
//	frame, _ := term.BeginFrame()
//	frame.PrintStyled(0, 0, "Hello, World!", NewStyle())
//	term.EndFrame(frame)
type Terminal struct {
	mu          sync.Mutex
	width       int
	height      int
	savedCursor Position
	altScreen   bool
	rawMode     bool
	closed      bool
	oldState    *term.State
	fd          int
	out         io.Writer // Output destination

	// Double-buffering
	buffered    bool
	backBuffer  [][]Cell
	frontBuffer [][]Cell
	virtualX    int
	virtualY    int

	// Performance optimization
	dirtyRegion DirtyRegion

	// Resize handling
	resizeChan      chan os.Signal
	stopResize      chan struct{}
	resizing        bool
	resizeCallbacks []func(width, height int)
	callbackMu      sync.RWMutex

	// Deprecated: Styles should be passed explicitly to render methods.
	// This field will be removed in v2.0.
	currentStyle Style

	// Performance metrics
	metrics        *RenderMetrics
	metricsEnabled bool

	// Session recording
	recorder *Recorder

	// Kitty keyboard protocol support
	kittySupported bool
	kittyEnabled   bool
}

// EndFrame finishes the frame, flushes the buffer to the terminal, and unlocks.
//
// Errors:
//   - Returns ErrInvalidFrame if the frame doesn't match this terminal
func (t *Terminal) EndFrame(f RenderFrame) error {
	// Ensure we are unlocking the same terminal we locked
	tf, ok := f.(*terminalRenderFrame)
	if !ok || tf.t != t {
		t.mu.Unlock() // Unlock anyway to prevent deadlock if misuse
		return ErrInvalidFrame
	}

	defer t.mu.Unlock()
	return t.flushInternal()
}

// Position represents a cursor position
type Position struct {
	X int
	Y int
}

// NewTerminal creates a new terminal instance
func NewTerminal() (*Terminal, error) {
	fd := int(os.Stdout.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal size: %w", err)
	}

	// Fallback for invalid terminal sizes (e.g. in CI/CD or piped output)
	if width <= 0 || height <= 1 {
		width = 80
		height = 24
	}

	t := &Terminal{
		fd:       fd,
		width:    width,
		height:   height,
		buffered: true,
		out:      os.Stdout,
		metrics:  NewRenderMetrics(),
	}

	t.initBuffers(width, height)
	return t, nil
}

// NewTestTerminal creates a terminal for testing with fixed size and custom output
func NewTestTerminal(width, height int, out io.Writer) *Terminal {
	t := &Terminal{
		width:    width,
		height:   height,
		buffered: true,
		out:      out,
		fd:       -1, // Invalid FD
		metrics:  NewRenderMetrics(),
	}
	t.initBuffers(width, height)
	return t
}

func (t *Terminal) initBuffers(width, height int) {
	t.backBuffer = make([][]Cell, height)
	t.frontBuffer = make([][]Cell, height)

	for y := 0; y < height; y++ {
		t.backBuffer[y] = make([]Cell, width)
		t.frontBuffer[y] = make([]Cell, width)
		for x := 0; x < width; x++ {
			// Initialize with spaces
			t.backBuffer[y][x] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
			// Initialize front buffer with nulls to force initial redraw
			t.frontBuffer[y][x] = Cell{Char: 0, Style: NewStyle(), Width: 0, Continuation: false}
		}
	}
}

// BeginFrame starts a new frame rendering sequence.
// It locks the terminal and returns a RenderFrame interface for drawing.
// The caller MUST call EndFrame to release the lock and flush changes.
//
// Errors:
//   - Returns ErrClosed if terminal has been closed
func (t *Terminal) BeginFrame() (RenderFrame, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, ErrClosed
	}

	// The initial frame covers the entire terminal
	return &terminalRenderFrame{
		t:      t,
		bounds: image.Rect(0, 0, t.width, t.height),
	}, nil
}

func (tf *terminalRenderFrame) Size() (width, height int) {
	return tf.bounds.Dx(), tf.bounds.Dy()
}

func (tf *terminalRenderFrame) SetCell(x, y int, char rune, style Style) error {
	globalX := tf.bounds.Min.X + x
	globalY := tf.bounds.Min.Y + y

	// Clip to frame bounds
	if !image.Pt(globalX, globalY).In(tf.bounds) {
		return ErrOutOfBounds
	}

	return tf.t.setCellInternal(globalX, globalY, char, style)
}

func (tf *terminalRenderFrame) PrintStyled(x, y int, text string, style Style) error {
	return tf.t.printInternal(tf.bounds.Min.X+x, tf.bounds.Min.Y+y, text, style, tf.bounds, true)
}

func (tf *terminalRenderFrame) PrintTruncated(x, y int, text string, style Style) error {
	return tf.t.printInternal(tf.bounds.Min.X+x, tf.bounds.Min.Y+y, text, style, tf.bounds, false)
}

func (tf *terminalRenderFrame) FillStyled(x, y, width, height int, char rune, style Style) error {
	// Calculate the fill rectangle in global coordinates
	globalFillRect := image.Rect(tf.bounds.Min.X+x, tf.bounds.Min.Y+y, tf.bounds.Min.X+x+width, tf.bounds.Min.Y+y+height)

	// Intersect with the frame's bounds to get the actual drawing area
	clippedRect := tf.bounds.Intersect(globalFillRect)

	// If the intersection is empty, do nothing
	if clippedRect.Empty() {
		return nil
	}

	return tf.t.fillInternal(clippedRect.Min.X, clippedRect.Min.Y, clippedRect.Dx(), clippedRect.Dy(), char, style)
}

// Fill fills the entire frame with the given character and style
func (tf *terminalRenderFrame) Fill(char rune, style Style) error {
	width, height := tf.Size()
	return tf.FillStyled(0, 0, width, height, char, style)
}

// Size returns the terminal dimensions
func (t *Terminal) Size() (width, height int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.width, t.height
}

// RefreshSize updates the cached terminal size and resizes buffers.
// It automatically calls all registered resize callbacks when the terminal size changes.
func (t *Terminal) RefreshSize() error {
	t.mu.Lock()

	// Skip in tests
	if t.fd == -1 {
		t.mu.Unlock()
		return nil
	}

	width, height, err := term.GetSize(t.fd)
	if err != nil {
		t.mu.Unlock()
		return err
	}

	sizeChanged := width != t.width || height != t.height

	if sizeChanged {
		t.width = width
		t.height = height
		t.resizeBuffers(width, height)

		// Clamp virtual cursor to new bounds
		if t.virtualX >= t.width {
			t.virtualX = t.width - 1
			if t.virtualX < 0 {
				t.virtualX = 0
			}
		}
		if t.virtualY >= t.height {
			t.virtualY = t.height - 1
			if t.virtualY < 0 {
				t.virtualY = 0
			}
		}

		// Force full redraw after resize
		if t.width > 0 && t.height > 0 {
			t.dirtyRegion.MarkRect(0, 0, t.width, t.height)
		}
	}

	t.mu.Unlock()

	// Call resize callbacks outside the lock to prevent deadlocks
	if sizeChanged {
		t.callbackMu.RLock()
		callbacks := make([]func(width, height int), 0, len(t.resizeCallbacks))
		for _, cb := range t.resizeCallbacks {
			if cb != nil {
				callbacks = append(callbacks, cb)
			}
		}
		t.callbackMu.RUnlock()

		for _, callback := range callbacks {
			callback(width, height)
		}
	}

	return nil
}

func (t *Terminal) resizeBuffers(width, height int) {
	newBack := make([][]Cell, height)
	newFront := make([][]Cell, height)

	for y := 0; y < height; y++ {
		newBack[y] = make([]Cell, width)
		newFront[y] = make([]Cell, width)
		for x := 0; x < width; x++ {
			if y < len(t.backBuffer) && x < len(t.backBuffer[y]) {
				newBack[y][x] = t.backBuffer[y][x]
			} else {
				newBack[y][x] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
			}
			// Invalidate front buffer to force redraw
			newFront[y][x] = Cell{Char: 0, Style: NewStyle(), Width: 0, Continuation: false}
		}
	}
	t.backBuffer = newBack
	t.frontBuffer = newFront
}

// SetStyle sets the current style for subsequent Print calls
// Deprecated: Use PrintStyled instead. Implicit style state will be removed in v2.0.
func (t *Terminal) SetStyle(style Style) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentStyle = style
}

// Clear clears the entire screen (fills buffer with spaces)
func (t *Terminal) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		fmt.Fprint(t.out, "\033[2J")
		return
	}

	emptyCell := Cell{Char: ' ', Style: t.currentStyle, Width: 1, Continuation: false}
	for y := 0; y < t.height; y++ {
		for x := 0; x < t.width; x++ {
			t.backBuffer[y][x] = emptyCell
		}
	}
	// Mark entire screen as dirty
	t.dirtyRegion.MarkRect(0, 0, t.width, t.height)
}

// ClearLine clears the current line
func (t *Terminal) ClearLine() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		fmt.Fprint(t.out, "\033[2K")
		return
	}

	if t.virtualY >= 0 && t.virtualY < t.height {
		emptyCell := Cell{Char: ' ', Style: t.currentStyle, Width: 1, Continuation: false}
		for x := 0; x < t.width; x++ {
			t.backBuffer[t.virtualY][x] = emptyCell
		}
	}
}

// ClearToEndOfLine clears from cursor to end of line
func (t *Terminal) ClearToEndOfLine() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		fmt.Fprint(t.out, "\033[K")
		return
	}

	if t.virtualY >= 0 && t.virtualY < t.height {
		emptyCell := Cell{Char: ' ', Style: t.currentStyle, Width: 1, Continuation: false}

		start := t.virtualX
		if start < 0 {
			start = 0
		}

		for x := start; x < t.width; x++ {
			t.backBuffer[t.virtualY][x] = emptyCell
		}
	}
}

// MoveCursor moves the cursor to the specified position
func (t *Terminal) MoveCursor(x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.moveCursorInternal(x, y)
}

func (t *Terminal) moveCursorInternal(x, y int) {
	if !t.buffered {
		fmt.Fprintf(t.out, "\033[%d;%dH", y+1, x+1)
		return
	}
	t.virtualX = x
	t.virtualY = y
}

// MoveCursorUp moves the cursor up by n lines
func (t *Terminal) MoveCursorUp(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		if n > 0 {
			fmt.Fprintf(t.out, "\033[%dA", n)
		}
		return
	}
	t.virtualY -= n
	if t.virtualY < 0 {
		t.virtualY = 0
	}
}

// MoveCursorDown moves the cursor down by n lines
func (t *Terminal) MoveCursorDown(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		if n > 0 {
			fmt.Fprintf(t.out, "\033[%dB", n)
		}
		return
	}
	t.virtualY += n
	if t.virtualY >= t.height {
		t.virtualY = t.height - 1
	}
}

// MoveCursorRight moves the cursor right by n columns
func (t *Terminal) MoveCursorRight(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		if n > 0 {
			fmt.Fprintf(t.out, "\033[%dC", n)
		}
		return
	}
	t.virtualX += n
	if t.virtualX >= t.width {
		t.virtualX = t.width - 1
	}
}

// MoveCursorLeft moves the cursor left by n columns
func (t *Terminal) MoveCursorLeft(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		if n > 0 {
			fmt.Fprintf(t.out, "\033[%dD", n)
		}
		return
	}
	t.virtualX -= n
	if t.virtualX < 0 {
		t.virtualX = 0
	}
}

// SaveCursor saves the current cursor position
func (t *Terminal) SaveCursor() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		fmt.Fprint(t.out, "\033[s")
		return
	}
	t.savedCursor = Position{X: t.virtualX, Y: t.virtualY}
}

// RestoreCursor restores the saved cursor position
func (t *Terminal) RestoreCursor() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		fmt.Fprint(t.out, "\033[u")
		return
	}
	t.virtualX = t.savedCursor.X
	t.virtualY = t.savedCursor.Y
}

// CursorPosition returns the current virtual cursor position
func (t *Terminal) CursorPosition() (x, y int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.virtualX, t.virtualY
}

// HideCursor hides the cursor
func (t *Terminal) HideCursor() {
	fmt.Fprint(t.out, "\033[?25l")
}

// ShowCursor shows the cursor
func (t *Terminal) ShowCursor() {
	fmt.Fprint(t.out, "\033[?25h")
}

// EnableAlternateScreen switches to the alternate screen buffer
func (t *Terminal) EnableAlternateScreen() {
	if !t.altScreen {
		fmt.Fprint(t.out, "\033[?1049h")
		t.altScreen = true
	}
}

// EnableBracketedPaste enables bracketed paste mode.
// When enabled, pasted text is wrapped in escape sequences (\033[200~ ... \033[201~)
// allowing the application to distinguish pasted text from typed text.
// This prevents security issues where pasted newlines could execute commands.
//
// This should be called after EnableRawMode() and before reading input.
// Don't forget to call DisableBracketedPaste() to restore normal paste behavior.
func (t *Terminal) EnableBracketedPaste() {
	fmt.Fprint(t.out, "\033[?2004h")
}

// DisableBracketedPaste disables bracketed paste mode.
// This restores normal paste behavior where pasted text is treated as typed input.
func (t *Terminal) DisableBracketedPaste() {
	fmt.Fprint(t.out, "\033[?2004l")
}

// DetectKittyProtocol probes the terminal to detect Kitty keyboard protocol support.
// This should be called once at startup before enabling raw mode.
// Returns true if the terminal supports the protocol.
//
// The detection works by:
// 1. Sending a query for progressive enhancement support (\x1b[?u)
// 2. Sending a device attributes query (\x1b[c)
// 3. Checking if both responses are received within a timeout
//
// This is the same approach used by Gemini CLI.
func (t *Terminal) DetectKittyProtocol() bool {
	if t.fd == -1 {
		return false // Test mode
	}

	// Need raw mode for detection
	oldState, err := term.MakeRaw(t.fd)
	if err != nil {
		return false
	}
	defer term.Restore(t.fd, oldState)

	// Send query: progressive enhancement query + device attributes query
	fmt.Print("\x1b[?u\x1b[c")

	// Read response with timeout
	responseChan := make(chan string, 1)
	go func() {
		buf := make([]byte, 256)
		response := ""
		deadline := time.Now().Add(200 * time.Millisecond)
		for time.Now().Before(deadline) {
			os.Stdin.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
			n, err := os.Stdin.Read(buf)
			if err != nil {
				break
			}
			response += string(buf[:n])
			// Check if we have both responses
			if strings.Contains(response, "\x1b[?") && strings.Contains(response, "c") {
				break
			}
		}
		os.Stdin.SetReadDeadline(time.Time{}) // Clear deadline
		responseChan <- response
	}()

	select {
	case response := <-responseChan:
		// Check for progressive enhancement response (CSI ? <flags> u) AND device attributes (CSI ? ... c)
		hasProgressiveEnhancement := strings.Contains(response, "\x1b[?") && strings.Contains(response, "u")
		hasDeviceAttributes := strings.Contains(response, "\x1b[?") && strings.Contains(response, "c")
		t.kittySupported = hasProgressiveEnhancement && hasDeviceAttributes
	case <-time.After(250 * time.Millisecond):
		t.kittySupported = false
	}

	return t.kittySupported
}

// EnableEnhancedKeyboard enables enhanced keyboard mode (CSI u / kitty keyboard protocol).
// This allows detection of modifier keys with Enter, Tab, and other special keys.
// For example, Shift+Enter will be reported as a distinct key event (ESC[13;2u).
//
// If DetectKittyProtocol() was called and returned false, this does nothing.
// Otherwise it enables the protocol (useful when you know the terminal supports it).
//
// Supported terminals: kitty, WezTerm, foot, ghostty, iTerm2 (3.5+), and others.
// Unsupported terminals will silently ignore this escape sequence.
//
// Call DisableEnhancedKeyboard() before exiting to restore normal keyboard mode.
func (t *Terminal) EnableEnhancedKeyboard() {
	if t.kittyEnabled {
		return
	}
	// Enable kitty keyboard protocol with flags:
	// Bit 0 (1): Disambiguate escape codes - report modifier keys with special keys
	// Bit 1 (2): Report event types (press, repeat, release)
	// Bit 2 (4): Report alternate keys
	// Bit 3 (8): Report all keys as escape codes
	// Bit 4 (16): Report associated text
	// We use flag 1 for basic modifier detection (e.g., Shift+Enter)
	fmt.Print("\033[>1u")
	t.kittyEnabled = true
}

// DisableEnhancedKeyboard disables enhanced keyboard mode.
// This restores normal keyboard reporting.
func (t *Terminal) DisableEnhancedKeyboard() {
	if !t.kittyEnabled {
		return
	}
	fmt.Print("\033[<u")
	t.kittyEnabled = false
}

// IsKittyProtocolSupported returns true if Kitty keyboard protocol is supported.
// Only valid after DetectKittyProtocol() has been called.
func (t *Terminal) IsKittyProtocolSupported() bool {
	return t.kittySupported
}

// IsKittyProtocolEnabled returns true if Kitty keyboard protocol is currently enabled.
func (t *Terminal) IsKittyProtocolEnabled() bool {
	return t.kittyEnabled
}

// DisableAlternateScreen switches back to the main screen buffer
func (t *Terminal) DisableAlternateScreen() {
	if t.altScreen {
		fmt.Fprint(t.out, "\033[?1049l")
		t.altScreen = false
	}
}

// EnableRawMode enables raw terminal mode
func (t *Terminal) EnableRawMode() error {
	if t.rawMode {
		return nil
	}

	// Skip in tests
	if t.fd == -1 {
		t.rawMode = true
		return nil
	}

	oldState, err := term.MakeRaw(t.fd)
	if err != nil {
		return fmt.Errorf("failed to enable raw mode: %w", err)
	}

	t.oldState = oldState
	t.rawMode = true
	return nil
}

// DisableRawMode disables raw terminal mode
func (t *Terminal) DisableRawMode() error {
	if !t.rawMode {
		return nil
	}

	if t.fd == -1 {
		t.rawMode = false
		return nil
	}

	if t.oldState != nil {
		if err := term.Restore(t.fd, t.oldState); err != nil {
			return fmt.Errorf("failed to restore terminal: %w", err)
		}
		t.oldState = nil
	}

	t.rawMode = false
	return nil
}

// Print outputs text at the current cursor position using the current style
// Deprecated: Use PrintStyled instead. Implicit style state will be removed in v2.0.
func (t *Terminal) Print(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.printInternal(t.virtualX, t.virtualY, text, t.currentStyle, image.Rect(0, 0, t.width, t.height), true)
}

// PrintStyled outputs text at the current cursor position using the provided style
func (t *Terminal) PrintStyled(text string, style Style) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.printInternal(t.virtualX, t.virtualY, text, style, image.Rect(0, 0, t.width, t.height), true)
}

// printInternal is the internal implementation of printing
// It assumes the lock is already held
func (t *Terminal) printInternal(startX, startY int, text string, style Style, clipRect image.Rectangle, wrap bool) error {
	if !t.buffered {
		// Non-buffered mode doesn't support clipping via buffer, direct print
		// This will draw outside the clipRect if it's smaller than terminal size
		var output strings.Builder
		output.WriteString(fmt.Sprintf("\033[%d;%dH", startY+1, startX+1))
		if !style.IsEmpty() {
			output.WriteString(style.String())
		}
		output.WriteString(text)
		if !style.IsEmpty() {
			output.WriteString("\033[0m")
		}

		outputStr := output.String()
		fmt.Fprint(t.out, outputStr)

		// Record for non-buffered mode
		if t.recorder != nil {
			t.recorder.RecordOutput(outputStr)
		}
		return nil
	}

	// For buffered mode, record the logical output immediately
	// This captures timing of Print() calls, not EndFrame() calls
	if t.recorder != nil {
		var output strings.Builder
		output.WriteString(fmt.Sprintf("\033[%d;%dH", startY+1, startX+1))
		if !style.IsEmpty() {
			output.WriteString(style.String())
		}
		output.WriteString(text)
		if !style.IsEmpty() {
			output.WriteString("\033[0m")
		}
		t.recorder.RecordOutput(output.String())
	}

	currentX := startX
	currentY := startY

	for _, r := range text {
		if r == '\n' {
			currentX = clipRect.Min.X // New line starts at the beginning of the clipRect
			currentY++
			continue
		}

		// Get the display width of the rune
		charWidth := runewidth.RuneWidth(r)

		// Check if character would wrap or overflow current line in clipRect
		if currentX+charWidth > clipRect.Max.X {
			if wrap {
				// Auto-wrap: move to next line
				currentX = clipRect.Min.X
				currentY++
			} else {
				// Truncate: skip characters that would go past the edge
				continue
			}
		}

		// If currentY is outside clipRect, stop drawing
		if currentY >= clipRect.Max.Y {
			break
		}

		// Only draw if within the clipRect and terminal bounds
		if currentX >= clipRect.Min.X && currentX < clipRect.Max.X &&
			currentY >= clipRect.Min.Y && currentY < clipRect.Max.Y &&
			currentX >= 0 && currentX < t.width &&
			currentY >= 0 && currentY < t.height {

			// Set the main character cell
			t.backBuffer[currentY][currentX] = Cell{
				Char:         r,
				Style:        style,
				Width:        charWidth,
				Continuation: false,
			}
			t.dirtyRegion.Mark(currentX, currentY)

			// For wide characters (width 2), mark the next cell as continuation
			if charWidth == 2 && currentX+1 < clipRect.Max.X && currentX+1 < t.width {
				t.backBuffer[currentY][currentX+1] = Cell{
					Char:         0,
					Style:        style,
					Width:        0,
					Continuation: true,
				}
				t.dirtyRegion.Mark(currentX+1, currentY)
			}
		}

		currentX += charWidth
	}

	// Update virtual cursor position to end of printed text
	t.virtualX = currentX
	t.virtualY = currentY

	return nil
}

// Println outputs text with a newline
// Deprecated: Use PrintStyled instead.
func (t *Terminal) Println(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.printInternal(t.virtualX, t.virtualY, text, t.currentStyle, image.Rect(0, 0, t.width, t.height), true)

	t.virtualX = 0
	t.virtualY++
	if t.virtualY >= t.height {
		t.scrollUp()
		t.virtualY = t.height - 1
	}
}

func (t *Terminal) scrollUp() {
	// Create new buffer for the last line
	newLine := make([]Cell, t.width)
	for x := 0; x < t.width; x++ {
		newLine[x] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
	}

	// Shift lines up
	copy(t.backBuffer, t.backBuffer[1:])
	t.backBuffer[t.height-1] = newLine
}

// PrintAt prints text at a specific position
// Deprecated: Use RenderFrame.PrintStyled instead.
func (t *Terminal) PrintAt(x, y int, text string) {
	t.MoveCursor(x, y)
	t.Print(text)
}

// PrintAtStyled prints text at a specific position with a specific style
func (t *Terminal) PrintAtStyled(x, y int, text string, style Style) {
	t.MoveCursor(x, y)
	t.PrintStyled(text, style)
}

// SetCell directly sets a cell in the buffer
func (t *Terminal) SetCell(x, y int, char rune, style Style) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.setCellInternal(x, y, char, style)
}

func (t *Terminal) setCellInternal(x, y int, char rune, style Style) error {
	if !t.buffered {
		// Fallback for non-buffered
		fmt.Fprintf(t.out, "\033[%d;%dH%s%c\033[0m", y+1, x+1, style.String(), char)
		return nil
	}

	if x < 0 || x >= t.width || y < 0 || y >= t.height {
		return ErrOutOfBounds
	}

	// Get the display width of the character
	charWidth := runewidth.RuneWidth(char)

	// Check if character fits
	if x+charWidth > t.width {
		return ErrOutOfBounds
	}

	// Set the main character cell
	t.backBuffer[y][x] = Cell{
		Char:         char,
		Style:        style,
		Width:        charWidth,
		Continuation: false,
	}
	t.dirtyRegion.Mark(x, y)

	// For wide characters (width 2), mark the next cell as continuation
	if charWidth == 2 && x+1 < t.width {
		t.backBuffer[y][x+1] = Cell{
			Char:         0,
			Style:        style,
			Width:        0,
			Continuation: true,
		}
		t.dirtyRegion.Mark(x+1, y)
	}

	return nil
}

// Fill fills a rectangular area with a character
// Deprecated: Use FillStyled instead.
func (t *Terminal) Fill(x, y, width, height int, char rune) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fillInternal(x, y, width, height, char, t.currentStyle)
}

// FillStyled fills a rectangular area with a character and a specific style
func (t *Terminal) FillStyled(x, y, width, height int, char rune, style Style) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fillInternal(x, y, width, height, char, style)
}

func (t *Terminal) fillInternal(x, y, width, height int, char rune, style Style) error {
	if !t.buffered {
		line := strings.Repeat(string(char), width)
		if !style.IsEmpty() {
			fmt.Fprint(t.out, style.String())
		}
		for i := 0; i < height; i++ {
			fmt.Fprintf(t.out, "\033[%d;%dH%s", y+i+1, x+1, line)
		}
		if !style.IsEmpty() {
			fmt.Fprint(t.out, "\033[0m")
		}
		return nil
	}

	// Get character width once
	charWidth := runewidth.RuneWidth(char)

	for j := 0; j < height; j++ {
		for i := 0; i < width; i++ {
			// Validation
			if x+i >= 0 && x+i < t.width && y+j >= 0 && y+j < t.height {
				// Only set if not part of a wide character (or if it's a space)
				t.backBuffer[y+j][x+i] = Cell{
					Char:         char,
					Style:        style,
					Width:        charWidth,
					Continuation: false,
				}
			}
		}
	}
	// Mark the entire rectangle as dirty
	t.dirtyRegion.MarkRect(x, y, width, height)
	return nil
}

// BypassInput updates the terminal state to reflect input that was already echoed by the OS.
// This keeps the virtual buffers in sync with the physical screen.
func (t *Terminal) BypassInput(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		return
	}

	for _, r := range text {
		if r == '\n' {
			t.virtualX = 0
			t.virtualY++
			if t.virtualY >= t.height {
				t.scrollBothBuffers()
				t.virtualY = t.height - 1
			}
			continue
		}

		if t.virtualY >= t.height {
			t.scrollBothBuffers()
			t.virtualY = t.height - 1
		}

		if t.virtualX >= 0 && t.virtualX < t.width && t.virtualY >= 0 && t.virtualY < t.height {
			charWidth := runewidth.RuneWidth(r)
			cell := Cell{Char: r, Style: NewStyle(), Width: charWidth, Continuation: false}
			t.backBuffer[t.virtualY][t.virtualX] = cell
			t.frontBuffer[t.virtualY][t.virtualX] = cell

			// Handle wide characters
			if charWidth == 2 && t.virtualX+1 < t.width {
				contCell := Cell{Char: 0, Style: NewStyle(), Width: 0, Continuation: true}
				t.backBuffer[t.virtualY][t.virtualX+1] = contCell
				t.frontBuffer[t.virtualY][t.virtualX+1] = contCell
			}
		}
		t.virtualX++
		if t.virtualX >= t.width {
			t.virtualX = 0
			t.virtualY++
		}
	}
}

func (t *Terminal) scrollBothBuffers() {
	// Scroll back buffer
	newBackLine := make([]Cell, t.width)
	for x := 0; x < t.width; x++ {
		newBackLine[x] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
	}
	copy(t.backBuffer, t.backBuffer[1:])
	t.backBuffer[t.height-1] = newBackLine

	// Scroll front buffer
	newFrontLine := make([]Cell, t.width)
	for x := 0; x < t.width; x++ {
		newFrontLine[x] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
	}
	copy(t.frontBuffer, t.frontBuffer[1:])
	t.frontBuffer[t.height-1] = newFrontLine
}

// Reset resets all terminal attributes
// Deprecated: Use explicit styles.
func (t *Terminal) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.currentStyle = NewStyle()
	if !t.buffered {
		fmt.Fprint(t.out, "\033[0m")
	}
}

// Flush calculates the difference between buffers and updates the terminal
func (t *Terminal) Flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flushInternal()
}

func (t *Terminal) flushInternal() error {
	if !t.buffered {
		// os.Stdout.Sync() // Cannot sync generic writer
		return nil
	}

	// Early exit if nothing changed
	if t.dirtyRegion.Empty() {
		if t.metricsEnabled {
			t.metrics.RecordSkippedFrame()
		}
		return nil
	}

	// Start timing if metrics enabled
	var startTime time.Time
	if t.metricsEnabled {
		startTime = time.Now()
	}

	var output strings.Builder
	cellsUpdated := 0
	ansiCodes := 0

	// Track state to minimize ANSI codes
	currentY, currentX := -1, -1
	var currentStyle Style // Empty style
	currentURL := ""       // Track current hyperlink URL

	// Only scan the dirty region
	minY := max(0, t.dirtyRegion.MinY)
	maxY := min(t.height-1, t.dirtyRegion.MaxY)
	minX := max(0, t.dirtyRegion.MinX)
	maxX := min(t.width-1, t.dirtyRegion.MaxX)

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			cell := t.backBuffer[y][x]
			oldCell := t.frontBuffer[y][x]

			// Skip continuation cells - they're already handled by their wide character
			if cell.Continuation {
				// Still need to update front buffer
				if cell != oldCell {
					t.frontBuffer[y][x] = cell
				}
				continue
			}

			if cell != oldCell {
				// Move cursor if needed
				if y != currentY || x != currentX {
					// Optimization: If we are just 1 char ahead, no need to move
					if y == currentY && x == currentX {
						// Already there
					} else {
						output.WriteString(fmt.Sprintf("\033[%d;%dH", y+1, x+1))
						if t.metricsEnabled {
							ansiCodes++
						}
					}
					currentY = y
					currentX = x
				}

				// Handle hyperlink URL changes (OSC 8)
				if cell.Style.URL != currentURL {
					// End current hyperlink if one is active
					if currentURL != "" {
						output.WriteString("\033]8;;\033\\") // OSC 8 end
						if t.metricsEnabled {
							ansiCodes++
						}
					}
					// Start new hyperlink if URL is set
					if cell.Style.URL != "" {
						output.WriteString(fmt.Sprintf("\033]8;;%s\033\\", cell.Style.URL)) // OSC 8 start
						if t.metricsEnabled {
							ansiCodes++
						}
					}
					currentURL = cell.Style.URL
				}

				// Update style if needed
				if cell.Style != currentStyle {
					output.WriteString(cell.Style.String())
					currentStyle = cell.Style
					if t.metricsEnabled {
						ansiCodes++
					}
				}

				// Write char
				output.WriteRune(cell.Char)
				// Move cursor by the character's display width
				currentX += cell.Width

				// Update front buffer
				t.frontBuffer[y][x] = cell

				if t.metricsEnabled {
					cellsUpdated++
				}
			}
		}
	}

	// End any active hyperlink
	if currentURL != "" {
		output.WriteString("\033]8;;\033\\") // OSC 8 end
		if t.metricsEnabled {
			ansiCodes++
		}
	}

	// Restore cursor to virtual position
	if currentX != t.virtualX || currentY != t.virtualY {
		output.WriteString(fmt.Sprintf("\033[%d;%dH", t.virtualY+1, t.virtualX+1))
		if t.metricsEnabled {
			ansiCodes++
		}
	}

	// Reset style at end to be safe
	if !currentStyle.IsEmpty() {
		output.WriteString("\033[0m")
		if t.metricsEnabled {
			ansiCodes++
		}
	}

	outputStr := output.String()
	bytesWritten := len(outputStr)

	if _, err := fmt.Fprint(t.out, outputStr); err != nil {
		// Leave dirty region intact so caller can retry
		return err
	}

	// Note: Recording happens at Print() level, not here
	// This ensures we capture timing of logical operations, not frame flushes

	// Record metrics if enabled
	if t.metricsEnabled {
		duration := time.Since(startTime)
		dirtyArea := (t.dirtyRegion.MaxX - t.dirtyRegion.MinX + 1) *
			(t.dirtyRegion.MaxY - t.dirtyRegion.MinY + 1)
		t.metrics.RecordFrame(cellsUpdated, ansiCodes, bytesWritten, duration, dirtyArea)
	}

	// Clear the dirty region for next frame
	t.dirtyRegion.Clear()

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// WatchResize starts watching for terminal resize signals (SIGWINCH)
// and automatically updates the terminal dimensions when the window is resized.
// Call StopWatchResize() or Close() to stop watching.
func (t *Terminal) WatchResize() {
	t.mu.Lock()
	if t.resizing || t.fd == -1 {
		t.mu.Unlock()
		return
	}

	t.resizeChan = make(chan os.Signal, 1)
	t.stopResize = make(chan struct{})
	t.resizing = true
	t.mu.Unlock()

	signal.Notify(t.resizeChan, syscall.SIGWINCH)

	go func() {
		for {
			select {
			case <-t.resizeChan:
				t.RefreshSize()
			case <-t.stopResize:
				return
			}
		}
	}()
}

// StopWatchResize stops watching for resize signals
func (t *Terminal) StopWatchResize() {
	t.mu.Lock()
	if !t.resizing {
		t.mu.Unlock()
		return
	}

	t.resizing = false
	if t.resizeChan != nil {
		signal.Stop(t.resizeChan)
		close(t.stopResize)
		t.resizeChan = nil
	}
	t.mu.Unlock()
}

// OnResize registers a callback to be called when the terminal is resized.
// The callback receives the new width and height.
// Returns a function that can be called to unregister the callback.
//
// Example:
//
//	unregister := terminal.OnResize(func(width, height int) {
//	    // Update layout with new dimensions
//	})
//	defer unregister()
func (t *Terminal) OnResize(callback func(width, height int)) func() {
	t.callbackMu.Lock()
	defer t.callbackMu.Unlock()

	t.resizeCallbacks = append(t.resizeCallbacks, callback)
	index := len(t.resizeCallbacks) - 1

	// Return unregister function
	return func() {
		t.callbackMu.Lock()
		defer t.callbackMu.Unlock()

		// Set to nil instead of removing to avoid index shifts
		if index < len(t.resizeCallbacks) {
			t.resizeCallbacks[index] = nil
		}
	}
}

// ClearResizeCallbacks removes all registered resize callbacks
func (t *Terminal) ClearResizeCallbacks() {
	t.callbackMu.Lock()
	defer t.callbackMu.Unlock()
	t.resizeCallbacks = nil
}

// EnableMetrics turns on performance metrics collection.
// When enabled, the terminal will track rendering statistics including:
// - Cells updated per frame
// - ANSI escape codes emitted
// - Bytes written to terminal
// - Frame render times
// - Dirty region sizes
//
// Metrics have minimal overhead but if you need maximum performance,
// keep them disabled (default).
func (t *Terminal) EnableMetrics() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metricsEnabled = true
}

// DisableMetrics turns off performance metrics collection.
func (t *Terminal) DisableMetrics() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metricsEnabled = false
}

// GetMetrics returns a snapshot of current rendering metrics.
// This is thread-safe and returns a copy of the metrics.
func (t *Terminal) GetMetrics() MetricsSnapshot {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.metrics.Snapshot()
}

// ResetMetrics clears all accumulated metrics.
func (t *Terminal) ResetMetrics() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.metrics.Reset()
}

// Close cleans up terminal state and marks the terminal as closed.
// After Close() is called, the terminal should not be reused.
func (t *Terminal) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil // Already closed
	}

	// Stop resize watching
	if t.resizing && t.resizeChan != nil {
		t.mu.Unlock() // Unlock before calling StopWatchResize
		t.StopWatchResize()
		t.mu.Lock()
	}

	t.closed = true
	t.ShowCursor()
	t.DisableAlternateScreen()
	t.DisableRawMode()

	// Reset style (inline to avoid mutex deadlock)
	t.currentStyle = NewStyle()
	if !t.buffered {
		fmt.Fprint(t.out, "\033[0m")
	}

	// Don't flush on close, might re-print garbage
	fmt.Fprint(t.out, "\033[2J") // Clear screen on exit
	return nil
}
