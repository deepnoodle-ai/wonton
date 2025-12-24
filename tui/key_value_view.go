package tui

// keyValueView displays a label: value pair
type keyValueView struct {
	label      string
	value      string
	labelStyle Style
	valueStyle Style
	separator  string
	width      int
}

// KeyValue creates a key-value pair display.
//
// Example:
//
//	KeyValue("Name", "John Doe")
//	KeyValue("Status", "Active").LabelFg(ColorYellow).ValueFg(ColorGreen)
func KeyValue(label, value string) *keyValueView {
	return &keyValueView{
		label:      label,
		value:      value,
		labelStyle: NewStyle().WithBold(),
		valueStyle: NewStyle(),
		separator:  ": ",
	}
}

// LabelFg sets the label foreground color.
func (k *keyValueView) LabelFg(c Color) *keyValueView {
	k.labelStyle = k.labelStyle.WithForeground(c)
	return k
}

// ValueFg sets the value foreground color.
func (k *keyValueView) ValueFg(c Color) *keyValueView {
	k.valueStyle = k.valueStyle.WithForeground(c)
	return k
}

// LabelStyle sets the complete label style.
func (k *keyValueView) LabelStyle(s Style) *keyValueView {
	k.labelStyle = s
	return k
}

// ValueStyle sets the complete value style.
func (k *keyValueView) ValueStyle(s Style) *keyValueView {
	k.valueStyle = s
	return k
}

// Separator sets the separator string (default ": ").
func (k *keyValueView) Separator(sep string) *keyValueView {
	k.separator = sep
	return k
}

// Width sets a fixed width.
func (k *keyValueView) Width(w int) *keyValueView {
	k.width = w
	return k
}

// Dim makes the value dimmed.
func (k *keyValueView) Dim() *keyValueView {
	k.valueStyle = k.valueStyle.WithDim()
	return k
}

func (k *keyValueView) size(maxWidth, maxHeight int) (int, int) {
	labelW, _ := MeasureText(k.label)
	sepW, _ := MeasureText(k.separator)
	valueW, _ := MeasureText(k.value)
	w := labelW + sepW + valueW
	if k.width > 0 {
		w = k.width
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (k *keyValueView) render(ctx *RenderContext) {
	w, h := ctx.Size()
	if w == 0 || h == 0 {
		return
	}

	x := 0

	// Draw label
	ctx.PrintStyled(x, 0, k.label, k.labelStyle)
	labelW, _ := MeasureText(k.label)
	x += labelW

	// Draw separator
	ctx.PrintStyled(x, 0, k.separator, k.labelStyle)
	sepW, _ := MeasureText(k.separator)
	x += sepW

	// Draw value
	ctx.PrintTruncated(x, 0, k.value, k.valueStyle)
}
