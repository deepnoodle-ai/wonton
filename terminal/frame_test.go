package terminal

import (
	"bytes"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/require"
)

func TestNewFrame(t *testing.T) {
	frame := NewFrame(10, 5, 20, 10)
	assert.Equal(t, 10, frame.X)
	assert.Equal(t, 5, frame.Y)
	assert.Equal(t, 20, frame.Width)
	assert.Equal(t, 10, frame.Height)
	assert.Equal(t, SingleBorder, frame.Border)
	assert.Equal(t, AlignLeft, frame.TitleAlign)
}

func TestFrame_WithBorderStyle(t *testing.T) {
	frame := NewFrame(0, 0, 10, 10).WithBorderStyle(DoubleBorder)
	assert.Equal(t, DoubleBorder, frame.Border)
}

func TestFrame_WithColor(t *testing.T) {
	style := NewStyle().WithForeground(ColorRed)
	frame := NewFrame(0, 0, 10, 10).WithColor(style)
	assert.Equal(t, ColorRed, frame.BorderStyle.Foreground)
}

func TestFrame_WithTitle(t *testing.T) {
	frame := NewFrame(0, 0, 10, 10).WithTitle("Test", AlignCenter)
	assert.Equal(t, "Test", frame.Title)
	assert.Equal(t, AlignCenter, frame.TitleAlign)
}

func TestFrame_WithTitleStyle(t *testing.T) {
	style := NewStyle().WithBold()
	frame := NewFrame(0, 0, 10, 10).WithTitleStyle(style)
	assert.True(t, frame.TitleStyle.Bold)
}

func TestFrame_Draw_TooSmall(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	// Width < 2 should not draw
	frame := NewFrame(0, 0, 1, 10)
	require.NotPanics(t, func() {
		frame.Draw(renderFrame)
	})

	// Height < 2 should not draw
	frame = NewFrame(0, 0, 10, 1)
	require.NotPanics(t, func() {
		frame.Draw(renderFrame)
	})

	term.EndFrame(renderFrame)
}

func TestFrame_Draw_BasicBorder(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 5, 3)
	frame.Draw(renderFrame)

	term.EndFrame(renderFrame)

	// Check corners
	assert.Equal(t, '┌', term.backBuffer[0][0].Char)
	assert.Equal(t, '┐', term.backBuffer[0][4].Char)
	assert.Equal(t, '└', term.backBuffer[2][0].Char)
	assert.Equal(t, '┘', term.backBuffer[2][4].Char)

	// Check sides
	assert.Equal(t, '│', term.backBuffer[1][0].Char)
	assert.Equal(t, '│', term.backBuffer[1][4].Char)

	// Check horizontal
	assert.Equal(t, '─', term.backBuffer[0][1].Char)
	assert.Equal(t, '─', term.backBuffer[2][1].Char)
}

func TestFrame_Draw_DoubleBorder(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 5, 3).WithBorderStyle(DoubleBorder)
	frame.Draw(renderFrame)

	term.EndFrame(renderFrame)

	// Check double border corners
	assert.Equal(t, '╔', term.backBuffer[0][0].Char)
	assert.Equal(t, '╗', term.backBuffer[0][4].Char)
	assert.Equal(t, '╚', term.backBuffer[2][0].Char)
	assert.Equal(t, '╝', term.backBuffer[2][4].Char)
}

func TestFrame_Draw_RoundedBorder(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 5, 3).WithBorderStyle(RoundedBorder)
	frame.Draw(renderFrame)

	term.EndFrame(renderFrame)

	// Check rounded corners
	assert.Equal(t, '╭', term.backBuffer[0][0].Char)
	assert.Equal(t, '╮', term.backBuffer[0][4].Char)
	assert.Equal(t, '╰', term.backBuffer[2][0].Char)
	assert.Equal(t, '╯', term.backBuffer[2][4].Char)
}

func TestFrame_Draw_ThickBorder(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 5, 3).WithBorderStyle(ThickBorder)
	frame.Draw(renderFrame)

	term.EndFrame(renderFrame)

	// Check thick border corners
	assert.Equal(t, '┏', term.backBuffer[0][0].Char)
	assert.Equal(t, '┓', term.backBuffer[0][4].Char)
	assert.Equal(t, '┗', term.backBuffer[2][0].Char)
	assert.Equal(t, '┛', term.backBuffer[2][4].Char)
}

func TestFrame_Draw_ASCIIBorder(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 5, 3).WithBorderStyle(ASCIIBorder)
	frame.Draw(renderFrame)

	term.EndFrame(renderFrame)

	// Check ASCII border
	assert.Equal(t, '+', term.backBuffer[0][0].Char)
	assert.Equal(t, '+', term.backBuffer[0][4].Char)
	assert.Equal(t, '+', term.backBuffer[2][0].Char)
	assert.Equal(t, '+', term.backBuffer[2][4].Char)
	assert.Equal(t, '|', term.backBuffer[1][0].Char)
	assert.Equal(t, '-', term.backBuffer[0][1].Char)
}

func TestFrame_Draw_WithLeftAlignedTitle(t *testing.T) {
	term := NewTestTerminal(30, 10, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 15, 3).WithTitle("Test", AlignLeft)
	frame.Draw(renderFrame)

	term.EndFrame(renderFrame)

	// Title should appear near the start
	// Format is: "┌ Test ──────┐" (corner, space, title, space, dashes, corner)
	assert.Equal(t, '┌', term.backBuffer[0][0].Char)
	assert.Equal(t, ' ', term.backBuffer[0][1].Char)
	assert.Equal(t, 'T', term.backBuffer[0][2].Char)
	assert.Equal(t, 'e', term.backBuffer[0][3].Char)
	assert.Equal(t, 's', term.backBuffer[0][4].Char)
	assert.Equal(t, 't', term.backBuffer[0][5].Char)
	assert.Equal(t, ' ', term.backBuffer[0][6].Char)
}

func TestFrame_Draw_WithCenteredTitle(t *testing.T) {
	term := NewTestTerminal(30, 10, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 20, 3).WithTitle("Test", AlignCenter)
	frame.Draw(renderFrame)

	term.EndFrame(renderFrame)

	// Title should be roughly centered
	// Find where 'T' appears
	titlePos := -1
	for i := 0; i < 20; i++ {
		if term.backBuffer[0][i].Char == 'T' {
			titlePos = i
			break
		}
	}
	assert.Greater(t, titlePos, 5, "Title should not be at the start")
	assert.Less(t, titlePos, 15, "Title should not be at the end")
}

func TestFrame_Draw_WithRightAlignedTitle(t *testing.T) {
	term := NewTestTerminal(30, 10, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 20, 3).WithTitle("Test", AlignRight)
	frame.Draw(renderFrame)

	term.EndFrame(renderFrame)

	// Title should be near the end
	titlePos := -1
	for i := 0; i < 20; i++ {
		if term.backBuffer[0][i].Char == 'T' {
			titlePos = i
			break
		}
	}
	assert.Greater(t, titlePos, 10, "Title should be in the right half")
}

func TestFrame_Clear(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	// First, fill the area with something
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			renderFrame.SetCell(x, y, 'X', NewStyle())
		}
	}

	frame := NewFrame(0, 0, 10, 5)
	frame.Draw(renderFrame)
	frame.Clear(renderFrame)

	term.EndFrame(renderFrame)

	// Inside should be cleared (spaces), border should remain
	assert.Equal(t, '┌', term.backBuffer[0][0].Char) // Top-left corner
	assert.Equal(t, ' ', term.backBuffer[1][1].Char) // Inside
	assert.Equal(t, ' ', term.backBuffer[2][2].Char) // Inside
	assert.Equal(t, '│', term.backBuffer[1][0].Char) // Left border
}

func TestFrame_Clear_TooSmall(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	frame := NewFrame(0, 0, 1, 5)
	require.NotPanics(t, func() {
		frame.Clear(renderFrame)
	})

	term.EndFrame(renderFrame)
}

func TestBox_NewBox(t *testing.T) {
	content := []string{"Line 1", "Line 2"}
	box := NewBox(content)
	assert.Equal(t, content, box.Content)
	assert.Equal(t, SingleBorder, box.Border)
	assert.Equal(t, 1, box.Padding)
}

func TestBox_Draw_EmptyContent(t *testing.T) {
	term := NewTestTerminal(20, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	box := NewBox([]string{})
	require.NotPanics(t, func() {
		box.Draw(renderFrame, 0, 0)
	})

	term.EndFrame(renderFrame)
}

func TestBox_Draw_BasicBox(t *testing.T) {
	term := NewTestTerminal(30, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	box := NewBox([]string{"Hello", "World"})
	box.Draw(renderFrame, 0, 0)

	term.EndFrame(renderFrame)

	// Check that border is drawn
	assert.Equal(t, '┌', term.backBuffer[0][0].Char)

	// Check content is present
	foundHello := false
	foundWorld := false
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if term.backBuffer[y][x].Char == 'H' {
				// Check if "Hello" follows
				if x+4 < 20 &&
					term.backBuffer[y][x+1].Char == 'e' &&
					term.backBuffer[y][x+2].Char == 'l' &&
					term.backBuffer[y][x+3].Char == 'l' &&
					term.backBuffer[y][x+4].Char == 'o' {
					foundHello = true
				}
			}
			if term.backBuffer[y][x].Char == 'W' {
				// Check if "World" follows
				if x+4 < 20 &&
					term.backBuffer[y][x+1].Char == 'o' &&
					term.backBuffer[y][x+2].Char == 'r' &&
					term.backBuffer[y][x+3].Char == 'l' &&
					term.backBuffer[y][x+4].Char == 'd' {
					foundWorld = true
				}
			}
		}
	}
	assert.True(t, foundHello, "Should find 'Hello' in the box")
	assert.True(t, foundWorld, "Should find 'World' in the box")
}

func TestBox_Draw_WithPadding(t *testing.T) {
	term := NewTestTerminal(30, 20, &bytes.Buffer{})
	renderFrame, _ := term.BeginFrame()

	box := NewBox([]string{"Test"})
	box.Padding = 2
	box.Draw(renderFrame, 0, 0)

	term.EndFrame(renderFrame)

	// With padding=2, content should be further from borders
	// Find the 'T' and check it's not immediately after the border
	foundTest := false
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if term.backBuffer[y][x].Char == 'T' {
				// Should have padding (spaces) before it
				assert.Greater(t, x, 2, "Content should be offset by padding")
				foundTest = true
				break
			}
		}
	}
	assert.True(t, foundTest)
}

func TestAlignment_Constants(t *testing.T) {
	assert.Equal(t, Alignment(0), AlignLeft)
	assert.Equal(t, Alignment(1), AlignCenter)
	assert.Equal(t, Alignment(2), AlignRight)
}

func TestBorderStyle_SingleBorder(t *testing.T) {
	assert.Equal(t, "┌", SingleBorder.TopLeft)
	assert.Equal(t, "┐", SingleBorder.TopRight)
	assert.Equal(t, "└", SingleBorder.BottomLeft)
	assert.Equal(t, "┘", SingleBorder.BottomRight)
	assert.Equal(t, "─", SingleBorder.Horizontal)
	assert.Equal(t, "│", SingleBorder.Vertical)
}

func TestBorderStyle_DoubleBorder(t *testing.T) {
	assert.Equal(t, "╔", DoubleBorder.TopLeft)
	assert.Equal(t, "╗", DoubleBorder.TopRight)
	assert.Equal(t, "╚", DoubleBorder.BottomLeft)
	assert.Equal(t, "╝", DoubleBorder.BottomRight)
	assert.Equal(t, "═", DoubleBorder.Horizontal)
	assert.Equal(t, "║", DoubleBorder.Vertical)
}

func TestBorderStyle_RoundedBorder(t *testing.T) {
	assert.Equal(t, "╭", RoundedBorder.TopLeft)
	assert.Equal(t, "╮", RoundedBorder.TopRight)
	assert.Equal(t, "╰", RoundedBorder.BottomLeft)
	assert.Equal(t, "╯", RoundedBorder.BottomRight)
}

func TestBorderStyle_ThickBorder(t *testing.T) {
	assert.Equal(t, "┏", ThickBorder.TopLeft)
	assert.Equal(t, "┓", ThickBorder.TopRight)
	assert.Equal(t, "┗", ThickBorder.BottomLeft)
	assert.Equal(t, "┛", ThickBorder.BottomRight)
	assert.Equal(t, "━", ThickBorder.Horizontal)
	assert.Equal(t, "┃", ThickBorder.Vertical)
}

func TestBorderStyle_ASCIIBorder(t *testing.T) {
	assert.Equal(t, "+", ASCIIBorder.TopLeft)
	assert.Equal(t, "+", ASCIIBorder.TopRight)
	assert.Equal(t, "+", ASCIIBorder.BottomLeft)
	assert.Equal(t, "+", ASCIIBorder.BottomRight)
	assert.Equal(t, "-", ASCIIBorder.Horizontal)
	assert.Equal(t, "|", ASCIIBorder.Vertical)
}
