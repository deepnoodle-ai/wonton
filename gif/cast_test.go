package gif_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/gif"
	"github.com/deepnoodle-ai/wonton/termsession"
)

func TestDefaultCastOptions(t *testing.T) {
	opts := gif.DefaultCastOptions()

	assert.Equal(t, opts.Speed, 1.0)
	assert.Equal(t, opts.MaxIdle, 2.0)
	assert.Equal(t, opts.FPS, 10)
	assert.Equal(t, opts.Padding, 8)
	assert.Equal(t, opts.FontSize, 14.0)
}

func createTestCastFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "test_cast_*.cast")
	assert.NoError(t, err)

	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)

	err = tmpFile.Close()
	assert.NoError(t, err)

	return tmpFile.Name()
}

func TestGetCastInfo(t *testing.T) {
	// Prepare test data
	castContent := `{"version": 2, "width": 80, "height": 24, "timestamp": 1600000000, "title": "Test Recording"}
[0.5, "o", "Hello"]
[1.0, "o", " World"]
`
	filename := createTestCastFile(t, castContent)
	defer os.Remove(filename)

	// Test GetCastInfo
	info, err := gif.GetCastInfo(filename)
	assert.NoError(t, err)

	assert.Equal(t, info.Width, 80)
	assert.Equal(t, info.Height, 24)
	assert.Equal(t, info.Title, "Test Recording")
	assert.Equal(t, info.EventCount, 2)
	assert.InDelta(t, info.Duration, 1.0, 0.001)
}

func TestGetCastInfo_InvalidFile(t *testing.T) {
	_, err := gif.GetCastInfo("non_existent_file.cast")
	assert.Error(t, err)
}

func TestRenderCast(t *testing.T) {
	// Prepare test data
	castContent := `{"version": 2, "width": 40, "height": 10, "timestamp": 1600000000}
[0.1, "o", "First"]
[0.3, "o", "Second"]
`
	filename := createTestCastFile(t, castContent)
	defer os.Remove(filename)

	opts := gif.DefaultCastOptions()
	opts.FPS = 10 // 100ms per frame

	g, err := gif.RenderCast(filename, opts)
	assert.NoError(t, err)
	assert.NotNil(t, g)

	// Check GIF properties
	// Width = (40 * charWidth) + (2 * padding)
	// Height = (10 * charHeight) + (2 * padding)
	// We just verify it's not empty and has frames
	assert.Greater(t, g.Width(), 0)
	assert.Greater(t, g.Height(), 0)
	assert.Greater(t, g.FrameCount(), 0)
}

func TestRenderCast_InvalidFile(t *testing.T) {
	opts := gif.DefaultCastOptions()
	_, err := gif.RenderCast("non_existent_file.cast", opts)
	assert.Error(t, err)
}

func TestRenderCastEvents(t *testing.T) {
	header := &termsession.RecordingHeader{
		Version: 2,
		Width:   20,
		Height:  5,
	}

	events := []termsession.RecordingEvent{
		{Time: 0.1, Type: "o", Data: "A"},
		{Time: 0.2, Type: "o", Data: "B"},
		{Time: 10.0, Type: "o", Data: "C"}, // Large gap to test MaxIdle
	}

	opts := gif.DefaultCastOptions()
	opts.MaxIdle = 1.0
	opts.FPS = 10
	opts.Cols = 30 // Override width
	opts.Rows = 10 // Override height

	g, err := gif.RenderCastEvents(header, events, opts)
	assert.NoError(t, err)
	assert.NotNil(t, g)

	// Verify MaxIdle effect
	// Gap of 9.8s should be reduced to 1.0s
	// Total duration should be roughly 0.1 + 0.1 + 1.0 = 1.2s
	// At 10 FPS, that's roughly 12 frames
	assert.Greater(t, g.FrameCount(), 0)

	// Test validation of options
	opts.Speed = -1
	opts.MaxIdle = -1
	opts.FPS = -1
	opts.FontSize = -1

	g2, err := gif.RenderCastEvents(header, events, opts)
	assert.NoError(t, err)
	assert.NotNil(t, g2)
}

func TestRenderCast_EmptyEvents(t *testing.T) {
	header := &termsession.RecordingHeader{
		Version: 2,
		Width:   20,
		Height:  5,
	}
	events := []termsession.RecordingEvent{}

	opts := gif.DefaultCastOptions()
	g, err := gif.RenderCastEvents(header, events, opts)
	assert.NoError(t, err)

	// Should produce at least one frame
	assert.Equal(t, g.FrameCount(), 1)
}

// Custom marshaling check
func TestRecordingEventJSON(t *testing.T) {
	event := termsession.RecordingEvent{
		Time: 1.5,
		Type: "o",
		Data: "test",
	}

	data, err := json.Marshal(event)
	assert.NoError(t, err)

	// Should be [1.5,"o","test"]
	expected := `[1.5,"o","test"]`
	assert.Equal(t, string(data), expected)
}
