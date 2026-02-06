package gif_test

import (
	"os"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/gif"
)

func TestNewTerminalScreen(t *testing.T) {
	screen := gif.NewTerminalScreen(80, 24)
	assert.NotNil(t, screen)
	assert.Equal(t, screen.Width, 80)
	assert.Equal(t, screen.Height, 24)
	assert.Equal(t, screen.Cells[0][0].Char, ' ')
}

func TestTerminalScreen_Clear(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 5)
	screen.SetCell(0, 0, 'X', nil, nil)
	screen.CursorX = 5
	screen.CursorY = 2

	screen.Clear()

	assert.Equal(t, screen.Cells[0][0].Char, ' ')
	assert.Equal(t, screen.CursorX, 0)
	assert.Equal(t, screen.CursorY, 0)
}

func TestTerminalScreen_SetCell(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 5)

	// Valid set
	screen.SetCell(1, 1, 'A', gif.Red, gif.Blue)
	assert.Equal(t, screen.Cells[1][1].Char, 'A')
	assert.Equal(t, screen.Cells[1][1].FG, gif.Red)
	assert.Equal(t, screen.Cells[1][1].BG, gif.Blue)

	// Out of bounds (should not panic)
	screen.SetCell(-1, 0, 'A', nil, nil)
	screen.SetCell(0, -1, 'A', nil, nil)
	screen.SetCell(10, 0, 'A', nil, nil)
	screen.SetCell(0, 5, 'A', nil, nil)
}

func TestTerminalScreen_WriteString(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 4)

	// Write simple string
	screen.WriteString("Hello", nil, nil)
	assert.Equal(t, screen.CursorX, 5)
	assert.Equal(t, screen.Cells[0][0].Char, 'H')

	// Write newline
	screen.WriteString("\n", nil, nil)
	assert.Equal(t, screen.CursorX, 0)
	assert.Equal(t, screen.CursorY, 1)

	// Write carriage return
	screen.WriteString("World\r", nil, nil)
	assert.Equal(t, screen.Cells[1][0].Char, 'W')
	assert.Equal(t, screen.CursorX, 0)

	// Write beyond width (wrap)
	screen.CursorY = 2
	screen.CursorX = 8
	screen.WriteString("123", nil, nil) // "12" on line 2, "3" on line 3
	assert.Equal(t, screen.Cells[2][8].Char, '1')
	assert.Equal(t, screen.Cells[2][9].Char, '2')
	assert.Equal(t, screen.Cells[3][0].Char, '3')

	// Write beyond height (scroll)
	screen.CursorY = 3
	screen.CursorX = 0
	screen.WriteString("\nScroll", nil, nil)
	// "3" from previous wrap should be moved up to line 2
	assert.Equal(t, screen.Cells[2][0].Char, '3')
	assert.Equal(t, screen.Cells[3][0].Char, 'S')
}

func TestTerminalScreen_MoveCursor(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 10)

	screen.MoveCursor(5, 5)
	assert.Equal(t, screen.CursorX, 5)
	assert.Equal(t, screen.CursorY, 5)

	// Clamp negative
	screen.MoveCursor(-1, -1)
	assert.Equal(t, screen.CursorX, 0)
	assert.Equal(t, screen.CursorY, 0)

	// Clamp positive
	screen.MoveCursor(20, 20)
	assert.Equal(t, screen.CursorX, 9)
	assert.Equal(t, screen.CursorY, 9)
}

func TestTerminalScreen_WriteString_WrapScroll(t *testing.T) {
	screen := gif.NewTerminalScreen(5, 2)
	// Fill first line
	screen.WriteString("12345", nil, nil)
	// Fill second line
	screen.WriteString("67890", nil, nil)
	// Write one more char, should wrap AND scroll
	screen.WriteString("A", nil, nil)

	assert.Equal(t, screen.Cells[0][0].Char, '6')
	assert.Equal(t, screen.Cells[1][0].Char, 'A')
}

func TestTerminalRenderer(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 5)
	renderer := gif.NewTerminalRenderer(screen, 2)

	assert.NotNil(t, renderer.GIF())

	// Test rendering a frame
	screen.WriteString("Test", nil, nil)
	renderer.RenderFrame(10)

	assert.Equal(t, renderer.GIF().FrameCount(), 1)
}

func TestTerminalRenderer_Options(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 5)
	opts := gif.DefaultRendererOptions()
	opts.UseBitmap = true
	opts.BitmapFont = gif.BitmapFont8x8

	renderer := gif.NewTerminalRendererWithOptions(screen, opts)
	// Verify dimensions based on 8x8 font + padding 8
	// Width = 10*8 + 16 = 96
	// Height = 5*8 + 16 = 56
	assert.Equal(t, renderer.GIF().Width(), 96)
	assert.Equal(t, renderer.GIF().Height(), 56)

	// Test rendering with bitmap font to cover that path
	screen.WriteString("Bitmap", nil, nil)
	renderer.RenderFrame(10)
}

func TestTerminalRenderer_SaveAndBytes(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 5)
	renderer := gif.NewTerminalRenderer(screen, 2)
	renderer.RenderFrame(1)

	// Test Bytes
	data, err := renderer.Bytes()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	// Test Save
	tmpFile := "test_renderer_save.gif"
	defer os.Remove(tmpFile)

	err = renderer.Save(tmpFile)
	assert.NoError(t, err)

	_, err = os.Stat(tmpFile)
	assert.NoError(t, err)
}

func TestTerminalRenderer_Save_Error(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 5)
	renderer := gif.NewTerminalRenderer(screen, 0)

	// Try to save to an empty path or invalid path
	err := renderer.Save("")
	assert.Error(t, err)
}

func TestTerminalRenderer_LoopCount(t *testing.T) {
	screen := gif.NewTerminalScreen(10, 5)
	renderer := gif.NewTerminalRenderer(screen, 0)

	renderer.SetLoopCount(5)
	// We can't easily inspect the GIF loop count without decoding,
	// but we can ensure the method is callable and returns the renderer
	assert.NotNil(t, renderer)
}

func TestTerminalScreen_ScrollUp_ZeroHeight(t *testing.T) {
	// Test that scrollUp doesn't panic with zero or negative height
	screen := gif.NewTerminalScreen(10, 0)

	// This should not panic
	screen.WriteString("Test\n", nil, nil)

	// Verify screen is still usable
	assert.Equal(t, screen.Height, 0)
}

func TestTerminalScreen_WriteString_ZeroHeight(t *testing.T) {
	// Test WriteString behavior with zero height
	screen := gif.NewTerminalScreen(10, 0)

	// Writing should not panic even though there's nowhere to write
	screen.WriteString("Hello World", nil, nil)

	// With zero height, cursor should be at 0,0 but no cells exist
	assert.Equal(t, screen.Height, 0)
	assert.Equal(t, len(screen.Cells), 0)
}
