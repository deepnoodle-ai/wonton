package terminal

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func drawFrame(t *testing.T, term *Terminal, draw func(f RenderFrame)) {
	t.Helper()
	frame, err := term.BeginFrame()
	assert.NoError(t, err)
	draw(frame)
	assert.NoError(t, term.EndFrame(frame))
}

func rowChars(term *Terminal, y, n int) string {
	var b strings.Builder
	for x := 0; x < n; x++ {
		c := term.GetCell(x, y)
		if c.Continuation {
			continue
		}
		b.WriteRune(c.Char)
	}
	return b.String()
}

func TestPrintTabTakesNoCell(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(20, 2, &out)
	drawFrame(t, term, func(f RenderFrame) {
		f.PrintStyled(0, 0, "a\tb", NewStyle())
		f.PrintStyled(0, 1, "a\t", NewStyle())
		f.SetCell(2, 1, 'X', NewStyle())
	})
	assert.Equal(t, "ab", rowChars(term, 0, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 20; x++ {
			assert.True(t, term.GetCell(x, y).Char != '\t', "tab stored in cell (%d,%d)", x, y)
		}
	}
	// A raw tab on flush would move the real cursor to column 8.
	assert.False(t, strings.Contains(out.String(), "\t"), "flush emitted a tab: %q", out.String())
}

func TestFillClearsStraddledWideCharacters(t *testing.T) {
	term := NewTestTerminal(10, 1, &bytes.Buffer{})
	drawFrame(t, term, func(f RenderFrame) {
		f.PrintStyled(4, 0, "世", NewStyle()) // columns 4-5
		f.PrintStyled(0, 0, "界", NewStyle()) // columns 0-1
	})
	drawFrame(t, term, func(f RenderFrame) {
		f.FillStyled(1, 0, 4, 1, '.', NewStyle()) // columns 1-4
	})
	// Neither glyph survives as half a cell pair.
	assert.False(t, term.GetCell(0, 0).Width == 2, "orphaned lead at 0: %+v", term.GetCell(0, 0))
	assert.False(t, term.GetCell(5, 0).Continuation, "orphaned continuation at 5: %+v", term.GetCell(5, 0))
}

func TestFillWideCharacterOddWidth(t *testing.T) {
	term := NewTestTerminal(10, 1, &bytes.Buffer{})
	drawFrame(t, term, func(f RenderFrame) {
		f.FillStyled(0, 0, 3, 1, '世', NewStyle())
	})
	// The third column has no room for a second wide glyph.
	c := term.GetCell(2, 0)
	assert.False(t, c.Width == 2, "wide lead with no continuation at 2: %+v", c)
}

func TestSubFrameSetCellWideCharacterStaysInside(t *testing.T) {
	term := NewTestTerminal(10, 1, &bytes.Buffer{})
	drawFrame(t, term, func(f RenderFrame) {
		f.SetCell(5, 0, 'x', NewStyle())
		sub := f.SubFrame(image.Rect(0, 0, 5, 1))
		err := sub.SetCell(4, 0, '世', NewStyle())
		assert.Error(t, err)
	})
	assert.Equal(t, 'x', term.GetCell(5, 0).Char)
}

func TestRestyleContinuationCellRestylesGlyph(t *testing.T) {
	term := NewTestTerminal(10, 1, &bytes.Buffer{})
	bold := NewStyle().WithBold()
	drawFrame(t, term, func(f RenderFrame) {
		f.PrintStyled(0, 0, "世", NewStyle())
	})
	assert.NoError(t, term.RestyleCell(1, 0, bold))
	assert.True(t, term.GetCell(0, 0).Style.Equal(bold), "lead cell not restyled")
}

func TestOneColumnSubFrameSkipsWideCharacter(t *testing.T) {
	term := NewTestTerminal(10, 3, &bytes.Buffer{})
	drawFrame(t, term, func(f RenderFrame) {
		f.SetCell(1, 0, 'x', NewStyle())
		f.SetCell(1, 1, 'y', NewStyle())
		f.SubFrame(image.Rect(0, 0, 1, 3)).PrintStyled(0, 0, "世a", NewStyle())
	})
	for y := 0; y < 3; y++ {
		assert.False(t, term.GetCell(0, y).Width == 2, "wide glyph in 1-column frame at row %d", y)
	}
	assert.Equal(t, 'x', term.GetCell(1, 0).Char)
	assert.Equal(t, 'y', term.GetCell(1, 1).Char)
}

func TestPrintHyperlinkFallbackDropsStyleURL(t *testing.T) {
	var out bytes.Buffer
	term := NewTestTerminal(60, 1, &out)
	link := NewHyperlink("https://example.com", "site")
	link.Style = link.Style.WithURL("https://evil.example")
	drawFrame(t, term, func(f RenderFrame) {
		assert.NoError(t, f.PrintHyperlinkFallback(0, 0, link))
	})
	assert.False(t, strings.Contains(out.String(), "\x1b]8;"), "fallback emitted OSC 8: %q", out.String())
}

func TestFrameTitleThatFillsTopBorder(t *testing.T) {
	term := NewTestTerminal(20, 5, &bytes.Buffer{})
	f := NewFrame(0, 0, 8, 3).WithTitle("abcd", AlignLeft) // " abcd " is 6 = Width-2
	drawFrame(t, term, func(fr RenderFrame) { f.Draw(fr) })
	assert.Equal(t, "┌ abcd ┐", rowChars(term, 0, 8))
}
