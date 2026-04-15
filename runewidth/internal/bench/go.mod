module github.com/deepnoodle-ai/wonton/runewidth/internal/bench

go 1.25

require (
	github.com/deepnoodle-ai/wonton v0.0.0
	github.com/mattn/go-runewidth v0.0.21
	github.com/rivo/uniseg v0.4.7
)

require github.com/clipperhouse/uax29/v2 v2.2.0 // indirect

replace github.com/deepnoodle-ai/wonton => ../../..
