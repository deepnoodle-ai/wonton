package tui

// Declarations belong to the composed view, including leaves outside its frame.
// Drawing collects visible glyphs separately, so clipping cannot hide a conflict.
func collectSourceBindings(view View, layout *sourceLayout) {
	var children []View
	switch v := view.(type) {
	case *TextView:
		if v.sourceMarked {
			layout.bind(v.content, v.sourceOffset)
		}
	case *MarkdownView:
		if v.sourceMarked {
			layout.bind(v.content, v.sourceOffset)
		}
	case *StackView:
		children = v.children
	case *GroupView:
		children = v.children
	case *ZStackView:
		children = v.children
	case *paddingView:
		children = []View{v.inner}
	case *sizeView:
		children = []View{v.inner}
	case *BorderedView:
		children = []View{v.inner}
	case *PanelView:
		children = []View{v.content}
	case *ScrollView:
		children = []View{v.inner}
	case *focusableWrapper:
		children = []View{v.inner}
	case *AnimatedBorderedView:
		children = []View{v.inner}
	case interface{ sourceChildren() []View }:
		children = v.sourceChildren()
	}
	for _, child := range children {
		collectSourceBindings(child, layout)
	}
}
func (f *ForEachView[T]) sourceChildren() []View  { return f.buildStack().children }
func (f *HForEachView[T]) sourceChildren() []View { return f.buildStack().children }
