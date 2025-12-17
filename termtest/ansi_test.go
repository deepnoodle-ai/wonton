package termtest

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestANSICursorMovement(t *testing.T) {
	t.Run("cursor up", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 5)
		s.Write([]byte("\x1b[2A")) // Move up 2

		x, y := s.Cursor()
		assert.Equal(t, 5, x)
		assert.Equal(t, 3, y)
	})

	t.Run("cursor down", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 3)
		s.Write([]byte("\x1b[3B")) // Move down 3

		x, y := s.Cursor()
		assert.Equal(t, 5, x)
		assert.Equal(t, 6, y)
	})

	t.Run("cursor forward", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 3)
		s.Write([]byte("\x1b[4C")) // Move forward 4

		x, y := s.Cursor()
		assert.Equal(t, 9, x)
		assert.Equal(t, 3, y)
	})

	t.Run("cursor back", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(10, 3)
		s.Write([]byte("\x1b[3D")) // Move back 3

		x, y := s.Cursor()
		assert.Equal(t, 7, x)
		assert.Equal(t, 3, y)
	})

	t.Run("cursor position", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.Write([]byte("\x1b[5;10H")) // Row 5, column 10 (1-based)

		x, y := s.Cursor()
		assert.Equal(t, 9, x)  // 0-based
		assert.Equal(t, 4, y)  // 0-based
	})

	t.Run("cursor position default", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 5)
		s.Write([]byte("\x1b[H")) // Home position (1,1)

		x, y := s.Cursor()
		assert.Equal(t, 0, x)
		assert.Equal(t, 0, y)
	})

	t.Run("cursor horizontal absolute", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 5)
		s.Write([]byte("\x1b[15G")) // Column 15 (1-based)

		x, y := s.Cursor()
		assert.Equal(t, 14, x) // 0-based
		assert.Equal(t, 5, y)
	})
}

func TestANSIEraseInDisplay(t *testing.T) {
	t.Run("clear to end of screen", func(t *testing.T) {
		s := NewScreen(10, 3)
		s.WriteString("AAAAAAAAAA")
		s.SetCursor(0, 1)
		s.WriteString("BBBBBBBBBB")
		s.SetCursor(0, 2)
		s.WriteString("CCCCCCCCCC")

		s.SetCursor(5, 1)
		s.Write([]byte("\x1b[0J")) // Clear from cursor to end

		assert.Equal(t, "AAAAAAAAAA", s.Row(0))
		assert.Equal(t, "BBBBB", s.Row(1))
		assert.Equal(t, "", s.Row(2))
	})

	t.Run("clear to start of screen", func(t *testing.T) {
		s := NewScreen(10, 3)
		s.WriteString("AAAAAAAAAA")
		s.SetCursor(0, 1)
		s.WriteString("BBBBBBBBBB")
		s.SetCursor(0, 2)
		s.WriteString("CCCCCCCCCC")

		s.SetCursor(5, 1)
		s.Write([]byte("\x1b[1J")) // Clear from start to cursor

		assert.Equal(t, "", s.Row(0))
		// Row 1 up to cursor position cleared
		assert.Equal(t, "CCCCCCCCCC", s.Row(2))
	})

	t.Run("clear entire screen", func(t *testing.T) {
		s := NewScreen(10, 3)
		s.WriteString("AAAAAAAAAA")
		s.SetCursor(0, 1)
		s.WriteString("BBBBBBBBBB")

		s.Write([]byte("\x1b[2J")) // Clear entire screen

		assert.Equal(t, "", s.Row(0))
		assert.Equal(t, "", s.Row(1))
		assert.Equal(t, "", s.Row(2))
	})
}

func TestANSIEraseInLine(t *testing.T) {
	t.Run("clear to end of line", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.WriteString("Hello World")
		s.SetCursor(5, 0)
		s.Write([]byte("\x1b[0K"))

		assert.Equal(t, "Hello", s.Row(0))
	})

	t.Run("clear to start of line", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.WriteString("Hello World")
		s.SetCursor(5, 0)
		s.Write([]byte("\x1b[1K"))

		row := s.Row(0)
		assert.Contains(t, row, "World")
	})

	t.Run("clear entire line", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.WriteString("Hello World")
		s.SetCursor(5, 0)
		s.Write([]byte("\x1b[2K"))

		assert.Equal(t, "", s.Row(0))
	})
}

func TestANSISGR(t *testing.T) {
	t.Run("reset", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[1mBold\x1b[0mNormal"))

		boldCell := s.Cell(0, 0)
		assert.True(t, boldCell.Style.Bold)

		normalCell := s.Cell(4, 0)
		assert.False(t, normalCell.Style.Bold)
	})

	t.Run("bold", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[1mX"))

		cell := s.Cell(0, 0)
		assert.True(t, cell.Style.Bold)
	})

	t.Run("italic", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[3mX"))

		cell := s.Cell(0, 0)
		assert.True(t, cell.Style.Italic)
	})

	t.Run("underline", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[4mX"))

		cell := s.Cell(0, 0)
		assert.True(t, cell.Style.Underline)
	})

	t.Run("foreground color", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[31mX")) // Red

		cell := s.Cell(0, 0)
		assert.Equal(t, ColorBasic, cell.Style.Foreground.Type)
		assert.Equal(t, uint8(1), cell.Style.Foreground.Value) // Red = 1
	})

	t.Run("background color", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[44mX")) // Blue background

		cell := s.Cell(0, 0)
		assert.Equal(t, ColorBasic, cell.Style.Background.Type)
		assert.Equal(t, uint8(4), cell.Style.Background.Value) // Blue = 4
	})

	t.Run("256 color foreground", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[38;5;196mX")) // Color 196

		cell := s.Cell(0, 0)
		assert.Equal(t, Color256, cell.Style.Foreground.Type)
		assert.Equal(t, uint8(196), cell.Style.Foreground.Value)
	})

	t.Run("RGB color foreground", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[38;2;255;128;64mX")) // RGB

		cell := s.Cell(0, 0)
		assert.Equal(t, ColorRGB, cell.Style.Foreground.Type)
		assert.Equal(t, uint8(255), cell.Style.Foreground.R)
		assert.Equal(t, uint8(128), cell.Style.Foreground.G)
		assert.Equal(t, uint8(64), cell.Style.Foreground.B)
	})

	t.Run("bright foreground colors", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[91mX")) // Bright red

		cell := s.Cell(0, 0)
		assert.Equal(t, ColorBasic, cell.Style.Foreground.Type)
		assert.Equal(t, uint8(9), cell.Style.Foreground.Value) // Bright red = 8 + 1
	})

	t.Run("default colors", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[31m\x1b[39mX")) // Set red then default

		cell := s.Cell(0, 0)
		assert.Equal(t, ColorDefault, cell.Style.Foreground.Type)
	})
}

func TestANSISaveCursor(t *testing.T) {
	t.Run("save and restore with CSI", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 3)
		s.Write([]byte("\x1b[s"))      // Save
		s.Write([]byte("\x1b[10;15H")) // Move
		s.Write([]byte("\x1b[u"))      // Restore

		x, y := s.Cursor()
		assert.Equal(t, 5, x)
		assert.Equal(t, 3, y)
	})

	t.Run("save and restore with ESC 7/8", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(7, 4)
		s.Write([]byte("\x1b7"))       // Save
		s.Write([]byte("\x1b[10;15H")) // Move
		s.Write([]byte("\x1b8"))       // Restore

		x, y := s.Cursor()
		assert.Equal(t, 7, x)
		assert.Equal(t, 4, y)
	})
}

func TestANSIInsertDeleteLines(t *testing.T) {
	t.Run("insert lines", func(t *testing.T) {
		s := NewScreen(10, 5)
		s.WriteString("Line0\nLine1\nLine2\nLine3\nLine4")
		s.SetCursor(0, 2)
		s.Write([]byte("\x1b[2L")) // Insert 2 lines

		assert.Equal(t, "Line0", s.Row(0))
		assert.Equal(t, "Line1", s.Row(1))
		assert.Equal(t, "", s.Row(2)) // Inserted
		assert.Equal(t, "", s.Row(3)) // Inserted
		assert.Equal(t, "Line2", s.Row(4))
	})

	t.Run("delete lines", func(t *testing.T) {
		s := NewScreen(10, 5)
		s.WriteString("Line0\nLine1\nLine2\nLine3\nLine4")
		s.SetCursor(0, 1)
		s.Write([]byte("\x1b[2M")) // Delete 2 lines

		assert.Equal(t, "Line0", s.Row(0))
		assert.Equal(t, "Line3", s.Row(1))
		assert.Equal(t, "Line4", s.Row(2))
		assert.Equal(t, "", s.Row(3))
	})
}

func TestANSIInsertDeleteChars(t *testing.T) {
	t.Run("insert characters", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.WriteString("ABCDEFG")
		s.SetCursor(3, 0)
		s.Write([]byte("\x1b[2@")) // Insert 2 chars

		assert.Equal(t, "ABC  DEFG", s.Row(0))
	})

	t.Run("delete characters", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.WriteString("ABCDEFG")
		s.SetCursor(2, 0)
		s.Write([]byte("\x1b[2P")) // Delete 2 chars

		assert.Equal(t, "ABEFG", s.Row(0))
	})
}

func TestANSIEraseCharacters(t *testing.T) {
	s := NewScreen(20, 5)
	s.WriteString("ABCDEFG")
	s.SetCursor(2, 0)
	s.Write([]byte("\x1b[3X")) // Erase 3 chars

	assert.Equal(t, "AB   FG", s.Row(0))
}

func TestANSIOSCIgnored(t *testing.T) {
	s := NewScreen(20, 5)
	// OSC sequences should be parsed and ignored
	s.Write([]byte("\x1b]0;Title\x07Hello"))

	assert.Equal(t, "Hello", s.Row(0))
}

func TestANSIPrivateModes(t *testing.T) {
	s := NewScreen(20, 5)
	// Private modes like cursor visibility should be ignored
	s.Write([]byte("\x1b[?25lHello\x1b[?25h"))

	assert.Equal(t, "Hello", s.Row(0))
}

func TestANSIMixedContent(t *testing.T) {
	s := NewScreen(40, 10)

	// Simulate typical terminal output
	s.Write([]byte("\x1b[2J\x1b[H"))                         // Clear and home
	s.Write([]byte("\x1b[1;32mSuccess:\x1b[0m Test passed")) // Green bold "Success:"
	s.Write([]byte("\x1b[2;1H"))                             // Move to line 2
	s.Write([]byte("\x1b[31mError:\x1b[0m Something failed"))

	assert.Contains(t, s.Text(), "Success:")
	assert.Contains(t, s.Text(), "Test passed")
	assert.Contains(t, s.Text(), "Error:")
	assert.Contains(t, s.Text(), "Something failed")

	// Check styling
	successCell := s.Cell(0, 0)
	assert.True(t, successCell.Style.Bold)
	assert.Equal(t, ColorBasic, successCell.Style.Foreground.Type)
	assert.Equal(t, uint8(2), successCell.Style.Foreground.Value) // Green
}

// ANSI edge cases and malformed sequences

func TestANSIIncompleteSequences(t *testing.T) {
	t.Run("ESC at end of string", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("Hello\x1b"))
		// ESC by itself is skipped
		assert.Equal(t, "Hello", s.Row(0))
	})

	t.Run("ESC[ at end of string", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("Hello\x1b["))
		// Incomplete CSI outputs the [ since no command found
		// This is reasonable behavior - we don't buffer across writes
		assert.Contains(t, s.Row(0), "Hello")
	})

	t.Run("ESC[ with params but no command", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("Hello\x1b[1;2"))
		// Incomplete CSI outputs the raw chars
		assert.Contains(t, s.Row(0), "Hello")
	})
}

func TestANSIUnknownSequences(t *testing.T) {
	t.Run("unknown CSI command", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[999zHello"))
		// Unknown CSI commands are parsed and ignored
		assert.Equal(t, "Hello", s.Row(0))
	})

	t.Run("unknown ESC sequence", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1bZHello"))
		// Unknown ESC sequences: ESC is skipped, but following char is written
		// This matches the behavior - only known ESC sequences are consumed
		assert.Contains(t, s.Row(0), "Hello")
		assert.Contains(t, s.Row(0), "Z")
	})
}

func TestANSICursorMovementBoundaries(t *testing.T) {
	t.Run("cursor up past top", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 2)
		s.Write([]byte("\x1b[100A")) // Move up 100

		x, y := s.Cursor()
		assert.Equal(t, 5, x)
		assert.Equal(t, 0, y) // Clamped to top
	})

	t.Run("cursor down past bottom", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 7)
		s.Write([]byte("\x1b[100B")) // Move down 100

		x, y := s.Cursor()
		assert.Equal(t, 5, x)
		assert.Equal(t, 9, y) // Clamped to bottom
	})

	t.Run("cursor right past edge", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(15, 5)
		s.Write([]byte("\x1b[100C")) // Move right 100

		x, y := s.Cursor()
		assert.Equal(t, 19, x) // Clamped to right edge
		assert.Equal(t, 5, y)
	})

	t.Run("cursor left past edge", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 5)
		s.Write([]byte("\x1b[100D")) // Move left 100

		x, y := s.Cursor()
		assert.Equal(t, 0, x) // Clamped to left edge
		assert.Equal(t, 5, y)
	})
}

func TestANSICursorNextPrevLine(t *testing.T) {
	t.Run("cursor next line", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(10, 3)
		s.Write([]byte("\x1b[2E")) // Next line x2

		x, y := s.Cursor()
		assert.Equal(t, 0, x) // Column reset to 0
		assert.Equal(t, 5, y)
	})

	t.Run("cursor previous line", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(10, 5)
		s.Write([]byte("\x1b[2F")) // Previous line x2

		x, y := s.Cursor()
		assert.Equal(t, 0, x) // Column reset to 0
		assert.Equal(t, 3, y)
	})

	t.Run("next line at bottom", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 9)
		s.Write([]byte("\x1b[100E"))

		x, y := s.Cursor()
		assert.Equal(t, 0, x)
		assert.Equal(t, 9, y) // Clamped
	})

	t.Run("previous line at top", func(t *testing.T) {
		s := NewScreen(20, 10)
		s.SetCursor(5, 0)
		s.Write([]byte("\x1b[100F"))

		x, y := s.Cursor()
		assert.Equal(t, 0, x)
		assert.Equal(t, 0, y) // Clamped
	})
}

func TestANSILinePositionAbsolute(t *testing.T) {
	s := NewScreen(20, 10)
	s.SetCursor(15, 3)
	s.Write([]byte("\x1b[7d")) // Line position absolute to row 7 (1-based)

	x, y := s.Cursor()
	assert.Equal(t, 15, x) // X unchanged
	assert.Equal(t, 6, y)  // 0-based row 6
}

func TestANSICursorPositionF(t *testing.T) {
	s := NewScreen(20, 10)
	s.Write([]byte("\x1b[5;10f")) // Row 5, column 10 (1-based), using 'f' command

	x, y := s.Cursor()
	assert.Equal(t, 9, x) // 0-based
	assert.Equal(t, 4, y) // 0-based
}

func TestANSIEraseJ3(t *testing.T) {
	// ESC[3J - Clear entire screen and scrollback
	s := NewScreen(10, 3)
	s.WriteString("ABC")
	s.Write([]byte("\x1b[3J"))

	assert.Equal(t, "", s.Row(0))
	assert.Equal(t, "", s.Row(1))
	assert.Equal(t, "", s.Row(2))
}

func TestANSIOSCWithST(t *testing.T) {
	s := NewScreen(20, 5)
	// OSC with ST terminator (ESC \) instead of BEL
	s.Write([]byte("\x1b]0;Title\x1b\\Hello"))

	assert.Equal(t, "Hello", s.Row(0))
}

func TestANSITerminalReset(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1mBold\x1bc"))                 // Set bold, then reset terminal
	s.Write([]byte("Normal"))

	cell := s.Cell(0, 0)
	assert.False(t, cell.Style.Bold) // Style should be reset
	assert.Equal(t, 'N', cell.Char)
}

func TestANSIReverseIndex(t *testing.T) {
	t.Run("reverse index scrolls when at top", func(t *testing.T) {
		s := NewScreen(10, 3)
		s.WriteString("Line1     ")
		s.SetCursor(0, 1)
		s.WriteString("Line2     ")
		s.SetCursor(0, 2)
		s.WriteString("Line3     ")

		s.SetCursor(0, 0)
		s.Write([]byte("\x1bM")) // Reverse index

		// Should scroll down, cursor stays at 0
		x, y := s.Cursor()
		assert.Equal(t, 0, x)
		assert.Equal(t, 0, y)
		assert.Equal(t, "", s.Row(0)) // New blank line
		assert.Equal(t, "Line1", s.Row(1))
	})

	t.Run("reverse index moves cursor up when not at top", func(t *testing.T) {
		s := NewScreen(10, 3)
		s.SetCursor(0, 2)
		s.Write([]byte("\x1bM"))

		_, y := s.Cursor()
		assert.Equal(t, 1, y)
	})
}

func TestANSIIndex(t *testing.T) {
	t.Run("index scrolls when at bottom", func(t *testing.T) {
		s := NewScreen(10, 3)
		s.WriteString("Line1     ")
		s.SetCursor(0, 1)
		s.WriteString("Line2     ")
		s.SetCursor(0, 2)
		s.WriteString("Line3     ")

		s.SetCursor(0, 2)
		s.Write([]byte("\x1bD")) // Index

		assert.Equal(t, "Line2", s.Row(0))
		assert.Equal(t, "Line3", s.Row(1))
		assert.Equal(t, "", s.Row(2)) // Scrolled in blank line
	})

	t.Run("index moves cursor down when not at bottom", func(t *testing.T) {
		s := NewScreen(10, 3)
		s.SetCursor(0, 0)
		s.Write([]byte("\x1bD"))

		_, y := s.Cursor()
		assert.Equal(t, 1, y)
	})
}

func TestANSINextLine(t *testing.T) {
	s := NewScreen(20, 5)
	s.SetCursor(10, 2)
	s.Write([]byte("\x1bE")) // Next line

	x, y := s.Cursor()
	assert.Equal(t, 0, x) // Column reset
	assert.Equal(t, 3, y)
}

func TestANSICharacterSetDesignation(t *testing.T) {
	s := NewScreen(20, 5)
	// Character set designation (ESC ( 0) should be ignored
	s.Write([]byte("\x1b(0Hello"))
	assert.Equal(t, "Hello", s.Row(0))
}

func TestANSISGRAllAttributes(t *testing.T) {
	t.Run("dim", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[2mX"))
		assert.True(t, s.Cell(0, 0).Style.Dim)
	})

	t.Run("blink", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[5mX"))
		assert.True(t, s.Cell(0, 0).Style.Blink)
	})

	t.Run("reverse", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[7mX"))
		assert.True(t, s.Cell(0, 0).Style.Reverse)
	})

	t.Run("hidden", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[8mX"))
		assert.True(t, s.Cell(0, 0).Style.Hidden)
	})

	t.Run("strike", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[9mX"))
		assert.True(t, s.Cell(0, 0).Style.Strike)
	})

	t.Run("double underline", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[21mX"))
		assert.True(t, s.Cell(0, 0).Style.Underline) // Treated as underline
	})
}

func TestANSISGRResetAttributes(t *testing.T) {
	t.Run("reset bold and dim", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[1;2mA\x1b[22mB"))
		assert.True(t, s.Cell(0, 0).Style.Bold)
		assert.True(t, s.Cell(0, 0).Style.Dim)
		assert.False(t, s.Cell(1, 0).Style.Bold)
		assert.False(t, s.Cell(1, 0).Style.Dim)
	})

	t.Run("reset italic", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[3mA\x1b[23mB"))
		assert.True(t, s.Cell(0, 0).Style.Italic)
		assert.False(t, s.Cell(1, 0).Style.Italic)
	})

	t.Run("reset underline", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[4mA\x1b[24mB"))
		assert.True(t, s.Cell(0, 0).Style.Underline)
		assert.False(t, s.Cell(1, 0).Style.Underline)
	})

	t.Run("reset blink", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[5mA\x1b[25mB"))
		assert.True(t, s.Cell(0, 0).Style.Blink)
		assert.False(t, s.Cell(1, 0).Style.Blink)
	})

	t.Run("reset reverse", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[7mA\x1b[27mB"))
		assert.True(t, s.Cell(0, 0).Style.Reverse)
		assert.False(t, s.Cell(1, 0).Style.Reverse)
	})

	t.Run("reset hidden", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[8mA\x1b[28mB"))
		assert.True(t, s.Cell(0, 0).Style.Hidden)
		assert.False(t, s.Cell(1, 0).Style.Hidden)
	})

	t.Run("reset strike", func(t *testing.T) {
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[9mA\x1b[29mB"))
		assert.True(t, s.Cell(0, 0).Style.Strike)
		assert.False(t, s.Cell(1, 0).Style.Strike)
	})
}

func TestANSISGRBrightBackgroundColors(t *testing.T) {
	for i := 0; i < 8; i++ {
		code := 100 + i
		s := NewScreen(20, 5)
		s.Write([]byte("\x1b[" + string(rune('0'+code/100)) + string(rune('0'+(code/10)%10)) + string(rune('0'+code%10)) + "mX"))
		cell := s.Cell(0, 0)
		assert.Equal(t, ColorBasic, cell.Style.Background.Type, "bright bg %d", code)
		assert.Equal(t, uint8(8+i), cell.Style.Background.Value, "bright bg %d", code)
	}
}

func TestANSISGR256ColorBackground(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[48;5;196mX")) // Background color 196

	cell := s.Cell(0, 0)
	assert.Equal(t, Color256, cell.Style.Background.Type)
	assert.Equal(t, uint8(196), cell.Style.Background.Value)
}

func TestANSISGRRGBBackground(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[48;2;100;150;200mX")) // RGB background

	cell := s.Cell(0, 0)
	assert.Equal(t, ColorRGB, cell.Style.Background.Type)
	assert.Equal(t, uint8(100), cell.Style.Background.R)
	assert.Equal(t, uint8(150), cell.Style.Background.G)
	assert.Equal(t, uint8(200), cell.Style.Background.B)
}

func TestANSISGRColonSeparatedRGB(t *testing.T) {
	s := NewScreen(20, 5)
	// Some terminals use colon instead of semicolon for RGB
	s.Write([]byte("\x1b[38:2:128:64:32mX"))

	cell := s.Cell(0, 0)
	assert.Equal(t, ColorRGB, cell.Style.Foreground.Type)
	assert.Equal(t, uint8(128), cell.Style.Foreground.R)
	assert.Equal(t, uint8(64), cell.Style.Foreground.G)
	assert.Equal(t, uint8(32), cell.Style.Foreground.B)
}

func TestANSISGRDefaultBackground(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[44m\x1b[49mX")) // Set blue then default

	cell := s.Cell(0, 0)
	assert.Equal(t, ColorDefault, cell.Style.Background.Type)
}

func TestANSISGRMultipleAttributes(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1;3;4;31mX")) // Bold, italic, underline, red

	cell := s.Cell(0, 0)
	assert.True(t, cell.Style.Bold)
	assert.True(t, cell.Style.Italic)
	assert.True(t, cell.Style.Underline)
	assert.Equal(t, ColorBasic, cell.Style.Foreground.Type)
	assert.Equal(t, uint8(1), cell.Style.Foreground.Value) // Red
}

func TestANSISGREmptyReset(t *testing.T) {
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[1mBold\x1b[m")) // ESC[m without param = reset
	s.Write([]byte("Normal"))

	assert.True(t, s.Cell(0, 0).Style.Bold)  // 'B' is bold
	assert.False(t, s.Cell(4, 0).Style.Bold) // 'N' is not
}

func TestANSIInsertCharsAtEnd(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("ABCDEFGHIJ")
	s.SetCursor(8, 0)
	s.Write([]byte("\x1b[5@")) // Insert 5 chars (more than remaining space)

	// Characters should shift but wrap is limited
	row := s.Row(0)
	assert.Contains(t, row, "ABCDEFGH")
}

func TestANSIDeleteCharsAtEnd(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("ABCDEFGHIJ")
	s.SetCursor(8, 0)
	s.Write([]byte("\x1b[5P")) // Delete 5 chars (more than remaining)

	// When deleting more chars than available at cursor position,
	// the blanks fill from width-n position, clearing more content
	row := s.Row(0)
	assert.Equal(t, "ABCDE", row)
}

func TestANSIEraseCharsAtEnd(t *testing.T) {
	s := NewScreen(10, 5)
	s.WriteString("ABCDEFGHIJ")
	s.SetCursor(8, 0)
	s.Write([]byte("\x1b[5X")) // Erase 5 chars (more than remaining)

	row := s.Row(0)
	// Only last 2 chars should be erased
	assert.Equal(t, "ABCDEFGH", row)
}

func TestANSIInsertLinesAtBottom(t *testing.T) {
	s := NewScreen(10, 3)
	s.WriteString("Line0\nLine1\nLine2")
	s.SetCursor(0, 2) // Bottom line
	s.Write([]byte("\x1b[2L"))

	// Inserting lines at bottom pushes content off
	assert.Equal(t, "Line0", s.Row(0))
	assert.Equal(t, "Line1", s.Row(1))
	assert.Equal(t, "", s.Row(2))
}

func TestANSIDeleteLinesAtBottom(t *testing.T) {
	s := NewScreen(10, 3)
	s.WriteString("Line0\nLine1\nLine2")
	s.SetCursor(0, 2)
	s.Write([]byte("\x1b[1M"))

	// Delete at bottom just clears
	assert.Equal(t, "Line0", s.Row(0))
	assert.Equal(t, "Line1", s.Row(1))
	assert.Equal(t, "", s.Row(2))
}

func TestANSIWindowManipulation(t *testing.T) {
	// ESC[t commands should be ignored
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[8;50;100tHello"))
	assert.Equal(t, "Hello", s.Row(0))
}

func TestANSIDeviceStatusReport(t *testing.T) {
	// ESC[n commands should be ignored
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[6nHello"))
	assert.Equal(t, "Hello", s.Row(0))
}

func TestANSIDeviceAttributes(t *testing.T) {
	// ESC[c commands should be ignored
	s := NewScreen(20, 5)
	s.Write([]byte("\x1b[cHello"))
	assert.Equal(t, "Hello", s.Row(0))
}

func TestANSISetResetMode(t *testing.T) {
	s := NewScreen(20, 5)
	// Various mode commands should be parsed without error
	s.Write([]byte("\x1b[?1hHello"))  // Set mode
	s.Write([]byte("\x1b[?1l"))       // Reset mode
	s.Write([]byte("\x1b[?1049h"))    // Alternate screen
	s.Write([]byte("\x1b[?1049l"))    // Back to main screen
	assert.Contains(t, s.Row(0), "Hello")
}

func TestANSIScrollRegion(t *testing.T) {
	// ESC[r should be parsed (currently ignored)
	s := NewScreen(20, 10)
	s.Write([]byte("\x1b[2;8rHello"))
	assert.Equal(t, "Hello", s.Row(0))
}

func TestANSIUTF8InSequences(t *testing.T) {
	s := NewScreen(30, 5)
	// Mix UTF-8 with ANSI sequences
	s.Write([]byte("\x1b[1m日本語\x1b[0mABC"))

	assert.Contains(t, s.Row(0), "日本語")
	assert.Contains(t, s.Row(0), "ABC")
	assert.True(t, s.Cell(0, 0).Style.Bold)
}

func TestANSIRapidSequences(t *testing.T) {
	s := NewScreen(80, 24)
	// Many rapid cursor movements
	for i := 0; i < 100; i++ {
		s.Write([]byte("\x1b[H\x1b[2J")) // Clear and home
		s.Write([]byte("X"))
	}
	// Should end with just "X" at home
	assert.Equal(t, "X", s.Row(0))
}

func TestParseParams(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		result := parseParams("")
		assert.Nil(t, result)
	})

	t.Run("single param", func(t *testing.T) {
		result := parseParams("5")
		assert.Equal(t, []int{5}, result)
	})

	t.Run("multiple params", func(t *testing.T) {
		result := parseParams("1;2;3")
		assert.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("with private mode prefix", func(t *testing.T) {
		result := parseParams("?25")
		assert.Equal(t, []int{25}, result)
	})

	t.Run("empty params become 0", func(t *testing.T) {
		result := parseParams(";5;")
		assert.Equal(t, 3, len(result))
		assert.Equal(t, 0, result[0])
		assert.Equal(t, 5, result[1])
		assert.Equal(t, 0, result[2])
	})
}

func TestGetParam(t *testing.T) {
	args := []int{0, 5, 10}

	assert.Equal(t, 1, getParam(args, 0, 1))   // 0 treated as default
	assert.Equal(t, 5, getParam(args, 1, 1))   // Use actual value
	assert.Equal(t, 10, getParam(args, 2, 1))  // Use actual value
	assert.Equal(t, 99, getParam(args, 3, 99)) // Out of bounds uses default
}
