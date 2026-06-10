package tui

import "fmt"

// MeterView displays a labeled meter/gauge
type MeterView struct {
	label      string
	value      int
	max        int
	width      int
	filledChar rune
	emptyChar  rune
	style      Style
	labelStyle Style
	showValue  bool
}

// Meter creates a labeled meter/gauge view.
//
// Example:
//
//	Meter("CPU", 75, 100).Width(20)
func Meter(label string, value, max int) *MeterView {
	return &MeterView{
		label:      label,
		value:      value,
		max:        max,
		width:      10,
		filledChar: '█',
		emptyChar:  '·',
		style:      NewStyle().WithForeground(ColorGreen),
		labelStyle: NewStyle(),
		showValue:  true,
	}
}

// Width sets the width of the bar portion.
func (m *MeterView) Width(w int) *MeterView {
	m.width = w
	return m
}

// FilledChar sets the filled character.
func (m *MeterView) FilledChar(c rune) *MeterView {
	m.filledChar = c
	return m
}

// EmptyChar sets the empty character.
func (m *MeterView) EmptyChar(c rune) *MeterView {
	m.emptyChar = c
	return m
}

// Fg sets the bar foreground color.
func (m *MeterView) Fg(c Color) *MeterView {
	m.style = m.style.WithForeground(c)
	return m
}

// LabelFg sets the label foreground color.
func (m *MeterView) LabelFg(c Color) *MeterView {
	m.labelStyle = m.labelStyle.WithForeground(c)
	return m
}

// Style sets the bar style.
func (m *MeterView) Style(s Style) *MeterView {
	m.style = s
	return m
}

// ShowValue enables/disables value display.
func (m *MeterView) ShowValue(show bool) *MeterView {
	m.showValue = show
	return m
}

func (m *MeterView) size(maxWidth, maxHeight int) (int, int) {
	labelW, _ := MeasureText(m.label)
	w := labelW + 2 + m.width // label + ": " + bar
	if m.showValue {
		w += len(fmt.Sprintf(" %d%%", 100))
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (m *MeterView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	x := 0

	// Draw label
	if m.label != "" {
		ctx.PrintStyled(x, 0, m.label+": ", m.labelStyle)
		labelW, _ := MeasureText(m.label)
		x += labelW + 2
	}

	// Calculate fill
	barWidth := m.width
	fillWidth := 0
	if m.max > 0 {
		fillWidth = (m.value * barWidth) / m.max
		if fillWidth > barWidth {
			fillWidth = barWidth
		}
	}

	// Draw empty background
	emptyStyle := NewStyle().WithForeground(ColorBrightBlack)
	for i := 0; i < barWidth; i++ {
		ctx.SetCell(x+i, 0, m.emptyChar, emptyStyle)
	}

	// Draw filled portion
	for i := 0; i < fillWidth; i++ {
		ctx.SetCell(x+i, 0, m.filledChar, m.style)
	}
	x += barWidth

	// Draw value
	if m.showValue && m.max > 0 {
		percent := (m.value * 100) / m.max
		ctx.PrintStyled(x, 0, fmt.Sprintf(" %d%%", percent), m.style)
	}
}
