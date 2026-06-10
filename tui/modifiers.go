package tui

// This file contains modifier methods that extend view types with builder-style APIs.
// The actual view implementations are in their respective files:
// - padding_view.go
// - size_view.go
// - bordered_view.go

// Padding modifier methods for stack types

// Padding adds equal padding on all sides of a Stack.
func (v *StackView) Padding(n int) View {
	return Padding(n, v)
}

// PaddingHV adds horizontal and vertical padding to a Stack.
func (v *StackView) PaddingHV(h, vpad int) View {
	return PaddingHV(h, vpad, v)
}

// PaddingLTRB adds specific padding to each side of a Stack.
func (v *StackView) PaddingLTRB(left, top, right, bottom int) View {
	return PaddingLTRB(left, top, right, bottom, v)
}

// Padding adds equal padding on all sides of a Group.
func (h *GroupView) Padding(n int) View {
	return Padding(n, h)
}

// PaddingHV adds horizontal and vertical padding to a Group.
func (h *GroupView) PaddingHV(hpad, v int) View {
	return PaddingHV(hpad, v, h)
}

// PaddingLTRB adds specific padding to each side of a Group.
func (h *GroupView) PaddingLTRB(left, top, right, bottom int) View {
	return PaddingLTRB(left, top, right, bottom, h)
}

// Padding adds equal padding on all sides of a ZStack.
func (z *ZStackView) Padding(n int) View {
	return Padding(n, z)
}

// Size modifier methods for view types

// Width sets a fixed width for a TextView.
func (t *TextView) Width(w int) View {
	return Width(w, t)
}

// Height sets a fixed height for a TextView.
func (t *TextView) Height(h int) View {
	return Height(h, t)
}

// MaxWidth sets a maximum width for a TextView.
func (t *TextView) MaxWidth(w int) View {
	return MaxWidth(w, t)
}

// Bordered modifier methods for stack types

// Bordered wraps a Stack with a border.
func (v *StackView) Bordered() *BorderedView {
	return Bordered(v)
}

// Bordered wraps a Group with a border.
func (h *GroupView) Bordered() *BorderedView {
	return Bordered(h)
}

// Bordered wraps a ZStack with a border.
func (z *ZStackView) Bordered() *BorderedView {
	return Bordered(z)
}

// Background modifier

// Background wraps a view with a background fill.
func Background(char rune, style Style, inner View) View {
	return &ZStackView{
		children: []View{
			&FillView{char: char, style: style},
			inner,
		},
		alignment: AlignLeft,
	}
}

// Bg adds a background color to a Stack.
func (v *StackView) Bg(c Color) View {
	return Background(' ', NewStyle().WithBackground(c), v)
}

// Bg adds a background color to a Group.
func (h *GroupView) Bg(c Color) View {
	return Background(' ', NewStyle().WithBackground(c), h)
}
