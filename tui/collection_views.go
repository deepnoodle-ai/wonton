package tui

// ForEach maps a slice of items to views using a mapper function.
// The resulting views are arranged in a VStack by default.
//
// Example:
//
//	ForEach(app.items, func(item Item, i int) View {
//	    return Text("%d. %s", i+1, item.Name)
//	})
func ForEach[T any](items []T, mapper func(item T, index int) View) *forEachView[T] {
	return &forEachView[T]{
		items:     items,
		mapper:    mapper,
		separator: nil,
	}
}

// forEachView represents a collection of views generated from items
type forEachView[T any] struct {
	items     []T
	mapper    func(item T, index int) View
	separator View
	cached    *stack // cached result for rendering
}

// Separator sets a view to be rendered between each item.
func (f *forEachView[T]) Separator(sep View) *forEachView[T] {
	f.separator = sep
	return f
}

func (f *forEachView[T]) buildStack() *stack {
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
	return f.cached
}

func (f *forEachView[T]) size(maxWidth, maxHeight int) (int, int) {
	// Clear cache to rebuild with fresh mapper calls
	f.cached = nil
	return f.buildStack().size(maxWidth, maxHeight)
}

func (f *forEachView[T]) render(ctx *RenderContext) {
	f.buildStack().render(ctx)
}

// Gap sets the spacing between items (like VStack.Gap).
func (f *forEachView[T]) Gap(n int) *forEachView[T] {
	f.buildStack().gap = n
	return f
}

// HForEach is like ForEach but arranges items horizontally in an HStack.
//
// Example:
//
//	HForEach(app.tabs, func(tab Tab, i int) View {
//	    return Text(tab.Title).Padding(1)
//	})
func HForEach[T any](items []T, mapper func(item T, index int) View) *hForEachView[T] {
	return &hForEachView[T]{
		items:     items,
		mapper:    mapper,
		separator: nil,
	}
}

// hForEachView arranges mapped views horizontally
type hForEachView[T any] struct {
	items     []T
	mapper    func(item T, index int) View
	separator View
	cached    *group
}

// Separator sets a view to be rendered between each item.
func (f *hForEachView[T]) Separator(sep View) *hForEachView[T] {
	f.separator = sep
	return f
}

func (f *hForEachView[T]) buildStack() *group {
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
	return f.cached
}

func (f *hForEachView[T]) size(maxWidth, maxHeight int) (int, int) {
	f.cached = nil
	return f.buildStack().size(maxWidth, maxHeight)
}

func (f *hForEachView[T]) render(ctx *RenderContext) {
	f.buildStack().render(ctx)
}

// Gap sets the spacing between items (like HStack.Gap).
func (f *hForEachView[T]) Gap(n int) *hForEachView[T] {
	f.buildStack().gap = n
	return f
}
