// Package tuitest renders tui views to a termtest.Screen for tests.
//
// It is a separate package so that the core tui package, and every program
// that imports it, does not depend on termtest.
//
// Example:
//
//	func TestButton(t *testing.T) {
//	    btn := tui.Button("Submit", func() {})
//	    screen := tuitest.Screen(btn, tui.WithWidth(20))
//	    termtest.AssertRowContains(t, screen, 0, "Submit")
//	}
package tuitest

import (
	"github.com/deepnoodle-ai/wonton/termtest"
	"github.com/deepnoodle-ai/wonton/tui"
)

// Screen renders view as tui.Sprint does and returns the result on a
// termtest.Screen for assertions. Color is on unless opts turn it off, so
// tests can check styles. The screen is as wide as tui.WithWidth sets, or 80
// columns. It is as high as tui.WithHeight sets, or the view's height plus
// one line for the trailing newline. With tui.WithRawMode(true), the screen
// models a raw-mode terminal (see termtest.WithRawMode), so a line that does
// not start at column 0 shows.
func Screen(view tui.View, opts ...tui.PrintOption) *termtest.Screen {
	opts = append([]tui.PrintOption{tui.WithColor(true)}, opts...)
	settings := tui.SprintSettings(opts...)

	height := settings.Height
	if height == 0 {
		_, viewHeight := tui.Measure(view, settings.Width, 0)
		height = max(viewHeight, 1) + 1
	}

	output := tui.Sprint(view, opts...)
	screen := termtest.NewScreen(settings.Width, height, termtest.WithRawMode(settings.RawMode))
	screen.Write([]byte(output))
	return screen
}
