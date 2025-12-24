package tui

// This file contains modifier methods that extend view types with builder-style APIs.
// The actual view implementations are in their respective files:
// - padding_view.go
// - size_view.go
// - bordered_view.go

// Padding modifier methods for stack types

// Padding adds equal padding on all sides of a Stack.
func (v *stack) Padding(n int) View {
	return Padding(n, v)
}

// PaddingHV adds horizontal and vertical padding to a Stack.
func (v *stack) PaddingHV(h, vpad int) View {
	return PaddingHV(h, vpad, v)
}

// PaddingLTRB adds specific padding to each side of a Stack.
func (v *stack) PaddingLTRB(left, top, right, bottom int) View {
	return PaddingLTRB(left, top, right, bottom, v)
}

// Padding adds equal padding on all sides of a Group.
func (h *group) Padding(n int) View {
	return Padding(n, h)
}

// PaddingHV adds horizontal and vertical padding to a Group.
func (h *group) PaddingHV(hpad, v int) View {
	return PaddingHV(hpad, v, h)
}

// PaddingLTRB adds specific padding to each side of a Group.
func (h *group) PaddingLTRB(left, top, right, bottom int) View {
	return PaddingLTRB(left, top, right, bottom, h)
}

// Padding adds equal padding on all sides of a ZStack.
func (z *zStack) Padding(n int) View {
	return Padding(n, z)
}

// Size modifier methods for view types

// Width sets a fixed width for a textView.
func (t *textView) Width(w int) View {
	return Width(w, t)
}

// Height sets a fixed height for a textView.
func (t *textView) Height(h int) View {
	return Height(h, t)
}

// MaxWidth sets a maximum width for a textView.
func (t *textView) MaxWidth(w int) View {
	return MaxWidth(w, t)
}

// Bordered modifier methods for stack types

// Bordered wraps a Stack with a border.
func (v *stack) Bordered() *borderedView {
	return Bordered(v)
}

// Bordered wraps a Group with a border.
func (h *group) Bordered() *borderedView {
	return Bordered(h)
}

// Bordered wraps a ZStack with a border.
func (z *zStack) Bordered() *borderedView {
	return Bordered(z)
}

// Background modifier

// Background wraps a view with a background fill.
func Background(char rune, style Style, inner View) View {
	return &zStack{
		children: []View{
			&fillView{char: char, style: style},
			inner,
		},
		alignment: AlignLeft,
	}
}

// Bg adds a background color to a Stack.
func (v *stack) Bg(c Color) View {
	return Background(' ', NewStyle().WithBackground(c), v)
}

// Bg adds a background color to a Group.
func (h *group) Bg(c Color) View {
	return Background(' ', NewStyle().WithBackground(c), h)
}
