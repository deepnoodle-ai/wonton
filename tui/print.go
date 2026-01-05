package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deepnoodle-ai/wonton/termtest"
	"golang.org/x/term"
)

// PrintConfig configures printing functions like Print, Fprint, and Sprint.
// All fields are optional with sensible zero-value defaults.
type PrintConfig struct {
	Width   int       // 0 = auto (terminal width or 80). Positive values set the width.
	Height  int       // 0 = auto (based on view size). Positive values set a fixed height.
	Output  io.Writer // nil = os.Stdout. Specify where to write output.
	RawMode bool      // false = use \n line endings. true = use \r\n for raw terminal mode.
}

func (c PrintConfig) withDefaults() PrintConfig {
	if c.Width == 0 {
		c.Width = 80
		if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
			if w, _, err := term.GetSize(fd); err == nil && w > 0 {
				c.Width = w
			}
		}
	}
	if c.Output == nil {
		c.Output = os.Stdout
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
//	tui.Print(view, tui.PrintConfig{Width: 60})
func Print(view View, cfgs ...PrintConfig) error {
	cfg := PrintConfig{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
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
	output := renderToANSI(terminal, cfg.Width, height, cfg.RawMode)

	// Write to output
	_, err = io.WriteString(cfg.Output, output)
	return err
}

// renderToANSI converts the terminal buffer to a string with ANSI escape codes.
// If rawMode is true, uses \r\n line endings (required in raw terminal mode where
// \n alone only moves down without returning to column 0).
func renderToANSI(t *Terminal, width, height int, rawMode bool) string {
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

		// Add newline between lines, but not after the last line
		// (a trailing newline would cause scrollUp in the test screen)
		if y < height-1 {
			output.WriteString(lineEnding)
		}
	}

	return output.String()
}

// Fprint renders a view to the specified writer.
func Fprint(w io.Writer, view View, cfgs ...PrintConfig) error {
	cfg := PrintConfig{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	cfg.Output = w
	return Print(view, cfg)
}

// Sprint renders a view to a string with ANSI escape codes.
func Sprint(view View, cfgs ...PrintConfig) string {
	var buf strings.Builder
	cfg := PrintConfig{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	cfg.Output = &buf
	Print(view, cfg)
	return buf.String()
}

// SprintScreen renders a view and returns a termtest.Screen for assertions.
// This is a convenience function for testing that combines Sprint with
// termtest.Screen parsing, making it easy to write precise visual tests.
//
// Example:
//
//	func TestButton(t *testing.T) {
//	    btn := Button("Submit", func() {})
//	    screen := SprintScreen(btn, PrintConfig{Width: 20})
//	    termtest.AssertRowContains(t, screen, 0, "Submit")
//	}
func SprintScreen(view View, cfgs ...PrintConfig) *termtest.Screen {
	cfg := PrintConfig{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
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
	output := Sprint(view, cfg)

	// Create screen and write output
	screen := termtest.NewScreen(cfg.Width, height)
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
//	live := tui.NewLivePrinter(tui.PrintConfig{Width: 60})
//	defer live.Stop()
//
//	for i := 0; i <= 100; i++ {
//	    live.Update(tui.Text("Progress: %d%%", i))
//	    time.Sleep(50 * time.Millisecond)
//	}
type LivePrinter struct {
	config       PrintConfig
	lastHeight   int
	started      bool
	frameCount   uint64
	hiddenCursor bool
	lastLines    []string // Previous frame's lines for diffing
}

// NewLivePrinter creates a new LivePrinter for updating a region in place.
func NewLivePrinter(cfgs ...PrintConfig) *LivePrinter {
	cfg := PrintConfig{}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}
	cfg = cfg.withDefaults()
	return &LivePrinter{
		config: cfg,
	}
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
	// Measure the view
	_, viewHeight := view.size(lp.config.Width, 0)
	if viewHeight == 0 {
		viewHeight = 1
	}

	height := lp.config.Height
	if height == 0 {
		height = viewHeight
	}

	// If we've already rendered, move cursor back up
	if lp.started && lp.lastHeight > 0 {
		// Move cursor up to the start of the previous render.
		// After rendering N lines (with newlines between but not at end),
		// the cursor is on line N. To get back to line 1, move up (N-1) lines.
		if lp.lastHeight > 1 {
			fmt.Fprintf(lp.config.Output, "\033[%dA", lp.lastHeight-1)
		}
		// Move to beginning of line
		fmt.Fprint(lp.config.Output, "\r")
	}

	// Hide cursor on first update for cleaner display
	if !lp.started {
		fmt.Fprint(lp.config.Output, "\033[?25l")
		lp.hiddenCursor = true
		lp.started = true
	}

	// Create terminal buffer and render
	var buf strings.Builder
	terminal := NewTestTerminal(lp.config.Width, height, &buf)

	frame, err := terminal.BeginFrame()
	if err != nil {
		return fmt.Errorf("failed to begin frame: %w", err)
	}

	frame.Fill(' ', NewStyle())
	ctx := NewRenderContext(frame, lp.frameCount)
	if fm != nil {
		ctx = ctx.WithFocusManager(fm)
	}
	lp.frameCount++
	view.size(lp.config.Width, height)
	view.render(ctx)
	terminal.EndFrame(frame)

	// Convert to individual lines for diffing
	newLines := renderToLines(terminal, lp.config.Width, height)

	// Build output with line-level diffing
	var output strings.Builder

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

		if newLine != oldLine || y >= len(lp.lastLines) {
			// Line changed - clear and rewrite
			output.WriteString("\r\033[2K") // Move to column 0 and clear line
			output.WriteString(newLine)
		}

		// Move to next line (except for last line)
		if y < height-1 {
			output.WriteString("\n")
		}
	}

	// If the new content is shorter than before, clear the extra lines
	if height < lp.lastHeight {
		output.WriteString("\033[0J") // Clear from cursor to end of screen
	}

	// Store lines for next frame's diffing
	lp.lastLines = newLines
	lp.lastHeight = height

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

	_, err = io.WriteString(lp.config.Output, finalOutput)
	return err
}

// Stop finalizes the live region, moving the cursor below the content
// and restoring cursor visibility.
func (lp *LivePrinter) Stop() {
	if lp.hiddenCursor {
		// Show cursor again
		fmt.Fprint(lp.config.Output, "\033[?25h")
		lp.hiddenCursor = false
	}
	// Ensure we're on a new line after the content
	if lp.started && lp.lastHeight > 0 {
		fmt.Fprintln(lp.config.Output)
	}
}

// Clear removes the live region content and resets state.
func (lp *LivePrinter) Clear() {
	if lp.started && lp.lastHeight > 0 {
		// Move up and clear
		if lp.lastHeight > 1 {
			fmt.Fprintf(lp.config.Output, "\033[%dA", lp.lastHeight-1)
		}
		fmt.Fprint(lp.config.Output, "\r\033[0J")
	}
	lp.lastHeight = 0
	lp.lastLines = nil // Reset diff state
	lp.started = false
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

			char := cell.Char
			if char == 0 {
				char = ' '
			}
			line.WriteRune(char)
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
//	}, tui.PrintConfig{Width: 40})
func Live(fn func(update func(View)), cfgs ...PrintConfig) error {
	lp := NewLivePrinter(cfgs...)
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
