package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/deepnoodle-ai/wonton/internal/terminalstate"
	"github.com/deepnoodle-ai/wonton/terminal"
	"golang.org/x/term"
)

// Common errors returned by Prompt
var (
	// ErrInterrupted is returned when the user presses Ctrl+C.
	ErrInterrupted = errors.New("interrupted")

	// ErrEOF is returned when the user presses Ctrl+D on empty input.
	ErrEOF = errors.New("end of input")
)

// AutocompleteFunc is called when the user presses Tab to get completion suggestions.
// It receives the current input and cursor position, and returns a list of completions
// and the index in the input string where replacement should start from.
// Both cursorPos and replaceFrom are byte offsets into input.
//
// Example for file completion:
//
//	func(input string, cursorPos int) ([]string, int) {
//	    // Find @ before cursor
//	    atIdx := strings.LastIndex(input[:cursorPos], "@")
//	    if atIdx == -1 {
//	        return nil, 0
//	    }
//	    prefix := input[atIdx+1 : cursorPos]
//	    matches := findFiles(prefix)
//	    return matches, atIdx
//	}
type AutocompleteFunc func(input string, cursorPos int) (completions []string, replaceFrom int)

// PromptOption configures the Prompt function.
type PromptOption func(*promptConfig)

type promptConfig struct {
	history      *[]string
	autocomplete AutocompleteFunc
	multiLine    bool
	placeholder  string
	validator    func(string) error
	maxLength    int
	mask         string
	masked       bool // set by WithMask, so WithMask("") hides input
	promptStyle  Style
	inputStyle   Style
	output       io.Writer
	input        io.Reader // nil = os.Stdin
	frame        bool
}

func defaultPromptConfig() promptConfig {
	return promptConfig{
		output:      os.Stdout,
		frame:       true,
		promptStyle: NewStyle(),
		inputStyle:  NewStyle(),
	}
}

// WithHistory enables command history navigation with up/down arrows.
// The provided slice is updated with new entries.
func WithHistory(history *[]string) PromptOption {
	return func(c *promptConfig) {
		c.history = history
	}
}

// WithAutocomplete enables tab completion.
// The function receives the current input and cursor position,
// returns a list of completions and the start index to replace from.
func WithAutocomplete(fn AutocompleteFunc) PromptOption {
	return func(c *promptConfig) {
		c.autocomplete = fn
	}
}

// WithMultiLine allows multi-line input via Shift+Enter or Ctrl+J.
// Enter submits; Shift+Enter/Ctrl+J inserts newline.
func WithMultiLine(enabled bool) PromptOption {
	return func(c *promptConfig) {
		c.multiLine = enabled
	}
}

// WithPlaceholder shows hint text when input is empty.
func WithPlaceholder(text string) PromptOption {
	return func(c *promptConfig) {
		c.placeholder = text
	}
}

// WithValidator validates input before submission.
// Return non-nil error to prevent submission and show error message.
func WithValidator(fn func(string) error) PromptOption {
	return func(c *promptConfig) {
		c.validator = fn
	}
}

// WithMaxLength limits input length.
func WithMaxLength(n int) PromptOption {
	return func(c *promptConfig) {
		c.maxLength = n
	}
}

// WithMask masks input (for passwords). Empty string = show nothing.
func WithMask(char string) PromptOption {
	return func(c *promptConfig) {
		c.mask = char
		c.masked = true
	}
}

// WithPromptStyle styles the prompt text.
func WithPromptStyle(style Style) PromptOption {
	return func(c *promptConfig) {
		c.promptStyle = style
	}
}

// WithInputStyle styles the user's input text.
func WithInputStyle(style Style) PromptOption {
	return func(c *promptConfig) {
		c.inputStyle = style
	}
}

// WithPromptOutput sets the output writer for the prompt. Default is os.Stdout.
// When w is a terminal, the prompt takes its width from w.
func WithPromptOutput(w io.Writer) PromptOption {
	return func(c *promptConfig) {
		c.output = w
	}
}

// WithPromptInput sets the reader the prompt reads keys from. Default is
// os.Stdin.
//
// When r is a terminal (it has an Fd method and the descriptor is a
// terminal), Prompt puts it in raw mode while it reads. When r is not a
// terminal, Prompt still runs the full line editor and decodes the bytes as
// a terminal in raw mode sends them: "\r" is Enter, "\x7f" is Backspace,
// "\x1b[D" is Left, "\x03" is Ctrl+C. This lets a program or a test drive
// the prompt from a pipe or a bytes.Reader. Prompt reads no bytes past the
// key that ends the input, so one reader can feed several prompts in turn.
//
// Without this option, a non-terminal os.Stdin gets a plain line read with
// no editing.
func WithPromptInput(r io.Reader) PromptOption {
	return func(c *promptConfig) {
		c.input = r
	}
}

// WithPromptFrame sets whether the prompt draws a line above and below the
// input while the user types. Default is true. With false, the prompt shows
// only the prompt text and the input, as a plain REPL prompt does.
func WithPromptFrame(enabled bool) PromptOption {
	return func(c *promptConfig) {
		c.frame = enabled
	}
}

// promptState holds the runtime state during input
type promptState struct {
	config     promptConfig
	prompt     string
	input      []rune
	cursorPos  int
	historyIdx int // -1 = not navigating
	savedInput []rune
	acActive   bool
	acMatches  []string
	acSelected int
	acFrom     int
	// crlf is set when the input terminal is in raw mode and the output
	// is a terminal. Raw mode turns off output post-processing, so "\n"
	// alone does not return the cursor to column 0.
	crlf bool
}

// Prompt reads a line of input from the user with rich editing support.
// It temporarily enables raw mode, handles input, and restores terminal state.
// Returns the entered text or an error.
//
// The input region is as wide as the terminal: the output's width when the
// output is a terminal, else the input's, else 80 columns. On Unix it
// follows a resize of the terminal while the prompt is open.
//
// Errors:
//   - ErrInterrupted: User pressed Ctrl+C
//   - ErrEOF: User pressed Ctrl+D on empty input, or the input ended
//   - Other errors from terminal operations
//
// Example:
//
//	input, err := tui.Prompt("> ",
//	    tui.WithHistory(&history),
//	    tui.WithAutocomplete(completeFn),
//	    tui.WithMultiLine(true),
//	)
//
// A plain REPL prompt with no line above and below the input:
//
//	input, err := tui.Prompt("> ", tui.WithPromptFrame(false))
//
// WithPromptInput and WithPromptOutput set the streams, for example to
// drive the prompt from a test.
func Prompt(prompt string, opts ...PromptOption) (string, error) {
	cfg := defaultPromptConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	in := cfg.input
	if in == nil {
		in = os.Stdin
	}
	keys := in
	raw := false
	if fd, ok := terminalFd(in); ok {
		oldState, err := term.MakeRaw(fd)
		if err != nil {
			return readLineFallback(in)
		}
		defer term.Restore(fd, oldState)
		raw = true

		// Enable bracketed paste mode
		fmt.Fprint(cfg.output, "\033[?2004h")
		defer fmt.Fprint(cfg.output, "\033[?2004l")

		// Enable enhanced keyboard mode for Shift+Enter detection
		fmt.Fprint(cfg.output, terminalstate.KeyboardEnable())
		defer fmt.Fprint(cfg.output, terminalstate.KeyboardRelease())
	} else if cfg.input == nil {
		// os.Stdin is piped or redirected: a plain line read.
		return readLineFallback(in)
	} else {
		// An explicit non-terminal reader: run the line editor on its bytes.
		keys = &scriptedKeys{r: in}
	}

	// Initialize state
	state := &promptState{
		config:     cfg,
		prompt:     prompt,
		input:      []rune{},
		cursorPos:  0,
		historyIdx: -1,
		crlf:       raw && isTerminal(cfg.output),
	}

	// Create live printer for the input region
	width, fromTerminal := promptWidth(cfg.output, in)
	region := &promptRegion{
		state: state,
		lp:    NewLivePrinter(WithWidth(width), WithOutput(cfg.output)),
	}
	lp := region.lp
	defer lp.Stop() // restores the cursor the live printer hid

	// Render initial view
	lp.Update(state.buildView())

	stopResize := func() {}
	if fromTerminal {
		stopResize = watchPromptResize(func() {
			if w, ok := promptWidth(cfg.output, in); ok {
				region.resize(w)
			}
		})
	}
	defer stopResize()

	// Input loop
	decoder := terminal.NewKeyDecoder(keys)
	for {
		event, err := decoder.ReadKeyEvent()
		if err != nil {
			stopResize()
			lp.Clear()
			if err == io.EOF {
				return "", ErrEOF
			}
			return "", err
		}

		result, done, keyErr := region.handleKey(event, prompt)
		if done {
			// Stop the watcher first, so a resize cannot redraw the
			// cleared region.
			stopResize()
			lp.Clear()
			state.printFinal(cfg.output, prompt)
			return result, keyErr
		}
	}
}

// promptRegion is the prompt's state and the live printer that draws it.
// mu guards both between the input loop and the resize watcher.
type promptRegion struct {
	mu    sync.Mutex
	state *promptState
	lp    *LivePrinter
}

// handleKey applies a key event to the state and redraws the region unless
// the prompt is done.
func (r *promptRegion) handleKey(event terminal.KeyEvent, prompt string) (string, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result, done, err := r.state.handleKey(event, prompt)
	if !done {
		r.lp.Update(r.state.buildView())
	}
	return result, done, err
}

// resize redraws the region at the given terminal width.
func (r *promptRegion) resize(width int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lp.SetWidth(width)
	r.lp.Update(r.state.buildView())
}

// defaultPromptWidth is the prompt's width when neither stream is a
// terminal.
const defaultPromptWidth = 80

// promptWidth returns the width of the prompt's input region: the output's
// terminal width, else the input's terminal width, else
// defaultPromptWidth. fromTerminal reports whether a terminal gave it.
func promptWidth(out io.Writer, in io.Reader) (width int, fromTerminal bool) {
	if w, _, ok := terminalSize(out); ok {
		return w, true
	}
	if fd, ok := terminalFd(in); ok {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			return w, true
		}
	}
	return defaultPromptWidth, false
}

// terminalFd returns the file descriptor of v when v has one and it is a
// terminal.
// isTerminal reports whether v is a file attached to a terminal.
func isTerminal(v any) bool {
	_, ok := terminalFd(v)
	return ok
}

func terminalFd(v any) (int, bool) {
	f, ok := v.(interface{ Fd() uintptr })
	if !ok {
		return 0, false
	}
	fd := int(f.Fd())
	return fd, term.IsTerminal(fd)
}

// watchPromptResize calls onResize on its own goroutine each time the
// terminal is resized, until stop is called. stop waits for the goroutine
// to exit and is safe to call more than once. On platforms with no resize
// signal it does nothing.
func watchPromptResize(onResize func()) (stop func()) {
	sigs := promptResizeSignals()
	if len(sigs) == 0 {
		return func() {}
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	done := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		for {
			select {
			case <-ch:
				onResize()
			case <-done:
				return
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			signal.Stop(ch)
			close(done)
			<-exited
		})
	}
}

// scriptedKeys feeds a non-terminal input to the key decoder. It reads one
// byte at a time, so the decoder never buffers bytes past the key that ends
// the prompt, and the next prompt on the same reader gets them. It reports
// that more input is always coming, so the decoder reads an escape sequence
// whole instead of taking its ESC for the Escape key.
type scriptedKeys struct {
	r io.Reader
}

func (s *scriptedKeys) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return s.r.Read(p)
}

// WaitReady implements the key decoder's optional ready check.
func (s *scriptedKeys) WaitReady(time.Duration) bool { return true }

// printFinal prints the prompt and the entered input on their own line after
// the live region is cleared. Masked input stays masked. On a raw-mode
// terminal the line ends with "\r\n", so the next output starts in
// column 0. Other output gets "\n".
func (s *promptState) printFinal(w io.Writer, prompt string) {
	fmt.Fprint(w, s.config.promptStyle.String())
	fmt.Fprint(w, prompt)
	fmt.Fprint(w, "\033[0m")
	fmt.Fprint(w, s.getDisplayInput())
	if s.crlf {
		fmt.Fprint(w, "\r\n")
	} else {
		fmt.Fprint(w, "\n")
	}
}

// handleKey processes a key event and returns (result, done, error)
func (s *promptState) handleKey(event terminal.KeyEvent, prompt string) (string, bool, error) {
	// Handle paste events
	if event.Paste != "" {
		s.insertString(event.Paste)
		s.acActive = false
		return "", false, nil
	}

	// Handle autocomplete if active
	if s.acActive {
		switch event.Key {
		case terminal.KeyArrowDown:
			s.acSelected = (s.acSelected + 1) % len(s.acMatches)
			return "", false, nil
		case terminal.KeyArrowUp:
			s.acSelected--
			if s.acSelected < 0 {
				s.acSelected = len(s.acMatches) - 1
			}
			return "", false, nil
		case terminal.KeyTab, terminal.KeyEnter:
			// Apply selection
			s.applyAutocomplete()
			return "", false, nil
		case terminal.KeyEscape:
			s.acActive = false
			return "", false, nil
		default:
			// Any other key dismisses autocomplete
			s.acActive = false
		}
	}

	// Handle special keys
	switch event.Key {
	case terminal.KeyEnter:
		// Shift+Enter for newline in multi-line mode
		if s.config.multiLine && event.Shift {
			s.insertRune('\n')
			return "", false, nil
		}
		// Submit
		result := string(s.input)
		if s.config.validator != nil {
			if err := s.config.validator(result); err != nil {
				// TODO: Show error message
				return "", false, nil
			}
		}
		// Add to history
		if s.config.history != nil && result != "" {
			*s.config.history = append(*s.config.history, result)
		}
		return result, true, nil

	case terminal.KeyCtrlJ:
		// Ctrl+J for newline in multi-line mode
		if s.config.multiLine {
			s.insertRune('\n')
			return "", false, nil
		}

	case terminal.KeyCtrlC:
		return "", true, ErrInterrupted

	case terminal.KeyCtrlD:
		if len(s.input) == 0 {
			return "", true, ErrEOF
		}
		// Delete character at cursor
		s.deleteChar()

	case terminal.KeyBackspace:
		s.backspace()

	case terminal.KeyArrowLeft:
		if s.cursorPos > 0 {
			s.cursorPos--
		}

	case terminal.KeyArrowRight:
		if s.cursorPos < len(s.input) {
			s.cursorPos++
		}

	case terminal.KeyHome, terminal.KeyCtrlA:
		s.cursorPos = 0

	case terminal.KeyEnd, terminal.KeyCtrlE:
		s.cursorPos = len(s.input)

	case terminal.KeyCtrlU:
		// Clear line
		s.input = []rune{}
		s.cursorPos = 0

	case terminal.KeyCtrlW:
		// Delete word backward
		s.deleteWordBackward()

	case terminal.KeyCtrlK:
		// Delete to end of line
		s.input = s.input[:s.cursorPos]

	case terminal.KeyArrowUp:
		s.historyUp()

	case terminal.KeyArrowDown:
		s.historyDown()

	case terminal.KeyTab:
		if s.config.autocomplete != nil {
			s.triggerAutocomplete()
		}

	case terminal.KeyDelete:
		s.deleteChar()

	default:
		// Regular character
		if event.Rune != 0 {
			s.insertRune(event.Rune)
		}
	}

	return "", false, nil
}

// insertRune inserts a rune at the cursor position
func (s *promptState) insertRune(r rune) {
	// Check max length
	if s.config.maxLength > 0 && len(s.input) >= s.config.maxLength {
		return
	}

	// Exit history mode
	s.historyIdx = -1

	// Insert rune at cursor
	s.input = append(s.input[:s.cursorPos], append([]rune{r}, s.input[s.cursorPos:]...)...)
	s.cursorPos++

	// Auto-trigger autocomplete if configured
	if s.config.autocomplete != nil {
		s.tryAutoTriggerAutocomplete()
	}
}

// insertString inserts a string at the cursor position
func (s *promptState) insertString(str string) {
	for _, r := range str {
		s.insertRune(r)
	}
}

// backspace deletes the character before the cursor
func (s *promptState) backspace() {
	if s.cursorPos > 0 {
		s.input = append(s.input[:s.cursorPos-1], s.input[s.cursorPos:]...)
		s.cursorPos--
		s.historyIdx = -1

		// Update autocomplete if active
		if s.config.autocomplete != nil {
			s.tryAutoTriggerAutocomplete()
		}
	}
}

// deleteChar deletes the character at the cursor
func (s *promptState) deleteChar() {
	if s.cursorPos < len(s.input) {
		s.input = append(s.input[:s.cursorPos], s.input[s.cursorPos+1:]...)
	}
}

// deleteWordBackward deletes the word before the cursor
func (s *promptState) deleteWordBackward() {
	if s.cursorPos == 0 {
		return
	}

	// Find start of current word
	pos := s.cursorPos - 1

	// Skip trailing whitespace
	for pos > 0 && unicode.IsSpace(s.input[pos]) {
		pos--
	}

	// Delete word characters
	for pos > 0 && !unicode.IsSpace(s.input[pos-1]) {
		pos--
	}

	// Delete from pos to cursor
	s.input = append(s.input[:pos], s.input[s.cursorPos:]...)
	s.cursorPos = pos
	s.historyIdx = -1
}

// historyUp navigates to previous history entry
func (s *promptState) historyUp() {
	if s.config.history == nil || len(*s.config.history) == 0 {
		return
	}

	// Save current input on first history navigation
	if s.historyIdx == -1 {
		s.savedInput = make([]rune, len(s.input))
		copy(s.savedInput, s.input)
		s.historyIdx = len(*s.config.history)
	}

	if s.historyIdx > 0 {
		s.historyIdx--
		s.input = []rune((*s.config.history)[s.historyIdx])
		s.cursorPos = len(s.input)
	}
}

// historyDown navigates to next history entry
func (s *promptState) historyDown() {
	if s.config.history == nil || s.historyIdx == -1 {
		return
	}

	s.historyIdx++
	if s.historyIdx >= len(*s.config.history) {
		// Restore saved input
		s.input = s.savedInput
		s.historyIdx = -1
	} else {
		s.input = []rune((*s.config.history)[s.historyIdx])
	}
	s.cursorPos = len(s.input)
}

// runAutocomplete calls the AutocompleteFunc with a byte cursor offset, as
// its documentation shows, and converts the returned byte offset back to a
// rune index into s.input.
func (s *promptState) runAutocomplete() ([]string, int) {
	input := string(s.input)
	cursor := len(string(s.input[:s.cursorPos]))
	completions, from := s.config.autocomplete(input, cursor)
	from = min(max(from, 0), cursor)
	// An offset inside a rune counts its bytes as runes; keep it at or
	// before the cursor.
	return completions, min(utf8.RuneCountInString(input[:from]), s.cursorPos)
}

// triggerAutocomplete activates autocomplete
func (s *promptState) triggerAutocomplete() {
	completions, from := s.runAutocomplete()
	if len(completions) == 0 {
		return
	}

	s.acActive = true
	s.acMatches = completions
	s.acSelected = 0
	s.acFrom = from
}

// tryAutoTriggerAutocomplete checks if autocomplete should be triggered automatically
func (s *promptState) tryAutoTriggerAutocomplete() {
	// Check if autocomplete function would return results
	completions, from := s.runAutocomplete()

	if len(completions) == 0 {
		// No matches, deactivate if active
		s.acActive = false
		return
	}

	// Activate or update autocomplete
	s.acActive = true
	s.acMatches = completions
	s.acSelected = 0
	s.acFrom = from
}

// applyAutocomplete inserts the selected completion
func (s *promptState) applyAutocomplete() {
	if !s.acActive || len(s.acMatches) == 0 {
		return
	}

	completion := []rune(s.acMatches[s.acSelected])

	// Replace from acFrom to cursor with completion
	before := s.input[:s.acFrom]
	after := s.input[s.cursorPos:]
	if m := s.config.maxLength; m > 0 && len(before)+len(completion)+len(after) > m {
		completion = completion[:max(m-len(before)-len(after), 0)]
	}
	s.input = append(before, append(completion, after...)...)
	s.cursorPos = len(before) + len(completion)

	s.acActive = false
}

// buildView creates the complete view for the prompt region
func (s *promptState) buildView() View {
	// Build input content
	var inputContent View

	if len(s.input) == 0 && s.config.placeholder != "" {
		// Show placeholder
		inputContent = Group(
			Text("%s", s.prompt).Style(s.config.promptStyle),
			Text("%s", s.config.placeholder).Dim(),
		)
	} else {
		// Get display input (may be masked for passwords)
		displayInput := s.getDisplayInput()

		// Split into lines for proper multiline rendering
		lines := strings.Split(displayInput, "\n")

		// Find which line the cursor is on and position within that line
		cursorLine := 0
		cursorCol := s.cursorPos
		pos := 0
		for i, line := range lines {
			lineLen := len([]rune(line))
			if pos+lineLen >= s.cursorPos {
				cursorLine = i
				cursorCol = s.cursorPos - pos
				break
			}
			pos += lineLen + 1 // +1 for the newline
			cursorLine = i + 1
			cursorCol = 0
		}
		if cursorLine >= len(lines) {
			// The display is shorter than the input (WithMask("") shows
			// nothing); keep the cursor at the end of the last line.
			cursorLine = len(lines) - 1
			cursorCol = len([]rune(lines[cursorLine]))
		}

		// Build each line as a view
		lineViews := make([]View, len(lines))
		for i, line := range lines {
			lineRunes := []rune(line)

			if i == cursorLine {
				// This line has the cursor
				var beforeCursor, cursorChar, afterCursor string

				if cursorCol < len(lineRunes) {
					beforeCursor = string(lineRunes[:cursorCol])
					cursorChar = string(lineRunes[cursorCol])
					afterCursor = string(lineRunes[cursorCol+1:])
				} else {
					beforeCursor = line
					cursorChar = " "
					afterCursor = ""
				}

				if i == 0 {
					// First line includes prompt
					lineViews[i] = Group(
						Text("%s", s.prompt).Style(s.config.promptStyle),
						Text("%s", beforeCursor).Style(s.config.inputStyle),
						Text("%s", cursorChar).Reverse(),
						Text("%s", afterCursor).Style(s.config.inputStyle),
					)
				} else {
					// Continuation line - indent to match prompt
					indent := strings.Repeat(" ", len([]rune(s.prompt)))
					lineViews[i] = Group(
						Text("%s", indent),
						Text("%s", beforeCursor).Style(s.config.inputStyle),
						Text("%s", cursorChar).Reverse(),
						Text("%s", afterCursor).Style(s.config.inputStyle),
					)
				}
			} else {
				// Line without cursor
				if i == 0 {
					lineViews[i] = Group(
						Text("%s", s.prompt).Style(s.config.promptStyle),
						Text("%s", line).Style(s.config.inputStyle),
					)
				} else {
					indent := strings.Repeat(" ", len([]rune(s.prompt)))
					lineViews[i] = Group(
						Text("%s", indent),
						Text("%s", line).Style(s.config.inputStyle),
					)
				}
			}
		}

		if len(lineViews) == 1 {
			inputContent = lineViews[0]
		} else {
			inputContent = Stack(lineViews...)
		}
	}

	// Build the input region, with a line above and below unless the
	// frame is off
	inputRegion := inputContent
	if s.config.frame {
		inputRegion = Stack(
			Divider(),
			inputContent,
			Divider(),
		)
	}

	// If no autocomplete, just return input region
	if !s.acActive || len(s.acMatches) == 0 {
		return inputRegion
	}

	// Build autocomplete dropdown
	maxShow := 5
	items := []View{}

	// Scroll the window so the selected match stays visible.
	start := max(s.acSelected-maxShow+1, 0)
	for i := start; i < len(s.acMatches) && i < start+maxShow; i++ {
		var item View
		if i == s.acSelected {
			item = Group(
				Text("→").Fg(ColorCyan),
				Text(" %s", s.acMatches[i]),
			)
		} else {
			item = Text("  %s", s.acMatches[i]).Dim()
		}
		items = append(items, item)
	}

	if len(s.acMatches) > maxShow {
		items = append(items, Text("  ... +%d more", len(s.acMatches)-maxShow).Dim())
	}

	dropdown := Bordered(
		Stack(items...),
	).Border(&RoundedBorder).BorderFg(ColorCyan)

	return Stack(inputRegion, dropdown)
}

// getDisplayInput returns the input as it should be displayed (with masking if configured)
func (s *promptState) getDisplayInput() string {
	if s.config.masked || s.config.mask != "" {
		return strings.Repeat(s.config.mask, len(s.input))
	}
	return string(s.input)
}

// readLineFallback is a simple fallback for non-terminal input
func readLineFallback(in io.Reader) (string, error) {
	var line strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := in.Read(buf)
		if err != nil {
			if err == io.EOF && line.Len() > 0 {
				return line.String(), nil
			}
			return "", err
		}
		if n > 0 {
			if buf[0] == '\n' {
				return line.String(), nil
			}
			line.WriteByte(buf[0])
		}
	}
}
