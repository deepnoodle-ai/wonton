package tui

import (
	"bytes"
	"image"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNewRenderContext(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 42)

	assert.NotNil(t, ctx)
	assert.Equal(t, uint64(42), ctx.Frame())
	w, h := ctx.Size()
	assert.Equal(t, 80, w)
	assert.Equal(t, 24, h)
}

func TestRenderContext_Frame(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 100)

	assert.Equal(t, uint64(100), ctx.Frame())
}

func TestRenderContext_RenderFrame(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	rf := ctx.RenderFrame()
	assert.NotNil(t, rf)
	w, h := rf.Size()
	assert.Equal(t, 80, w)
	assert.Equal(t, 24, h)
}

func TestRenderContext_Bounds(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	bounds := ctx.Bounds()
	assert.Equal(t, image.Rect(0, 0, 80, 24), bounds)
}

func TestRenderContext_Size(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(100, 50, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	w, h := ctx.Size()
	assert.Equal(t, 100, w)
	assert.Equal(t, 50, h)
}

func TestRenderContext_SubContext(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 42)

	// Create a sub-context for a smaller region
	subCtx := ctx.SubContext(image.Rect(10, 5, 50, 15))

	assert.NotNil(t, subCtx)
	assert.Equal(t, uint64(42), subCtx.Frame()) // Frame counter preserved
	w, h := subCtx.Size()
	assert.Equal(t, 40, w) // 50-10
	assert.Equal(t, 10, h) // 15-5
}

func TestRenderContext_SubContextNested(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 1)

	// Create nested sub-contexts
	sub1 := ctx.SubContext(image.Rect(10, 10, 70, 20))
	sub2 := sub1.SubContext(image.Rect(5, 2, 55, 8))

	assert.NotNil(t, sub2)
	assert.Equal(t, uint64(1), sub2.Frame())
	w, h := sub2.Size()
	assert.Equal(t, 50, w) // 55-5
	assert.Equal(t, 6, h)  // 8-2
}

func TestRenderContext_SubContextClipping(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Create a sub-context that extends beyond parent bounds
	subCtx := ctx.SubContext(image.Rect(70, 20, 100, 30))

	// Should be clipped to parent bounds
	w, h := subCtx.Size()
	assert.Equal(t, 10, w) // clipped: 80-70
	assert.Equal(t, 4, h)  // clipped: 24-20
}

func TestRenderContext_SetCell(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Should not panic
	ctx.SetCell(5, 5, 'X', NewStyle())
}

func TestRenderContext_PrintStyled(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Should not panic
	ctx.PrintStyled(0, 0, "Hello World", NewStyle())
}

func TestRenderContext_PrintTruncated(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Should not panic
	ctx.PrintTruncated(0, 0, "Hello World", NewStyle())
}

func TestRenderContext_FillStyled(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Should not panic
	ctx.FillStyled(0, 0, 10, 5, ' ', NewStyle().WithBackground(ColorBlue))
}

func TestRenderContext_Fill(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Should not panic
	ctx.Fill('*', NewStyle())
}

func TestRenderContext_PrintHyperlink(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	link := NewHyperlink("https://example.com", "Example")
	// Should not panic
	ctx.PrintHyperlink(0, 0, link)
}

func TestRenderContext_PrintHyperlinkFallback(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	link := NewHyperlink("https://example.com", "Example")
	// Should not panic
	ctx.PrintHyperlinkFallback(0, 0, link)
}

func TestRenderContext_AbsoluteBounds(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Create sub-context
	subCtx := ctx.SubContext(image.Rect(10, 5, 50, 15))

	// Absolute bounds should reflect the position in the original frame
	bounds := subCtx.AbsoluteBounds()
	assert.NotNil(t, bounds)
}

func TestRenderContext_WithFrame(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 99)

	// Create a sub-frame and use it
	subFrame := frame.SubFrame(image.Rect(0, 0, 40, 12))
	newCtx := ctx.WithFrame(subFrame)

	assert.NotNil(t, newCtx)
	assert.Equal(t, uint64(99), newCtx.Frame()) // Frame counter preserved
	w, h := newCtx.Size()
	assert.Equal(t, 40, w)
	assert.Equal(t, 12, h)
}

func TestRenderContext_ZeroSizeSubContext(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Create a zero-size sub-context
	subCtx := ctx.SubContext(image.Rect(10, 10, 10, 10))

	w, h := subCtx.Size()
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestRenderContext_OutOfBoundsSubContext(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// Create a sub-context that is completely outside the bounds
	subCtx := ctx.SubContext(image.Rect(100, 100, 150, 150))

	w, h := subCtx.Size()
	assert.Equal(t, 0, w)
	assert.Equal(t, 0, h)
}

func TestRenderContext_SubContextPreservesFrameCount(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 12345)

	sub1 := ctx.SubContext(image.Rect(0, 0, 40, 12))
	sub2 := sub1.SubContext(image.Rect(0, 0, 20, 6))
	sub3 := sub2.SubContext(image.Rect(0, 0, 10, 3))

	// All should have the same frame count
	assert.Equal(t, uint64(12345), ctx.Frame())
	assert.Equal(t, uint64(12345), sub1.Frame())
	assert.Equal(t, uint64(12345), sub2.Frame())
	assert.Equal(t, uint64(12345), sub3.Frame())
}

// Focus manager tests for RenderContext

func TestRenderContext_FocusManager_NilByDefault(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	ctx := NewRenderContext(frame, 0)

	// FocusManager should be nil by default
	assert.Nil(t, ctx.FocusManager())
}

func TestRenderContext_WithFocusManager(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	fm := NewFocusManager()
	ctx := NewRenderContext(frame, 42).WithFocusManager(fm)

	assert.NotNil(t, ctx.FocusManager())
	assert.Equal(t, fm, ctx.FocusManager())
	assert.Equal(t, uint64(42), ctx.Frame()) // Other fields preserved
}

func TestRenderContext_SubContextPreservesFocusManager(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	fm := NewFocusManager()
	ctx := NewRenderContext(frame, 0).WithFocusManager(fm)

	sub1 := ctx.SubContext(image.Rect(0, 0, 40, 12))
	sub2 := sub1.SubContext(image.Rect(0, 0, 20, 6))

	// All contexts should share the same focus manager
	assert.Equal(t, fm, ctx.FocusManager())
	assert.Equal(t, fm, sub1.FocusManager())
	assert.Equal(t, fm, sub2.FocusManager())
}

func TestRenderContext_WithFramePreservesFocusManager(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	fm := NewFocusManager()
	ctx := NewRenderContext(frame, 99).WithFocusManager(fm)

	subFrame := frame.SubFrame(image.Rect(0, 0, 40, 12))
	newCtx := ctx.WithFrame(subFrame)

	// Focus manager should be preserved
	assert.Equal(t, fm, newCtx.FocusManager())
	assert.Equal(t, uint64(99), newCtx.Frame()) // Frame count also preserved
}

func TestRenderContext_FocusManagerUsage(t *testing.T) {
	var buf bytes.Buffer
	terminal := NewTestTerminal(80, 24, &buf)
	frame, err := terminal.BeginFrame()
	assert.NoError(t, err)
	defer terminal.EndFrame(frame)

	fm := NewFocusManager()
	ctx := NewRenderContext(frame, 0).WithFocusManager(fm)

	// Verify the focus manager can be used through the context
	mock := &mockFocusableForCtxTest{id: "test"}
	ctx.FocusManager().Register(mock)

	assert.Equal(t, "test", ctx.FocusManager().GetFocusedID())
	assert.True(t, mock.focused)
}

// mockFocusableForCtxTest is a simple focusable for context tests
type mockFocusableForCtxTest struct {
	id      string
	focused bool
}

func (m *mockFocusableForCtxTest) FocusID() string                { return m.id }
func (m *mockFocusableForCtxTest) IsFocused() bool                { return m.focused }
func (m *mockFocusableForCtxTest) SetFocused(focused bool)        { m.focused = focused }
func (m *mockFocusableForCtxTest) FocusBounds() image.Rectangle   { return image.Rect(0, 0, 10, 10) }
func (m *mockFocusableForCtxTest) HandleKeyEvent(e KeyEvent) bool { return false }
