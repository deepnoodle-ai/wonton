package tui

import (
	"github.com/deepnoodle-ai/wonton/terminal"
)

// Re-export types from terminal package for backward compatibility
type (
	// Style represents text styling attributes
	Style = terminal.Style
	// Color represents a terminal color
	Color = terminal.Color
	// RGB represents an RGB color value
	RGB = terminal.RGB
	// RenderFrame represents a rendering surface for a single frame
	RenderFrame = terminal.RenderFrame
	// BorderStyle defines the characters used for drawing borders
	BorderStyle = terminal.BorderStyle
	// Cell represents a single character cell on the terminal
	Cell = terminal.Cell
	// DirtyRegion tracks the rectangular area that has been modified
	DirtyRegion = terminal.DirtyRegion
	// Terminal represents the terminal abstraction
	Terminal = terminal.Terminal
	// RenderMetrics tracks performance statistics for the rendering system
	RenderMetrics = terminal.RenderMetrics
	// Hyperlink represents a clickable hyperlink in the terminal
	Hyperlink = terminal.Hyperlink
	// Alignment represents text alignment
	Alignment = terminal.Alignment
	// RecordingHeader represents asciinema v2 header
	RecordingHeader = terminal.RecordingHeader
	// RecordingEvent represents a single recording event
	RecordingEvent = terminal.RecordingEvent
	// Recorder handles session recording
	Recorder = terminal.Recorder
	// RecordingOptions configures recording behavior
	RecordingOptions = terminal.RecordingOptions
	// Frame is a bordered frame/box
	Frame = terminal.Frame
	// Box is a simple box primitive
	Box = terminal.Box
	// MetricsSnapshot is a point-in-time snapshot of rendering metrics
	MetricsSnapshot = terminal.MetricsSnapshot
)

// Re-export color constants from terminal
const (
	ColorDefault       = terminal.ColorDefault
	ColorBlack         = terminal.ColorBlack
	ColorRed           = terminal.ColorRed
	ColorGreen         = terminal.ColorGreen
	ColorYellow        = terminal.ColorYellow
	ColorBlue          = terminal.ColorBlue
	ColorMagenta       = terminal.ColorMagenta
	ColorCyan          = terminal.ColorCyan
	ColorWhite         = terminal.ColorWhite
	ColorBrightBlack   = terminal.ColorBrightBlack
	ColorBrightRed     = terminal.ColorBrightRed
	ColorBrightGreen   = terminal.ColorBrightGreen
	ColorBrightYellow  = terminal.ColorBrightYellow
	ColorBrightBlue    = terminal.ColorBrightBlue
	ColorBrightMagenta = terminal.ColorBrightMagenta
	ColorBrightCyan    = terminal.ColorBrightCyan
	ColorBrightWhite   = terminal.ColorBrightWhite
)

// Re-export color functions from terminal
var (
	NewRGB          = terminal.NewRGB
	Gradient        = terminal.Gradient
	RainbowGradient = terminal.RainbowGradient
	SmoothRainbow   = terminal.SmoothRainbow
	MultiGradient   = terminal.MultiGradient
)

// Re-export style constructor
var NewStyle = terminal.NewStyle

// Re-export border styles from terminal
var (
	SingleBorder  = terminal.SingleBorder
	DoubleBorder  = terminal.DoubleBorder
	RoundedBorder = terminal.RoundedBorder
	ThickBorder   = terminal.ThickBorder
	ASCIIBorder   = terminal.ASCIIBorder
)

// Re-export alignment constants from terminal
const (
	AlignLeft   = terminal.AlignLeft
	AlignCenter = terminal.AlignCenter
	AlignRight  = terminal.AlignRight
)

// Re-export terminal functions
var (
	NewTerminal      = terminal.NewTerminal
	NewTestTerminal  = terminal.NewTestTerminal
	NewRenderMetrics = terminal.NewRenderMetrics
	NewHyperlink     = terminal.NewHyperlink
	OSC8Start        = terminal.OSC8Start
	OSC8End          = terminal.OSC8End
	NewFrame         = terminal.NewFrame
)

// Re-export error types from terminal
var (
	ErrOutOfBounds   = terminal.ErrOutOfBounds
	ErrNotInRawMode  = terminal.ErrNotInRawMode
	ErrClosed        = terminal.ErrClosed
	ErrInvalidFrame  = terminal.ErrInvalidFrame
	ErrAlreadyActive = terminal.ErrAlreadyActive
)
