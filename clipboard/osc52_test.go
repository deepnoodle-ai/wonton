package clipboard

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestWriteOSC52WrapsBase64InTheSequence(t *testing.T) {
	var buf bytes.Buffer
	assert.NoError(t, WriteOSC52(&buf, "hello"))

	assert.Equal(t, buf.String(), "\033]52;c;"+base64.StdEncoding.EncodeToString([]byte("hello"))+"\a")
}

func TestWriteOSC52CarriesTextTheTerminalWouldOtherwiseEat(t *testing.T) {
	// Newlines, escapes and non-ASCII all have to survive: base64 is what
	// makes the payload opaque to the terminal's own parser.
	text := "line one\nline two\ttabbed\n\033[31mnot a color\033[0m\né 世界"

	var buf bytes.Buffer
	assert.NoError(t, WriteOSC52(&buf, text))

	body := strings.TrimSuffix(strings.TrimPrefix(buf.String(), "\033]52;c;"), "\a")
	decoded, err := base64.StdEncoding.DecodeString(body)
	assert.NoError(t, err)
	assert.Equal(t, string(decoded), text)
}

func TestWriteOSC52RefusesOversizedText(t *testing.T) {
	var buf bytes.Buffer
	err := WriteOSC52(&buf, strings.Repeat("x", OSC52Limit+1))

	assert.True(t, errors.Is(err, ErrTooLarge), "the error must identify itself")
	assert.Equal(t, buf.Len(), 0, "nothing may be written when the text is refused")
}

func TestWriteOSC52AcceptsTextExactlyAtTheLimit(t *testing.T) {
	var buf bytes.Buffer
	assert.NoError(t, WriteOSC52(&buf, strings.Repeat("x", OSC52Limit)))
	assert.True(t, buf.Len() > OSC52Limit, "the encoded payload is larger than the text")
}

func TestWriteOSC52ReportsAWriteFailure(t *testing.T) {
	err := WriteOSC52(failingWriter{}, "hello")
	assert.Error(t, err)
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("no") }
