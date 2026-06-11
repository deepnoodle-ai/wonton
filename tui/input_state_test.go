package tui

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// newTestInputState registers an input with the given config and returns its
// state, bypassing rendering.
func newTestInputState(t *testing.T, cfg inputConfig) *inputState {
	t.Helper()
	reg := &inputRegistryImpl{inputs: make(map[string]*inputState)}
	state := reg.Register("test-input", cfg, nil)
	// The focus manager only delivers keys to a focused element; the
	// underlying textInput also gates HandleKey on focus.
	state.SetFocused(true)
	return state
}

func typeString(s *inputState, text string) {
	for _, r := range text {
		s.HandleKeyEvent(KeyEvent{Rune: r})
	}
}

// --- OnKey hook ---

func TestInputState_OnKey_InterceptsBeforeEditing(t *testing.T) {
	var text string
	var seen []KeyEvent
	state := newTestInputState(t, inputConfig{
		binding: &text,
		onKey: func(e KeyEvent) bool {
			seen = append(seen, e)
			return e.Rune == 'x' // claim only 'x'
		},
	})

	// Claimed key is consumed and never edits the text.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Rune: 'x'}))
	assert.Equal(t, "", text)

	// Unclaimed key falls through to normal editing.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Rune: 'a'}))
	assert.Equal(t, "a", text)

	assert.Equal(t, 2, len(seen), "OnKey should see every key")
}

func TestInputState_OnKey_SeesEnterBeforeSubmit(t *testing.T) {
	var text string
	submitted := false
	state := newTestInputState(t, inputConfig{
		binding:  &text,
		onSubmit: func(string) { submitted = true },
		onKey: func(e KeyEvent) bool {
			return e.Key == KeyEnter // claim Enter
		},
	})

	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyEnter}))
	assert.False(t, submitted, "OnKey claimed Enter; OnSubmit must not fire")
}

func TestInputState_OnKey_DoesNotSeePaste(t *testing.T) {
	var text string
	hookCalled := false
	state := newTestInputState(t, inputConfig{
		binding: &text,
		onKey:   func(KeyEvent) bool { hookCalled = true; return true },
	})

	state.HandleKeyEvent(KeyEvent{Paste: "pasted"})
	assert.False(t, hookCalled, "paste events bypass OnKey")
	assert.Equal(t, "pasted", text)
}

// --- Multiline arrow boundary bubbling ---

func TestInputState_MultilineArrows_BubbleAtBoundary(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{binding: &text, multiline: true})

	typeString(state, "line1")
	state.HandleKeyEvent(KeyEvent{Key: KeyEnter, Shift: true})
	typeString(state, "line2")
	assert.Equal(t, "line1\nline2", text)

	// Cursor is at the end (last line): Down can't move, so it bubbles.
	assert.False(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))

	// Up moves the cursor to line1 and is consumed.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))

	// Now on the first line: Up can't move, so it bubbles.
	assert.False(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))

	// Down moves back to line2 and is consumed.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))
}

func TestInputState_SingleLineArrows_Bubble(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{binding: &text})
	typeString(state, "hello")

	assert.False(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.False(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))
}

// --- History ---

func TestInputState_History_RecallAndRestore(t *testing.T) {
	var text string
	history := []string{"first", "second", "third"}
	state := newTestInputState(t, inputConfig{binding: &text, history: history})

	typeString(state, "draft")

	// Up walks backward from the newest entry.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, "third", text)
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, "second", text)
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, "first", text)

	// Up at the oldest entry is consumed but does nothing.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, "first", text)

	// Down walks forward and finally restores the draft.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))
	assert.Equal(t, "second", text)
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))
	assert.Equal(t, "third", text)
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))
	assert.Equal(t, "draft", text)

	// Back on the live draft: Down bubbles.
	assert.False(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))
}

func TestInputState_History_SubmitResets(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding:  &text,
		history:  []string{"old"},
		onSubmit: func(string) {},
	})

	state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, "old", text)

	state.HandleKeyEvent(KeyEvent{Key: KeyEnter})

	// After submit, recall starts from the newest entry again.
	state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, "old", text)
	assert.Equal(t, 0, state.historyIdx)
}

func TestInputState_History_MultilineNavigatesTextFirst(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding:   &text,
		multiline: true,
		history:   []string{"previous command"},
	})

	typeString(state, "line1")
	state.HandleKeyEvent(KeyEvent{Key: KeyEnter, Shift: true})
	typeString(state, "line2")

	// Cursor on the last line: Up moves the cursor, not history.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, "line1\nline2", text)

	// Cursor on the first line: Up now recalls history.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, "previous command", text)

	// Down past the newest entry restores the multiline draft.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))
	assert.Equal(t, "line1\nline2", text)
}

func TestInputState_History_OnChangeFiresOnRecall(t *testing.T) {
	var text string
	var changes []string
	state := newTestInputState(t, inputConfig{
		binding:  &text,
		history:  []string{"recalled"},
		onChange: func(v string) { changes = append(changes, v) },
	})

	state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, []string{"recalled"}, changes)
}

func TestInputState_History_ShrunkSliceResetsIndex(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding: &text,
		history: []string{"a", "b", "c"},
	})

	state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, "c", text)

	// Simulate the app replacing history with a shorter slice on re-render.
	state.history = []string{"a"}
	state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, "a", text)
}

// --- Completion ---

func TestInputState_Completion_SingleCandidateAccepted(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding: &text,
		onComplete: func(value string) []string {
			assert.Equal(t, "/he", value)
			return []string{"/help"}
		},
	})

	typeString(state, "/he")
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyTab}))
	assert.Equal(t, "/help", text)
	assert.True(t, state.completions == nil, "single candidate should not enter cycling mode")
}

func TestInputState_Completion_CyclesCandidates(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding: &text,
		onComplete: func(string) []string {
			return []string{"/help", "/hello", "/health"}
		},
	})

	typeString(state, "/he")

	// First Tab shows the first candidate.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyTab}))
	assert.Equal(t, "/help", text)

	// Tab and Down advance; Shift+Tab and Up go back.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyTab}))
	assert.Equal(t, "/hello", text)
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown}))
	assert.Equal(t, "/health", text)
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyTab}))
	assert.Equal(t, "/help", text, "cycling wraps around")
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyTab, Shift: true}))
	assert.Equal(t, "/health", text)
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, "/hello", text)
}

func TestInputState_Completion_EscRestoresOriginal(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding:    &text,
		onComplete: func(string) []string { return []string{"/help", "/hello"} },
	})

	typeString(state, "/he")
	state.HandleKeyEvent(KeyEvent{Key: KeyTab})
	assert.Equal(t, "/help", text)

	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyEscape}))
	assert.Equal(t, "/he", text)
	assert.True(t, state.completions == nil)
}

func TestInputState_Completion_TypingAcceptsCandidate(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding:    &text,
		onComplete: func(string) []string { return []string{"/help", "/hello"} },
	})

	typeString(state, "/he")
	state.HandleKeyEvent(KeyEvent{Key: KeyTab})
	assert.Equal(t, "/help", text)

	// A normal key accepts the shown candidate and is applied on top.
	state.HandleKeyEvent(KeyEvent{Rune: ' '})
	assert.Equal(t, "/help ", text)
	assert.True(t, state.completions == nil, "typing exits cycling mode")
}

func TestInputState_Completion_EnterAcceptsAndSubmits(t *testing.T) {
	var text string
	var submitted string
	state := newTestInputState(t, inputConfig{
		binding:    &text,
		onComplete: func(string) []string { return []string{"/help", "/hello"} },
		onSubmit:   func(v string) { submitted = v },
	})

	typeString(state, "/he")
	state.HandleKeyEvent(KeyEvent{Key: KeyTab})
	state.HandleKeyEvent(KeyEvent{Key: KeyTab})
	assert.Equal(t, "/hello", text)

	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyEnter}))
	assert.Equal(t, "/hello", submitted)
	assert.True(t, state.completions == nil)
}

func TestInputState_Completion_NoCandidatesStillClaimsTab(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding:    &text,
		onComplete: func(string) []string { return nil },
	})

	typeString(state, "zzz")
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyTab}))
	assert.Equal(t, "zzz", text)
}

func TestInputState_Completion_TabIgnoredWithoutOnComplete(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{binding: &text})

	// Without OnComplete the field doesn't use Tab, so it propagates
	// (allowing focus traversal).
	assert.False(t, state.HandleKeyEvent(KeyEvent{Key: KeyTab}))
}

// --- End-to-end key routing through InlineApp ---

// replScenarioApp models a REPL-style CLI (like Dive's): a single focused
// multiline InputField, app-level handling for keys the input doesn't use,
// and an OnKey hook claiming Tab.
type replScenarioApp struct {
	input   string
	history []string
	appKeys []KeyEvent
	tabbed  bool
}

func (a *replScenarioApp) LiveView() View {
	return InputField(&a.input).
		ID("main-input").
		Multiline(true).
		History(a.history).
		OnKey(func(e KeyEvent) bool {
			if e.Key == KeyTab {
				a.tabbed = true
				return true
			}
			return false
		})
}

func (a *replScenarioApp) HandleEvent(event Event) []Cmd {
	if k, ok := event.(KeyEvent); ok {
		a.appKeys = append(a.appKeys, k)
	}
	return nil
}

func TestInlineApp_KeyRouting_ReplScenario(t *testing.T) {
	var buf bytes.Buffer
	runner := NewInlineApp(WithInlineOutput(&buf), WithInlineWidth(80))

	app := &replScenarioApp{}
	runner.app = app
	runner.live = NewLivePrinter(WithWidth(80), WithOutput(&buf))

	// First render registers and auto-focuses the input.
	runner.render()

	// Typing goes to the focused input and is NOT delivered to the app.
	runner.processEvent(KeyEvent{Rune: 'h'})
	assert.Equal(t, "h", app.input)
	assert.Equal(t, 0, len(app.appKeys))

	// ArrowUp with no history and nowhere for the cursor to go: the input
	// doesn't use it, so it propagates to the app's HandleEvent.
	runner.processEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, 1, len(app.appKeys))
	assert.Equal(t, KeyArrowUp, app.appKeys[0].Key)

	// Tab is delegated to the focused input first, where OnKey claims it.
	runner.processEvent(KeyEvent{Key: KeyTab})
	assert.True(t, app.tabbed)
	assert.Equal(t, 1, len(app.appKeys), "claimed Tab must not reach the app")

	// With History configured, the input consumes ArrowUp to recall.
	app.history = []string{"previous command"}
	runner.render()
	runner.processEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, "previous command", app.input)
	assert.Equal(t, 1, len(app.appKeys), "recall must not reach the app")
}

func TestInputState_Completion_ArrowsCycleBeforeHistory(t *testing.T) {
	var text string
	state := newTestInputState(t, inputConfig{
		binding:    &text,
		history:    []string{"old command"},
		onComplete: func(string) []string { return []string{"/help", "/hello"} },
	})

	typeString(state, "/he")
	state.HandleKeyEvent(KeyEvent{Key: KeyTab})

	// While cycling candidates, Up cycles instead of recalling history.
	assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp}))
	assert.Equal(t, "/hello", text)
}
