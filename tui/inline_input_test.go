package tui

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
)

// Key bytes as a terminal in raw mode sends them.
const (
	keyEnter     = "\r"
	keyBackspace = "\x7f"
	keyLeft      = "\x1b[D"
	keyUp        = "\x1b[A"
	keyHome      = "\x01" // Ctrl+A
	keyCtrlC     = "\x03"
	keyCtrlD     = "\x04"
	keyCtrlJ     = "\n"
)

// promptKeys runs Prompt on the given key bytes and returns its result.
func promptKeys(t *testing.T, keys string, opts ...PromptOption) (string, error) {
	t.Helper()
	var out bytes.Buffer
	opts = append([]PromptOption{
		WithPromptInput(strings.NewReader(keys)),
		WithPromptOutput(&out),
	}, opts...)
	return Prompt("> ", opts...)
}

// snapshotReader calls snap the first time the reader reaches it, then
// reports EOF so an io.MultiReader moves on. Prompt renders each key before
// it reads the next one, so snap sees the live region after the keys before
// it.
type snapshotReader struct {
	snap func()
	done bool
}

func (r *snapshotReader) Read([]byte) (int, error) {
	if !r.done {
		r.done = true
		r.snap()
	}
	return 0, io.EOF
}

// livePromptRows returns the rows of the live region that Prompt draws for
// the given keys on a screen wider than the prompt.
func livePromptRows(t *testing.T, keys string, opts ...PromptOption) []string {
	t.Helper()
	screen := termtest.NewScreen(120, 10)
	var rows []string
	in := io.MultiReader(
		strings.NewReader(keys),
		&snapshotReader{snap: func() {
			rows = strings.Split(strings.TrimRight(screen.Text(), "\n"), "\n")
		}},
		strings.NewReader(keyEnter),
	)
	opts = append([]PromptOption{WithPromptInput(in), WithPromptOutput(screen)}, opts...)
	_, err := Prompt("> ", opts...)
	assert.NoError(t, err)
	return rows
}

func TestPrompt_TypeAndSubmit(t *testing.T) {
	var out bytes.Buffer
	got, err := Prompt("> ",
		WithPromptInput(strings.NewReader("hello world"+keyEnter)),
		WithPromptOutput(&out),
	)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", got)
	// The submitted line stays in the output after the live region clears.
	assert.True(t, strings.HasSuffix(out.String(), "hello world\n\x1b[?25h"))
}

func TestPrompt_EditingKeys(t *testing.T) {
	tests := []struct {
		name string
		keys string
		want string
	}{
		{"backspace", "hellp" + keyBackspace + "o", "hello"},
		{"left arrow", "helo" + keyLeft + "l", "hello"},
		{"home", "ello" + keyHome + "h", "hello"},
		{"ctrl+d deletes at cursor", "hello!" + keyLeft + keyCtrlD, "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := promptKeys(t, tt.keys+keyEnter)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPrompt_HistoryAcrossPrompts(t *testing.T) {
	// One reader feeds three prompts: each prompt must leave the bytes
	// after its Enter for the next one.
	in := strings.NewReader("first" + keyEnter + "second" + keyEnter + keyUp + keyUp + keyEnter)
	var history []string
	var got []string
	for range 3 {
		line, err := Prompt("> ",
			WithPromptInput(in),
			WithPromptOutput(io.Discard),
			WithHistory(&history),
		)
		assert.NoError(t, err)
		got = append(got, line)
	}
	assert.Equal(t, []string{"first", "second", "first"}, got)
	assert.Equal(t, []string{"first", "second", "first"}, history)
}

func TestPrompt_MultiLineInput(t *testing.T) {
	got, err := promptKeys(t, "one"+keyCtrlJ+"two"+keyEnter, WithMultiLine(true))
	assert.NoError(t, err)
	assert.Equal(t, "one\ntwo", got)
}

func TestPrompt_Autocomplete(t *testing.T) {
	complete := func(input string, cursor int) ([]string, int) {
		at := strings.LastIndex(input[:cursor], "@")
		if at < 0 {
			return nil, 0
		}
		return []string{"main.go"}, at + 1
	}
	got, err := promptKeys(t, "open @ma\t"+keyEnter, WithAutocomplete(complete))
	assert.NoError(t, err)
	assert.Equal(t, "open @main.go", got)
}

func TestPrompt_EndErrors(t *testing.T) {
	tests := []struct {
		name string
		keys string
		want error
	}{
		{"ctrl+c", "abc" + keyCtrlC, ErrInterrupted},
		{"ctrl+d on empty input", keyCtrlD, ErrEOF},
		{"input ends before enter", "abc", ErrEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := promptKeys(t, tt.keys)
			assert.True(t, errors.Is(err, tt.want), "got error %v", err)
			assert.Equal(t, "", got)
		})
	}
}

func TestPrompt_Frame(t *testing.T) {
	t.Run("on by default", func(t *testing.T) {
		rows := livePromptRows(t, "hi")
		assert.Equal(t, strings.Repeat("─", 80), rows[0])
		assert.Equal(t, "> hi", rows[1])
		assert.Equal(t, strings.Repeat("─", 80), rows[2])
	})
	t.Run("off", func(t *testing.T) {
		rows := livePromptRows(t, "hi", WithPromptFrame(false))
		assert.Equal(t, []string{"> hi"}, rows)
	})
}

func TestPrompt_Width(t *testing.T) {
	t.Run("80 columns when no stream is a terminal", func(t *testing.T) {
		width, fromTerminal := promptWidth(&bytes.Buffer{}, strings.NewReader(""))
		assert.Equal(t, 80, width)
		assert.False(t, fromTerminal)
	})
	t.Run("long input stays within the width", func(t *testing.T) {
		rows := livePromptRows(t, strings.Repeat("a", 100), WithPromptFrame(false))
		assert.Equal(t, 1, len(rows))
		assert.Equal(t, "> "+strings.Repeat("a", 78), rows[0])
	})
	t.Run("a resize redraws the region at the new width", func(t *testing.T) {
		screen := termtest.NewScreen(120, 10)
		state := &promptState{config: defaultPromptConfig(), prompt: "> ", input: []rune("hi"), cursorPos: 2, historyIdx: -1}
		region := &promptRegion{state: state, lp: NewLivePrinter(WithWidth(80), WithOutput(screen))}
		region.lp.Update(state.buildView())
		rows := strings.Split(strings.TrimRight(screen.Text(), "\n"), "\n")
		assert.Equal(t, strings.Repeat("─", 80), rows[0])

		region.resize(40)
		rows = strings.Split(strings.TrimRight(screen.Text(), "\n"), "\n")
		assert.Equal(t, []string{strings.Repeat("─", 40), "> hi", strings.Repeat("─", 40)}, rows)
	})
}

func TestReadLineFallback(t *testing.T) {
	in := strings.NewReader("one\ntwo")
	got, err := readLineFallback(in)
	assert.NoError(t, err)
	assert.Equal(t, "one", got)
	got, err = readLineFallback(in)
	assert.NoError(t, err)
	assert.Equal(t, "two", got)
	_, err = readLineFallback(in)
	assert.True(t, errors.Is(err, io.EOF))
}

// Test the autocomplete function signature
func TestAutocompleteFunc(t *testing.T) {
	fn := func(input string, cursorPos int) ([]string, int) {
		// Find @ before cursor
		atIdx := strings.LastIndex(input[:cursorPos], "@")
		if atIdx == -1 {
			return nil, 0
		}
		prefix := input[atIdx+1 : cursorPos]

		// Mock file matches
		files := []string{"app.go", "app_test.go", "application.go"}
		var matches []string
		for _, f := range files {
			if strings.HasPrefix(f, prefix) {
				matches = append(matches, f)
			}
		}
		return matches, atIdx
	}

	// Test with @ prefix
	completions, from := fn("hello @ap", 9)
	assert.Equal(t, 3, len(completions))
	assert.Equal(t, 6, from) // Position of @
	assert.Equal(t, "app.go", completions[0])

	// Test without @ prefix
	completions, from = fn("hello", 5)
	assert.Equal(t, 0, len(completions))
}

// Test prompt state operations
func TestPromptState_InsertRune(t *testing.T) {
	cfg := defaultPromptConfig()
	state := &promptState{
		config:    cfg,
		input:     []rune("hello"),
		cursorPos: 5,
	}

	state.insertRune('!')
	assert.Equal(t, "hello!", string(state.input))
	assert.Equal(t, 6, state.cursorPos)

	// Insert in middle
	state.cursorPos = 2
	state.insertRune('X')
	assert.Equal(t, "heXllo!", string(state.input))
	assert.Equal(t, 3, state.cursorPos)
}

func TestPromptState_Backspace(t *testing.T) {
	cfg := defaultPromptConfig()
	state := &promptState{
		config:    cfg,
		input:     []rune("hello"),
		cursorPos: 5,
	}

	state.backspace()
	assert.Equal(t, "hell", string(state.input))
	assert.Equal(t, 4, state.cursorPos)

	// Backspace in middle
	state.cursorPos = 2
	state.backspace()
	assert.Equal(t, "hll", string(state.input))
	assert.Equal(t, 1, state.cursorPos)

	// Backspace at start (no-op)
	state.cursorPos = 0
	state.backspace()
	assert.Equal(t, "hll", string(state.input))
	assert.Equal(t, 0, state.cursorPos)
}

func TestPromptState_DeleteChar(t *testing.T) {
	cfg := defaultPromptConfig()
	state := &promptState{
		config:    cfg,
		input:     []rune("hello"),
		cursorPos: 0,
	}

	state.deleteChar()
	assert.Equal(t, "ello", string(state.input))
	assert.Equal(t, 0, state.cursorPos)

	// Delete in middle
	state.cursorPos = 1
	state.deleteChar()
	assert.Equal(t, "elo", string(state.input))
	assert.Equal(t, 1, state.cursorPos)

	// Delete at end (no-op)
	state.cursorPos = 3
	state.deleteChar()
	assert.Equal(t, "elo", string(state.input))
	assert.Equal(t, 3, state.cursorPos)
}

func TestPromptState_DeleteWordBackward(t *testing.T) {
	cfg := defaultPromptConfig()
	state := &promptState{
		config:    cfg,
		input:     []rune("hello world test"),
		cursorPos: 16,
	}

	state.deleteWordBackward()
	assert.Equal(t, "hello world ", string(state.input))
	assert.Equal(t, 12, state.cursorPos)

	state.deleteWordBackward()
	assert.Equal(t, "hello ", string(state.input))
	assert.Equal(t, 6, state.cursorPos)

	state.deleteWordBackward()
	assert.Equal(t, "", string(state.input))
	assert.Equal(t, 0, state.cursorPos)
}

func TestPromptState_History(t *testing.T) {
	history := []string{"first", "second", "third"}
	cfg := defaultPromptConfig()
	cfg.history = &history

	state := &promptState{
		config:     cfg,
		input:      []rune("current"),
		cursorPos:  7,
		historyIdx: -1,
	}

	// Go up once
	state.historyUp()
	assert.Equal(t, "third", string(state.input))
	assert.Equal(t, 2, state.historyIdx)
	assert.Equal(t, "current", string(state.savedInput))

	// Go up again
	state.historyUp()
	assert.Equal(t, "second", string(state.input))
	assert.Equal(t, 1, state.historyIdx)

	// Go up to first
	state.historyUp()
	assert.Equal(t, "first", string(state.input))
	assert.Equal(t, 0, state.historyIdx)

	// Can't go further up
	state.historyUp()
	assert.Equal(t, "first", string(state.input))
	assert.Equal(t, 0, state.historyIdx)

	// Go down
	state.historyDown()
	assert.Equal(t, "second", string(state.input))
	assert.Equal(t, 1, state.historyIdx)

	// Go to end (restore saved)
	state.historyDown()
	state.historyDown()
	assert.Equal(t, "current", string(state.input))
	assert.Equal(t, -1, state.historyIdx)
}

func TestPromptState_Autocomplete(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.autocomplete = func(input string, cursorPos int) ([]string, int) {
		return []string{"apple", "application", "apply"}, 0
	}

	state := &promptState{
		config:    cfg,
		input:     []rune("ap"),
		cursorPos: 2,
	}

	// Trigger autocomplete
	state.triggerAutocomplete()
	assert.Equal(t, true, state.acActive)
	assert.Equal(t, 3, len(state.acMatches))
	assert.Equal(t, 0, state.acSelected)

	// Apply first match
	state.applyAutocomplete()
	assert.Equal(t, "apple", string(state.input))
	assert.Equal(t, false, state.acActive)
}

func TestPromptState_MaxLength(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.maxLength = 5

	state := &promptState{
		config:    cfg,
		input:     []rune("hello"),
		cursorPos: 5,
	}

	// Can't insert more
	state.insertRune('!')
	assert.Equal(t, "hello", string(state.input))
	assert.Equal(t, 5, state.cursorPos)
}

func TestPromptState_Mask(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.mask = "*"

	state := &promptState{
		config:    cfg,
		input:     []rune("password"),
		cursorPos: 8,
	}

	display := state.getDisplayInput()
	assert.Equal(t, "********", display)
}

func TestPromptState_Validator(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.validator = func(s string) error {
		if len(s) < 3 {
			return errors.New("too short")
		}
		return nil
	}

	state := &promptState{
		config:    cfg,
		input:     []rune("ab"),
		cursorPos: 2,
	}

	// Validator should prevent submission
	err := cfg.validator(string(state.input))
	assert.NotNil(t, err)

	state.input = []rune("abc")
	err = cfg.validator(string(state.input))
	assert.Nil(t, err)
}

func TestPromptState_AutocompleteRespectsMaxLength(t *testing.T) {
	cfg := defaultPromptConfig()
	cfg.maxLength = 4
	state := &promptState{
		config:    cfg,
		input:     []rune("ap"),
		cursorPos: 2,
		acActive:  true,
		acMatches: []string{"application"},
	}
	state.applyAutocomplete()
	assert.Equal(t, string(state.input), "appl")
	assert.Equal(t, state.cursorPos, 4)
}

func TestPromptState_EmptyMaskKeepsCursor(t *testing.T) {
	cfg := defaultPromptConfig()
	WithMask("")(&cfg)
	state := &promptState{config: cfg, prompt: "> ", input: []rune("secret"), cursorPos: 6}
	out := Sprint(state.buildView(), WithWidth(20), WithColor(true))
	assert.Contains(t, out, "\x1b[0;7m \x1b[0m", "the reverse-video cursor is drawn")
	assert.NotContains(t, out, "secret")
}

func TestPromptState_AutocompleteShowsSelectionPastFifth(t *testing.T) {
	state := &promptState{
		config:     defaultPromptConfig(),
		prompt:     "> ",
		acActive:   true,
		acMatches:  []string{"m0", "m1", "m2", "m3", "m4", "m5", "m6"},
		acSelected: 6,
	}
	out := Sprint(state.buildView(), WithWidth(20))
	assert.Contains(t, out, "→ m6")
	assert.NotContains(t, out, "m0")
}
