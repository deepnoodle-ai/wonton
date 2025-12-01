package tui

import "image"

// animatedTextView displays text with per-character animation (declarative view)
type animatedTextView struct {
	text      string
	animation TextAnimation
	frame     uint64
	style     Style // fallback style if no animation
	width     int
}

// AnimatedTextView creates a declarative animated text view.
// The frame parameter should come from TickEvent.Frame for animation.
//
// Example:
//
//	AnimatedTextView("Hello World", CreateRainbowText("", 3), app.frame)
//	AnimatedTextView("Pulsing", CreatePulseText(NewRGB(0, 255, 255), 12), app.frame)
func AnimatedTextView(text string, animation TextAnimation, frame uint64) *animatedTextView {
	return &animatedTextView{
		text:      text,
		animation: animation,
		frame:     frame,
		style:     NewStyle(),
	}
}

// Width sets a fixed width for the animated text.
func (a *animatedTextView) Width(w int) *animatedTextView {
	a.width = w
	return a
}

// Style sets the fallback style (used when animation is nil).
func (a *animatedTextView) Style(s Style) *animatedTextView {
	a.style = s
	return a
}

func (a *animatedTextView) size(maxWidth, maxHeight int) (int, int) {
	w, _ := MeasureText(a.text)
	if a.width > 0 {
		w = a.width
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (a *animatedTextView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	subFrame := frame.SubFrame(bounds)
	runes := []rune(a.text)
	totalChars := len(runes)
	maxWidth := bounds.Dx()

	for i, r := range runes {
		if i >= maxWidth {
			break
		}
		var style Style
		if a.animation != nil {
			style = a.animation.GetStyle(a.frame, i, totalChars)
		} else {
			style = a.style
		}
		subFrame.SetCell(i, 0, r, style)
	}
}

// panelView displays a filled rectangle with optional border
type panelView struct {
	content     View
	width       int
	height      int
	fillChar    rune
	borderStyle borderStyleType
	bgStyle     Style
	borderColor Color
	title       string
}

type borderStyleType int

const (
	BorderNone borderStyleType = iota
	BorderSingle
	BorderDouble
	BorderRounded
	BorderHeavy
)

// Panel creates a filled box/panel view with optional content.
//
// Example:
//
//	Panel(nil).Width(20).Height(5).Bg(ColorBlue)
//	Panel(Text("Hello")).Border(BorderSingle)
func Panel(content View) *panelView {
	return &panelView{
		content:     content,
		fillChar:    ' ',
		borderStyle: BorderNone,
		bgStyle:     NewStyle(),
		borderColor: ColorDefault,
	}
}

// Width sets the box width.
func (b *panelView) Width(w int) *panelView {
	b.width = w
	return b
}

// Height sets the box height.
func (b *panelView) Height(h int) *panelView {
	b.height = h
	return b
}

// FillChar sets the character used to fill the box.
func (b *panelView) FillChar(c rune) *panelView {
	b.fillChar = c
	return b
}

// Border sets the border style.
func (b *panelView) Border(style borderStyleType) *panelView {
	b.borderStyle = style
	return b
}

// BorderColor sets the border color.
func (b *panelView) BorderColor(c Color) *panelView {
	b.borderColor = c
	return b
}

// Bg sets the background color.
func (b *panelView) Bg(c Color) *panelView {
	b.bgStyle = b.bgStyle.WithBackground(c)
	return b
}

// Fg sets the foreground color (for fill character).
func (b *panelView) Fg(c Color) *panelView {
	b.bgStyle = b.bgStyle.WithForeground(c)
	return b
}

// Style sets the complete background style.
func (b *panelView) Style(s Style) *panelView {
	b.bgStyle = s
	return b
}

// Title sets a title for the box (displayed in top border).
func (b *panelView) Title(title string) *panelView {
	b.title = title
	return b
}

func (b *panelView) getBorderChars() (tl, tr, bl, br, h, v rune) {
	switch b.borderStyle {
	case BorderSingle:
		return '┌', '┐', '└', '┘', '─', '│'
	case BorderDouble:
		return '╔', '╗', '╚', '╝', '═', '║'
	case BorderRounded:
		return '╭', '╮', '╰', '╯', '─', '│'
	case BorderHeavy:
		return '┏', '┓', '┗', '┛', '━', '┃'
	default:
		return ' ', ' ', ' ', ' ', ' ', ' '
	}
}

func (b *panelView) size(maxWidth, maxHeight int) (int, int) {
	w := b.width
	h := b.height

	// If content provided and no explicit size, size to content
	if b.content != nil && (w == 0 || h == 0) {
		if sizer, ok := b.content.(interface{ size(int, int) (int, int) }); ok {
			cw, ch := sizer.size(maxWidth, maxHeight)
			if w == 0 {
				w = cw
				if b.borderStyle != BorderNone {
					w += 2 // border
				}
			}
			if h == 0 {
				h = ch
				if b.borderStyle != BorderNone {
					h += 2 // border
				}
			}
		}
	}

	if w == 0 {
		w = 10
	}
	if h == 0 {
		h = 3
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (b *panelView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	subFrame := frame.SubFrame(bounds)
	width := bounds.Dx()
	height := bounds.Dy()

	// Fill background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			subFrame.SetCell(x, y, b.fillChar, b.bgStyle)
		}
	}

	// Draw border if specified
	if b.borderStyle != BorderNone && width >= 2 && height >= 2 {
		tl, tr, bl, br, h, v := b.getBorderChars()
		borderStyle := b.bgStyle
		if b.borderColor != ColorDefault {
			borderStyle = borderStyle.WithForeground(b.borderColor)
		}

		// Top and bottom
		for x := 1; x < width-1; x++ {
			subFrame.SetCell(x, 0, h, borderStyle)
			subFrame.SetCell(x, height-1, h, borderStyle)
		}

		// Left and right
		for y := 1; y < height-1; y++ {
			subFrame.SetCell(0, y, v, borderStyle)
			subFrame.SetCell(width-1, y, v, borderStyle)
		}

		// Corners
		subFrame.SetCell(0, 0, tl, borderStyle)
		subFrame.SetCell(width-1, 0, tr, borderStyle)
		subFrame.SetCell(0, height-1, bl, borderStyle)
		subFrame.SetCell(width-1, height-1, br, borderStyle)

		// Title
		if b.title != "" && width > 4 {
			titleText := " " + b.title + " "
			titleW, _ := MeasureText(titleText)
			if titleW > width-2 {
				titleW = width - 2
			}
			startX := (width - titleW) / 2
			subFrame.PrintTruncated(startX, 0, titleText, borderStyle)
		}
	}

	// Render content if provided
	if b.content != nil {
		contentBounds := bounds
		if b.borderStyle != BorderNone {
			contentBounds = image.Rect(
				bounds.Min.X+1,
				bounds.Min.Y+1,
				bounds.Max.X-1,
				bounds.Max.Y-1,
			)
		}
		if renderer, ok := b.content.(interface {
			render(RenderFrame, image.Rectangle)
		}); ok {
			renderer.render(frame, contentBounds)
		}
	}
}

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

func (k *keyValueView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	subFrame := frame.SubFrame(bounds)
	x := 0

	// Draw label
	subFrame.PrintStyled(x, 0, k.label, k.labelStyle)
	labelW, _ := MeasureText(k.label)
	x += labelW

	// Draw separator
	subFrame.PrintStyled(x, 0, k.separator, k.labelStyle)
	sepW, _ := MeasureText(k.separator)
	x += sepW

	// Draw value
	subFrame.PrintTruncated(x, 0, k.value, k.valueStyle)
}

// toggleView displays an on/off toggle switch
type toggleView struct {
	value      *bool
	onLabel    string
	offLabel   string
	onStyle    Style
	offStyle   Style
	onChange   func(bool)
	showLabels bool
}

// Toggle creates an on/off toggle switch.
// value should be a pointer to a bool controlling the toggle state.
//
// Example:
//
//	Toggle(&app.darkMode).OnChange(func(v bool) { app.updateTheme() })
func Toggle(value *bool) *toggleView {
	return &toggleView{
		value:      value,
		onLabel:    "ON",
		offLabel:   "OFF",
		onStyle:    NewStyle().WithForeground(ColorGreen).WithBold(),
		offStyle:   NewStyle().WithForeground(ColorBrightBlack),
		showLabels: true,
	}
}

// OnLabel sets the label for the ON state.
func (t *toggleView) OnLabel(label string) *toggleView {
	t.onLabel = label
	return t
}

// OffLabel sets the label for the OFF state.
func (t *toggleView) OffLabel(label string) *toggleView {
	t.offLabel = label
	return t
}

// OnStyle sets the style for the ON state.
func (t *toggleView) OnStyle(s Style) *toggleView {
	t.onStyle = s
	return t
}

// OffStyle sets the style for the OFF state.
func (t *toggleView) OffStyle(s Style) *toggleView {
	t.offStyle = s
	return t
}

// OnChange sets a callback when the toggle is clicked.
func (t *toggleView) OnChange(fn func(bool)) *toggleView {
	t.onChange = fn
	return t
}

// ShowLabels enables/disables showing ON/OFF labels.
func (t *toggleView) ShowLabels(show bool) *toggleView {
	t.showLabels = show
	return t
}

func (t *toggleView) size(maxWidth, maxHeight int) (int, int) {
	// [●] ON  or  [○] OFF
	w := 3 // switch chars
	if t.showLabels {
		onW, _ := MeasureText(t.onLabel)
		offW, _ := MeasureText(t.offLabel)
		if onW > offW {
			w += 1 + onW
		} else {
			w += 1 + offW
		}
	}
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	return w, 1
}

func (t *toggleView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	subFrame := frame.SubFrame(bounds)
	isOn := t.value != nil && *t.value

	var switchChar string
	var style Style
	var label string

	if isOn {
		switchChar = "●"
		style = t.onStyle
		label = t.onLabel
	} else {
		switchChar = "○"
		style = t.offStyle
		label = t.offLabel
	}

	// Draw switch
	text := "[" + switchChar + "]"
	if t.showLabels {
		text += " " + label
	}
	subFrame.PrintStyled(0, 0, text, style)

	// Register click region
	if t.value != nil {
		interactiveRegistry.RegisterButton(bounds, func() {
			*t.value = !*t.value
			if t.onChange != nil {
				t.onChange(*t.value)
			}
		})
	}
}

// styledButtonView displays a button with dimensions and styling
type styledButtonView struct {
	label       string
	callback    func()
	width       int
	height      int
	style       Style
	hoverStyle  Style
	borderStyle borderStyleType
	centered    bool
}

// StyledButton creates a styled button with dimensions.
// Unlike Clickable, this draws a filled box with centered text.
//
// Example:
//
//	StyledButton("Submit", func() { app.submit() }).Width(20).Height(3).Bg(ColorBlue)
func StyledButton(label string, callback func()) *styledButtonView {
	return &styledButtonView{
		label:       label,
		callback:    callback,
		width:       0,
		height:      1,
		style:       NewStyle().WithBackground(ColorBlue).WithForeground(ColorWhite),
		hoverStyle:  NewStyle().WithBackground(ColorCyan).WithForeground(ColorBlack),
		borderStyle: BorderNone,
		centered:    true,
	}
}

// Width sets the button width.
func (s *styledButtonView) Width(w int) *styledButtonView {
	s.width = w
	return s
}

// Height sets the button height.
func (s *styledButtonView) Height(h int) *styledButtonView {
	s.height = h
	return s
}

// Bg sets the background color.
func (s *styledButtonView) Bg(c Color) *styledButtonView {
	s.style = s.style.WithBackground(c)
	return s
}

// Fg sets the foreground color.
func (s *styledButtonView) Fg(c Color) *styledButtonView {
	s.style = s.style.WithForeground(c)
	return s
}

// Style sets the complete style.
func (s *styledButtonView) Style(st Style) *styledButtonView {
	s.style = st
	return s
}

// HoverStyle sets the style when hovered (if hover tracking is enabled).
func (s *styledButtonView) HoverStyle(st Style) *styledButtonView {
	s.hoverStyle = st
	return s
}

// Bold makes the label bold.
func (s *styledButtonView) Bold() *styledButtonView {
	s.style = s.style.WithBold()
	return s
}

// Border sets the border style.
func (s *styledButtonView) Border(style borderStyleType) *styledButtonView {
	s.borderStyle = style
	return s
}

// Centered sets whether the label is centered (default true).
func (s *styledButtonView) Centered(centered bool) *styledButtonView {
	s.centered = centered
	return s
}

func (s *styledButtonView) size(maxWidth, maxHeight int) (int, int) {
	labelW, _ := MeasureText(s.label)
	w := s.width
	if w == 0 {
		w = labelW + 2 // padding
	}
	h := s.height
	if h == 0 {
		h = 1
	}

	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}
	return w, h
}

func (s *styledButtonView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() {
		return
	}

	subFrame := frame.SubFrame(bounds)
	width := bounds.Dx()
	height := bounds.Dy()

	// Fill background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			subFrame.SetCell(x, y, ' ', s.style)
		}
	}

	// Calculate text position
	labelW, _ := MeasureText(s.label)
	textX := 0
	textY := height / 2
	if s.centered {
		textX = (width - labelW) / 2
		if textX < 0 {
			textX = 0
		}
	}

	// Draw label
	subFrame.PrintTruncated(textX, textY, s.label, s.style)

	// Register click region
	if s.callback != nil {
		interactiveRegistry.RegisterButton(bounds, s.callback)
	}
}
