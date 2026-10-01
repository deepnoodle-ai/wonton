package tui

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestInputPastePreservesNativeInsertion(t *testing.T) {
	for _, decision := range []PasteHandlerDecision{PasteReject, PasteModified} {
		t.Run(fmt.Sprint(decision), func(t *testing.T) {
			text := "前 draft 後"
			changes, calls := 0, 0
			state := newTestInputState(t, inputConfig{
				binding:  &text,
				onChange: func(string) { changes++ },
				onPaste: func(info PasteInfo) (PasteHandlerDecision, string) {
					calls++
					if info.Content != "世界\n" || info.ByteCount != 7 || info.LineCount != 2 {
						t.Fatalf("incorrect paste info: %+v", info)
					}
					return decision, ""
				},
			})
			state.input.CursorPos = len("前 ")
			if !state.HandleKeyEvent(KeyEvent{Paste: "世界\n"}) {
				t.Fatal("consumed paste bubbled to the application")
			}
			if text != "前 draft 後" || state.input.CursorPos != len("前 ") || changes != 0 || calls != 1 {
				t.Fatalf("consumption changed draft/cursor: %q, %d, changes=%d, calls=%d", text, state.input.CursorPos, changes, calls)
			}
			state.HandleKeyEvent(KeyEvent{Rune: '✓'})
			if text != "前 ✓draft 後" || changes != 1 || calls != 1 {
				t.Fatalf("subsequent typing lost native insertion: %q", text)
			}
		})
	}
}

type pasteHookApp struct {
	text           string
	field          bool
	calls, bubbled int
}

func (a *pasteHookApp) View() View {
	hook := func(info PasteInfo) (PasteHandlerDecision, string) {
		a.calls++
		return PasteModified, strings.ToUpper(info.Content)
	}
	if a.field {
		return InputField(&a.text).ID("paste").OnPaste(hook)
	}
	return Input(&a.text).ID("paste").OnPaste(hook)
}

func (a *pasteHookApp) HandleEvent(e Event) []Cmd {
	if key, ok := e.(KeyEvent); ok && key.Paste != "" {
		a.bubbled++
	}
	return nil
}

func TestPasteHookThroughFocusedViews(t *testing.T) {
	for _, field := range []bool{true, false} {
		t.Run(fmt.Sprint(field), func(t *testing.T) {
			a := &pasteHookApp{text: "前 後", field: field}
			r := NewRuntime(NewTestTerminal(80, 24, &bytes.Buffer{}), a, 30)
			if err := r.renderChecked(); err != nil {
				t.Fatal(err)
			}
			r.focusMgr.GetFocused().(*inputState).input.CursorPos = len("前 ")
			r.processEvent(KeyEvent{Paste: "hello"})
			if a.text != "前 HELLO後" || a.calls != 1 || a.bubbled != 0 {
				t.Fatalf("focused paste did not preserve surrounding text: %+v", a)
			}
		})
	}
}

func ExampleInputFieldView_OnPaste() {
	var draft string
	handler := func(info PasteInfo) (PasteHandlerDecision, string) {
		if info.ByteCount > 1024 {
			return PasteReject, ""
		}
		return PasteModified, strings.TrimSpace(info.Content)
	}
	// Return this field from the application's View to handle real paste.
	field := InputField(&draft).OnPaste(handler)
	_ = field
	decision, text := handler(PasteInfo{Content: "  hello  ", ByteCount: 9, LineCount: 1})
	fmt.Println(decision == PasteModified, text)
	// Output: true hello
}
