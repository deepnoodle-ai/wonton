package tui

import "image"

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

// Size sets both width and height at once.
func (b *panelView) Size(w, h int) *panelView {
	b.width = w
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

func (b *panelView) render(ctx *RenderContext) {
	width, height := ctx.Size()
	if width == 0 || height == 0 {
		return
	}

	// Fill background
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ctx.SetCell(x, y, b.fillChar, b.bgStyle)
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
			ctx.SetCell(x, 0, h, borderStyle)
			ctx.SetCell(x, height-1, h, borderStyle)
		}

		// Left and right
		for y := 1; y < height-1; y++ {
			ctx.SetCell(0, y, v, borderStyle)
			ctx.SetCell(width-1, y, v, borderStyle)
		}

		// Corners
		ctx.SetCell(0, 0, tl, borderStyle)
		ctx.SetCell(width-1, 0, tr, borderStyle)
		ctx.SetCell(0, height-1, bl, borderStyle)
		ctx.SetCell(width-1, height-1, br, borderStyle)

		// Title
		if b.title != "" && width > 4 {
			titleText := " " + b.title + " "
			titleW, _ := MeasureText(titleText)
			if titleW > width-2 {
				titleW = width - 2
			}
			startX := (width - titleW) / 2
			ctx.PrintTruncated(startX, 0, titleText, borderStyle)
		}
	}

	// Render content if provided
	if b.content != nil {
		contentCtx := ctx
		if b.borderStyle != BorderNone {
			contentCtx = ctx.SubContext(image.Rect(1, 1, width-1, height-1))
		}
		b.content.render(contentCtx)
	}
}
