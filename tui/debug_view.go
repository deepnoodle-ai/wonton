package tui

import (
	"fmt"
	"image"
	"time"
)

// DebugInfo holds runtime debug information.
type DebugInfo struct {
	// FPS is the current frames per second.
	FPS float64

	// FrameTime is the time taken for the last frame.
	FrameTime time.Duration

	// FrameCount is the total number of frames rendered.
	FrameCount uint64

	// TerminalSize is the current terminal dimensions.
	TerminalSize image.Point

	// LastEvent is a description of the most recent event.
	LastEvent string

	// LastEventTime is when the last event occurred.
	LastEventTime time.Time

	// EventCount is the total number of events processed.
	EventCount uint64

	// Custom allows applications to add custom debug values.
	Custom map[string]string
}

// debugView displays debug information as an overlay.
type debugView struct {
	info     *DebugInfo
	position DebugPosition
	style    Style
	bgStyle  Style
}

// DebugPosition specifies where the debug overlay appears.
type DebugPosition int

const (
	DebugTopLeft DebugPosition = iota
	DebugTopRight
	DebugBottomLeft
	DebugBottomRight
)

// Debug creates a debug overlay view that displays runtime statistics.
//
// Example:
//
//	type MyApp struct {
//	    debug *tui.DebugInfo
//	}
//
//	func (app *MyApp) View() tui.View {
//	    return tui.ZStack(
//	        app.mainContent(),
//	        tui.Debug(&app.debug).Position(tui.DebugTopRight),
//	    )
//	}
//
//	func (app *MyApp) HandleEvent(event tui.Event) []tui.Cmd {
//	    app.debug.Update(event) // Call this to track events
//	    // ...
//	}
func Debug(info *DebugInfo) *debugView {
	return &debugView{
		info:     info,
		position: DebugTopRight,
		style:    NewStyle().WithForeground(ColorBrightWhite),
		bgStyle:  NewStyle().WithBackground(ColorBlack),
	}
}

// Position sets where the debug overlay appears.
func (d *debugView) Position(pos DebugPosition) *debugView {
	d.position = pos
	return d
}

// Style sets the text style for the debug overlay.
func (d *debugView) Style(s Style) *debugView {
	d.style = s
	return d
}

// BackgroundStyle sets the background style.
func (d *debugView) BackgroundStyle(s Style) *debugView {
	d.bgStyle = s
	return d
}

func (d *debugView) size(maxWidth, maxHeight int) (int, int) {
	// Debug overlay is sized to fit its content
	// Typical content:
	// FPS: 60.0
	// Frame: 1234
	// Event: KeyEvent(a)
	// Size: 80x24

	lines := d.buildLines()
	width := 0
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}

	h := len(lines)
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}

	w := width + 2 // padding
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}

	return w, h
}

func (d *debugView) buildLines() []string {
	if d.info == nil {
		return []string{"[debug: no info]"}
	}

	lines := []string{
		fmt.Sprintf("FPS: %.1f", d.info.FPS),
		fmt.Sprintf("Frame: %d", d.info.FrameCount),
	}

	if d.info.FrameTime > 0 {
		lines = append(lines, fmt.Sprintf("Time: %v", d.info.FrameTime.Round(time.Microsecond)))
	}

	if d.info.LastEvent != "" {
		lines = append(lines, fmt.Sprintf("Event: %s", d.info.LastEvent))
	}

	lines = append(lines, fmt.Sprintf("Events: %d", d.info.EventCount))

	if d.info.TerminalSize.X > 0 || d.info.TerminalSize.Y > 0 {
		lines = append(lines, fmt.Sprintf("Size: %dx%d", d.info.TerminalSize.X, d.info.TerminalSize.Y))
	}

	// Add custom values
	for k, v := range d.info.Custom {
		lines = append(lines, fmt.Sprintf("%s: %s", k, v))
	}

	return lines
}

func (d *debugView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() || d.info == nil {
		return
	}

	lines := d.buildLines()
	width := bounds.Dx()
	height := bounds.Dy()

	// Calculate overlay size
	overlayWidth := 0
	for _, line := range lines {
		if len(line) > overlayWidth {
			overlayWidth = len(line)
		}
	}
	overlayWidth += 2 // padding
	overlayHeight := len(lines)

	// Calculate position based on DebugPosition
	var overlayX, overlayY int
	switch d.position {
	case DebugTopLeft:
		overlayX = bounds.Min.X
		overlayY = bounds.Min.Y
	case DebugTopRight:
		overlayX = bounds.Max.X - overlayWidth
		if overlayX < bounds.Min.X {
			overlayX = bounds.Min.X
		}
		overlayY = bounds.Min.Y
	case DebugBottomLeft:
		overlayX = bounds.Min.X
		overlayY = bounds.Max.Y - overlayHeight
		if overlayY < bounds.Min.Y {
			overlayY = bounds.Min.Y
		}
	case DebugBottomRight:
		overlayX = bounds.Max.X - overlayWidth
		if overlayX < bounds.Min.X {
			overlayX = bounds.Min.X
		}
		overlayY = bounds.Max.Y - overlayHeight
		if overlayY < bounds.Min.Y {
			overlayY = bounds.Min.Y
		}
	}

	// Draw background
	bgChar := ' '
	for y := 0; y < overlayHeight && y < height; y++ {
		for x := 0; x < overlayWidth && x < width; x++ {
			frame.SetCell(overlayX+x, overlayY+y, bgChar, d.bgStyle)
		}
	}

	// Draw text
	for y, line := range lines {
		if y >= height {
			break
		}
		text := " " + line // Left padding
		if len(text) > overlayWidth {
			text = text[:overlayWidth]
		}
		frame.PrintStyled(overlayX, overlayY+y, text, d.style)
	}
}

// NewDebugInfo creates a new DebugInfo instance.
func NewDebugInfo() *DebugInfo {
	return &DebugInfo{
		Custom: make(map[string]string),
	}
}

// Update updates the debug info based on an event.
// Call this from your HandleEvent to track events.
func (d *DebugInfo) Update(event Event) {
	d.EventCount++
	d.LastEventTime = event.Timestamp()

	switch e := event.(type) {
	case TickEvent:
		d.FrameCount = e.Frame
	case KeyEvent:
		if e.Paste != "" {
			d.LastEvent = fmt.Sprintf("Paste(%d chars)", len(e.Paste))
		} else if e.Key != 0 {
			d.LastEvent = fmt.Sprintf("Key(%v)", e.Key)
		} else {
			d.LastEvent = fmt.Sprintf("Key('%c')", e.Rune)
		}
	case MouseEvent:
		d.LastEvent = fmt.Sprintf("Mouse(%v@%d,%d)", e.Type, e.X, e.Y)
	case ResizeEvent:
		d.TerminalSize = image.Pt(e.Width, e.Height)
		d.LastEvent = fmt.Sprintf("Resize(%dx%d)", e.Width, e.Height)
	default:
		d.LastEvent = fmt.Sprintf("%T", event)
	}
}

// SetFPS sets the FPS value.
func (d *DebugInfo) SetFPS(fps float64) {
	d.FPS = fps
}

// SetFrameTime sets the frame time.
func (d *DebugInfo) SetFrameTime(t time.Duration) {
	d.FrameTime = t
}

// Set sets a custom debug value.
func (d *DebugInfo) Set(key, value string) {
	if d.Custom == nil {
		d.Custom = make(map[string]string)
	}
	d.Custom[key] = value
}

// Clear removes a custom debug value.
func (d *DebugInfo) Clear(key string) {
	delete(d.Custom, key)
}

// DebugWrapper wraps an Application to automatically track debug info.
type DebugWrapper struct {
	app           Application
	Info          *DebugInfo
	enabled       bool
	lastFrameTime time.Time
	frameCount    int
	fpsUpdateTime time.Time
}

// WrapWithDebug wraps an application to automatically collect debug info.
//
// Example:
//
//	wrapper := tui.WrapWithDebug(&MyApp{})
//	// Use wrapper.Info in your View to display debug overlay
//	tui.Run(wrapper)
func WrapWithDebug(app Application) *DebugWrapper {
	return &DebugWrapper{
		app:           app,
		Info:          NewDebugInfo(),
		enabled:       true,
		fpsUpdateTime: time.Now(),
	}
}

// Enable enables or disables debug tracking.
func (w *DebugWrapper) Enable(enabled bool) {
	w.enabled = enabled
}

// View implements Application.
func (w *DebugWrapper) View() View {
	now := time.Now()

	// Calculate FPS
	if w.enabled {
		w.frameCount++
		if now.Sub(w.fpsUpdateTime) >= time.Second {
			w.Info.FPS = float64(w.frameCount) / now.Sub(w.fpsUpdateTime).Seconds()
			w.frameCount = 0
			w.fpsUpdateTime = now
		}

		if !w.lastFrameTime.IsZero() {
			w.Info.FrameTime = now.Sub(w.lastFrameTime)
		}
		w.lastFrameTime = now
	}

	return w.app.View()
}

// HandleEvent implements EventHandler.
func (w *DebugWrapper) HandleEvent(event Event) []Cmd {
	if w.enabled {
		w.Info.Update(event)
	}

	if handler, ok := w.app.(EventHandler); ok {
		return handler.HandleEvent(event)
	}
	return nil
}
