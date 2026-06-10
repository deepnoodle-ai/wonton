package tui

// ListItem represents an item in a list. It is the shared item type for the
// list-family views: SelectList, FilterableList, CheckboxList, RadioList, and
// FilePicker.
type ListItem struct {
	Label string
	Value interface{}
	Icon  string // Optional icon/prefix
}
