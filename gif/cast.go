package gif

import (
	"fmt"
	"strconv"
	"strings"

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
	Cols      int       // Override terminal columns (0 = use recording dimensions). Setting Cols or Rows fixes the size: "r" resize events are then ignored.
	Rows      int       // Override terminal rows (0 = use recording dimensions). Setting Cols or Rows fixes the size: "r" resize events are then ignored.
	Speed     float64   // Playback speed multiplier (1.0 = normal, 2.0 = double speed)
	MaxIdle   float64   // Maximum idle time between events in seconds (default: 2.0, prevents long pauses)
	FPS       int       // Target frames per second for output GIF (default: 10)
	Padding   int       // Padding around terminal content in pixels (default: 8)
	Font      *FontFace // Custom TTF/OTF font (nil = use default Inconsolata)
	FontSize  float64   // Font size in points when using default font (default: 14)
	UseBitmap bool      // Force bitmap font instead of TTF (faster but lower quality)
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
	cols, rows := 0, 0
	if header != nil {
		cols, rows = header.Width, header.Height
	}
	fixedSize := opts.Cols > 0 || opts.Rows > 0
	if opts.Cols > 0 {
		cols = opts.Cols
	}
	if opts.Rows > 0 {
		rows = opts.Rows
	}
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	// Size the canvas to fit the largest terminal size in the recording, so
	// that "r" (resize) events do not draw outside it.
	canvasCols, canvasRows := cols, rows
	if !fixedSize {
		for _, event := range events {
			if c, r, ok := parseResize(event); ok {
				canvasCols = max(canvasCols, c)
				canvasRows = max(canvasRows, r)
			}
		}
	}

	// Create emulator and renderer
	emulator := NewEmulator(canvasCols, canvasRows)
	rendererOpts := RendererOptions{
		Font:       opts.Font,
		FontSize:   opts.FontSize,
		UseBitmap:  opts.UseBitmap,
		BitmapFont: BitmapFont8x16, // Default bitmap font when UseBitmap=true
		Padding:    opts.Padding,
	}
	renderer := NewTerminalRendererWithOptions(emulator.Screen(), rendererOpts)
	renderer.SetLoopCount(0) // Loop forever
	if cols != canvasCols || rows != canvasRows {
		emulator.Resize(cols, rows)
		renderer.screen = emulator.Screen()
	}

	// Process events and render frames. A frame shows the screen as it was
	// before the current event, for as long as it was visible: the time
	// since the previous frame. So render first, then apply the event.
	frameInterval := 1.0 / float64(opts.FPS)
	lastFrameTime := 0.0
	var adjustedTime float64
	var lastEventTime float64
	dirty := false

	for _, event := range events {
		// Apply speed and max idle adjustments
		timeDelta := event.Time - lastEventTime
		if timeDelta > opts.MaxIdle {
			timeDelta = opts.MaxIdle
		}
		adjustedTime += timeDelta / opts.Speed
		lastEventTime = event.Time

		// Render the current screen if enough time has passed
		if adjustedTime-lastFrameTime >= frameInterval {
			delay := int((adjustedTime-lastFrameTime)*100 + 0.5) // Convert to centiseconds
			if delay < 1 {
				delay = 1
			}
			renderer.RenderFrame(delay)
			lastFrameTime = adjustedTime
			dirty = false
		}

		switch event.Type {
		case "o":
			emulator.ProcessOutput(event.Data)
			dirty = true
		case "r":
			if c, r, ok := parseResize(event); ok && !fixedSize {
				emulator.Resize(c, r)
				renderer.screen = emulator.Screen()
				dirty = true
			}
		}
	}

	// Render the final screen, or ensure at least one frame exists
	if dirty || renderer.GIF().FrameCount() == 0 {
		renderer.RenderFrame(10) // 100ms final frame
	}

	return renderer.GIF(), nil
}

// CastInfo contains metadata about a terminal recording without rendering it.
// This is useful for inspecting recording properties before conversion.
type CastInfo struct {
	Width      int     // Terminal width in columns
	Height     int     // Terminal height in rows
	Duration   float64 // Total recording duration in seconds
	EventCount int     // Number of events in the recording
	Title      string  // Recording title from metadata
	Timestamp  int64   // Unix timestamp when recorded
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

// parseResize parses an asciinema "r" event, whose data is "COLSxROWS".
func parseResize(event termsession.RecordingEvent) (cols, rows int, ok bool) {
	if event.Type != "r" {
		return 0, 0, false
	}
	c, r, found := strings.Cut(event.Data, "x")
	if !found {
		return 0, 0, false
	}
	cols, err1 := strconv.Atoi(c)
	rows, err2 := strconv.Atoi(r)
	if err1 != nil || err2 != nil || cols <= 0 || rows <= 0 {
		return 0, 0, false
	}
	return cols, rows, true
}
