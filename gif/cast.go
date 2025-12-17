package gif

import (
	"fmt"

	"github.com/deepnoodle-ai/wonton/termsession"
)

// CastOptions configures cast-to-GIF conversion.
type CastOptions struct {
	Cols      int            // Override columns (0 = from file)
	Rows      int            // Override rows (0 = from file)
	Speed     float64        // Playback speed multiplier (default: 1.0)
	MaxIdle   float64        // Max idle time between events in seconds (default: 2.0)
	FPS       int            // Target frames per second (default: 10)
	Padding   int            // Padding around terminal in pixels (default: 8)
	Font      *FontFace      // Custom font (nil = default TTF)
	FontSize  float64        // Font size if using default TTF (default: 14)
	UseBitmap bool           // Force bitmap font instead of TTF
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

// RenderCast converts a .cast file to an animated GIF.
// The returned GIF can be saved with gif.Save() or encoded with gif.Encode().
func RenderCast(castFile string, opts CastOptions) (*GIF, error) {
	header, events, err := termsession.LoadCastFile(castFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load cast file: %w", err)
	}

	return RenderCastEvents(header, events, opts)
}

// RenderCastEvents converts cast events to an animated GIF.
// This is useful when you already have the header and events loaded.
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

// CastInfo returns information about a .cast file without rendering it.
type CastInfo struct {
	Width       int
	Height      int
	Duration    float64
	EventCount  int
	Title       string
	Timestamp   int64
}

// GetCastInfo returns information about a .cast file.
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
