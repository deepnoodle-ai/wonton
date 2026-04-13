package gif

import (
	"fmt"
	"testing"
)

// These tests intentionally capture the gif package's current grapheme-cluster
// behavior so it can be evaluated before changing the storage/rendering model.
// Run with:
//
//	go test -v ./gif -run GraphemeCharacterization
func TestTerminalScreen_GraphemeCharacterization_CurrentBehavior(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantCursor int
		wantCells  []rune
	}{
		{
			name:       "heart VS16",
			input:      "\u2764\uFE0F",
			wantCursor: 2,
			wantCells:  []rune{'\u2764', '\uFE0F'},
		},
		{
			name:       "hash keycap",
			input:      "#\uFE0F\u20E3",
			wantCursor: 3,
			wantCells:  []rune{'#', '\uFE0F', '\u20E3'},
		},
		{
			name:       "JP flag",
			input:      "\U0001F1EF\U0001F1F5",
			wantCursor: 2,
			wantCells:  []rune{'\U0001F1EF', '\U0001F1F5'},
		},
		{
			name:       "family of four",
			input:      "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466",
			wantCursor: 7,
			wantCells:  []rune{'\U0001F468', '\u200D', '\U0001F469', '\u200D', '\U0001F467', '\u200D', '\U0001F466'},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			screen := NewTerminalScreen(20, 3)
			screen.WriteString(tc.input, White, Black)

			if screen.CursorX != tc.wantCursor {
				t.Fatalf("cursorX: want %d, got %d", tc.wantCursor, screen.CursorX)
			}

			got := make([]rune, len(tc.wantCells))
			for i := range tc.wantCells {
				got[i] = screen.Cells[0][i].Char
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.wantCells) {
				t.Fatalf("cells: want %v, got %v", tc.wantCells, got)
			}

			t.Logf("input=%q cursorX=%d cells=%s", tc.input, screen.CursorX, dumpTerminalCells(screen, 0, len(tc.wantCells)))
		})
	}
}

func TestEmulator_GraphemeCharacterization_CurrentBehavior(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		wantCursor int
		wantCells  []rune
	}{
		{
			name:       "heart VS16",
			input:      "\u2764\uFE0F",
			wantCursor: 2,
			wantCells:  []rune{'\u2764', '\uFE0F'},
		},
		{
			name:       "hash keycap",
			input:      "#\uFE0F\u20E3",
			wantCursor: 3,
			wantCells:  []rune{'#', '\uFE0F', '\u20E3'},
		},
		{
			name:       "JP flag",
			input:      "\U0001F1EF\U0001F1F5",
			wantCursor: 2,
			wantCells:  []rune{'\U0001F1EF', '\U0001F1F5'},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			em := NewEmulator(20, 3)
			em.ProcessOutput(tc.input)

			screen := em.Screen()
			if screen.CursorX != tc.wantCursor {
				t.Fatalf("cursorX: want %d, got %d", tc.wantCursor, screen.CursorX)
			}

			got := make([]rune, len(tc.wantCells))
			for i := range tc.wantCells {
				got[i] = screen.Cells[0][i].Char
			}
			if fmt.Sprint(got) != fmt.Sprint(tc.wantCells) {
				t.Fatalf("cells: want %v, got %v", tc.wantCells, got)
			}

			t.Logf("input=%q cursorX=%d cells=%s", tc.input, screen.CursorX, dumpTerminalCells(screen, 0, len(tc.wantCells)))
		})
	}
}

func dumpTerminalCells(screen *TerminalScreen, row, count int) string {
	if row < 0 || row >= screen.Height {
		return "<row out of bounds>"
	}
	if count > screen.Width {
		count = screen.Width
	}

	out := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			out += " | "
		}
		r := screen.Cells[row][i].Char
		out += fmt.Sprintf("%d:%q(U+%04X)", i, r, r)
	}
	return out
}
