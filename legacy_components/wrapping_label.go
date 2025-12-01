package tui

// import (
// 	"image"
// 	"strings"

// 	"github.com/mattn/go-runewidth"
// )

// // WrappingLabel is a text label that automatically wraps text to fit width constraints.
// // It implements the Measurable interface for dynamic layout.
// type WrappingLabel struct {
// 	BaseWidget
// 	Text         string
// 	Style        Style
// 	Align        Alignment
// 	wrappedLines []string
// 	lastWidth    int
// }

// // NewWrappingLabel creates a new wrapping label
// func NewWrappingLabel(text string) *WrappingLabel {
// 	wl := &WrappingLabel{
// 		BaseWidget: NewBaseWidget(),
// 		Text:       text,
// 		Style:      NewStyle(),
// 		Align:      AlignLeft,
// 	}
// 	// Set initial min size to something small, preferred to full text
// 	w, h := MeasureText(text)
// 	wl.SetMinSize(image.Point{X: 1, Y: 1}) // Can shrink to 1x1
// 	wl.SetPreferredSize(image.Point{X: w, Y: h})
// 	return wl
// }

// // Measure implements the Measurable interface.
// func (wl *WrappingLabel) Measure(constraints SizeConstraints) image.Point {
// 	// Determine target width for wrapping
// 	targetWidth := constraints.MaxWidth

// 	// If unconstrained width, don't wrap (use infinite width)
// 	if !constraints.HasMaxWidth() {
// 		// Return natural size
// 		w, h := MeasureText(wl.Text)
// 		return ApplyConstraints(image.Point{X: w, Y: h}, constraints)
// 	}

// 	// Re-wrap if width changed or we have no lines
// 	if targetWidth != wl.lastWidth || len(wl.wrappedLines) == 0 {
// 		wrapped := WrapText(wl.Text, targetWidth)
// 		wl.wrappedLines = strings.Split(wrapped, "\n")
// 		wl.lastWidth = targetWidth
// 	}

// 	// Measure wrapped text
// 	maxW := 0
// 	for _, line := range wl.wrappedLines {
// 		w := runewidth.StringWidth(line)
// 		if w > maxW {
// 			maxW = w
// 		}
// 	}
// 	h := len(wl.wrappedLines)

// 	return ApplyConstraints(image.Point{X: maxW, Y: h}, constraints)
// }

// // WithStyle sets the label's style
// func (wl *WrappingLabel) WithStyle(style Style) *WrappingLabel {
// 	wl.Style = style
// 	wl.MarkDirty()
// 	return wl
// }

// // WithAlign sets the text alignment
// func (wl *WrappingLabel) WithAlign(align Alignment) *WrappingLabel {
// 	wl.Align = align
// 	wl.MarkDirty()
// 	return wl
// }

// // Draw renders the label
// func (wl *WrappingLabel) Draw(frame RenderFrame) {
// 	if !wl.visible {
// 		return
// 	}

// 	bounds := wl.GetBounds()
// 	width := bounds.Dx()
// 	height := bounds.Dy()

// 	if width <= 0 || height <= 0 {
// 		return
// 	}

// 	// Determine if we're drawing in a positioned SubFrame
// 	frameWidth, frameHeight := frame.Size()
// 	inSubFrame := (frameWidth == width && frameHeight == height)

// 	// Get lines to draw
// 	lines := wl.wrappedLines

// 	// If width doesn't match last measure (e.g. legacy layout or resize), re-wrap
// 	if width != wl.lastWidth {
// 		wrapped := WrapText(wl.Text, width)
// 		lines = strings.Split(wrapped, "\n")
// 		// Update cache
// 		wl.wrappedLines = lines
// 		wl.lastWidth = width
// 	}

// 	// Draw lines
// 	for i, line := range lines {
// 		if i >= height {
// 			break
// 		}

// 		// Align text
// 		alignedLine := AlignText(line, width, wl.Align)

// 		// Calculate draw position
// 		var startX, startY int
// 		if inSubFrame {
// 			startX, startY = 0, 0
// 		} else {
// 			startX, startY = bounds.Min.X, bounds.Min.Y
// 		}

// 		// Draw line
// 		// Note: AlignText pads with spaces, so we just draw it
// 		for j, r := range alignedLine {
// 			if j >= width {
// 				break
// 			}
// 			frame.SetCell(startX+j, startY+i, r, wl.Style)
// 		}
// 	}

// 	wl.ClearDirty()
// }

// // HandleKey handles keyboard events
// func (wl *WrappingLabel) HandleKey(event KeyEvent) bool {
// 	return false
// }
