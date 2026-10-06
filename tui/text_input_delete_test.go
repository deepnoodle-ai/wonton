package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTextInputKillLine(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		key                 Key
		cursor, wantCursor  int
		multiline           bool
	}{
		{"beginning", "abc def", " def", KeyCtrlU, 3, 0, false},
		{"end", "abc def", "abc", KeyCtrlK, 3, 3, false},
		{"unicode beginning", "前e\u0301👩🏽‍💻後", "後", KeyCtrlU, len("前e\u0301👩🏽‍💻"), 0, false},
		{"unicode end", "前e\u0301👩🏽‍💻後", "前", KeyCtrlK, len("前"), len("前"), false},
		{"logical beginning", "first\n前e\u0301👩🏽‍💻後\nlast", "first\n後\nlast", KeyCtrlU, len("first\n前e\u0301👩🏽‍💻"), len("first\n"), true},
		{"logical end", "first\n前e\u0301👩🏽‍💻後\nlast", "first\n前\nlast", KeyCtrlK, len("first\n前"), len("first\n前"), true},
		{"at newline", "first\nlast", "first\nlast", KeyCtrlK, 5, 5, true},
		{"at line start", "first\nlast", "first\nlast", KeyCtrlU, 6, 6, true},
		{"single line removes newline", "first\nlast", "fi", KeyCtrlK, 2, 2, false},
		{"CRLF is one cluster", "a\r\nb\nc", "a\nc", KeyCtrlK, 1, 1, true},
		{"empty", "", "", KeyCtrlU, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := newTextInput().WithMultilineMode(tc.multiline)
			input.SetFocused(true)
			input.SetValue(tc.before)
			input.CursorPos = tc.cursor
			assert.True(t, input.HandleKey(KeyEvent{Key: tc.key}))
			assert.Equal(t, tc.after, input.Value())
			assert.Equal(t, tc.wantCursor, input.CursorPos)
		})
	}
}

func BenchmarkTextInputKillLine(b *testing.B) {
	for _, size := range []int{8192, 16384, 65536, 262144} {
		for _, key := range []Key{KeyCtrlU, KeyCtrlK} {
			b.Run(fmt.Sprintf("%v/%d", key, size), func(b *testing.B) {
				value := "prefix" + strings.Repeat("x", size) + "suffix"
				input := newTextInput().WithMultilineMode(true)
				input.SetFocused(true)
				b.ReportAllocs()
				for b.Loop() {
					input.SetValue(value)
					if key == KeyCtrlK {
						input.CursorPos = len("prefix")
					} else {
						input.CursorPos = len(value) - len("suffix")
					}
					input.HandleKey(KeyEvent{Key: key})
					if len(input.Value()) != len("prefix") {
						b.Fatal("selected range was not cleared")
					}
				}
			})
		}
	}
}

func TestTextInputKillLinePasteSegments(t *testing.T) {
	for _, key := range []Key{KeyCtrlU, KeyCtrlK} {
		for _, inside := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/inside=%t", key, inside), func(t *testing.T) {
				input := newTextInput().WithMultilineMode(true).WithPastePlaceholderMode(true)
				input.SetFocused(true)
				input.SetValue("keep\n前")
				input.HandlePaste("first\npaste")
				input.insertAtCursor("中")
				input.HandlePaste("second\npaste")
				input.insertAtCursor("後\nlast")
				beforeFirst := len("keep\n前")
				afterSecond := len("keep\n前[pasted 2 lines]中[pasted 2 lines]")
				want, cursor := "keep\n後\nlast", len("keep\n")
				input.CursorPos = afterSecond
				if key == KeyCtrlK {
					input.CursorPos = beforeFirst
					want, cursor = "keep\n前\nlast", beforeFirst
				}
				if inside {
					if key == KeyCtrlU {
						input.CursorPos -= 3
					} else {
						input.CursorPos += 3
					}
				}
				changes := 0
				input.OnChange = func(value string) {
					changes++
					assert.Equal(t, want, value)
				}
				input.HandleKey(KeyEvent{Key: key})
				assert.Equal(t, want, input.Value())
				assert.Equal(t, want, input.DisplayText())
				assert.Equal(t, cursor, input.CursorPos)
				assert.Equal(t, 1, len(input.segments))
				assert.Equal(t, 1, changes)
			})
		}
	}
}

func TestTextInputKillLinePreservesUntouchedPaste(t *testing.T) {
	input := newTextInput().WithMultilineMode(true).WithPastePlaceholderMode(true)
	input.SetFocused(true)
	input.HandlePaste("kept\npaste")
	input.insertAtCursor("\nremove suffix")
	input.CursorPos = len("[pasted 2 lines]\nremove")
	input.HandleKey(KeyEvent{Key: KeyCtrlU})
	assert.Equal(t, "kept\npaste\n suffix", input.Value())
	assert.Equal(t, "[pasted 2 lines]\n suffix", input.DisplayText())
	assert.Equal(t, len("[pasted 2 lines]\n"), input.CursorPos)
	assert.True(t, input.segments[0].isPaste)
}

func TestInputStateKillLargePastedLine(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		for _, key := range []Key{KeyCtrlU, KeyCtrlK} {
			t.Run(fmt.Sprintf("%v/multiline=%t", key, multiline), func(t *testing.T) {
				var text string
				changes := 0
				state := newTestInputState(t, inputConfig{
					binding: &text, multiline: multiline,
					onChange: func(string) { changes++ },
				})
				payload := strings.Repeat("x", 262144)
				state.HandleKeyEvent(KeyEvent{Paste: "前" + payload + "後"})
				want := "後"
				state.input.CursorPos = len("前" + payload)
				if key == KeyCtrlK {
					state.input.CursorPos = len("前")
					want = "前"
				}
				assert.True(t, state.HandleKeyEvent(KeyEvent{Key: key}))
				assert.Equal(t, want, text)
				assert.Equal(t, 2, changes, "paste and range deletion notify once each")
				state.HandleKeyEvent(KeyEvent{Rune: '✓'})
				if key == KeyCtrlU {
					want = "✓" + want
				} else {
					want += "✓"
				}
				assert.Equal(t, want, text)
			})
		}
	}
}

func TestInputStateKillLineCompletionAndHistory(t *testing.T) {
	text := "draft"
	changes := 0
	state := newTestInputState(t, inputConfig{
		binding:    &text,
		onChange:   func(string) { changes++ },
		onComplete: func(string) []string { return []string{"completed draft", "other"} },
		history:    []string{"previous entry"},
	})
	state.HandleKeyEvent(KeyEvent{Key: KeyTab})
	state.input.CursorPos = len("completed")
	state.HandleKeyEvent(KeyEvent{Key: KeyCtrlK})
	assert.Equal(t, "completed", text)
	assert.True(t, state.completions == nil)
	assert.Equal(t, 2, changes)
	state.HandleKeyEvent(KeyEvent{Key: KeyArrowUp})
	assert.Equal(t, "previous entry", text)
	state.input.CursorPos = len("previous")
	state.HandleKeyEvent(KeyEvent{Key: KeyCtrlU})
	assert.Equal(t, " entry", text)
	state.HandleKeyEvent(KeyEvent{Key: KeyArrowDown})
	assert.Equal(t, "completed", text, "history restores the edited live draft")
	assert.Equal(t, 5, changes)
}

func TestTextInputKillWord(t *testing.T) {
	for _, tc := range []struct {
		name, prefix, suffix, wantPrefix string
	}{
		{"word", "keep word", "", "keep "},
		{"middle", "keep word", "suffix", "keep "},
		{"whitespace", "keep word \t\n", "suffix", "keep "},
		{"punctuation", "keep word!?...", "suffix", "keep "},
		{"underscore and digits", "keep word_123", "!", "keep "},
		{"punctuation before word", "keep.word", "!", "keep."},
		{"non-ASCII suffix", "keep word世界👩🏽‍💻", "後", "keep "},
		{"non-ASCII separator", "keep世界word", "後", "keep世界"},
		{"combining suffix", "keep cafe\u0301", "後", "keep "},
		{"combining separator", "keep cafe\u0301word", "後", "keep cafe\u0301"},
		{"only non-word", "世界👩🏽‍💻 \t\n", "後", ""},
		{"multiline word", "first\nword", "\nlast", "first\n"},
		{"cross newline", "first\n", "last", ""},
		{"empty", "", "unchanged", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := newTextInput().WithMultilineMode(true)
			input.SetFocused(true)
			input.SetValue(tc.prefix + tc.suffix)
			input.CursorPos = len(tc.prefix)
			input.HandleKey(KeyEvent{Key: KeyCtrlW})
			assert.Equal(t, tc.wantPrefix+tc.suffix, input.Value())
			assert.Equal(t, len(tc.wantPrefix), input.CursorPos)
		})
	}
}

func TestTextInputKillWordPasteSegments(t *testing.T) {
	for _, tc := range []struct {
		name, suffix, want string
		inside             bool
	}{
		{"trailing punctuation", "!? ", "keep ", false},
		{"trailing word", "word", "keep beforefirst\npastesecond\npaste", false},
		{"inside placeholder", "", "keep beforefirst\npaste", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := newTextInput().WithPastePlaceholderMode(true)
			input.SetFocused(true)
			input.SetValue("keep before")
			input.HandlePaste("first\npaste")
			input.HandlePaste("second\npaste")
			input.insertAtCursor(tc.suffix)
			cursor := input.CursorPos
			input.insertAtCursor("後")
			input.CursorPos = cursor
			if tc.inside {
				input.CursorPos -= len(" lines]")
			}
			input.HandleKey(KeyEvent{Key: KeyCtrlW})
			assert.Equal(t, tc.want+"後", input.Value())
			if tc.name == "trailing word" {
				assert.Equal(t, "keep before[pasted 2 lines][pasted 2 lines]後", input.DisplayText())
				assert.Equal(t, len("keep before[pasted 2 lines][pasted 2 lines]"), input.CursorPos)
			} else if tc.inside {
				assert.Equal(t, "keep before[pasted 2 lines]後", input.DisplayText())
				assert.Equal(t, len("keep before[pasted 2 lines]"), input.CursorPos)
			} else {
				assert.Equal(t, len("keep "), input.CursorPos)
			}
		})
	}
}

func TestInputStateKillLargePastedWord(t *testing.T) {
	for _, multiline := range []bool{false, true} {
		t.Run(fmt.Sprint(multiline), func(t *testing.T) {
			var text string
			changes := 0
			state := newTestInputState(t, inputConfig{
				binding: &text, multiline: multiline,
				onChange: func(string) { changes++ },
			})
			payload := "keep " + strings.Repeat("x", 262144)
			state.HandleKeyEvent(KeyEvent{Paste: payload + "後"})
			state.input.CursorPos = len(payload)
			assert.True(t, state.HandleKeyEvent(KeyEvent{Key: KeyCtrlW}))
			assert.Equal(t, "keep 後", text)
			assert.Equal(t, len("keep "), state.input.CursorPos)
			assert.Equal(t, 2, changes)
			state.HandleKeyEvent(KeyEvent{Rune: '✓'})
			assert.Equal(t, "keep ✓後", text)
		})
	}
}

func BenchmarkTextInputKillWord(b *testing.B) {
	for _, size := range []int{8192, 16384, 65536, 262144} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			value := "keep " + strings.Repeat("x", size) + "suffix"
			input := newTextInput().WithMultilineMode(true)
			input.SetFocused(true)
			b.ReportAllocs()
			for b.Loop() {
				input.SetValue(value)
				input.CursorPos = len(value) - len("suffix")
				input.HandleKey(KeyEvent{Key: KeyCtrlW})
				if input.Value() != "keep suffix" || input.CursorPos != len("keep ") {
					b.Fatal("word was not cleared")
				}
			}
		})
	}
}
