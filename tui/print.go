package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/runewidth"
	"github.com/deepnoodle-ai/wonton/termtest"
	"golang.org/x/term"
)

// printConfig configures printing functions like Print, Fprint, and Sprint.
// All fields are optional with sensible zero-value defaults.
type printConfig struct {
	Width   int       // 0 = auto (terminal width or 80). Positive values set the width.
	Height  int       // 0 = auto (based on view size). Positive values set a fixed height.
	Output  io.Writer // nil = os.Stdout. Specify where to write output.
	RawMode bool      // false = use \n line endings. true = use \r\n for raw terminal mode.
	Color   *bool     // nil = color.Enabled. false = plain text with no escape codes.
}

// PrintOption is a functional option for the printing functions:
// Print, Fprint, Sprint, SprintScreen, NewLivePrinter, and Live.
type PrintOption func(*printConfig)

// WithWidth sets the rendering width in columns.
// Default is the width of the output when it is a terminal, or 80 otherwise.
func WithWidth(width int) PrintOption {
	return func(c *printConfig) {
		c.Width = width
	}
}

// WithHeight sets a fixed rendering height in lines.
// Default is the view's natural height.
func WithHeight(height int) PrintOption {
	return func(c *printConfig) {
		c.Height = height
	}
}

// WithOutput sets where output is written. Default is os.Stdout.
func WithOutput(w io.Writer) PrintOption {
	return func(c *printConfig) {
		c.Output = w
	}
}

// WithRawMode controls line endings: when enabled, lines end with \r\n as
// required in raw terminal mode, where \n alone only moves down without
// returning to column 0. Default is \n.
func WithRawMode(enabled bool) PrintOption {
	return func(c *printConfig) {
		c.RawMode = enabled
	}
}

// WithColor controls whether Print, Fprint, and Sprint write styles as ANSI
// escape codes. When disabled, the output is plain text. Default is
// color.Enabled, which is off when stdout is not a terminal or when NO_COLOR
// or CLICOLOR=0 is set. SprintScreen defaults to on, since it exists to
// inspect styles in tests. LivePrinter ignores this option.
func WithColor(enabled bool) PrintOption {
	return func(c *printConfig) {
		c.Color = &enabled
	}
}

func newPrintConfig(opts []PrintOption) printConfig {
	var cfg printConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

func (c printConfig) withDefaults() printConfig {
	if c.Output == nil {
		c.Output = os.Stdout
	}
	if c.Width == 0 {
		c.Width = 80
		// Measure the destination, not stdout: a file or buffer gets the
		// default width even when stdout is a terminal.
		if f, ok := c.Output.(*os.File); ok {
			if fd := int(f.Fd()); term.IsTerminal(fd) {
				if w, _, err := term.GetSize(fd); err == nil && w > 0 {
					c.Width = w
				}
			}
		}
	}
	if c.Color == nil {
		enabled := color.Enabled
		c.Color = &enabled
	}
	return c
}

// Print renders a view to the terminal without taking over the screen.
// This outputs the view inline, preserving scroll history and existing content.
//
// Unlike Run(), Print does not:
//   - Enable alternate screen mode
//   - Enable raw mode or handle keyboard input
//   - Clear the screen
//   - Start an event loop
//
// This is useful for CLI tools that want to display styled output once
// and then exit, without the full TUI experience.
//
// Example:
//
//	view := tui.Stack(
//	    tui.Text("Hello").Bold(),
//	    tui.Text("World").Foreground(tui.ColorRed),
//	)
//	tui.Print(view)
//
// With configuration:
//
//	tui.Print(view, tui.WithWidth(60))
//
// Styles are written as ANSI escape codes only when color is enabled; see
// WithColor. Print leaves the cursor at the end of the last line. Use Println
// to end the output with a newline.
func Print(view View, opts ...PrintOption) error {
	return printWith(view, newPrintConfig(opts))
}

func printWith(view View, cfg printConfig) error {
	cfg = cfg.withDefaults()

	// Measure the view to get its natural height
	_, viewHeight := view.size(cfg.Width, 0)
	if viewHeight == 0 {
		viewHeight = 1 // At least one line
	}

	// Use configured height or view's natural height
	height := cfg.Height
	if height == 0 {
		height = viewHeight
	}

	// Create an in-memory terminal buffer
	var buf strings.Builder
	terminal := NewTestTerminal(cfg.Width, height, &buf)

	// Render the view to the buffer
	frame, err := terminal.BeginFrame()
	if err != nil {
		return fmt.Errorf("failed to begin frame: %w", err)
	}

	// Clear the frame
	frame.Fill(' ', NewStyle())

	// Create render context and render the view
	ctx := NewRenderContext(frame, 0)
	view.size(cfg.Width, height) // Measure phase
	view.render(ctx)             // Render phase

	// End the frame (this populates the buffer)
	terminal.EndFrame(frame)

	// Convert buffer to ANSI output
	output := renderToANSI(terminal, cfg.Width, height, cfg.RawMode, *cfg.Color)

	// Write to output
	_, err = io.WriteString(cfg.Output, output)
	return err
}

// renderToANSI converts the terminal buffer to a string with ANSI escape codes.
// If rawMode is true, uses \r\n line endings (required in raw terminal mode where
// \n alone only moves down without returning to column 0). If useColor is
// false, styles are dropped and the result is plain text.
func renderToANSI(t *Terminal, width, height int, rawMode, useColor bool) string {
	var output strings.Builder

	var currentStyle Style
	styleSet := false

	// Line ending depends on mode
	lineEnding := "\n"
	if rawMode {
		lineEnding = "\r\n"
	}

	for y := 0; y < height; y++ {
		lineHasContent := false
		lastContentX := -1

		// Find the last non-space character on this line. Styled spaces count
		// as content only when they're visible, i.e. when color is on.
		for x := width - 1; x >= 0; x-- {
			cell := t.GetCell(x, y)
			if (cell.Char != ' ' && cell.Char != 0) || (useColor && cell.Style != NewStyle()) {
				lastContentX = x
				break
			}
		}

		for x := 0; x <= lastContentX || !lineHasContent; x++ {
			if x >= width {
				break
			}

			cell := t.GetCell(x, y)

			// Skip continuation cells (part of wide characters)
			if cell.Continuation {
				continue
			}

			// Update style if needed
			if useColor && (!styleSet || cell.Style != currentStyle) {
				// Reset then apply new style
				if styleSet {
					output.WriteString("\033[0m")
				}
				if cell.Style != NewStyle() {
					output.WriteString(cell.Style.String())
				}
				currentStyle = cell.Style
				styleSet = true
			}

			writeCellGlyph(&output, cell)
			lineHasContent = true

			// Stop if we've passed the last content
			if x >= lastContentX && lastContentX >= 0 {
				break
			}
		}

		// Reset style at end of line and add newline
		if styleSet && currentStyle != NewStyle() {
			output.WriteString("\033[0m")
			styleSet = false
			currentStyle = NewStyle()
		}

		// Add newline between lines, but not after the last line
		// (a trailing newline would cause scrollUp in the test screen)
		if y < height-1 {
			output.WriteString(lineEnding)
		}
	}

	return output.String()
}

// Fprint renders a view to the specified writer. The width defaults to the
// writer's terminal width when w is a terminal, or 80 otherwise.
func Fprint(w io.Writer, view View, opts ...PrintOption) error {
	cfg := newPrintConfig(opts)
	cfg.Output = w
	return printWith(view, cfg)
}

// Println is like Print but ends the output with a newline, which is what
// CLI output usually wants.
func Println(view View, opts ...PrintOption) error {
	return printlnWith(view, newPrintConfig(opts))
}

// Fprintln is like Fprint but ends the output with a newline.
//
// Example:
//
//	tui.Fprintln(w, tui.Table(columns, nil).Rows(rows))
func Fprintln(w io.Writer, view View, opts ...PrintOption) error {
	cfg := newPrintConfig(opts)
	cfg.Output = w
	return printlnWith(view, cfg)
}

func printlnWith(view View, cfg printConfig) error {
	cfg = cfg.withDefaults()
	if err := printWith(view, cfg); err != nil {
		return err
	}
	lineEnding := "\n"
	if cfg.RawMode {
		lineEnding = "\r\n"
	}
	_, err := io.WriteString(cfg.Output, lineEnding)
	return err
}

// Sprint renders a view to a string. Styles are included as ANSI escape codes
// only when color is enabled; see WithColor.
func Sprint(view View, opts ...PrintOption) string {
	var buf strings.Builder
	cfg := newPrintConfig(opts)
	cfg.Output = &buf
	printWith(view, cfg)
	return buf.String()
}

// SprintScreen renders a view and returns a termtest.Screen for assertions.
// This is a convenience function for testing that combines Sprint with
// termtest.Screen parsing, making it easy to write precise visual tests.
// With WithRawMode(true), the screen models a raw-mode terminal (see
// termtest.WithRawMode), so a line that does not start at column 0 shows.
//
// Example:
//
//	func TestButton(t *testing.T) {
//	    btn := Button("Submit", func() {})
//	    screen := SprintScreen(btn, WithWidth(20))
//	    termtest.AssertRowContains(t, screen, 0, "Submit")
//	}
func SprintScreen(view View, opts ...PrintOption) *termtest.Screen {
	var buf strings.Builder
	cfg := newPrintConfig(opts)
	cfg.Output = &buf
	if cfg.Color == nil {
		enabled := true
		cfg.Color = &enabled
	}
	cfg = cfg.withDefaults()

	// Get view dimensions
	_, viewHeight := view.size(cfg.Width, 0)
	if viewHeight == 0 {
		viewHeight = 1
	}

	// Use configured height or view's natural height + 1 for trailing newline
	height := cfg.Height
	if height == 0 {
		height = viewHeight + 1
	}

	// Render to string
	printWith(view, cfg)
	output := buf.String()

	// Create screen and write output
	screen := termtest.NewScreen(cfg.Width, height, termtest.WithRawMode(cfg.RawMode))
	screen.Write([]byte(output))
	return screen
}

// LivePrinter renders views to a fixed region of the terminal that updates in place.
// This is useful for progress bars, status displays, and loading indicators that
// need to update without scrolling the terminal.
//
// # Rendering Optimization
//
// LivePrinter uses several techniques to minimize flicker and improve performance:
//
//   - Line-level diffing: Each frame is compared line-by-line against the previous
//     frame. Only lines that actually changed are redrawn. This is similar to how
//     the Terminal package uses cell-level diffing, but operates at a coarser
//     granularity for simpler inline updates.
//
//   - Synchronized output mode: Updates are wrapped in DEC private mode 2026
//     escape sequences (\033[?2026h ... \033[?2026l). This tells the terminal
//     to buffer all changes and render them atomically, eliminating partial-frame
//     flicker. Supported by most modern terminals (iTerm2, kitty, alacritty,
//     Windows Terminal, WezTerm, foot). Terminals that don't support it simply
//     ignore the escape sequences.
//
//   - Cursor hiding: The cursor is hidden during updates and restored on Stop().
//
// # Thread Safety
//
// LivePrinter is NOT thread-safe. All calls to Update, Clear, and Stop should
// be made from the same goroutine, or protected by external synchronization.
// For thread-safe live updates, use InlineApp which handles synchronization.
//
// # Example
//
//	live := tui.NewLivePrinter(tui.WithWidth(60))
//	defer live.Stop()
//
//	for i := 0; i <= 100; i++ {
//	    live.Update(tui.Text("Progress: %d%%", i))
//	    time.Sleep(50 * time.Millisecond)
//	}
type LivePrinter struct {
	config       printConfig
	maxHeight    int // Terminal height limit. 0 = unconstrained.
	lastHeight   int
	started      bool
	frameCount   uint64
	hiddenCursor bool
	lastLines    []string    // Previous frame's lines for diffing
	drawnWidth   int         // Width the previous frame was drawn at; 0 before the first frame
	rewriteAll   bool        // Next frame writes every row (set on a resize)
	unwrapped    bool        // Count the old frame as lastHeight rows, not as rewrapped (see keepRowsAbove)
	reg          *registries // Set by InlineApp so views register with its stores; nil = process default
}

// NewLivePrinter creates a new LivePrinter for updating a region in place.
func NewLivePrinter(opts ...PrintOption) *LivePrinter {
	return &LivePrinter{
		config: newPrintConfig(opts).withDefaults(),
	}
}

// SetWidth updates the width of the live printer.
// This is useful when the terminal is resized.
// The next Update() will use the new width.
func (lp *LivePrinter) SetWidth(w int) {
	lp.config.Width = w
}

// keepRowsAbove makes the next erase of the old frame (the next frame
// after a width change, or a Clear) treat it as lastHeight rows from the
// cursor up, instead of the rows the terminal would rewrap it into.
// InlineApp calls it when it inserted rows above the live region shortly
// before it heard of a width change: the terminal may have narrowed first,
// so those rows sit among the rewrapped rows of the old frame, and only the
// lastHeight rows below them are the old frame. Erasing fewer rows can
// leave a few rewrapped rows of the old frame above the new one; erasing
// more would erase printed rows.
func (lp *LivePrinter) keepRowsAbove() {
	lp.unwrapped = true
}

// rewriteNextFrame makes the next frame write every row, also rows whose
// text did not change. InlineApp calls it on every resize: a terminal may
// change rows around the cursor when it resizes (tmux does, for example,
// while its size catches up with a drag), and Print no longer redraws the
// live region, so only the next frame can repair it.
func (lp *LivePrinter) rewriteNextFrame() {
	lp.rewriteAll = true
}

// SetMaxHeight sets the maximum height (in lines) for the live region.
// When the view's natural height exceeds this limit, it is capped to prevent
// the live region from exceeding the terminal height, which would corrupt
// cursor positioning. 0 means unconstrained (the default).
func (lp *LivePrinter) SetMaxHeight(h int) {
	lp.maxHeight = h
}

// Update renders a new view, replacing the previous content in place.
// The cursor moves back to overwrite the previous output.
//
// Optimization: Uses line-level diffing to only update lines that changed,
// similar to how the Terminal package uses cell-level diffing. This reduces
// the amount of data written and minimizes flicker.
func (lp *LivePrinter) Update(view View) error {
	return lp.update(view, true, nil)
}

// UpdateWithFocus renders a view with focus management support.
// The focus manager is passed through the render context to focusable components.
func (lp *LivePrinter) UpdateWithFocus(view View, fm *FocusManager) error {
	return lp.update(view, true, fm)
}

// UpdateNoSync is like Update but without synchronized output mode wrapping.
// This is useful when the caller is already managing sync mode to avoid nested
// escape sequences.
//
// Use cases:
//   - InlineApp.Print() uses this to atomically clear, print, and re-render
//   - Batch operations that need to wrap multiple updates in a single sync block
//
// Most users should use Update() instead, which handles sync mode automatically.
func (lp *LivePrinter) UpdateNoSync(view View) error {
	return lp.update(view, false, nil)
}

// update is the internal implementation shared by Update, UpdateWithFocus, and UpdateNoSync.
func (lp *LivePrinter) update(view View, useSync bool, fm *FocusManager) error {
	checked := &checkedLiveOutput{destination: lp.config.Output}
	// Measure the view
	_, viewHeight := view.size(lp.config.Width, 0)
	if viewHeight == 0 {
		viewHeight = 1
	}

	renderHeight := lp.config.Height
	if renderHeight == 0 {
		renderHeight = viewHeight
	}

	lp.beginFrame(checked)

	// Create terminal buffer and render at the full (uncapped) height.
	// This lets the view lay out all its content naturally.
	var buf strings.Builder
	terminal := NewTestTerminal(lp.config.Width, renderHeight, &buf)

	frame, err := terminal.BeginFrame()
	if err != nil {
		return fmt.Errorf("failed to begin frame: %w", err)
	}

	frame.Fill(' ', NewStyle())
	ctx := NewRenderContext(frame, lp.frameCount)
	if fm != nil {
		ctx = ctx.WithFocusManager(fm)
	}
	if lp.reg != nil {
		ctx = ctx.withRegistries(lp.reg)
	}
	lp.frameCount++
	view.size(lp.config.Width, renderHeight)
	view.render(ctx)
	terminal.EndFrame(frame)

	// Convert to individual lines for diffing
	allLines := renderToLines(terminal, lp.config.Width, renderHeight)

	// Cap the displayed height to maxHeight to prevent the live region from
	// exceeding the terminal. When capped, show the BOTTOM of the content
	// so the most recent output (streaming text, latest tool call, input field)
	// remains visible.
	displayHeight := renderHeight
	var newLines []string
	if lp.maxHeight > 0 && displayHeight > lp.maxHeight {
		displayHeight = lp.maxHeight
		newLines = allLines[len(allLines)-displayHeight:]
	} else {
		newLines = allLines
	}
	return lp.writeFrame(checked, newLines, useSync)
}

// redraw writes lines, a frame this printer drew before, without rendering a
// view. InlineApp uses it to restore the live region after scrollback output
// without calling the application's LiveView off the event loop.
//
// drawnAt is the width the lines were drawn at. If it is not the current
// width (the width changed and no frame has been drawn since), the lines
// were rendered for the old width and may wrap on screen. The printer then
// keeps drawnAt as the frame's width and asks for every row again, so the
// next frame erases this one and writes every row at the new width.
func (lp *LivePrinter) redraw(lines []string, useSync bool, drawnAt int) error {
	if len(lines) == 0 {
		return nil
	}
	checked := &checkedLiveOutput{destination: lp.config.Output}
	lp.beginFrame(checked)
	err := lp.writeFrame(checked, lines, useSync)
	if drawnAt != 0 && drawnAt != lp.config.Width {
		lp.drawnWidth = drawnAt
		lp.rewriteAll = true
	}
	return err
}

// beginFrame moves the cursor to the top of the live region, and hides the
// cursor on the first frame.
func (lp *LivePrinter) beginFrame(checked io.Writer) {
	// If we've already rendered, move cursor back up
	if lp.started && lp.lastHeight > 0 {
		lp.moveToFrameTop(checked)
		if lp.drawnWidth != 0 && lp.drawnWidth != lp.config.Width {
			// After a resize the old frame may sit on different rows than
			// this frame overwrites, so erase all of it first.
			fmt.Fprint(checked, "\033[J")
			lp.lastLines = nil
		}
	}

	// Hide cursor on first update for cleaner display
	if !lp.started {
		fmt.Fprint(checked, "\033[?25l")
		lp.hiddenCursor = true
		lp.started = true
	}
}

// moveToFrameTop moves the cursor to column 0 of the first row of the
// previous frame.
func (lp *LivePrinter) moveToFrameTop(w io.Writer) {
	// After drawing N rows (newlines between, none after), the cursor is
	// on row N, so the top is N-1 rows up.
	if rows := lp.frameRows(); rows > 1 {
		fmt.Fprintf(w, "\033[%dA", rows-1)
	}
	fmt.Fprint(w, "\r")
}

// frameRows returns how many terminal rows the previous frame occupies at
// the current width. A terminal that reflows on resize (ghostty, iTerm2,
// kitty, WezTerm, Terminal.app) rewraps each line that no longer fits onto
// more rows, so each line takes ceil(lineWidth/width) rows, and at least
// one. The last line ends at the cursor, which reflow keeps after the last
// character, so it is counted from the cursor column instead.
func (lp *LivePrinter) frameRows() int {
	if len(lp.lastLines) == 0 || lp.unwrapped {
		return lp.lastHeight
	}
	width := lp.config.Width
	if width <= 0 {
		return len(lp.lastLines)
	}
	rows := 0
	last := len(lp.lastLines) - 1
	for _, line := range lp.lastLines[:last] {
		if w := liveLineWidth(line); w > width {
			rows += (w + width - 1) / width
		} else {
			rows++
		}
	}
	return rows + lp.cursorColumn(liveLineWidth(lp.lastLines[last]))/width + 1
}

// cursorColumn returns the cursor's offset in the last line of the previous
// frame, a line w cells wide. After w cells the cursor sits after the text,
// except when the text filled its last row: then the cursor waits on the
// last cell with a pending wrap.
func (lp *LivePrinter) cursorColumn(w int) int {
	drawn := lp.drawnWidth
	if drawn <= 0 {
		drawn = lp.config.Width
	}
	if w > 0 && drawn > 0 && w%drawn == 0 {
		return w - 1
	}
	return w
}

// liveLineWidth returns the display width of a line the live printer drew,
// skipping its escape sequences: CSI (SGR styles), OSC (OSC 8 hyperlinks),
// and other ESC sequences such as a charset choice (ESC ( B).
func liveLineWidth(line string) int {
	if !strings.Contains(line, "\x1b") {
		return runewidth.StringWidth(line)
	}
	var text strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] != 0x1b {
			text.WriteByte(line[i])
			continue
		}
		if i+1 >= len(line) {
			break
		}
		switch line[i+1] {
		case '[': // CSI: parameters, then a final byte 0x40-0x7E
			i += 2
			for i < len(line) && (line[i] < 0x40 || line[i] > 0x7e) {
				i++
			}
		case ']': // OSC: ends at BEL or ST (ESC \)
			i += 2
			for i < len(line) && line[i] != 0x07 && !(line[i] == 0x1b && i+1 < len(line) && line[i+1] == '\\') {
				i++
			}
			if i < len(line) && line[i] == 0x1b {
				i++
			}
		default: // ESC, intermediate bytes 0x20-0x2F, then a final byte
			i++
			for i < len(line) && line[i] >= 0x20 && line[i] <= 0x2f {
				i++
			}
		}
	}
	return runewidth.StringWidth(text.String())
}

// writeFrame writes newLines over the previous frame, redrawing only the
// lines that changed.
func (lp *LivePrinter) writeFrame(checked *checkedLiveOutput, newLines []string, useSync bool) error {
	height := len(newLines)

	// Build output with line-level diffing
	var output strings.Builder

	// When height changes, disable line-level diffing for this frame.
	// This ensures we rewrite all lines, maintaining correct cursor positioning.
	// Without this, skipped lines during height transitions can cause cursor
	// misalignment, leading to scrollback contamination.
	heightChanging := height != lp.lastHeight || lp.rewriteAll
	lp.rewriteAll = false

	for y := 0; y < height; y++ {
		newLine := ""
		if y < len(newLines) {
			newLine = newLines[y]
		}

		// Check if this line changed from the previous frame
		oldLine := ""
		if y < len(lp.lastLines) {
			oldLine = lp.lastLines[y]
		}

		// Always rewrite when height is changing, or when line content differs
		if heightChanging || newLine != oldLine || y >= len(lp.lastLines) {
			// Line changed - clear and rewrite
			output.WriteString("\r\033[2K") // Move to column 0 and clear line
			output.WriteString(newLine)
		}

		// Move to next line (except for last line)
		if y < height-1 {
			output.WriteString("\n")
		}
	}

	// If the new content is shorter than before, clear the extra lines.
	// We move down one line first to avoid clearing the last content line,
	// then clear from cursor to end of screen. In raw mode "\n" keeps the
	// column, so "\r" is needed to clear the orphaned line from its start.
	if height < lp.lastHeight {
		output.WriteString("\n\r")    // Move to first orphaned line
		output.WriteString("\033[0J") // Clear from cursor to end of screen
		output.WriteString("\033[A")  // Move back up to last content line
	}

	// Store lines for next frame's diffing
	lp.lastLines = newLines
	lp.lastHeight = height
	lp.drawnWidth = lp.config.Width
	lp.unwrapped = false

	// Optionally wrap in synchronized output mode to prevent flicker.
	// This tells the terminal to buffer all changes and render atomically.
	// Supported by: iTerm2, kitty, alacritty, Windows Terminal, WezTerm, foot, etc.
	// Terminals that don't support it simply ignore the escape sequences.
	var finalOutput string
	if useSync {
		finalOutput = "\033[?2026h" + output.String() + "\033[?2026l"
	} else {
		finalOutput = output.String()
	}

	_, err := io.WriteString(checked, finalOutput)
	return errors.Join(err, checked.err)
}

// Stop finalizes the live region, moving the cursor to column 0 of the line
// below the content and restoring cursor visibility.
func (lp *LivePrinter) Stop() {
	if lp.hiddenCursor {
		// Show cursor again
		fmt.Fprint(lp.config.Output, "\033[?25h")
		lp.hiddenCursor = false
	}
	// Ensure we're on a new line after the content. In raw mode a bare
	// "\n" keeps the column, so "\r" is needed to start the line at column
	// 0. In cooked mode the tty adds the same "\r".
	if lp.started && lp.lastHeight > 0 {
		fmt.Fprint(lp.config.Output, "\r\n")
	}
}

// Clear removes the live region content and resets state.
func (lp *LivePrinter) Clear() {
	_ = lp.clearChecked()
}

func (lp *LivePrinter) clearChecked() error {
	var output strings.Builder
	if lp.started && lp.lastHeight > 0 {
		// Move up and clear
		lp.moveToFrameTop(&output)
		fmt.Fprint(&output, "\033[0J")
	}
	if output.Len() > 0 {
		checked := &checkedLiveOutput{destination: lp.config.Output}
		if _, err := io.WriteString(checked, output.String()); err != nil {
			return err
		}
	}
	lp.lastHeight = 0
	lp.lastLines = nil // Reset diff state
	lp.drawnWidth = 0
	lp.started = false
	return nil
}

// renderToANSILive is like renderToANSI but clears each line for live updates.
func renderToANSILive(t *Terminal, width, height int) string {
	var output strings.Builder

	var currentStyle Style
	styleSet := false

	for y := 0; y < height; y++ {
		// Move to column 0 and clear the line for clean updates
		// (in raw mode, \n doesn't do carriage return)
		output.WriteString("\r\033[2K")

		lineHasContent := false
		lastContentX := -1

		// Find the last non-space character on this line
		for x := width - 1; x >= 0; x-- {
			cell := t.GetCell(x, y)
			if cell.Char != ' ' || cell.Style != NewStyle() {
				lastContentX = x
				break
			}
		}

		for x := 0; x <= lastContentX || !lineHasContent; x++ {
			if x >= width {
				break
			}

			cell := t.GetCell(x, y)

			if cell.Continuation {
				continue
			}

			if !styleSet || cell.Style != currentStyle {
				if styleSet {
					output.WriteString("\033[0m")
				}
				if cell.Style != NewStyle() {
					output.WriteString(cell.Style.String())
				}
				currentStyle = cell.Style
				styleSet = true
			}

			writeCellGlyph(&output, cell)
			lineHasContent = true

			if x >= lastContentX && lastContentX >= 0 {
				break
			}
		}

		// Reset style at end of line
		if styleSet && currentStyle != NewStyle() {
			output.WriteString("\033[0m")
			styleSet = false
			currentStyle = NewStyle()
		}

		// Add newline for all lines except the last
		if y < height-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

// renderToLines converts the terminal buffer to a slice of ANSI-encoded lines.
// Each line is a complete ANSI string (with style codes) that can be compared
// for line-level diffing. This enables LivePrinter to only update changed lines,
// significantly reducing bandwidth and flicker when only part of the view changes.
//
// The output format matches what would be rendered to the terminal, so string
// comparison accurately detects visual changes.
func renderToLines(t *Terminal, width, height int) []string {
	lines := make([]string, height)

	for y := 0; y < height; y++ {
		var line strings.Builder
		var currentStyle Style
		styleSet := false

		lastContentX := -1
		// Find the last non-space character on this line
		for x := width - 1; x >= 0; x-- {
			cell := t.GetCell(x, y)
			if cell.Char != ' ' || cell.Style != NewStyle() {
				lastContentX = x
				break
			}
		}

		lineHasContent := false
		for x := 0; x <= lastContentX || !lineHasContent; x++ {
			if x >= width {
				break
			}

			cell := t.GetCell(x, y)

			if cell.Continuation {
				continue
			}

			if !styleSet || cell.Style != currentStyle {
				if styleSet {
					line.WriteString("\033[0m")
				}
				if cell.Style != NewStyle() {
					line.WriteString(cell.Style.String())
				}
				currentStyle = cell.Style
				styleSet = true
			}

			writeCellGlyph(&line, cell)
			lineHasContent = true

			if x >= lastContentX && lastContentX >= 0 {
				break
			}
		}

		// Reset style at end of line
		if styleSet && currentStyle != NewStyle() {
			line.WriteString("\033[0m")
		}

		lines[y] = line.String()
	}

	return lines
}

// writeCellGlyph writes a cell's glyph to w, emitting the leading rune plus
// any trailing runes that belong to the same grapheme cluster (VS16
// selectors, combining marks, ZWJ tails, the second half of a regional
// indicator flag, skin-tone modifiers, and so on). Empty cells (Char == 0)
// are written as a single space.
func writeCellGlyph(w *strings.Builder, cell Cell) {
	if cell.Char == 0 {
		w.WriteByte(' ')
		return
	}
	w.WriteRune(cell.Char)
	if cell.Trailing != "" {
		w.WriteString(cell.Trailing)
	}
}

// Live is a convenience function for simple live updates with a callback.
// It creates a LivePrinter, calls the provided function with an update callback,
// and automatically stops when done.
//
// Example:
//
//	tui.Live(func(update func(tui.View)) {
//	    for i := 0; i <= 100; i++ {
//	        update(tui.Text("Loading... %d%%", i))
//	        time.Sleep(50 * time.Millisecond)
//	    }
//	}, tui.WithWidth(40))
func Live(fn func(update func(View)), opts ...PrintOption) error {
	lp := NewLivePrinter(opts...)
	defer lp.Stop()

	var lastErr error
	update := func(view View) {
		if err := lp.Update(view); err != nil {
			lastErr = err
		}
	}

	fn(update)
	return lastErr
}

// checkedLiveOutput retains failures from cursor/protocol writes that precede
// the cell frame, including writers that report a short write without an error.
type checkedLiveOutput struct {
	destination io.Writer
	err         error
}

func (w *checkedLiveOutput) Write(data []byte) (int, error) {
	n, err := w.destination.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	w.err = errors.Join(w.err, err)
	return n, err
}
