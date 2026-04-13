package termtest

import (
	"fmt"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNewScreen(t *testing.T) {
	s := NewScreen(80, 24)
	w, h := s.Size()
	assert.Equal(t, 80, w)
	assert.Equal(t, 24, h)

	x, y := s.Cursor()
	assert.Equal(t, 0, x)
	assert.Equal(t, 0, y)
}

func TestScreenWriteString(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello")

	assert.Equal(t, "Hello", s.Row(0))

	x, y := s.Cursor()
	assert.Equal(t, 5, x)
	assert.Equal(t, 0, y)
}

func TestScreenWriteNewline(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Line1\nLine2\nLine3")

	assert.Equal(t, "Line1", s.Row(0))
	assert.Equal(t, "Line2", s.Row(1))
	assert.Equal(t, "Line3", s.Row(2))
}

func TestScreenWriteCarriageReturn(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello\rWorld")

	assert.Equal(t, "World", s.Row(0))
}

func TestScreenWrap(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("HelloWorld!")

	// "HelloWorld" fills row 0, "!" wraps to row 1
	assert.Equal(t, "HelloWorld", s.Row(0))
	assert.Equal(t, "!", s.Row(1))
}

func TestScreenClear(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello")
	s.Clear()

	assert.Equal(t, "", s.Row(0))
	x, y := s.Cursor()
	assert.Equal(t, 0, x)
	assert.Equal(t, 0, y)
}

func TestScreenClearLine(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Line1\nLine2")
	s.SetCursor(0, 0)
	s.ClearLine()

	assert.Equal(t, "", s.Row(0))
	assert.Equal(t, "Line2", s.Row(1))
}

func TestScreenClearToEndOfLine(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello World")
	s.SetCursor(5, 0)
	s.ClearToEndOfLine()

	assert.Equal(t, "Hello", s.Row(0))
}

func TestScreenContains(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello World")

	assert.True(t, s.Contains("Hello"))
	assert.True(t, s.Contains("World"))
	assert.True(t, s.Contains("lo Wo"))
	assert.False(t, s.Contains("Goodbye"))
}

func TestScreenSetCell(t *testing.T) {
	s := NewScreen(20, 5)
	s.SetCell(5, 2, 'X', Style{Bold: true})

	cell := s.Cell(5, 2)
	assert.Equal(t, 'X', cell.Char)
	assert.True(t, cell.Style.Bold)
}

func TestScreenSetCursor(t *testing.T) {
	s := NewScreen(20, 10)
	s.SetCursor(15, 8)

	x, y := s.Cursor()
	assert.Equal(t, 15, x)
	assert.Equal(t, 8, y)
}

func TestScreenSetCursorClamp(t *testing.T) {
	s := NewScreen(20, 10)

	// Test clamping to bounds
	s.SetCursor(-5, -3)
	x, y := s.Cursor()
	assert.Equal(t, 0, x)
	assert.Equal(t, 0, y)

	s.SetCursor(100, 50)
	x, y = s.Cursor()
	assert.Equal(t, 19, x)
	assert.Equal(t, 9, y)
}

func TestScreenScrollUp(t *testing.T) {
	s := NewScreen(20, 3)
	s.WriteString("Line1\nLine2\nLine3\nLine4")

	// After writing 4 lines to a 3-line screen, should have scrolled
	assert.Equal(t, "Line2", s.Row(0))
	assert.Equal(t, "Line3", s.Row(1))
	assert.Equal(t, "Line4", s.Row(2))
}

func TestScreenText(t *testing.T) {
	s := NewScreen(10, 3)
	s.WriteString("A\nB\nC")

	text := s.Text()
	assert.Equal(t, "A\nB\nC\n", text)
}

func TestScreenTextTrimsTrailingSpaces(t *testing.T) {
	s := NewScreen(20, 3)
	s.WriteString("Hello   ")
	s.SetCursor(0, 1)
	s.WriteString("World")

	text := s.Text()
	// Trailing spaces should be trimmed per line
	assert.Equal(t, "Hello\nWorld\n\n", text)
}

func TestScreenWideCharacters(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hi")
	s.WriteRune('日') // Wide character
	s.WriteString("!")

	// The row should show: "Hi日!"
	// But "日" takes 2 columns
	row := s.Row(0)
	assert.Contains(t, row, "Hi")
	assert.Contains(t, row, "日")
	assert.Contains(t, row, "!")
}

func TestScreenTab(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("A\tB")

	// Tab should move to column 8
	row := s.Row(0)
	assert.True(t, len(row) >= 9, "row should be at least 9 chars with tab")
}

// Edge cases for Screen

func TestScreenSmallDimensions(t *testing.T) {
	t.Run("1x1 screen", func(t *testing.T) {
		s := NewScreen(1, 1)
		w, h := s.Size()
		assert.Equal(t, 1, w)
		assert.Equal(t, 1, h)

		s.WriteString("X")
		assert.Equal(t, "X", s.Row(0))

		// Writing more should wrap but stay in bounds due to scroll
		s.WriteString("Y")
		assert.Equal(t, "Y", s.Row(0))
	})

	t.Run("1 column screen", func(t *testing.T) {
		s := NewScreen(1, 5)
		s.WriteString("ABCDE")

		// Each character wraps to next line
		assert.Equal(t, "A", s.Row(0))
		assert.Equal(t, "B", s.Row(1))
		assert.Equal(t, "C", s.Row(2))
		assert.Equal(t, "D", s.Row(3))
		assert.Equal(t, "E", s.Row(4))
	})

	t.Run("1 row screen", func(t *testing.T) {
		s := NewScreen(10, 1)
		s.WriteString("Hello World!!")

		// "Hello Worl" fills row, then wraps+scrolls
		// After scroll, "d!!" is written to the now-empty row
		assert.Equal(t, "d!!", s.Row(0))
	})
}

func TestScreenCellOutOfBounds(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("Hello")

	// Reading out of bounds should return default cell
	cell := s.Cell(-1, 0)
	assert.Equal(t, ' ', cell.Char)
	assert.Equal(t, 1, cell.Width)

	cell = s.Cell(100, 0)
	assert.Equal(t, ' ', cell.Char)

	cell = s.Cell(0, -1)
	assert.Equal(t, ' ', cell.Char)

	cell = s.Cell(0, 100)
	assert.Equal(t, ' ', cell.Char)
}

func TestScreenSetCellOutOfBounds(t *testing.T) {
	s := NewScreen(10, 5)

	// Setting out of bounds should be no-op
	s.SetCell(-1, 0, 'X', Style{})
	s.SetCell(100, 0, 'X', Style{})
	s.SetCell(0, -1, 'X', Style{})
	s.SetCell(0, 100, 'X', Style{})

	// Screen should still be empty
	AssertEmpty(t, s)
}

func TestScreenRowOutOfBounds(t *testing.T) {
	s := NewScreen(10, 3)
	s.WriteString("Line1\nLine2\nLine3")

	assert.Equal(t, "", s.Row(-1))
	assert.Equal(t, "", s.Row(100))
}

func TestScreenWideCharacterAtEdge(t *testing.T) {
	s := NewScreen(10, 3)

	// Write up to column 9, then a wide char
	s.WriteString("123456789")
	s.WriteRune('日') // Wide character at position 9

	// Wide char should wrap since it needs 2 columns
	assert.Equal(t, "123456789", s.Row(0))
	assert.Contains(t, s.Row(1), "日")
}

func TestScreenMultipleScrolls(t *testing.T) {
	s := NewScreen(10, 3)

	for i := 0; i < 10; i++ {
		s.WriteString("Line\n")
	}

	// After 10 newlines with 3-line screen, only last content visible
	x, y := s.Cursor()
	assert.Equal(t, 0, x)
	assert.Equal(t, 2, y) // Cursor at last row
}

func TestScreenEmptyWrite(t *testing.T) {
	s := NewScreen(10, 5)
	n, err := s.Write([]byte{})
	assert.Equal(t, 0, n)
	assert.NoError(t, err)
	AssertEmpty(t, s)
}

func TestScreenNullCharacters(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("A\x00B")

	// Null character should be rendered (it becomes rune 0, shown as space in Text())
	row := s.Row(0)
	assert.Contains(t, row, "A")
	assert.Contains(t, row, "B")
}

func TestScreenTabAtVariousPositions(t *testing.T) {
	tests := []struct {
		startCol    int
		expectedCol int
	}{
		{0, 8},
		{1, 8},
		{7, 8},
		{8, 16},
		{15, 16},
	}

	for _, tt := range tests {
		s := NewScreen(30, 5)
		s.SetCursor(tt.startCol, 0)
		s.WriteRune('\t')
		x, _ := s.Cursor()
		assert.Equal(t, tt.expectedCol, x, "tab from col %d", tt.startCol)
	}
}

func TestScreenTabAtEndOfLine(t *testing.T) {
	s := NewScreen(10, 5)
	s.SetCursor(9, 0)
	s.WriteRune('\t')

	// Tab at end should stay clamped to width-1
	x, y := s.Cursor()
	assert.Equal(t, 9, x)
	assert.Equal(t, 0, y)
}

func TestScreenCarriageReturnNewline(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello\r\nWorld")

	assert.Equal(t, "Hello", s.Row(0))
	assert.Equal(t, "World", s.Row(1))
}

func TestScreenTextWithEmptyLines(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("A\n\n\nB")

	text := s.Text()
	assert.Equal(t, "A\n\n\nB\n\n", text)
}

func TestScreenClearToStartOfLine(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello World")
	s.SetCursor(5, 0)
	s.ClearToStartOfLine()

	// Clears positions 0-5 (inclusive), leaving spaces before "World"
	// Row() only trims trailing spaces, not leading
	row := s.Row(0)
	assert.Equal(t, "      World", row)
}

func TestScreenClearToEndOfScreen(t *testing.T) {
	s := NewScreen(10, 3)
	s.WriteString("Line1     ")
	s.SetCursor(0, 1)
	s.WriteString("Line2     ")
	s.SetCursor(0, 2)
	s.WriteString("Line3     ")

	s.SetCursor(3, 1)
	s.ClearToEndOfScreen()

	assert.Equal(t, "Line1", s.Row(0))
	assert.Equal(t, "Lin", s.Row(1))
	assert.Equal(t, "", s.Row(2))
}

func TestScreenClearToStartOfScreen(t *testing.T) {
	s := NewScreen(10, 3)
	s.WriteString("Line1     ")
	s.SetCursor(0, 1)
	s.WriteString("Line2     ")
	s.SetCursor(0, 2)
	s.WriteString("Line3     ")

	s.SetCursor(3, 1)
	s.ClearToStartOfScreen()

	assert.Equal(t, "", s.Row(0))
	// Clears positions 0-3 (inclusive) on row 1, leaving "    2" with leading spaces
	// "Line2" becomes "    2" (4 spaces + "2")
	assert.Equal(t, "    2", s.Row(1))
	assert.Equal(t, "Line3", s.Row(2))
}

func TestScreenContainsMultiline(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hello\nWorld")

	// Text() includes newlines so multi-line patterns work
	assert.True(t, s.Contains("Hello\nWorld"))
}

// Unicode and wide character tests

func TestScreenWideCharacterWrapping(t *testing.T) {
	t.Run("wide char wraps at exact boundary", func(t *testing.T) {
		s := NewScreen(6, 3)
		// Write 5 regular chars, then wide char at position 5 (needs 2 cols)
		s.WriteString("12345")
		s.WriteRune('日')

		assert.Equal(t, "12345", s.Row(0))
		assert.Contains(t, s.Row(1), "日")
	})

	t.Run("wide char fits exactly", func(t *testing.T) {
		s := NewScreen(6, 3)
		s.WriteString("1234")
		s.WriteRune('日') // Takes cols 4-5

		assert.Contains(t, s.Row(0), "1234")
		assert.Contains(t, s.Row(0), "日")
	})
}

func TestScreenMultipleWideChars(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("日本語テスト")

	row := s.Row(0)
	assert.Contains(t, row, "日")
	assert.Contains(t, row, "本")
	assert.Contains(t, row, "語")
	assert.Contains(t, row, "テ")
	assert.Contains(t, row, "ス")
	assert.Contains(t, row, "ト")
}

func TestScreenContinuationCells(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteRune('日') // Wide character

	// First cell has the character
	cell0 := s.Cell(0, 0)
	assert.Equal(t, '日', cell0.Char)
	assert.Equal(t, 2, cell0.Width)

	// Second cell is continuation (width 0)
	cell1 := s.Cell(1, 0)
	assert.Equal(t, rune(0), cell1.Char)
	assert.Equal(t, 0, cell1.Width)
}

func TestScreenEmojiBasic(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Hi 👋!")

	row := s.Row(0)
	assert.Contains(t, row, "Hi")
	assert.Contains(t, row, "👋")
	assert.Contains(t, row, "!")
}

func TestScreenWriteStringPreservesGraphemeClusters(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		wantChar     rune
		wantTrailing string
		wantWidth    int
	}{
		{
			name:         "VS16 heart",
			input:        "\u2764\uFE0F",
			wantChar:     '\u2764',
			wantTrailing: "\uFE0F",
			wantWidth:    2,
		},
		{
			name:         "hash keycap",
			input:        "#\uFE0F\u20E3",
			wantChar:     '#',
			wantTrailing: "\uFE0F\u20E3",
			wantWidth:    2,
		},
		{
			name:         "JP flag",
			input:        "\U0001F1EF\U0001F1F5",
			wantChar:     '\U0001F1EF',
			wantTrailing: "\U0001F1F5",
			wantWidth:    2,
		},
		{
			name:         "rainbow flag",
			input:        "\U0001F3F3\uFE0F\u200D\U0001F308",
			wantChar:     '\U0001F3F3',
			wantTrailing: "\uFE0F\u200D\U0001F308",
			wantWidth:    2,
		},
		{
			name:         "family of four",
			input:        "\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466",
			wantChar:     '\U0001F468',
			wantTrailing: "\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466",
			wantWidth:    2,
		},
		{
			name:         "skin toned wave",
			input:        "\U0001F44B\U0001F3FD",
			wantChar:     '\U0001F44B',
			wantTrailing: "\U0001F3FD",
			wantWidth:    2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewScreen(20, 5)
			s.WriteString(tc.input + " x")

			assert.Equal(t, tc.input+" x", s.Row(0))

			cell := s.Cell(0, 0)
			assert.Equal(t, tc.wantChar, cell.Char)
			assert.Equal(t, tc.wantTrailing, cell.Trailing)
			assert.Equal(t, tc.wantWidth, cell.Width)

			cont := s.Cell(1, 0)
			assert.Equal(t, rune(0), cont.Char)
			assert.Equal(t, 0, cont.Width)
			assert.True(t, cont.Continuation)

			trailingCell := s.Cell(tc.wantWidth, 0)
			assert.Equal(t, ' ', trailingCell.Char)
		})
	}
}

func TestScreenANSIWritePreservesGraphemeClusters(t *testing.T) {
	s := NewScreen(20, 5)
	_, err := s.Write([]byte("\x1b[1m#\uFE0F\u20E3\x1b[0m \x1b[32m\U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466\x1b[0m"))
	assert.NoError(t, err)

	assert.Equal(t, "#\uFE0F\u20E3 \U0001F468\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466", s.Row(0))

	keycap := s.Cell(0, 0)
	assert.Equal(t, '#', keycap.Char)
	assert.Equal(t, "\uFE0F\u20E3", keycap.Trailing)
	assert.True(t, keycap.Style.Bold)

	family := s.Cell(3, 0)
	assert.Equal(t, '\U0001F468', family.Char)
	assert.Equal(t, "\u200D\U0001F469\u200D\U0001F467\u200D\U0001F466", family.Trailing)
	assert.Equal(t, Color{Type: ColorBasic, Value: 2}, family.Style.Foreground)
}

func TestScreenCellGlyph(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("A#\uFE0F\u20E3")

	assert.Equal(t, "A", s.CellGlyph(0, 0))
	assert.Equal(t, "#\uFE0F\u20E3", s.CellGlyph(1, 0))
	assert.Equal(t, "", s.CellGlyph(2, 0))
}

func TestScreenDeleteCharsAvoidsBrokenWideGlyphs(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("日B")
	_, err := s.Write([]byte("\x1b[2G\x1b[P"))
	assert.NoError(t, err)

	assert.Equal(t, " B", s.Row(0))
	assert.Equal(t, ' ', s.Cell(0, 0).Char)
	assert.Equal(t, 'B', s.Cell(1, 0).Char)
	assert.False(t, s.Cell(0, 0).Continuation)
}

func TestScreenInsertCharsAvoidsBrokenWideGlyphs(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("日B")
	_, err := s.Write([]byte("\x1b[2G\x1b[@"))
	assert.NoError(t, err)

	assert.Equal(t, "   B", s.Row(0))
	assert.Equal(t, 'B', s.Cell(3, 0).Char)
	assert.False(t, s.Cell(1, 0).Continuation)
}

func TestScreenEraseCharsAvoidsBrokenWideGlyphs(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("日B")
	_, err := s.Write([]byte("\x1b[2G\x1b[X"))
	assert.NoError(t, err)

	assert.Equal(t, "  B", s.Row(0))
	assert.Equal(t, 'B', s.Cell(2, 0).Char)
	assert.False(t, s.Cell(1, 0).Continuation)
}

func TestScreenMixedWidthCharacters(t *testing.T) {
	s := NewScreen(30, 5)
	s.WriteString("Hello世界ABC")

	row := s.Row(0)
	assert.Contains(t, row, "Hello")
	assert.Contains(t, row, "世")
	assert.Contains(t, row, "界")
	assert.Contains(t, row, "ABC")
}

func TestScreenKoreanCharacters(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("안녕하세요")

	row := s.Row(0)
	assert.Contains(t, row, "안")
	assert.Contains(t, row, "녕")
}

func TestScreenArabicCharacters(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("مرحبا")

	row := s.Row(0)
	// Arabic is RTL but we just check it renders
	assert.NotEqual(t, "", row)
}

func TestScreenCyrillicCharacters(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Привет мир")

	row := s.Row(0)
	assert.Contains(t, row, "Привет")
	assert.Contains(t, row, "мир")
}

func TestScreenGreekCharacters(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("Γειά σου κόσμε")

	row := s.Row(0)
	assert.Contains(t, row, "Γειά")
}

func TestScreenZeroWidthJoiner(t *testing.T) {
	// Zero-width joiner used in some emoji sequences
	s := NewScreen(30, 5)
	s.WriteString("A\u200BB") // Zero-width space between A and B

	row := s.Row(0)
	// Should contain both A and B
	assert.Contains(t, row, "A")
	assert.Contains(t, row, "B")
}

func TestScreenAccentedCharacters(t *testing.T) {
	s := NewScreen(30, 5)
	s.WriteString("café résumé naïve")

	row := s.Row(0)
	assert.Contains(t, row, "café")
	assert.Contains(t, row, "résumé")
	assert.Contains(t, row, "naïve")
}

func TestScreenMathematicalSymbols(t *testing.T) {
	s := NewScreen(30, 5)
	s.WriteString("∑∫∂√∞≠≈")

	row := s.Row(0)
	assert.Contains(t, row, "∑")
	assert.Contains(t, row, "∞")
}

func TestScreenBoxDrawingCharacters(t *testing.T) {
	s := NewScreen(30, 5)
	s.WriteString("┌─┐│└┘├┤┬┴┼")

	row := s.Row(0)
	assert.Contains(t, row, "┌")
	assert.Contains(t, row, "─")
	assert.Contains(t, row, "┐")
}

func TestScreenWideCharAtEndOverwrite(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("ABCDEFGHIJ")
	s.SetCursor(8, 0)
	s.WriteRune('日') // Wide char at position 8 needs 2 columns (8-9)

	// Wide char fits at positions 8-9, overwrites I and J
	assert.Contains(t, s.Row(0), "ABCDEFGH")
	assert.Contains(t, s.Row(0), "日")
}

func TestScreenOverwriteWideCharWithNarrow(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteRune('日') // Wide char at 0,1
	s.SetCursor(0, 0)
	s.WriteRune('A') // Overwrite first half

	// The wide char should be partially destroyed
	cell0 := s.Cell(0, 0)
	assert.Equal(t, 'A', cell0.Char)
}

func TestScreenOverwriteContinuationCell(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteRune('日') // Wide char at positions 0,1
	s.SetCursor(1, 0)
	s.WriteRune('X') // Write to continuation cell

	// Should replace the continuation
	cell1 := s.Cell(1, 0)
	assert.Equal(t, 'X', cell1.Char)
}

// Example demonstrates basic screen usage.
func ExampleScreen() {
	screen := NewScreen(40, 5)
	screen.Write([]byte("Hello, World!"))
	fmt.Println(screen.Row(0))
	// Output: Hello, World!
}

// Example of processing ANSI sequences for styled text.
func ExampleScreen_Write() {
	screen := NewScreen(40, 3)
	screen.Write([]byte("\x1b[1mBold\x1b[0m and \x1b[32mGreen\x1b[0m"))

	// Check the text content (ANSI removed)
	fmt.Println(screen.Row(0))

	// Check that first character is bold
	cell := screen.Cell(0, 0)
	fmt.Printf("Bold: %v\n", cell.Style.Bold)

	// Output:
	// Bold and Green
	// Bold: true
}

// Example of using Screen with cursor position.
func ExampleScreen_Cursor() {
	screen := NewScreen(20, 5)
	screen.Write([]byte("Hello"))

	x, y := screen.Cursor()
	fmt.Printf("Cursor at (%d, %d)\n", x, y)

	// Output:
	// Cursor at (5, 0)
}

// Example of checking screen content.
func ExampleScreen_Contains() {
	screen := NewScreen(40, 5)
	screen.Write([]byte("The quick brown fox"))

	fmt.Println(screen.Contains("quick"))
	fmt.Println(screen.Contains("lazy"))

	// Output:
	// true
	// false
}
