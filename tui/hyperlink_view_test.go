package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestHyperlinkView_ShowURLMeasuresDisplayWidth(t *testing.T) {
	// "日本" is 2 runes, 6 bytes, and 4 columns wide.
	view := Link("https://例え.jp/日本", "go").ShowURL()
	w, h := view.size(0, 0)
	assert.Equal(t, len("go ()")+len("https://")+4+len(".jp/")+4, w)
	assert.Equal(t, 1, h)
}
