package tui

import (
	"image"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestDebugInfo(t *testing.T) {
	info := NewDebugInfo()
	assert.NotNil(t, info)
	assert.NotNil(t, info.Custom)
}

func TestDebugInfoUpdate(t *testing.T) {
	info := NewDebugInfo()

	// Test KeyEvent
	info.Update(KeyEvent{Rune: 'a'})
	assert.Contains(t, info.LastEvent, "'a'")
	assert.Equal(t, uint64(1), info.EventCount)

	// Test ResizeEvent
	info.Update(ResizeEvent{Width: 80, Height: 24})
	assert.Equal(t, image.Pt(80, 24), info.TerminalSize)

	// Test MouseEvent
	info.Update(MouseEvent{Type: MouseClick, X: 10, Y: 5})
	assert.Contains(t, info.LastEvent, "Mouse")
}

func TestDebugInfoCustomValues(t *testing.T) {
	info := NewDebugInfo()

	info.Set("custom", "value")
	assert.Equal(t, "value", info.Custom["custom"])

	info.Clear("custom")
	_, exists := info.Custom["custom"]
	assert.False(t, exists)
}

func TestDebugViewSize(t *testing.T) {
	info := NewDebugInfo()
	info.FPS = 60.0
	info.FrameCount = 100

	view := Debug(info)
	w, h := view.size(100, 100)

	assert.True(t, w > 0, "width should be positive")
	assert.True(t, h > 0, "height should be positive")
}

func TestDebugViewPosition(t *testing.T) {
	info := NewDebugInfo()
	view := Debug(info)

	view.Position(DebugTopLeft)
	assert.Equal(t, DebugTopLeft, view.position)

	view.Position(DebugBottomRight)
	assert.Equal(t, DebugBottomRight, view.position)
}

func TestDebugViewBuildLines(t *testing.T) {
	info := NewDebugInfo()
	info.FPS = 30.0
	info.FrameCount = 500
	info.EventCount = 100
	info.Set("mode", "debug")

	view := Debug(info)
	lines := view.buildLines()

	assert.True(t, len(lines) >= 3, "should have at least FPS, Frame, Events lines")

	// Check that custom values are included
	found := false
	for _, line := range lines {
		if line == "mode: debug" {
			found = true
			break
		}
	}
	assert.True(t, found, "custom values should be in lines")
}

func TestDebugViewNilInfo(t *testing.T) {
	view := Debug(nil)
	lines := view.buildLines()
	assert.Len(t, lines, 1)
	assert.Contains(t, lines[0], "no info")
}

func TestDebugWrapper(t *testing.T) {
	app := &testDebugApp{}
	wrapper := WrapWithDebug(app)

	assert.NotNil(t, wrapper.Info)
	assert.True(t, wrapper.enabled)

	// Test View is passed through
	view := wrapper.View()
	assert.NotNil(t, view)

	// Check FPS is calculated after multiple calls
	time.Sleep(10 * time.Millisecond)
	wrapper.View()
	assert.True(t, wrapper.Info.FrameTime > 0)
}

func TestDebugWrapperHandleEvent(t *testing.T) {
	app := &testDebugApp{}
	wrapper := WrapWithDebug(app)

	cmds := wrapper.HandleEvent(KeyEvent{Rune: 'x'})
	assert.NotNil(t, cmds)
	assert.Len(t, cmds, 1)

	assert.Equal(t, uint64(1), wrapper.Info.EventCount)
}

func TestDebugWrapperDisable(t *testing.T) {
	app := &testDebugApp{}
	wrapper := WrapWithDebug(app)

	wrapper.Enable(false)

	// Events should not be tracked when disabled
	wrapper.HandleEvent(KeyEvent{Rune: 'a'})
	assert.Equal(t, uint64(0), wrapper.Info.EventCount)
}

// testDebugApp is a simple app for testing the debug wrapper.
type testDebugApp struct{}

func (a *testDebugApp) View() View {
	return Text("test")
}

func (a *testDebugApp) HandleEvent(event Event) []Cmd {
	return []Cmd{func() Event { return nil }}
}
