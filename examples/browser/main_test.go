package main

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/tui"
)

func TestURLBarEditsWholeRunes(t *testing.T) {
	app := &BrowserApp{urlInput: "héllo", urlCursor: len("hé")}
	key := func(k tui.Key) { app.handleURLBarInput(tui.KeyEvent{Key: k}) }

	key(tui.KeyBackspace)
	assert.Equal(t, app.urlInput, "hllo")
	assert.Equal(t, app.urlCursor, 1)

	app.handleURLBarInput(tui.KeyEvent{Rune: '日'})
	assert.Equal(t, app.urlInput, "h日llo")
	assert.Equal(t, app.urlCursor, len("h日"))

	key(tui.KeyArrowLeft)
	assert.Equal(t, app.urlCursor, 1)
	key(tui.KeyDelete)
	assert.Equal(t, app.urlInput, "hllo")
	key(tui.KeyArrowRight)
	assert.Equal(t, app.urlCursor, 2)
}
