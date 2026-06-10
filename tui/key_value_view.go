package tui

// KeyValueView displays a label: value pair
type KeyValueView struct {
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
func KeyValue(label, value string) *KeyValueView {
	return &KeyValueView{
		label:      label,
		value:      value,
		labelStyle: NewStyle().WithBold(),
		valueStyle: NewStyle(),
		separator:  ": ",
	}
}

// LabelFg sets the label foreground color.
func (k *KeyValueView) LabelFg(c Color) *KeyValueView {
	k.labelStyle = k.labelStyle.WithForeground(c)
	return k
}

// ValueFg sets the value foreground color.
func (k *KeyValueView) ValueFg(c Color) *KeyValueView {
	k.valueStyle = k.valueStyle.WithForeground(c)
	return k
}

// LabelStyle sets the complete label style.
func (k *KeyValueView) LabelStyle(s Style) *KeyValueView {
	k.labelStyle = s
	return k
}

// ValueStyle sets the complete value style.
func (k *KeyValueView) ValueStyle(s Style) *KeyValueView {
	k.valueStyle = s
	return k
}

// Separator sets the separator string (default ": ").
func (k *KeyValueView) Separator(sep string) *KeyValueView {
	k.separator = sep
	return k
}

// Width sets a fixed width.
func (k *KeyValueView) Width(w int) *KeyValueView {
	k.width = w
	return k
}

// Dim makes the value dimmed.
func (k *KeyValueView) Dim() *KeyValueView {
	k.valueStyle = k.valueStyle.WithDim()
	return k
}

func (k *KeyValueView) size(maxWidth, maxHeight int) (int, int) {
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

func (k *KeyValueView) render(ctx *RenderContext) {
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
