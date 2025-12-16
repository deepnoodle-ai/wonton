package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// PrintOption configures the Print function.
type PrintOption func(*printConfig)

type printConfig struct {
	width  int
	height int // 0 = auto (based on view size)
	output io.Writer
}

func defaultPrintConfig() printConfig {
	// Try to get terminal width, fall back to 80
	width := 80
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}
	return printConfig{
		width:  width,
		height: 0, // auto
		output: os.Stdout,
	}
}

// WithWidth sets the width for rendering. Default is terminal width or 80.
func WithWidth(width int) PrintOption {
	return func(c *printConfig) {
		if width > 0 {
			c.width = width
		}
	}
}

// WithHeight sets a fixed height for rendering.
// Default is 0 (auto), which uses the view's natural height.
func WithHeight(height int) PrintOption {
	return func(c *printConfig) {
		c.height = height
	}
}

// WithOutput sets the output writer. Default is os.Stdout.
func WithOutput(w io.Writer) PrintOption {
	return func(c *printConfig) {
		c.output = w
	}
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
// Options can customize the output:
//
//	tui.Print(view, tui.WithWidth(60))
func Print(view View, opts ...PrintOption) error {
	cfg := defaultPrintConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Measure the view to get its natural height
	_, viewHeight := view.size(cfg.width, 0)
	if viewHeight == 0 {
		viewHeight = 1 // At least one line
	}

	// Use configured height or view's natural height
	height := cfg.height
	if height == 0 {
		height = viewHeight
	}

	// Create an in-memory terminal buffer
	var buf strings.Builder
	terminal := NewTestTerminal(cfg.width, height, &buf)

	// Render the view to the buffer
	frame, err := terminal.BeginFrame()
	if err != nil {
		return fmt.Errorf("failed to begin frame: %w", err)
	}

	// Clear the frame
	frame.Fill(' ', NewStyle())

	// Create render context and render the view
	ctx := NewRenderContext(frame, 0)
	view.size(cfg.width, height) // Measure phase
	view.render(ctx)             // Render phase

	// End the frame (this populates the buffer)
	terminal.EndFrame(frame)

	// Convert buffer to ANSI output
	output := renderToANSI(terminal, cfg.width, height)

	// Write to output
	_, err = io.WriteString(cfg.output, output)
	return err
}

// renderToANSI converts the terminal buffer to a string with ANSI escape codes.
func renderToANSI(t *Terminal, width, height int) string {
	var output strings.Builder

	var currentStyle Style
	styleSet := false

	for y := 0; y < height; y++ {
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

			// Skip continuation cells (part of wide characters)
			if cell.Continuation {
				continue
			}

			// Update style if needed
			if !styleSet || cell.Style != currentStyle {
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

			char := cell.Char
			if char == 0 {
				char = ' '
			}
			output.WriteRune(char)
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

		// Add newline (except for the last line if it's empty)
		if y < height-1 || lineHasContent {
			output.WriteString("\n")
		}
	}

	return output.String()
}

// Fprint renders a view to the specified writer.
// This is a convenience wrapper around Print with WithOutput.
func Fprint(w io.Writer, view View, opts ...PrintOption) error {
	opts = append([]PrintOption{WithOutput(w)}, opts...)
	return Print(view, opts...)
}

// Sprint renders a view to a string with ANSI escape codes.
func Sprint(view View, opts ...PrintOption) string {
	var buf strings.Builder
	opts = append([]PrintOption{WithOutput(&buf)}, opts...)
	Print(view, opts...)
	return buf.String()
}

// LivePrinter renders views to a fixed region of the terminal that updates in place.
// This is useful for progress bars, status displays, and loading indicators that
// need to update without scrolling the terminal.
//
// Example:
//
//	live := tui.NewLivePrinter(tui.WithWidth(60))
//	defer live.Stop()
//
//	for i := 0; i <= 100; i++ {
//	    live.Update(tui.Text("Progress: %d%%", i))
//	    time.Sleep(50 * time.Millisecond)
//	}
type LivePrinter struct {
	config      printConfig
	lastHeight  int
	started     bool
	frameCount  uint64
	hiddenCursor bool
}

// NewLivePrinter creates a new LivePrinter for updating a region in place.
func NewLivePrinter(opts ...PrintOption) *LivePrinter {
	cfg := defaultPrintConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &LivePrinter{
		config: cfg,
	}
}

// Update renders a new view, replacing the previous content in place.
// The cursor moves back to overwrite the previous output.
func (lp *LivePrinter) Update(view View) error {
	// Measure the view
	_, viewHeight := view.size(lp.config.width, 0)
	if viewHeight == 0 {
		viewHeight = 1
	}

	height := lp.config.height
	if height == 0 {
		height = viewHeight
	}

	// If we've already rendered, move cursor back up
	if lp.started && lp.lastHeight > 0 {
		// Move cursor up to the start of the previous render.
		// After rendering N lines (with newlines between but not at end),
		// the cursor is on line N. To get back to line 1, move up (N-1) lines.
		if lp.lastHeight > 1 {
			fmt.Fprintf(lp.config.output, "\033[%dA", lp.lastHeight-1)
		}
		// Move to beginning of line
		fmt.Fprint(lp.config.output, "\r")
	}

	// Hide cursor on first update for cleaner display
	if !lp.started {
		fmt.Fprint(lp.config.output, "\033[?25l")
		lp.hiddenCursor = true
		lp.started = true
	}

	// Create terminal buffer and render
	var buf strings.Builder
	terminal := NewTestTerminal(lp.config.width, height, &buf)

	frame, err := terminal.BeginFrame()
	if err != nil {
		return fmt.Errorf("failed to begin frame: %w", err)
	}

	frame.Fill(' ', NewStyle())
	ctx := NewRenderContext(frame, lp.frameCount)
	lp.frameCount++
	view.size(lp.config.width, height)
	view.render(ctx)
	terminal.EndFrame(frame)

	// Convert to ANSI and output
	output := renderToANSILive(terminal, lp.config.width, height)

	// If the new content is shorter than before, we need to clear the extra lines
	if height < lp.lastHeight {
		// Clear from cursor to end of screen
		output += "\033[0J"
	}

	_, err = io.WriteString(lp.config.output, output)
	lp.lastHeight = height

	return err
}

// Stop finalizes the live region, moving the cursor below the content
// and restoring cursor visibility.
func (lp *LivePrinter) Stop() {
	if lp.hiddenCursor {
		// Show cursor again
		fmt.Fprint(lp.config.output, "\033[?25h")
		lp.hiddenCursor = false
	}
	// Ensure we're on a new line after the content
	if lp.started && lp.lastHeight > 0 {
		fmt.Fprintln(lp.config.output)
	}
}

// Clear removes the live region content and resets state.
func (lp *LivePrinter) Clear() {
	if lp.started && lp.lastHeight > 0 {
		// Move up and clear
		if lp.lastHeight > 1 {
			fmt.Fprintf(lp.config.output, "\033[%dA", lp.lastHeight-1)
		}
		fmt.Fprint(lp.config.output, "\r\033[0J")
	}
	lp.lastHeight = 0
	lp.started = false
}

// renderToANSILive is like renderToANSI but clears each line for live updates.
func renderToANSILive(t *Terminal, width, height int) string {
	var output strings.Builder

	var currentStyle Style
	styleSet := false

	for y := 0; y < height; y++ {
		// Clear the line first for clean updates
		output.WriteString("\033[2K")

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

			char := cell.Char
			if char == 0 {
				char = ' '
			}
			output.WriteRune(char)
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
