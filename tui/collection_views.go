package tui

// ForEach maps a slice of items to views using a mapper function.
// The resulting views are arranged in a Stack by default.
//
// Example:
//
//	ForEach(app.items, func(item Item, i int) View {
//	    return Text("%d. %s", i+1, item.Name)
//	})
func ForEach[T any](items []T, mapper func(item T, index int) View) *ForEachView[T] {
	return &ForEachView[T]{
		items:     items,
		mapper:    mapper,
		separator: nil,
	}
}

// ForEachView represents a collection of views generated from items
type ForEachView[T any] struct {
	items     []T
	mapper    func(item T, index int) View
	separator View
	gap       int
	cached    *StackView // cached result for rendering
}

// Separator sets a view to be rendered between each item.
func (f *ForEachView[T]) Separator(sep View) *ForEachView[T] {
	f.separator = sep
	return f
}

func (f *ForEachView[T]) buildStack() *StackView {
	if f.cached != nil {
		return f.cached
	}

	var views []View
	for i, item := range f.items {
		if i > 0 && f.separator != nil {
			views = append(views, f.separator)
		}
		views = append(views, f.mapper(item, i))
	}

	f.cached = Stack(views...)
	if f.gap > 0 {
		f.cached.Gap(f.gap)
	}
	return f.cached
}

func (f *ForEachView[T]) size(maxWidth, maxHeight int) (int, int) {
	// Clear cache to rebuild with fresh mapper calls
	f.cached = nil
	return f.buildStack().size(maxWidth, maxHeight)
}

func (f *ForEachView[T]) render(ctx *RenderContext) {
	f.buildStack().render(ctx)
}

// Gap sets the spacing between items (like Stack.Gap).
func (f *ForEachView[T]) Gap(n int) *ForEachView[T] {
	f.gap = n
	return f
}

// HForEach is like ForEach but arranges items horizontally in a Group.
//
// Example:
//
//	HForEach(app.tabs, func(tab Tab, i int) View {
//	    return Text(tab.Title).Padding(1)
//	})
func HForEach[T any](items []T, mapper func(item T, index int) View) *HForEachView[T] {
	return &HForEachView[T]{
		items:     items,
		mapper:    mapper,
		separator: nil,
	}
}

// HForEachView arranges mapped views horizontally
type HForEachView[T any] struct {
	items     []T
	mapper    func(item T, index int) View
	separator View
	gap       int
	cached    *GroupView
}

// Separator sets a view to be rendered between each item.
func (f *HForEachView[T]) Separator(sep View) *HForEachView[T] {
	f.separator = sep
	return f
}

func (f *HForEachView[T]) buildStack() *GroupView {
	if f.cached != nil {
		return f.cached
	}

	var views []View
	for i, item := range f.items {
		if i > 0 && f.separator != nil {
			views = append(views, f.separator)
		}
		views = append(views, f.mapper(item, i))
	}

	f.cached = Group(views...)
	if f.gap > 0 {
		f.cached.Gap(f.gap)
	}
	return f.cached
}

func (f *HForEachView[T]) size(maxWidth, maxHeight int) (int, int) {
	f.cached = nil
	return f.buildStack().size(maxWidth, maxHeight)
}

func (f *HForEachView[T]) render(ctx *RenderContext) {
	f.buildStack().render(ctx)
}

// Gap sets the spacing between items (like Group.Gap).
func (f *HForEachView[T]) Gap(n int) *HForEachView[T] {
	f.gap = n
	return f
}
