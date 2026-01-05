package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

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
	promptStyle  Style
	inputStyle   Style
	output       io.Writer
}

func defaultPromptConfig() promptConfig {
	return promptConfig{
		output:      os.Stdout,
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
func WithPromptOutput(w io.Writer) PromptOption {
	return func(c *promptConfig) {
		c.output = w
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
}

// Prompt reads a line of input from the user with rich editing support.
// It temporarily enables raw mode, handles input, and restores terminal state.
// Returns the entered text (trimmed) or an error.
//
// Errors:
//   - ErrInterrupted: User pressed Ctrl+C
//   - ErrEOF: User pressed Ctrl+D on empty input
//   - Other errors from terminal operations
//
// Example:
//
//	input, err := tui.Prompt("> ",
//	    tui.WithHistory(&history),
//	    tui.WithAutocomplete(completeFn),
//	    tui.WithMultiLine(true),
//	)
func Prompt(prompt string, opts ...PromptOption) (string, error) {
	cfg := defaultPromptConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Check if stdin is a terminal
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		// Fallback to simple line read
		return readLineFallback()
	}

	// Save terminal state
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return readLineFallback()
	}
	defer term.Restore(fd, oldState)

	// Enable bracketed paste mode
	fmt.Fprint(cfg.output, "\033[?2004h")
	defer fmt.Fprint(cfg.output, "\033[?2004l")

	// Enable enhanced keyboard mode for Shift+Enter detection
	fmt.Fprint(cfg.output, "\033[>1u")
	defer fmt.Fprint(cfg.output, "\033[<u")

	// Initialize state
	state := &promptState{
		config:     cfg,
		prompt:     prompt,
		input:      []rune{},
		cursorPos:  0,
		historyIdx: -1,
	}

	// Create live printer for the input region
	lp := NewLivePrinter(PrintConfig{Width: 80, Output: cfg.output})

	// Render initial view
	lp.Update(state.buildView())

	// Input loop
	decoder := terminal.NewKeyDecoder(os.Stdin)
	for {
		event, err := decoder.ReadKeyEvent()
		if err != nil {
			lp.Clear()
			if err == io.EOF {
				return "", ErrEOF
			}
			return "", err
		}

		// Handle the key event
		result, done, keyErr := state.handleKey(event, prompt)
		if done {
			lp.Clear()
			// Print final prompt + input on its own line
			fmt.Fprint(cfg.output, state.config.promptStyle.String())
			fmt.Fprint(cfg.output, prompt)
			fmt.Fprint(cfg.output, "\033[0m")
			fmt.Fprintln(cfg.output, string(state.input))
			return result, keyErr
		}

		// Re-render
		lp.Update(state.buildView())
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

// triggerAutocomplete activates autocomplete
func (s *promptState) triggerAutocomplete() {
	completions, from := s.config.autocomplete(string(s.input), s.cursorPos)
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
	completions, from := s.config.autocomplete(string(s.input), s.cursorPos)

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

	completion := s.acMatches[s.acSelected]

	// Replace from acFrom to cursor with completion
	before := s.input[:s.acFrom]
	after := s.input[s.cursorPos:]
	s.input = append(before, append([]rune(completion), after...)...)
	s.cursorPos = len(before) + len([]rune(completion))

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

	// Build the input region with dividers
	inputRegion := Stack(
		Divider(),
		inputContent,
		Divider(),
	)

	// If no autocomplete, just return input region
	if !s.acActive || len(s.acMatches) == 0 {
		return inputRegion
	}

	// Build autocomplete dropdown
	maxShow := 5
	items := []View{}

	for i := 0; i < len(s.acMatches) && i < maxShow; i++ {
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
	if s.config.mask != "" {
		return strings.Repeat(s.config.mask, len(s.input))
	}
	return string(s.input)
}

// readLineFallback is a simple fallback for non-terminal input
func readLineFallback() (string, error) {
	var line strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
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
