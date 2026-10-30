// Package terminal provides low-level terminal control, input decoding, styling, and rendering.
//
// This package enables building rich terminal user interfaces with features like:
//   - Double-buffered rendering with dirty region tracking for flicker-free updates
//   - Frame-based rendering API (BeginFrame/EndFrame) for atomic updates
//   - Keyboard input decoding supporting multi-byte UTF-8, ANSI escape sequences, and modifiers
//   - Mouse event handling with click, drag, hover, and scroll support
//   - Rich text styling with RGB colors, bold, italic, underline, and more
//   - Hyperlinks using OSC 8 protocol for clickable terminal links
//   - Raw mode and alternate screen buffer management
//   - Terminal resize detection and callbacks
//   - Performance metrics for monitoring rendering efficiency
//
// # Basic Rendering
//
// The recommended way to render content is using the frame-based API:
//
//	term, err := terminal.NewTerminal()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer term.Close()
//
//	// Enable alternate screen and raw mode for full-screen apps
//	term.EnableAlternateScreen()
//	term.EnableRawMode()
//
//	// Render a frame
//	frame, _ := term.BeginFrame()
//	style := terminal.NewStyle().WithForeground(terminal.ColorBlue).WithBold()
//	frame.PrintStyled(0, 0, "Hello, World!", style)
//	term.EndFrame(frame)
//
// # Input Handling
//
// The KeyDecoder decodes terminal input into structured events:
//
//	decoder := terminal.NewKeyDecoder(os.Stdin)
//	for {
//	    event, err := decoder.ReadEvent()
//	    if err != nil {
//	        break
//	    }
//	    switch e := event.(type) {
//	    case terminal.KeyEvent:
//	        if e.Key == terminal.KeyCtrlC {
//	            return
//	        }
//	        fmt.Printf("Key: %c\n", e.Rune)
//	    case terminal.MouseEvent:
//	        fmt.Printf("Mouse: %d,%d\n", e.X, e.Y)
//	    }
//	}
//
// # Styling
//
// Text can be styled using the Style type which supports colors, attributes, and hyperlinks:
//
//	// Basic colors
//	style := terminal.NewStyle().
//	    WithForeground(terminal.ColorRed).
//	    WithBackground(terminal.ColorWhite)
//
//	// RGB colors
//	rgb := terminal.NewRGB(100, 150, 200)
//	style = style.WithFgRGB(rgb)
//
//	// Text attributes
//	style = style.WithBold().WithItalic().WithUnderline()
//
//	// Apply to text
//	styledText := style.Apply("Hello")
//
// # Thread Safety
//
// Most Terminal methods are thread-safe, including Size, Clear, MoveCursor,
// SetCell, and the BeginFrame/EndFrame rendering pattern. However, some methods
// that emit raw escape sequences (HideCursor, ShowCursor, EnableAlternateScreen,
// DisableAlternateScreen, EnableMouseTracking, DisableMouseTracking, etc.) do not
// acquire locks to avoid overhead, as they're typically called during setup/teardown.
// For rendering, use the BeginFrame/EndFrame pattern which locks the terminal
// for exclusive access during frame composition.
//
// # Performance
//
// The terminal uses double-buffering and dirty region tracking to minimize
// the amount of data written to the terminal. Only changed cells are updated
// on each frame. Enable metrics to monitor rendering performance:
//
//	term.EnableMetrics()
//	// ... render frames ...
//	snapshot := term.GetMetrics()
//	fmt.Println(snapshot.String())
package terminal

import (
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/runewidth"
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

// Cell represents a single character cell on the terminal.
//
// For simple ASCII or BMP characters, Char holds the entire cell content and
// Trailing is empty. For multi-rune grapheme clusters (VS16 emoji, keycaps,
// ZWJ sequences, regional indicator flag pairs, skin-toned emoji, combining
// marks), Char is the first rune of the cluster and Trailing holds the
// remaining bytes of the cluster so they can be emitted together on render.
// Width is the display width of the whole cluster.
type Cell struct {
	Char         rune
	Trailing     string // remaining runes of a multi-rune grapheme cluster; usually empty
	Style        Style
	Width        int  // Display width of the cluster (0 for continuation cells, 1-2 for actual chars)
	Continuation bool // True if this cell is a continuation of a wide character
}

// Equal reports whether two cells render identically. RGB style overrides
// are compared by value (see Style.Equal), so cells styled with separately
// constructed but identical RGB values compare equal.
func (c Cell) Equal(other Cell) bool {
	return c.Char == other.Char &&
		c.Trailing == other.Trailing &&
		c.Width == other.Width &&
		c.Continuation == other.Continuation &&
		c.Style.Equal(other.Style)
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

	// Cell returns what is currently drawn at the given position, in the
	// frame's own coordinates. Out-of-bounds reads return a blank cell rather
	// than an error, so a caller sweeping a rectangle needs no bounds check.
	//
	// This reads the buffer being built, so it sees what has been drawn to
	// this frame so far and nothing that comes after. A view uses it to paint
	// over what its children drew — a selection highlight — and to read the
	// text back out.
	//
	// Performance: O(1)
	Cell(x, y int) Cell

	// RestyleCell replaces the style of the cell at the given position and
	// leaves its character exactly as it is. A multi-rune grapheme cluster
	// keeps every rune, and a wide character is restyled together with its
	// continuation cell so the whole glyph changes at once.
	//
	// This is the counterpart to Cell: it lets a view paint over what its
	// children drew — a selection highlight — without reproducing their text.
	// SetCell cannot do it, because a rune is not a grapheme cluster: writing
	// one back would drop the combining marks and variation selectors that
	// make up the rest of the cluster.
	//
	// Returns ErrOutOfBounds for a position outside the frame.
	//
	// Performance: O(1)
	RestyleCell(x, y int, style Style) error

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
// Thread Safety: Most methods are thread-safe. Methods for buffer operations
// (Size, Clear, MoveCursor, SetCell, Fill, BeginFrame/EndFrame) use locking.
// Methods that emit raw escape sequences during setup/teardown (HideCursor,
// ShowCursor, EnableAlternateScreen, EnableMouseTracking, etc.) do not lock,
// as they're designed to be called before/after the main rendering loop.
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
	resizeChan           chan os.Signal
	stopResize           chan struct{}
	resizing             bool
	resizeCallbacks      []resizeCallback
	nextResizeCallbackID int
	callbackMu           sync.RWMutex

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
	kittyFraming   bool

	// Cursor visibility state
	cursorHidden bool

	// Mode tracking for cleanup
	mouseMode      MouseMode // Which mouse reporting mode is enabled, if any
	bracketedPaste bool      // Bracketed paste is enabled

	// Frame state: true between BeginFrame and EndFrame (guarded by mu,
	// which is held for the duration of the frame)
	inFrame bool
}

// EndFrame finishes the frame, flushes the buffer to the terminal, and unlocks.
//
// Errors:
//   - Returns ErrInvalidFrame if the frame doesn't match this terminal,
//     or if no frame is currently open
func (t *Terminal) EndFrame(f RenderFrame) error {
	// Ensure we are unlocking the same terminal we locked
	tf, ok := f.(*terminalRenderFrame)
	if !ok || tf.t != t {
		// Invalid frame. If this terminal has an open frame, its BeginFrame
		// left the mutex held; release it so the error is recoverable
		// instead of deadlocking every subsequent operation.
		if t.inFrame {
			t.inFrame = false
			t.mu.Unlock()
		}
		return ErrInvalidFrame
	}
	if !t.inFrame {
		// EndFrame called twice, or without BeginFrame
		return ErrInvalidFrame
	}

	t.inFrame = false
	defer t.mu.Unlock()
	return t.flushInternal()
}

// Position represents a cursor position
type Position struct {
	X int
	Y int
}

// NewTerminal creates a new Terminal instance connected to the current terminal.
//
// The terminal is created with:
//   - Double-buffering enabled for flicker-free rendering
//   - Size detected from the current terminal dimensions
//   - Output directed to os.Stdout
//   - Default style (no colors or attributes)
//
// Call EnableRawMode() and EnableAlternateScreen() for full-screen applications.
// Always call Close() when done to restore terminal state.
//
// Example:
//
//	term, err := terminal.NewTerminal()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer term.Close()
//
//	term.EnableRawMode()
//	term.EnableAlternateScreen()
//	// ... use terminal ...
//
// Returns an error if the terminal size cannot be detected (e.g., when not
// running in a terminal). In such cases, defaults to 80x24.
//
// Note: NewTerminal uses os.Stdout's file descriptor for terminal size detection
// and raw mode. This works correctly when stdout is a TTY. If stdout is piped
// while stdin is a TTY, raw mode may fail. For such cases, consider checking
// term.IsTerminal(os.Stdin.Fd()) before enabling raw mode, as done in the tui
// package's Runtime.Run().
func NewTerminal() (*Terminal, error) {
	fd := int(os.Stdout.Fd())
	width, height, err := term.GetSize(fd)
	if err != nil {
		// Ignore error and use defaults
		width = 0
		height = 0
	}

	// Fallback for invalid terminal sizes (e.g. in CI/CD or piped output)
	if width <= 0 || height <= 1 {
		width = 80
		height = 24
	}

	t := &Terminal{
		fd:           fd,
		width:        width,
		height:       height,
		buffered:     true,
		out:          os.Stdout,
		currentStyle: NewStyle(),
		metrics:      NewRenderMetrics(),
	}

	t.initBuffers(width, height)
	return t, nil
}

// NewTestTerminal creates a Terminal for testing with fixed dimensions and custom output.
//
// Unlike NewTerminal(), this does not connect to a real terminal and does not
// query terminal size. It's useful for:
//   - Unit testing rendering code
//   - Capturing terminal output for inspection
//   - Testing without requiring a TTY
//
// Example:
//
//	var output strings.Builder
//	term := terminal.NewTestTerminal(80, 24, &output)
//
//	frame, _ := term.BeginFrame()
//	frame.PrintStyled(0, 0, "Test", terminal.NewStyle())
//	term.EndFrame(frame)
//
//	// Inspect output
//	result := output.String()
//
// Note: Some terminal operations that require a file descriptor (like raw mode
// or resize detection) will be no-ops in test mode.
func NewTestTerminal(width, height int, out io.Writer) *Terminal {
	t := &Terminal{
		width:        width,
		height:       height,
		buffered:     true,
		out:          out,
		fd:           -1, // Invalid FD
		currentStyle: NewStyle(),
		metrics:      NewRenderMetrics(),
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
	t.inFrame = true

	// The initial frame covers the entire terminal
	return &terminalRenderFrame{
		t:      t,
		bounds: image.Rect(0, 0, t.width, t.height),
	}, nil
}

func (tf *terminalRenderFrame) Size() (width, height int) {
	return tf.bounds.Dx(), tf.bounds.Dy()
}

func (tf *terminalRenderFrame) Cell(x, y int) Cell {
	globalX := tf.bounds.Min.X + x
	globalY := tf.bounds.Min.Y + y
	if !image.Pt(globalX, globalY).In(tf.bounds) {
		return Cell{Char: ' '}
	}
	// The frame holds t.mu for its whole life, so this must not re-lock.
	return tf.t.getCellInternal(globalX, globalY)
}

func (tf *terminalRenderFrame) RestyleCell(x, y int, style Style) error {
	globalX := tf.bounds.Min.X + x
	globalY := tf.bounds.Min.Y + y
	if !image.Pt(globalX, globalY).In(tf.bounds) {
		return ErrOutOfBounds
	}
	// The frame holds t.mu for its whole life, so this must not re-lock.
	return tf.t.restyleCellInternal(globalX, globalY, style)
}

func (tf *terminalRenderFrame) SetCell(x, y int, char rune, style Style) error {
	globalX := tf.bounds.Min.X + x
	globalY := tf.bounds.Min.Y + y

	// Clip to frame bounds. A wide character must fit entirely: its
	// continuation cell may not land outside the frame.
	if !image.Pt(globalX, globalY).In(tf.bounds) ||
		globalX+runewidth.RuneWidth(cellRune(char)) > tf.bounds.Max.X {
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
	if width <= 0 || height <= 0 {
		// Some PTYs report 0x0 before they are sized. Buffers with no rows
		// or columns cannot be drawn into, so keep the current size.
		t.mu.Unlock()
		return nil
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
			callbacks = append(callbacks, cb.fn)
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
		// A wide glyph whose continuation cell was cut off by a narrower
		// width cannot fit in the last column; replace it with a space.
		if width > 0 && newBack[y][width-1].Width > 1 {
			newBack[y][width-1] = Cell{Char: ' ', Style: newBack[y][width-1].Style, Width: 1, Continuation: false}
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
		t.writeScreenLocked("\033[2J")
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
		t.writeScreenLocked("\033[2K")
		return
	}

	if t.virtualY >= 0 && t.virtualY < t.height {
		emptyCell := Cell{Char: ' ', Style: t.currentStyle, Width: 1, Continuation: false}
		for x := 0; x < t.width; x++ {
			t.backBuffer[t.virtualY][x] = emptyCell
		}
		t.dirtyRegion.MarkRect(0, t.virtualY, t.width, 1)
	}
}

// ClearToEndOfLine clears from cursor to end of line
func (t *Terminal) ClearToEndOfLine() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		t.writeScreenLocked("\033[K")
		return
	}

	if t.virtualY >= 0 && t.virtualY < t.height {
		emptyCell := Cell{Char: ' ', Style: t.currentStyle, Width: 1, Continuation: false}

		start := t.virtualX
		if start < 0 {
			start = 0
		}

		// If the clear starts on the continuation cell of a wide character,
		// blank its lead cell too so no orphaned half-glyph remains.
		if start > 0 && start < t.width && t.backBuffer[t.virtualY][start].Continuation {
			t.backBuffer[t.virtualY][start-1] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
			t.dirtyRegion.Mark(start-1, t.virtualY)
		}

		for x := start; x < t.width; x++ {
			t.backBuffer[t.virtualY][x] = emptyCell
		}
		t.dirtyRegion.MarkRect(start, t.virtualY, t.width-start, 1)
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
		t.writeScreenLocked(fmt.Sprintf("\033[%d;%dH", y+1, x+1))
	}
	t.virtualX = x
	t.virtualY = y
}

// MoveCursorUp moves the cursor up by n lines
func (t *Terminal) MoveCursorUp(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		if n <= 0 {
			return
		}
		t.writeScreenLocked(fmt.Sprintf("\033[%dA", n))
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
		if n <= 0 {
			return
		}
		t.writeScreenLocked(fmt.Sprintf("\033[%dB", n))
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
		if n <= 0 {
			return
		}
		t.writeScreenLocked(fmt.Sprintf("\033[%dC", n))
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
		if n <= 0 {
			return
		}
		t.writeScreenLocked(fmt.Sprintf("\033[%dD", n))
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
		t.writeScreenLocked("\033[s")
	}
	t.savedCursor = Position{X: t.virtualX, Y: t.virtualY}
}

// RestoreCursor restores the saved cursor position
func (t *Terminal) RestoreCursor() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		t.writeScreenLocked("\033[u")
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
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cursorHidden = true
	t.writeScreenLocked("\033[?25l")
}

// ShowCursor shows the cursor
func (t *Terminal) ShowCursor() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.showCursorLocked()
}

func (t *Terminal) showCursorLocked() {
	t.cursorHidden = false
	t.writeScreenLocked("\033[?25h")
}

// EnableAlternateScreen switches to the alternate screen buffer
func (t *Terminal) EnableAlternateScreen() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.altScreen {
		t.writeScreenLocked("\033[?1049h")
		t.altScreen = true
	}
}

// writeScreenLocked writes s, text or a sequence that changes what the screen
// shows, straight to the output and records it, so a replay changes the screen
// the same way. Sequences that change input modes (mouse, bracketed paste,
// keyboard) are not recorded: replayed into a viewer's terminal, they would
// change how that terminal reports input. Caller must hold t.mu.
func (t *Terminal) writeScreenLocked(s string) {
	fmt.Fprint(t.out, s)
	if t.recorder != nil {
		t.recorder.RecordOutput(s)
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
	if t.bracketedPaste {
		return
	}
	fmt.Fprint(t.out, "\033[?2004h")
	t.bracketedPaste = true
}

// DisableBracketedPaste disables bracketed paste mode.
// This restores normal paste behavior where pasted text is treated as typed input.
func (t *Terminal) DisableBracketedPaste() {
	if !t.bracketedPaste {
		return
	}
	fmt.Fprint(t.out, "\033[?2004l")
	t.bracketedPaste = false
}

// shouldSkipTerminalQuery reports whether interactive OSC/CSI probe queries
// should be skipped based on the environment. Queries are unsafe under
// multiplexers that don't forward replies reliably (tmux, screen) and
// meaningless on dumb terminals or under Apple Terminal, which does not
// implement the kitty keyboard protocol.
func shouldSkipTerminalQuery() bool {
	switch os.Getenv("TERM_PROGRAM") {
	case "Apple_Terminal":
		return true
	}
	termVar := os.Getenv("TERM")
	if termVar == "" || termVar == "dumb" {
		return true
	}
	if strings.HasPrefix(termVar, "screen") || strings.HasPrefix(termVar, "tmux") {
		return true
	}
	return false
}

// DetectKittyProtocol probes the terminal to detect Kitty keyboard protocol support.
// This should be called once at startup before enabling raw mode.
// Returns true if the terminal supports the protocol.
//
// The detection works by:
//  1. Temporarily enabling raw mode
//  2. Sending a query for progressive enhancement support (\x1b[?u)
//  3. Sending a device attributes query (\x1b[c)
//  4. Reading the replies up to the end of the device attributes reply, for
//     up to 200ms, or 400ms if the terminal has started to answer
//
// Detection is skipped when TERM indicates a multiplexer (tmux/screen) or
// dumb terminal, or when TERM_PROGRAM identifies a terminal known not to
// support the protocol (e.g. Apple Terminal). In those cases the reply
// would be unreliable or absent and the probe bytes could leak to the
// user's shell.
//
// Caveats:
//   - Detection is supported on Unix terminals; on other platforms it returns
//     false immediately.
//   - Keys typed while the probe waits for the reply are dropped. Keys typed
//     after the reply arrives, or after the probe gives up, are not.
//   - When the probe gives up, it discards pending input. A reply that
//     arrives later is left for the input reader; KeyDecoder reads a whole
//     reply as KeyUnknown events, not as typed keys.
//
// This is the same approach used by Gemini CLI and other terminal applications.
func (t *Terminal) DetectKittyProtocol() bool {
	if t.fd == -1 || shouldSkipTerminalQuery() {
		t.kittySupported = false
		return false
	}

	t.kittySupported = probeKittyProtocol(os.Stdin, t.out, 200*time.Millisecond)
	return t.kittySupported
}

// EnableEnhancedKeyboard enables enhanced keyboard mode (CSI u / kitty keyboard protocol).
// This allows detection of modifier keys with Enter, Tab, and other special keys.
// For example, Shift+Enter will be reported as a distinct key event (ESC[13;2u).
//
// It enables the protocol whether or not DetectKittyProtocol() was called, so
// an application that would rather not pay for the probe — or whose terminal
// the probe refuses to ask, such as anything under tmux — can just call this.
// Terminals without the protocol ignore the sequence.
//
// Supported terminals: kitty, WezTerm, foot, ghostty, iTerm2 (3.5+), and others.
// Unsupported terminals will silently ignore this escape sequence.
//
// tmux ignores the Kitty protocol, so inside tmux this also asks for xterm's
// modifyOtherKeys mode 1, which tmux forwards when its extended-keys option is
// on; tmux then sends Shift+Enter as ESC[27;2;13~ (or ESC[13;2u with
// extended-keys-format csi-u), which KeyDecoder reads as Shift+Enter. With the
// option off, which is tmux's default, Shift+Enter still arrives as Enter.
//
// Call DisableEnhancedKeyboard() before exiting to restore normal keyboard mode.
func (t *Terminal) EnableEnhancedKeyboard() {
	if t.kittyEnabled {
		return
	}
	_ = t.enableKeyboard()
}

// DisableEnhancedKeyboard disables enhanced keyboard mode.
// This restores normal keyboard reporting.
func (t *Terminal) DisableEnhancedKeyboard() { _ = t.releaseKeyboard() }

// IsKittyProtocolSupported returns true if Kitty keyboard protocol is supported.
// Only valid after DetectKittyProtocol() has been called.
func (t *Terminal) IsKittyProtocolSupported() bool {
	return t.kittySupported
}

// IsKittyProtocolEnabled returns true if Kitty keyboard protocol is currently enabled.
func (t *Terminal) IsKittyProtocolEnabled() bool {
	return t.kittyEnabled
}

// MouseMode is how much a terminal has been asked to report about the pointer.
// Each level is a superset of the one before it.
type MouseMode int

const (
	MouseModeOff      MouseMode = iota // No reporting
	MouseModeButtons                   // ?1000: press, release, wheel
	MouseModeDrag                      // + ?1002: motion while a button is held
	MouseModeTracking                  // + ?1003: motion with no button held
)

// EnableMouseTracking enables full mouse event reporting in the terminal.
// This includes button press/release AND all mouse motion events (hover).
// Use EnableMouseButtons() if you only need click events without motion tracking.
func (t *Terminal) EnableMouseTracking() {
	if t.mouseMode != MouseModeOff {
		return
	}
	// Enable SGR extended mouse mode (supports coordinates beyond 223)
	fmt.Fprint(t.out, "\033[?1006h")
	// Enable mouse tracking - report button press and release
	fmt.Fprint(t.out, "\033[?1000h")
	// Enable all mouse motion tracking (including when no button is pressed)
	// This is needed for proper hover state detection
	fmt.Fprint(t.out, "\033[?1003h")
	t.mouseMode = MouseModeTracking
}

// EnableMouseButtons enables mouse button tracking without motion events.
// This reports button press/release events only, not mouse movement.
// Use this for better performance when hover detection is not needed.
func (t *Terminal) EnableMouseButtons() {
	if t.mouseMode != MouseModeOff {
		return
	}
	// Enable SGR extended mouse mode (supports coordinates beyond 223)
	fmt.Fprint(t.out, "\033[?1006h")
	// Enable mouse tracking - report button press and release only
	fmt.Fprint(t.out, "\033[?1000h")
	t.mouseMode = MouseModeButtons
}

// EnableMouseDrag enables mouse reporting for button press, release, wheel, and
// motion while a button is held — but not hover. This is what a view needs to
// implement drag-to-select, and it is far quieter than EnableMouseTracking:
// a terminal in ?1003 reports every pointer movement over the window.
func (t *Terminal) EnableMouseDrag() {
	if t.mouseMode != MouseModeOff {
		return
	}
	// Enable SGR extended mouse mode (supports coordinates beyond 223)
	fmt.Fprint(t.out, "\033[?1006h")
	// Enable mouse tracking - report button press and release
	fmt.Fprint(t.out, "\033[?1000h")
	// Enable button-event tracking - report motion while a button is held
	fmt.Fprint(t.out, "\033[?1002h")
	t.mouseMode = MouseModeDrag
}

// DisableMouseTracking disables mouse event reporting
func (t *Terminal) DisableMouseTracking() {
	if t.mouseMode == MouseModeOff {
		return
	}
	fmt.Fprint(t.out, "\033[?1000l")
	fmt.Fprint(t.out, "\033[?1002l")
	fmt.Fprint(t.out, "\033[?1003l")
	fmt.Fprint(t.out, "\033[?1006l")
	t.mouseMode = MouseModeOff
}

// DisableAlternateScreen switches back to the main screen buffer
func (t *Terminal) DisableAlternateScreen() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.disableAlternateScreenLocked()
}

func (t *Terminal) disableAlternateScreenLocked() {
	if t.altScreen {
		t.writeScreenLocked("\033[?1049l")
		t.altScreen = false
	}
}

// EnableRawMode enables raw terminal mode for character-by-character input.
//
// In raw mode:
//   - Input is not line-buffered (characters available immediately)
//   - No echo (typed characters don't appear automatically)
//   - No signal generation (Ctrl+C doesn't send SIGINT)
//   - Special characters (Ctrl+C, Ctrl+Z) are passed as input
//
// This is essential for interactive applications that need immediate key responses.
// Always call DisableRawMode() or Close() to restore normal terminal behavior.
//
// Example:
//
//	term, _ := terminal.NewTerminal()
//	defer term.Close() // Automatically disables raw mode
//
//	if err := term.EnableRawMode(); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Now you can read character-by-character input
//	decoder := terminal.NewKeyDecoder(os.Stdin)
//	event, _ := decoder.ReadEvent()
//
// Returns an error if raw mode cannot be enabled (e.g., not running in a terminal).
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
	// Text is drawn, never interpreted: drop control characters so it
	// cannot carry escape sequences to the terminal.
	text = stripTextControl(text)

	if !t.buffered {
		// Non-buffered mode doesn't support clipping via buffer, direct print
		// This will draw outside the clipRect if it's smaller than terminal size
		t.printUnbufferedLocked(startX, startY, text, style)
		return nil
	}

	// Buffered mode is recorded in flushInternal, from the bytes that
	// actually reach the terminal, so clipping, SetCell, and Fill are
	// captured too.

	currentX := startX
	currentY := startY

	// Iterate the input by grapheme cluster so multi-rune clusters — VS16
	// emoji, keycaps, ZWJ sequences, regional indicator flag pairs, skin-toned
	// emoji, and combining marks — land in a single Cell instead of having
	// their trailing runes stomped by whichever character comes next.
	skipLine := false // set when truncating: drop the remainder of the line

	for cluster, clusterWidth := range runewidth.Graphemes(text) {
		if cluster == "\n" || cluster == "\r\n" {
			skipLine = false
			currentX = clipRect.Min.X // New line starts at the beginning of the clipRect
			currentY++
			continue
		}
		if cluster == "\r" {
			skipLine = false
			currentX = clipRect.Min.X
			continue
		}
		if skipLine {
			continue
		}
		if cluster == "\t" {
			// A tab has display width 0 (see runewidth), which is how callers
			// such as tui measure text, so it takes no column here either.
			// Storing it in a cell would send a raw tab on flush and move the
			// real cursor to the next tab stop, out of step with the buffer.
			continue
		}

		charWidth := clusterWidth

		// Check if the cluster would wrap or overflow the current line.
		if currentX+charWidth > clipRect.Max.X {
			if wrap {
				// Auto-wrap: move to next line
				currentX = clipRect.Min.X
				currentY++
				if currentX+charWidth > clipRect.Max.X {
					// Wider than the whole clip rect (a wide glyph in a
					// 1-column frame): it can never fit, so drop it rather
					// than draw past the right edge.
					continue
				}
			} else {
				// Truncate: skip everything up to the next newline so a
				// narrower later cluster can't be drawn at the skipped
				// cluster's position (which would reorder the text).
				skipLine = true
				continue
			}
		}

		// If currentY is outside clipRect, stop drawing
		if currentY >= clipRect.Max.Y {
			break
		}

		// Split the cluster into its leading rune and any trailing runes.
		// Single-rune clusters (the common case) keep Trailing empty.
		first, firstSize := utf8.DecodeRuneInString(cluster)
		trailing := cluster[firstSize:]

		// Only draw if within the clipRect and terminal bounds
		if currentX >= clipRect.Min.X && currentX < clipRect.Max.X &&
			currentY >= clipRect.Min.Y && currentY < clipRect.Max.Y &&
			currentX >= 0 && currentX < t.width &&
			currentY >= 0 && currentY < t.height {

			// Clear any wide-char artifacts before writing
			oldCell := t.backBuffer[currentY][currentX]
			if oldCell.Width == 2 && currentX+1 < t.width {
				// Old cell was a wide char - clear its continuation
				t.backBuffer[currentY][currentX+1] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
				t.dirtyRegion.Mark(currentX+1, currentY)
			} else if oldCell.Continuation && currentX > 0 {
				// We're overwriting a continuation cell - clear the lead wide char
				t.backBuffer[currentY][currentX-1] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
				t.dirtyRegion.Mark(currentX-1, currentY)
			}

			// Set the main character cell
			t.backBuffer[currentY][currentX] = Cell{
				Char:         first,
				Trailing:     trailing,
				Style:        style,
				Width:        charWidth,
				Continuation: false,
			}
			t.dirtyRegion.Mark(currentX, currentY)

			// For wide characters (width 2), mark the next cell as continuation
			if charWidth == 2 && currentX+1 < clipRect.Max.X && currentX+1 < t.width {
				// Clear any wide-char lead at the continuation position
				nextCell := t.backBuffer[currentY][currentX+1]
				if nextCell.Width == 2 && currentX+2 < t.width {
					t.backBuffer[currentY][currentX+2] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
					t.dirtyRegion.Mark(currentX+2, currentY)
				}
				t.backBuffer[currentY][currentX+1] = Cell{
					Char:         0,
					Style:        style,
					Width:        0,
					Continuation: true,
				}
				t.dirtyRegion.Mark(currentX+1, currentY)
			} else if charWidth > 2 {
				t.blankTail(currentX, currentY, charWidth, clipRect.Max.X, style)
			}
		}

		currentX += charWidth
	}

	// Update virtual cursor position to end of printed text
	t.virtualX = currentX
	t.virtualY = currentY

	return nil
}

// printUnbufferedLocked writes text directly to the terminal from (startX,
// startY) and advances the virtual cursor past it. Caller must hold t.mu.
func (t *Terminal) printUnbufferedLocked(startX, startY int, text string, style Style) {
	if text == "" {
		// Nothing to draw, so the cursor stays where it is, as in
		// buffered mode.
		return
	}

	// Skip the clusters left of column 0, as buffered mode does. A newline
	// starts the next line at column 0, so it ends the skipped run.
	x := startX
	for cluster, clusterWidth := range runewidth.Graphemes(text) {
		if x >= 0 || cluster == "\n" {
			break
		}
		x += clusterWidth
		text = text[len(cluster):]
	}
	if text == "" {
		t.virtualX, t.virtualY = x, startY
		return
	}
	x = max(x, 0)

	// Start where the terminal really draws: it clamps a cursor move to
	// the screen. Visible text that starts past the right edge, such as
	// after a print that filled its line, goes on the next line as in
	// buffered mode. On the last line, a newline scrolls up to make one.
	y := min(max(startY, 0), t.height-1)
	scroll := false
	if x >= t.width && startsVisible(text) {
		x = 0
		if y == t.height-1 {
			scroll = true
		} else {
			y++
		}
	}

	// Raw mode usually turns off OPOST, so a bare newline moves down without
	// returning to column 0. Each newline is sent as CRLF. The style is reset
	// around it, so a line scrolled in on the last line is not filled with
	// the style's background, as buffered mode's scrollUp fills it blank.
	newline := "\r\n"
	if !style.IsEmpty() {
		newline = "\033[0m\r\n" + style.String()
	}

	var output strings.Builder
	output.WriteString(fmt.Sprintf("\033[%d;%dH", y+1, x+1))
	if scroll {
		output.WriteString("\n")
	}
	if !style.IsEmpty() {
		output.WriteString(style.String())
	}
	output.WriteString(strings.ReplaceAll(text, "\n", newline))
	if !style.IsEmpty() {
		output.WriteString("\033[0m")
	}

	t.writeScreenLocked(output.String())
	t.virtualX, t.virtualY = t.advanceUnbuffered(x, y, text)
}

// startsVisible reports whether text has a cluster that takes a column
// before its first newline. A newline starts the next line by itself.
func startsVisible(text string) bool {
	for cluster, clusterWidth := range runewidth.Graphemes(text) {
		if cluster == "\n" {
			return false
		}
		if clusterWidth > 0 {
			return true
		}
	}
	return false
}

// advanceUnbuffered returns the cursor position after text is written
// directly to the terminal from (x, y). It follows the buffered layout: each
// grapheme cluster takes its display width, a cluster that does not fit
// wraps to the next line, a newline starts the next line at column 0, and a
// tab takes no column. Text past the last line scrolls the screen, so y stays
// on the last line. As in buffered mode, x is t.width after text that fills
// its line, and the next print starts on the line below.
func (t *Terminal) advanceUnbuffered(x, y int, text string) (int, int) {
	nextLine := func() {
		x = 0
		if y < t.height-1 {
			y++
		}
	}
	for cluster, clusterWidth := range runewidth.Graphemes(text) {
		if cluster == "\n" {
			nextLine()
			continue
		}
		if x > 0 && x+clusterWidth > t.width {
			nextLine()
		}
		x += clusterWidth
	}
	return x, y
}

// blankTail writes blank cells over the columns after a glyph wider than two
// cells, up to maxX. runewidth gives U+2E3A and U+2E3B widths 3 and 4, but
// terminals draw them in one cell. Continuation cells would hide those
// columns from flush, so stale text would stay on screen; blanks draw over it.
func (t *Terminal) blankTail(x, y, width, maxX int, style Style) {
	for i := 1; i < width && x+i < maxX && x+i < t.width; i++ {
		if old := t.backBuffer[y][x+i]; old.Width == 2 && x+i+1 < t.width {
			// Don't orphan the continuation of a wide glyph cut in half.
			t.backBuffer[y][x+i+1] = Cell{Char: ' ', Style: NewStyle(), Width: 1}
			t.dirtyRegion.Mark(x+i+1, y)
		}
		t.backBuffer[y][x+i] = Cell{Char: ' ', Style: style, Width: 1}
		t.dirtyRegion.Mark(x+i, y)
	}
}

// Println outputs text with a newline
// Deprecated: Use PrintStyled instead.
func (t *Terminal) Println(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.printInternal(t.virtualX, t.virtualY, text, t.currentStyle, image.Rect(0, 0, t.width, t.height), true)

	if !t.buffered {
		// Write the newline from column 0 of the cursor's line, so on the
		// last line it scrolls the screen as buffered mode does.
		y := min(max(t.virtualY, 0), t.height-1)
		t.writeScreenLocked(fmt.Sprintf("\033[%d;1H\n", y+1))
	}

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

	// Mark entire screen as dirty
	t.dirtyRegion.MarkRect(0, 0, t.width, t.height)
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
	char = cellRune(char)
	if x < 0 || x >= t.width || y < 0 || y >= t.height {
		return ErrOutOfBounds
	}

	// Get the display width of the character
	charWidth := runewidth.RuneWidth(char)

	// Check if character fits
	if x+charWidth > t.width {
		return ErrOutOfBounds
	}

	if !t.buffered {
		// Fallback for non-buffered
		out := fmt.Sprintf("\033[%d;%dH%s%c\033[0m", y+1, x+1, style.String(), char)
		t.writeScreenLocked(out)
		return nil
	}

	// Clear any wide-char artifacts before writing:
	// 1. If we're overwriting a wide char, clear its continuation cell
	// 2. If we're overwriting a continuation cell, clear the lead wide char
	oldCell := t.backBuffer[y][x]
	if oldCell.Width == 2 && x+1 < t.width {
		// Old cell was a wide char - clear its continuation
		t.backBuffer[y][x+1] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
		t.dirtyRegion.Mark(x+1, y)
	} else if oldCell.Continuation && x > 0 {
		// We're overwriting a continuation cell - clear the lead wide char
		t.backBuffer[y][x-1] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
		t.dirtyRegion.Mark(x-1, y)
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
		// Also clear any wide-char lead at x+1 before making it a continuation
		nextCell := t.backBuffer[y][x+1]
		if nextCell.Width == 2 && x+2 < t.width {
			t.backBuffer[y][x+2] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
			t.dirtyRegion.Mark(x+2, y)
		}
		t.backBuffer[y][x+1] = Cell{
			Char:         0,
			Style:        style,
			Width:        0,
			Continuation: true,
		}
		t.dirtyRegion.Mark(x+1, y)
	} else if charWidth > 2 {
		t.blankTail(x, y, charWidth, t.width, style)
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
	char = cellRune(char)

	// Clip the rectangle to the terminal bounds rather than rejecting the
	// whole fill: the public Fill/FillStyled methods have no way to surface
	// an error, and a silent no-op for a rect one cell past the edge is
	// surprising. This also matches RenderFrame.FillStyled, which clips.
	if x < 0 {
		width += x
		x = 0
	}
	if y < 0 {
		height += y
		y = 0
	}
	width = min(width, t.width-x)
	height = min(height, t.height-y)
	if width <= 0 || height <= 0 {
		return nil
	}

	// Get character width once
	charWidth := runewidth.RuneWidth(char)
	step := charWidth
	if step < 1 {
		step = 1
	}

	if !t.buffered {
		var lineBuilder strings.Builder
		for i := 0; i < width; i += step {
			if charWidth > 1 && i+charWidth > width {
				for ; i < width; i++ {
					lineBuilder.WriteByte(' ')
				}
				break
			}
			lineBuilder.WriteRune(char)
			if charWidth > 2 {
				for k := 2; k < charWidth && i+k < width; k++ {
					lineBuilder.WriteByte(' ')
				}
			}
		}
		line := lineBuilder.String()

		var out strings.Builder
		if !style.IsEmpty() {
			out.WriteString(style.String())
		}
		for i := 0; i < height; i++ {
			fmt.Fprintf(&out, "\033[%d;%dH%s", y+i+1, x+1, line)
		}
		if !style.IsEmpty() {
			out.WriteString("\033[0m")
		}
		t.writeScreenLocked(out.String())
		return nil
	}

	for j := 0; j < height; j++ {
		row := t.backBuffer[y+j]
		// A wide glyph straddling either edge of the fill loses one half;
		// blank the other half so no orphaned lead or continuation remains.
		if row[x].Continuation && x > 0 {
			row[x-1] = Cell{Char: ' ', Style: NewStyle(), Width: 1}
			t.dirtyRegion.Mark(x-1, y+j)
		}
		if right := x + width; right < t.width && row[right].Continuation {
			row[right] = Cell{Char: ' ', Style: NewStyle(), Width: 1}
			t.dirtyRegion.Mark(right, y+j)
		}

		for i := 0; i < width; i += step {
			// Validation (should be covered by top-level check but good for safety)
			if x+i >= t.width {
				break
			}

			if charWidth > 1 && i+charWidth > width {
				// No room for the whole glyph: fill the remainder with spaces.
				for ; i < width; i++ {
					row[x+i] = Cell{Char: ' ', Style: style, Width: 1}
				}
				break
			}

			// Set the main character cell
			t.backBuffer[y+j][x+i] = Cell{
				Char:         char,
				Style:        style,
				Width:        charWidth,
				Continuation: false,
			}

			// Handle wide characters
			if charWidth == 2 && i+1 < width && x+i+1 < t.width {
				t.backBuffer[y+j][x+i+1] = Cell{
					Char:         0,
					Style:        style,
					Width:        0,
					Continuation: true,
				}
			} else if charWidth > 2 {
				t.blankTail(x+i, y+j, charWidth, x+width, style)
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

	clearArtifacts := func(buffer [][]Cell, x, y int) {
		oldCell := buffer[y][x]
		if oldCell.Width == 2 && x+1 < t.width {
			buffer[y][x+1] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
		} else if oldCell.Continuation && x > 0 {
			buffer[y][x-1] = Cell{Char: ' ', Style: NewStyle(), Width: 1, Continuation: false}
		}
	}

	for cluster, clusterWidth := range runewidth.Graphemes(text) {
		if cluster == "\r\n" || cluster == "\n" {
			t.virtualX = 0
			t.virtualY++
			if t.virtualY >= t.height {
				t.scrollBothBuffers()
				t.virtualY = t.height - 1
			}
			continue
		}
		if cluster == "\r" {
			t.virtualX = 0
			continue
		}

		if t.virtualY >= t.height {
			t.scrollBothBuffers()
			t.virtualY = t.height - 1
		}

		if clusterWidth > 0 && t.virtualX+clusterWidth > t.width {
			t.virtualX = 0
			t.virtualY++
			if t.virtualY >= t.height {
				t.scrollBothBuffers()
				t.virtualY = t.height - 1
			}
		}

		if t.virtualX >= 0 && t.virtualX < t.width && t.virtualY >= 0 && t.virtualY < t.height {
			first, firstSize := utf8.DecodeRuneInString(cluster)
			trailing := cluster[firstSize:]
			cell := Cell{
				Char:         first,
				Trailing:     trailing,
				Style:        NewStyle(),
				Width:        clusterWidth,
				Continuation: false,
			}

			clearArtifacts(t.backBuffer, t.virtualX, t.virtualY)
			clearArtifacts(t.frontBuffer, t.virtualX, t.virtualY)
			t.backBuffer[t.virtualY][t.virtualX] = cell
			t.frontBuffer[t.virtualY][t.virtualX] = cell

			// Handle wide characters
			if clusterWidth == 2 && t.virtualX+1 < t.width {
				contCell := Cell{Char: 0, Style: NewStyle(), Width: 0, Continuation: true}
				t.backBuffer[t.virtualY][t.virtualX+1] = contCell
				t.frontBuffer[t.virtualY][t.virtualX+1] = contCell
			}
		}
		t.virtualX += clusterWidth
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
		t.writeScreenLocked("\033[0m")
	}
}

// renderRowsLocked renders the given rows of the back buffer in full, each
// from column 0, and then moves the cursor to the virtual position. It is
// used to record whole rows. Caller must hold t.mu.
func (t *Terminal) renderRowsLocked(rows []int) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	var style Style
	for _, y := range rows {
		fmt.Fprintf(&b, "\033[%d;1H", y+1)
		for x := 0; x < t.width; x++ {
			cell := t.backBuffer[y][x]
			if cell.Continuation {
				continue
			}
			if !cell.Style.Equal(style) {
				if !style.IsEmpty() {
					b.WriteString("\033[0m")
				}
				b.WriteString(cell.Style.String())
				style = cell.Style
			}
			b.WriteRune(cell.Char)
			b.WriteString(cell.Trailing)
			if cell.Width > 1 && x+cell.Width < t.width {
				// Position the next cell explicitly, as flush does.
				fmt.Fprintf(&b, "\033[%d;%dH", y+1, x+cell.Width+1)
			}
		}
	}
	if !style.IsEmpty() {
		b.WriteString("\033[0m")
	}
	fmt.Fprintf(&b, "\033[%d;%dH", t.virtualY+1, t.virtualX+1)
	return b.String()
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

	wasHidden := t.cursorHidden

	var output strings.Builder

	// Synchronized output: the terminal holds the frame back until the end
	// marker, so a repaint that touches every visible cell — a page scroll, a
	// resize — lands at once instead of tearing. Terminals that do not
	// implement DEC 2026 ignore both markers.
	output.WriteString("\033[?2026h")

	// Hide cursor during rendering to prevent flicker.
	// The cursor moves around while writing cells, which can cause visible
	// flashing even if the app has called HideCursor(). We always hide it
	// during flush and restore visibility based on the app's desired state.
	if !wasHidden {
		output.WriteString("\033[?25l") // Hide cursor
	}
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

	var changedRows []int // rows with at least one changed cell, for recording

	for y := minY; y <= maxY; y++ {
		rowChanged := false
		// rewriteRest is set once a multi-rune cluster is written on this
		// row. The terminal may draw such a cluster wider than its cell
		// width, spilling over the cells after it, so the front buffer no
		// longer describes them; write them again even when unchanged, up
		// to the end of the row, past the dirty region.
		rewriteRest := false
		for x := minX; x <= maxX || (rewriteRest && x < t.width); x++ {
			cell := t.backBuffer[y][x]
			oldCell := t.frontBuffer[y][x]

			// Skip continuation cells - they're already handled by their wide
			// character (the front buffer is committed wholesale after the
			// write succeeds).
			if cell.Continuation {
				continue
			}

			if rewriteRest || !cell.Equal(oldCell) {
				if cell.Trailing != "" {
					rewriteRest = true
				}
				rowChanged = true
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
						output.WriteString(Start(cell.Style.URL)) // OSC 8 start
						if t.metricsEnabled {
							ansiCodes++
						}
					}
					currentURL = cell.Style.URL
				}

				// Update style if needed
				if !cell.Style.Equal(currentStyle) {
					output.WriteString(cell.Style.String())
					currentStyle = cell.Style
					if t.metricsEnabled {
						ansiCodes++
					}
				}

				// A multi-rune cluster may draw narrower than its cell width
				// (terminals size Indic clusters and some emoji sequences
				// differently), and its continuation cells are never written.
				// Erase the cells it spans first (ECH leaves the cursor in
				// place) so whatever was there in the last frame, such as an
				// old border, cannot show through.
				if cell.Width > 1 && cell.Trailing != "" {
					fmt.Fprintf(&output, "\033[%dX", cell.Width)
					if t.metricsEnabled {
						ansiCodes++
					}
				}

				// Write char, plus any trailing runes that belong to the same
				// grapheme cluster (VS16, combining marks, ZWJ extensions, the
				// second regional indicator of a flag, etc).
				output.WriteRune(cell.Char)
				if cell.Trailing != "" {
					output.WriteString(cell.Trailing)
				}

				// Update cursor position tracking.
				//
				// WIDE CHARACTER HANDLING:
				// For wide characters (width > 1, typically emojis and CJK characters),
				// we cannot assume the terminal correctly advanced its cursor by the
				// character's display width. Different terminals handle this inconsistently:
				//
				// - Some terminals (like iTerm2) correctly advance the cursor by 2 cells
				//   after printing a 2-cell-wide emoji.
				// - Other terminals (like VSCode's xterm.js) may advance by only 1 cell,
				//   or have other inconsistent behavior between cursor tracking and
				//   visual rendering.
				//
				// By setting currentX to -1 after wide characters, we force an explicit
				// cursor positioning command (CSI row;col H) before the next character
				// write. This ensures correct rendering regardless of the terminal's
				// cursor advancement behavior, at the cost of slightly more output bytes.
				//
				// Multi-rune grapheme clusters (cell.Trailing != "") get the same
				// treatment even at width 1: terminals disagree on how far keycaps,
				// VS16 emoji, and Indic clusters advance the cursor, and trusting
				// our width would shift every later cell on the row, leaving stale
				// glyphs (such as an old border) behind.
				if cell.Width > 1 || cell.Trailing != "" {
					currentX = -1 // Force cursor reposition on next character
				} else {
					currentX += cell.Width
				}

				if t.metricsEnabled {
					cellsUpdated++
				}
			}
		}
		if rowChanged {
			changedRows = append(changedRows, y)
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

	// Restore cursor visibility if it was visible before flush
	if !wasHidden {
		output.WriteString("\033[?25h") // Show cursor
	}

	output.WriteString("\033[?2026l") // End sync

	outputStr := output.String()
	bytesWritten := len(outputStr)

	if _, err := fmt.Fprint(t.out, outputStr); err != nil {
		// Leave the dirty region and front buffer intact so a retry
		// re-diffs and re-emits the cells that never reached the terminal.
		return err
	}

	// Commit the flushed region to the front buffer only now that the write
	// has succeeded.
	for y := minY; y <= maxY; y++ {
		copy(t.frontBuffer[y][minX:maxX+1], t.backBuffer[y][minX:maxX+1])
	}

	// Record what the terminal received. Recording at Print() time would
	// miss SetCell and Fill and ignore clipping.
	if t.recorder != nil {
		if t.recorder.redactSecrets {
			// Redaction runs on each event alone. A diff can hold a changed
			// value without the label drawn in an earlier frame ("Token: "),
			// so the pattern would not match. Record each changed row in
			// full instead, so the label and value are redacted together.
			t.recorder.RecordOutput(t.renderRowsLocked(changedRows))
		} else {
			t.recorder.RecordOutput(outputStr)
		}
	}

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
	if t.closed || t.resizing || t.fd == -1 {
		t.mu.Unlock()
		return
	}

	t.resizeChan = make(chan os.Signal, 1)
	t.stopResize = make(chan struct{})
	t.resizing = true

	// Set up platform-specific resize signal handling while holding t.mu
	// so StopWatchResize cannot nil out or stop t.resizeChan before signal.Notify runs.
	t.setupResizeSignal()

	// Capture the channels so the goroutine never reads the struct fields,
	// which StopWatchResize mutates under the lock (and a Stop/Watch restart
	// replaces). This avoids a data race and ensures the old goroutine can't
	// latch onto a newer watcher's channels.
	resizeCh := t.resizeChan
	stopCh := t.stopResize
	t.mu.Unlock()

	go func() {
		for {
			select {
			case <-resizeCh:
				t.RefreshSize()
			case <-stopCh:
				return
			}
		}
	}()
}

// StopWatchResize stops watching for resize signals
func (t *Terminal) StopWatchResize() {
	t.mu.Lock()
	t.stopWatchResizeLocked()
	t.mu.Unlock()
}

func (t *Terminal) stopWatchResizeLocked() {
	if !t.resizing {
		return
	}

	t.resizing = false
	if t.resizeChan != nil {
		signal.Stop(t.resizeChan)
		close(t.stopResize)
		t.resizeChan = nil
	}
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

	id := t.nextResizeCallbackID
	t.nextResizeCallbackID++
	t.resizeCallbacks = append(t.resizeCallbacks, resizeCallback{id: id, fn: callback})

	// Return unregister function. The unique ID ensures a stale unregister
	// function (e.g. kept across ClearResizeCallbacks) can never remove an
	// unrelated callback that happens to occupy the same slot.
	return func() {
		t.callbackMu.Lock()
		defer t.callbackMu.Unlock()

		for i, cb := range t.resizeCallbacks {
			if cb.id == id {
				t.resizeCallbacks = append(t.resizeCallbacks[:i], t.resizeCallbacks[i+1:]...)
				return
			}
		}
	}
}

// resizeCallback pairs a registered resize callback with a unique ID so
// unregister functions remove exactly the callback they registered.
type resizeCallback struct {
	id int
	fn func(width, height int)
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
//
// Close restores terminal modes in the correct order:
//  1. Stops any active recording
//  2. Disables special input modes (mouse, keyboard, paste)
//  3. Shows cursor and disables alternate screen
//  4. Restores normal terminal mode
func (t *Terminal) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil // Already closed
	}

	// Mark closed before temporarily releasing the lock below, so a
	// concurrent Close or BeginFrame can't slip in and run a second
	// teardown (or start a frame) mid-close.
	t.closed = true

	// Stop resize watching
	t.stopWatchResizeLocked()

	// Stop any active recording
	if t.recorder != nil {
		t.recorder.close()
		t.recorder = nil
	}

	// Disable special input modes (order matters for some terminals)
	t.DisableMouseTracking()
	t.DisableBracketedPaste()
	t.DisableEnhancedKeyboard()

	// Restore cursor and screen
	t.showCursorLocked()
	t.disableAlternateScreenLocked()
	t.DisableRawMode()

	// Reset style
	t.currentStyle = NewStyle()
	fmt.Fprint(t.out, "\033[0m")

	return nil
}

// PrintHyperlink is a convenience method for RenderFrame to print a hyperlink.
// It always uses OSC 8 format (terminals that don't support it will ignore the codes).
func (tf *terminalRenderFrame) PrintHyperlink(x, y int, link Hyperlink) error {
	// Validate the hyperlink
	if err := link.Validate(); err != nil {
		// Fall back to printing just the text if invalid, without any URL
		// the style carries, which was not validated
		return tf.PrintStyled(x, y, link.Text, link.Style.WithURL(""))
	}

	// Add the URL to the style and print the text
	// The flush logic will emit OSC 8 codes when the URL changes
	styleWithURL := link.Style.WithURL(link.URL)
	return tf.PrintStyled(x, y, link.Text, styleWithURL)
}

// PrintHyperlinkFallback prints a hyperlink using the fallback format (text + URL).
// Use this when you know the terminal doesn't support OSC 8, or when you want
// to explicitly show the URL.
func (tf *terminalRenderFrame) PrintHyperlinkFallback(x, y int, link Hyperlink) error {
	// Validate the hyperlink
	if err := link.Validate(); err != nil {
		// Fall back to printing just the text if invalid, without any URL
		// the style carries, which was not validated
		return tf.PrintStyled(x, y, link.Text, link.Style.WithURL(""))
	}

	// Print the styled text followed by the URL in parentheses. Clear any URL
	// the style carries: the fallback format shows the URL as text, and the
	// style's URL was not validated.
	text := link.Text + " (" + link.URL + ")"
	return tf.PrintStyled(x, y, text, link.Style.WithURL(""))
}

// Write writes raw bytes directly to the terminal output, making *Terminal an
// io.Writer. It is WriteRaw with the signature the standard library expects,
// so a terminal can be passed to anything that writes bytes — an escape
// sequence builder, a logger, clipboard.WriteOSC52.
//
// Like WriteRaw it bypasses the double buffer, so a caller sending an escape
// sequence should do it from the goroutine that owns the terminal, or it can
// land in the middle of a frame.
func (t *Terminal) Write(p []byte) (int, error) {
	if err := t.WriteRaw(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// WriteRaw writes raw bytes directly to the terminal output.
// This is thread-safe and bypasses the double-buffering system.
// Use this for playback or other cases where you need direct terminal output.
func (t *Terminal) WriteRaw(data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.out == nil {
		return ErrClosed
	}

	_, err := t.out.Write(data)
	if err != nil {
		return err
	}

	// Sync if output is a file
	if f, ok := t.out.(*os.File); ok {
		f.Sync()
	}

	return nil
}

// GetCell returns the cell at the given position from the back buffer.
// Returns an empty cell if the position is out of bounds.
// This is primarily useful for testing.
func (t *Terminal) GetCell(x, y int) Cell {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.getCellInternal(x, y)
}

// Cell returns the cell at the given position, satisfying RenderFrame. The
// terminal is itself a frame covering the whole screen, so this is GetCell.
func (t *Terminal) Cell(x, y int) Cell { return t.GetCell(x, y) }

// RestyleCell changes the style of one cell and leaves its character alone,
// satisfying RenderFrame.
func (t *Terminal) RestyleCell(x, y int, style Style) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.restyleCellInternal(x, y, style)
}

// restyleCellInternal restyles a cell in the back buffer without locking.
// Callers must already hold t.mu — a render frame does, for the whole frame.
func (t *Terminal) restyleCellInternal(x, y int, style Style) error {
	if !t.buffered {
		// Nothing is retained to restyle, and the character is not known here
		// well enough to write it back.
		return nil
	}
	if x < 0 || x >= t.width || y < 0 || y >= t.height {
		return ErrOutOfBounds
	}
	// The right half of a wide glyph is never drawn on its own (flush skips
	// continuation cells), so restyle the glyph from its lead cell.
	if t.backBuffer[y][x].Continuation && x > 0 {
		x--
	}
	t.backBuffer[y][x].Style = style
	t.dirtyRegion.Mark(x, y)
	// A wide character's continuation cell carries the style too. Without it
	// the right half of the glyph keeps the old background and the highlight
	// is drawn half-width.
	if t.backBuffer[y][x].Width == 2 && x+1 < t.width {
		t.backBuffer[y][x+1].Style = style
		t.dirtyRegion.Mark(x+1, y)
	}
	return nil
}

// getCellInternal reads the back buffer without locking. Callers must already
// hold t.mu — a render frame does, for the duration of the frame.
func (t *Terminal) getCellInternal(x, y int) Cell {
	if y < 0 || y >= len(t.backBuffer) {
		return Cell{Char: ' '}
	}
	if x < 0 || x >= len(t.backBuffer[y]) {
		return Cell{Char: ' '}
	}
	return t.backBuffer[y][x]
}

// IsAlternateScreen reports whether the alternate screen buffer is active.
func (t *Terminal) IsAlternateScreen() bool { return t.altScreen }

// IsRawMode reports whether raw mode is enabled.
func (t *Terminal) IsRawMode() bool { return t.rawMode }

// IsCursorHidden reports whether the cursor is hidden.
func (t *Terminal) IsCursorHidden() bool { return t.cursorHidden }

// IsMouseEnabled reports whether any mouse reporting mode is enabled.
func (t *Terminal) IsMouseEnabled() bool { return t.mouseMode != MouseModeOff }

// MouseMode reports which mouse reporting mode is enabled. Restoring reporting
// after releasing it needs the mode, not just the fact that it was on: coming
// back in the wrong one either floods the app with motion events it never asked
// for, or silently drops the hover reports it depends on.
func (t *Terminal) MouseMode() MouseMode { return t.mouseMode }

// Invalidate discards the terminal's knowledge of what is currently on screen,
// so the next flush redraws every cell rather than diffing against a picture
// that is no longer true.
//
// Call it after anything outside the terminal's control has written to the
// screen: a subprocess, a suspend-and-resume, a manual escape sequence. The
// back buffer — what the application wants on screen — is untouched.
func (t *Terminal) Invalidate() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.buffered {
		return
	}
	for y := range t.frontBuffer {
		for x := range t.frontBuffer[y] {
			t.frontBuffer[y][x] = Cell{Char: 0, Style: NewStyle(), Width: 0, Continuation: false}
		}
	}
	t.dirtyRegion.MarkRect(0, 0, t.width, t.height)
}
