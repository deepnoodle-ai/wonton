package tui

// listNav describes a cursor movement requested by a navigation key. It is
// the shared key handling for the list-family views: SelectListView,
// FilterableListView, CheckboxListView, RadioListView, TableView, and
// TreeView.
type listNav int

const (
	listNavNone listNav = iota
	listNavUp
	listNavDown
	listNavPageUp
	listNavPageDown
	listNavHome
	listNavEnd
)

// listNavForKey maps a key event to a navigation action using one standard
// key map: arrow keys, PageUp/PageDown, Home/End, and the vi-style j/k and
// g/G when allowVi is true. Views that route printable characters to a text
// input (e.g. a filter field) pass allowVi=false so letters reach the input.
func listNavForKey(event KeyEvent, allowVi bool) listNav {
	switch event.Key {
	case KeyArrowUp:
		return listNavUp
	case KeyArrowDown:
		return listNavDown
	case KeyPageUp:
		return listNavPageUp
	case KeyPageDown:
		return listNavPageDown
	case KeyHome:
		return listNavHome
	case KeyEnd:
		return listNavEnd
	}
	if allowVi {
		switch event.Rune {
		case 'k':
			return listNavUp
		case 'j':
			return listNavDown
		case 'g':
			return listNavHome
		case 'G':
			return listNavEnd
		}
	}
	return listNavNone
}

// moveListCursor applies a navigation action to a cursor over count items.
// page is the PageUp/PageDown step, typically the visible height. The new
// cursor is clamped to [0, count). The second result reports whether the
// cursor moved; an action that hits a boundary without moving returns false
// so the key can propagate, matching the previous arrow-key behavior.
func moveListCursor(nav listNav, cursor, count, page int) (int, bool) {
	if nav == listNavNone || count <= 0 {
		return cursor, false
	}
	if page < 1 {
		page = 1
	}
	next := cursor
	switch nav {
	case listNavUp:
		next = cursor - 1
	case listNavDown:
		next = cursor + 1
	case listNavPageUp:
		next = cursor - page
	case listNavPageDown:
		next = cursor + page
	case listNavHome:
		next = 0
	case listNavEnd:
		next = count - 1
	}
	if next < 0 {
		next = 0
	}
	if next > count-1 {
		next = count - 1
	}
	return next, next != cursor
}

// scrollIntoView adjusts scroll so cursor lies within the visible window
// [scroll, scroll+visible). A nil scroll or non-positive visible is a no-op.
func scrollIntoView(scroll *int, cursor, visible int) {
	if scroll == nil || visible <= 0 {
		return
	}
	if *scroll > cursor {
		*scroll = cursor
	}
	if cursor-*scroll >= visible {
		*scroll = cursor - visible + 1
	}
	if *scroll < 0 {
		*scroll = 0
	}
}
