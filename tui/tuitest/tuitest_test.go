package tuitest_test

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/termtest"
	"github.com/deepnoodle-ai/wonton/tui"
	"github.com/deepnoodle-ai/wonton/tui/tuitest"
)

func TestScreen_SizesToView(t *testing.T) {
	view := tui.Stack(tui.Text("one"), tui.Text("two"))
	screen := tuitest.Screen(view, tui.WithWidth(20))
	w, h := screen.Size()
	assert.Equal(t, w, 20)
	assert.Equal(t, h, 3) // two rows and the trailing newline
	termtest.AssertRow(t, screen, 0, "one")
	termtest.AssertRow(t, screen, 1, "two")
}

func TestScreen_DefaultsToEightyColumns(t *testing.T) {
	w, _ := tuitest.Screen(tui.Text("x")).Size()
	assert.Equal(t, w, 80)
}

func TestScreen_FixedHeight(t *testing.T) {
	_, h := tuitest.Screen(tui.Text("x"), tui.WithWidth(10), tui.WithHeight(5)).Size()
	assert.Equal(t, h, 5)
}

func TestScreen_ColorOnByDefault(t *testing.T) {
	view := tui.Text("hi").Fg(tui.ColorRed)
	screen := tuitest.Screen(view, tui.WithWidth(10))
	assert.Equal(t, screen.Cell(0, 0).Style.Foreground.Type, termtest.ColorBasic)

	plain := tuitest.Screen(view, tui.WithWidth(10), tui.WithColor(false))
	assert.Equal(t, plain.Cell(0, 0).Style.Foreground.Type, termtest.ColorDefault)
}
