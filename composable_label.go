package gooey

import (
	"image"
	"strings"
)

// ComposableLabel is a simple text label that implements ComposableWidget.
// It displays static or dynamic text within bounds assigned by its container.
type ComposableLabel struct {
	BaseWidget
	Text  string
	Style Style
	Align Alignment // Text alignment (AlignLeft, AlignCenter, AlignRight)
}

// NewComposableLabel creates a new composable label
func NewComposableLabel(text string) *ComposableLabel {
	label := &ComposableLabel{
		BaseWidget: NewBaseWidget(),
		Text:       text,
		Style:      NewStyle(),
		Align:      AlignLeft,
	}

	// Set minimum size based on text
	label.SetMinSize(image.Point{X: len(text), Y: 1})
	label.SetPreferredSize(image.Point{X: len(text), Y: 1})

	return label
}

// WithStyle sets the label's style
func (cl *ComposableLabel) WithStyle(style Style) *ComposableLabel {
	cl.Style = style
	cl.MarkDirty()
	return cl
}

// WithAlign sets the text alignment
func (cl *ComposableLabel) WithAlign(align Alignment) *ComposableLabel {
	cl.Align = align
	cl.MarkDirty()
	return cl
}

// SetText updates the label text
func (cl *ComposableLabel) SetText(text string) {
	if cl.Text != text {
		cl.Text = text
		cl.SetMinSize(image.Point{X: len(text), Y: 1})
		cl.SetPreferredSize(image.Point{X: len(text), Y: 1})
		cl.MarkDirty()
		if cl.parent != nil {
			cl.parent.MarkDirty()
		}
	}
}

// Draw renders the label within its assigned bounds
func (cl *ComposableLabel) Draw(frame RenderFrame) {
	if !cl.visible {
		return
	}

	bounds := cl.GetBounds()
	width := bounds.Dx()

	if width <= 0 {
		cl.ClearDirty()
		return
	}

	// Determine if we're drawing in a positioned SubFrame
	frameWidth, frameHeight := frame.Size()
	inSubFrame := (frameWidth == bounds.Dx() && frameHeight == bounds.Dy())

	// Prepare text with padding/alignment
	text := cl.Text
	if len(text) > width {
		// Truncate if too long
		text = text[:width]
	} else if len(text) < width {
		// Add padding based on alignment
		padding := width - len(text)
		switch cl.Align {
		case AlignLeft:
			text = text + strings.Repeat(" ", padding)
		case AlignRight:
			text = strings.Repeat(" ", padding) + text
		case AlignCenter:
			leftPad := padding / 2
			rightPad := padding - leftPad
			text = strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
		}
	}

	// Draw character by character to avoid wrapping
	// If we're in a SubFrame, draw at (0,0). Otherwise use absolute bounds.
	var startX, startY int
	if inSubFrame {
		startX, startY = 0, 0
	} else {
		startX, startY = bounds.Min.X, bounds.Min.Y
	}

	for i := 0; i < len(text) && i < width; i++ {
		frame.SetCell(startX+i, startY, rune(text[i]), cl.Style)
	}

	cl.ClearDirty()
}

// HandleKey handles keyboard events (labels don't handle keys by default)
func (cl *ComposableLabel) HandleKey(event KeyEvent) bool {
	return false
}

// ComposableMultiLineLabel displays multi-line text within bounds
type ComposableMultiLineLabel struct {
	BaseWidget
	Lines []string
	Style Style
	Align Alignment
}

// NewComposableMultiLineLabel creates a new multi-line label
func NewComposableMultiLineLabel(lines []string) *ComposableMultiLineLabel {
	label := &ComposableMultiLineLabel{
		BaseWidget: NewBaseWidget(),
		Lines:      lines,
		Style:      NewStyle(),
		Align:      AlignLeft,
	}

	// Calculate minimum size
	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	label.SetMinSize(image.Point{X: maxWidth, Y: len(lines)})
	label.SetPreferredSize(image.Point{X: maxWidth, Y: len(lines)})

	return label
}

// WithStyle sets the label's style
func (cml *ComposableMultiLineLabel) WithStyle(style Style) *ComposableMultiLineLabel {
	cml.Style = style
	cml.MarkDirty()
	return cml
}

// WithAlign sets the text alignment
func (cml *ComposableMultiLineLabel) WithAlign(align Alignment) *ComposableMultiLineLabel {
	cml.Align = align
	cml.MarkDirty()
	return cml
}

// SetLines updates the label lines
func (cml *ComposableMultiLineLabel) SetLines(lines []string) {
	cml.Lines = lines

	// Recalculate minimum size
	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	cml.SetMinSize(image.Point{X: maxWidth, Y: len(lines)})
	cml.SetPreferredSize(image.Point{X: maxWidth, Y: len(lines)})

	cml.MarkDirty()
	if cml.parent != nil {
		cml.parent.MarkDirty()
	}
}

// Draw renders the multi-line label within its assigned bounds
func (cml *ComposableMultiLineLabel) Draw(frame RenderFrame) {
	if !cml.visible {
		return
	}

	bounds := cml.GetBounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= 0 || height <= 0 {
		cml.ClearDirty()
		return
	}

	// Determine if we're drawing in a positioned SubFrame
	frameWidth, frameHeight := frame.Size()
	inSubFrame := (frameWidth == bounds.Dx() && frameHeight == bounds.Dy())

	// Draw each line
	for i, line := range cml.Lines {
		if i >= height {
			break // Don't draw beyond bounds
		}

		// Prepare text with padding/alignment
		text := line
		if len(text) > width {
			// Truncate if too long
			text = text[:width]
		} else if len(text) < width {
			// Add padding based on alignment
			padding := width - len(text)
			switch cml.Align {
			case AlignLeft:
				text = text + strings.Repeat(" ", padding)
			case AlignRight:
				text = strings.Repeat(" ", padding) + text
			case AlignCenter:
				leftPad := padding / 2
				rightPad := padding - leftPad
				text = strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
			}
		}

		// Draw character by character to avoid wrapping
		// If we're in a SubFrame, draw at (0,i). Otherwise use absolute bounds.
		var startX, startY int
		if inSubFrame {
			startX, startY = 0, 0
		} else {
			startX, startY = bounds.Min.X, bounds.Min.Y
		}

		for j := 0; j < len(text) && j < width; j++ {
			frame.SetCell(startX+j, startY+i, rune(text[j]), cml.Style)
		}
	}

	cml.ClearDirty()
}

// HandleKey handles keyboard events (labels don't handle keys by default)
func (cml *ComposableMultiLineLabel) HandleKey(event KeyEvent) bool {
	return false
}
