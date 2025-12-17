package gif

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/termsession"
)

// CastOptions configures the conversion of terminal recordings (.cast files)
// to animated GIFs. It provides control over dimensions, timing, rendering
// quality, and font selection.
//
// Use DefaultCastOptions to get sensible defaults, then customize as needed:
//
//	opts := gif.DefaultCastOptions()
//	opts.FontSize = 16
//	opts.FPS = 15
//	opts.Speed = 2.0  // Double speed playback
type CastOptions struct {
	Cols      int            // Override terminal columns (0 = use recording dimensions)
	Rows      int            // Override terminal rows (0 = use recording dimensions)
	Speed     float64        // Playback speed multiplier (1.0 = normal, 2.0 = double speed)
	MaxIdle   float64        // Maximum idle time between events in seconds (default: 2.0, prevents long pauses)
	FPS       int            // Target frames per second for output GIF (default: 10)
	Padding   int            // Padding around terminal content in pixels (default: 8)
	Font      *FontFace      // Custom TTF/OTF font (nil = use default Inconsolata)
	FontSize  float64        // Font size in points when using default font (default: 14)
	UseBitmap bool           // Force bitmap font instead of TTF (faster but lower quality)
}

// DefaultCastOptions returns sensible defaults for cast conversion.
func DefaultCastOptions() CastOptions {
	return CastOptions{
		Speed:    1.0,
		MaxIdle:  2.0,
		FPS:      10,
		Padding:  8,
		FontSize: 14,
	}
}

// RenderCast converts an asciinema .cast file (terminal recording) to an
// animated GIF. It processes the terminal output events, applies ANSI escape
// sequences, and renders each frame with proper timing.
//
// The function handles:
//   - ANSI color codes (16-color, 256-color, and true color)
//   - Cursor movement and text positioning
//   - Screen clearing and line editing
//   - Playback speed adjustment and idle time limiting
//
// Example:
//
//	opts := gif.DefaultCastOptions()
//	opts.FontSize = 16
//	opts.Speed = 1.5  // Play at 1.5x speed
//	g, err := gif.RenderCast("demo.cast", opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	g.Save("demo.gif")
//
// The returned GIF can be saved with Save() or encoded with Encode().
func RenderCast(castFile string, opts CastOptions) (*GIF, error) {
	header, events, err := termsession.LoadCastFile(castFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load cast file: %w", err)
	}

	return RenderCastEvents(header, events, opts)
}

// RenderCastEvents converts pre-loaded terminal recording events to an animated
// GIF. This function is useful when you already have the header and events in
// memory, or when you want to filter or modify events before rendering.
//
// See RenderCast for higher-level usage that loads from a file directly.
func RenderCastEvents(header *termsession.RecordingHeader, events []termsession.RecordingEvent, opts CastOptions) (*GIF, error) {
	// Apply defaults
	if opts.Speed <= 0 {
		opts.Speed = 1.0
	}
	if opts.MaxIdle <= 0 {
		opts.MaxIdle = 2.0
	}
	if opts.FPS <= 0 {
		opts.FPS = 10
	}
	if opts.FontSize <= 0 {
		opts.FontSize = 14
	}

	// Determine terminal dimensions
	cols := header.Width
	rows := header.Height
	if opts.Cols > 0 {
		cols = opts.Cols
	}
	if opts.Rows > 0 {
		rows = opts.Rows
	}

	// Create emulator and renderer
	emulator := NewEmulator(cols, rows)

	rendererOpts := RendererOptions{
		Font:      opts.Font,
		FontSize:  opts.FontSize,
		UseBitmap: opts.UseBitmap,
		Padding:   opts.Padding,
	}
	renderer := NewTerminalRendererWithOptions(emulator.Screen(), rendererOpts)
	renderer.SetLoopCount(0) // Loop forever

	// Process events and render frames
	frameInterval := 1.0 / float64(opts.FPS)
	lastFrameTime := 0.0
	var adjustedTime float64
	var lastEventTime float64

	for _, event := range events {
		// Apply speed and max idle adjustments
		timeDelta := event.Time - lastEventTime
		if timeDelta > opts.MaxIdle {
			timeDelta = opts.MaxIdle
		}
		adjustedTime += timeDelta / opts.Speed
		lastEventTime = event.Time

		// Only process output events
		if event.Type == "o" {
			emulator.ProcessOutput(event.Data)
		}

		// Render frame if enough time has passed
		if adjustedTime-lastFrameTime >= frameInterval {
			delay := int((adjustedTime - lastFrameTime) * 100) // Convert to centiseconds
			if delay < 1 {
				delay = 1
			}
			renderer.RenderFrame(delay)
			lastFrameTime = adjustedTime
		}
	}

	// Render final frame
	if lastFrameTime < adjustedTime {
		renderer.RenderFrame(10) // 100ms final frame
	}

	return renderer.GIF(), nil
}

// CastInfo contains metadata about a terminal recording without rendering it.
// This is useful for inspecting recording properties before conversion.
type CastInfo struct {
	Width       int     // Terminal width in columns
	Height      int     // Terminal height in rows
	Duration    float64 // Total recording duration in seconds
	EventCount  int     // Number of events in the recording
	Title       string  // Recording title from metadata
	Timestamp   int64   // Unix timestamp when recorded
}

// GetCastInfo extracts metadata from a .cast file without rendering it to a
// GIF. This is useful for displaying recording information or making decisions
// about rendering options.
//
// Example:
//
//	info, err := gif.GetCastInfo("demo.cast")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Recording: %dx%d, %.1fs duration, %d events\n",
//	    info.Width, info.Height, info.Duration, info.EventCount)
func GetCastInfo(castFile string) (*CastInfo, error) {
	header, events, err := termsession.LoadCastFile(castFile)
	if err != nil {
		return nil, err
	}

	return &CastInfo{
		Width:      header.Width,
		Height:     header.Height,
		Duration:   termsession.Duration(events),
		EventCount: len(events),
		Title:      header.Title,
		Timestamp:  header.Timestamp,
	}, nil
}
