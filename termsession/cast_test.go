package termsession

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestLoadCastFile(t *testing.T) {
	// Create a test cast file
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress: false,
		Title:    "Test Cast",
	})
	assert.NoError(t, err)

	r.RecordOutput("Hello, World!\n")
	r.RecordOutput("Line 2\n")
	err = r.Close()
	assert.NoError(t, err)

	// Load it back
	header, events, err := LoadCastFile(filename)
	assert.NoError(t, err)

	assert.Equal(t, 80, header.Width)
	assert.Equal(t, 24, header.Height)
	assert.Equal(t, "Test Cast", header.Title)
	assert.Equal(t, 2, header.Version)

	assert.Len(t, events, 2)
	assert.Equal(t, "o", events[0].Type)
	assert.Equal(t, "Hello, World!\n", events[0].Data)
	assert.Equal(t, "Line 2\n", events[1].Data)
}

func TestLoadCastFile_Gzip(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "test.cast")

	r, err := NewRecorder(filename, 80, 24, RecordingOptions{
		Compress: true,
	})
	assert.NoError(t, err)

	r.RecordOutput("Compressed content\n")
	err = r.Close()
	assert.NoError(t, err)

	// Load compressed file
	header, events, err := LoadCastFile(filename)
	assert.NoError(t, err)

	assert.Equal(t, 80, header.Width)
	assert.Len(t, events, 1)
	assert.Equal(t, "Compressed content\n", events[0].Data)
}

func TestLoadCastFile_NotFound(t *testing.T) {
	_, _, err := LoadCastFile("/nonexistent/file.cast")
	assert.Error(t, err)
}

func TestLoadCastFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "invalid.cast")

	err := os.WriteFile(filename, []byte("not json"), 0644)
	assert.NoError(t, err)

	_, _, err = LoadCastFile(filename)
	assert.Error(t, err)
}

func TestDuration(t *testing.T) {
	events := []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "a"},
		{Time: 1.5, Type: "o", Data: "b"},
		{Time: 3.0, Type: "o", Data: "c"},
	}

	d := Duration(events)
	assert.Equal(t, 3.0, d)
}

func TestDuration_Empty(t *testing.T) {
	var events []RecordingEvent
	d := Duration(events)
	assert.Equal(t, 0.0, d)
}

func TestOutputEvents(t *testing.T) {
	events := []RecordingEvent{
		{Time: 0.0, Type: "o", Data: "output1"},
		{Time: 0.5, Type: "i", Data: "input1"},
		{Time: 1.0, Type: "o", Data: "output2"},
		{Time: 1.5, Type: "i", Data: "input2"},
	}

	output := OutputEvents(events)

	assert.Len(t, output, 2)
	assert.Equal(t, "output1", output[0].Data)
	assert.Equal(t, "output2", output[1].Data)
}
