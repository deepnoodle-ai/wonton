package tui

import (
	"image"
	"strings"
	"unicode"

	"github.com/deepnoodle-ai/wonton/runewidth"
)

// MenuItem is one entry in a popup menu.
type MenuItem struct {
	// Label is the entry's text.
	Label string

	// Hint is shown right-aligned after the label, typically a keyboard
	// shortcut.
	Hint string

	// Disabled entries are shown dimmed and cannot be chosen.
	Disabled bool

	// Danger marks a destructive entry, such as one that closes or deletes
	// something. It is drawn in the menu's danger style.
	Danger bool

	// Separator makes the entry a horizontal rule. Its other fields are
	// ignored.
	Separator bool

	// Action runs when the entry is chosen, after the menu has closed.
	Action func()
}

// MenuSeparator returns a menu entry that draws a horizontal rule.
func MenuSeparator() MenuItem { return MenuItem{Separator: true} }

func (it MenuItem) selectable() bool { return !it.Separator && !it.Disabled }

// MenuState is a popup menu — a context menu opened at the pointer, or a
// dropdown opened under a button. The application owns it, opens it with
// Open, draws it with Menu on top of everything else, and gives it events
// first while it is open:
//
//	func (a *app) View() tui.View {
//	    return tui.ZStack(a.main(), tui.Menu(&a.menu))
//	}
//
//	func (a *app) HandleEvent(ev tui.Event) []tui.Cmd {
//	    switch e := ev.(type) {
//	    case tui.MouseEvent:
//	        if a.menu.HandleMouse(e) {
//	            return nil
//	        }
//	        if e.Type == tui.MouseClick && e.Button == tui.MouseButtonRight {
//	            a.menu.Open(e.X, e.Y, "", a.itemsFor(e.X, e.Y))
//	        }
//	    case tui.KeyEvent:
//	        if a.menu.HandleKey(e) {
//	            return nil
//	        }
//	    }
//	    return nil
//	}
//
// An open menu is modal: it takes every key and mouse event. A click outside
// it closes it without reaching what was clicked.
type MenuState struct {
	title    string
	items    []MenuItem
	selected int
	open     bool
	anchor   image.Point
	bounds   image.Rectangle // the box at the last render, in screen cells
	itemTop  int             // screen row of the first item
	shown    int             // items that fit in the box at the last render

	// dismissed is set when a press outside closed the menu, so the rest of
	// that press — drags, the release, the click — is swallowed too.
	dismissed bool
}

// Open shows the menu with its top-left corner at the screen position
// (x, y), moved as needed to fit on screen. A non-empty title is shown as a
// header above the items. The first item that can be chosen is selected.
func (m *MenuState) Open(x, y int, title string, items []MenuItem) {
	m.title = title
	m.items = items
	m.anchor = image.Pt(x, y)
	m.open = true
	m.dismissed = false
	m.bounds = image.Rectangle{}
	m.shown = len(items)
	m.selected = -1
	m.move(1)
}

// Close hides the menu without choosing anything.
func (m *MenuState) Close() {
	m.open = false
	m.bounds = image.Rectangle{}
}

// IsOpen reports whether the menu is showing.
func (m *MenuState) IsOpen() bool { return m.open }

// Items returns the menu's entries.
func (m *MenuState) Items() []MenuItem { return m.items }

// Selected returns the index of the highlighted entry, or -1.
func (m *MenuState) Selected() int { return m.selected }

// Bounds returns where the menu was drawn at the last render, in screen
// coordinates. It is empty while the menu is closed.
func (m *MenuState) Bounds() image.Rectangle { return m.bounds }

// Choose closes the menu and runs entry i's action, if it can be chosen.
func (m *MenuState) Choose(i int) bool {
	if i < 0 || i >= len(m.items) || !m.items[i].selectable() {
		return false
	}
	m.Close()
	if action := m.items[i].Action; action != nil {
		action()
	}
	return true
}

// move steps the selection by dir (+1 or -1) to the next shown entry that
// can be chosen, wrapping around.
func (m *MenuState) move(dir int) {
	n := m.shown
	if n == 0 {
		m.selected = -1
		return
	}
	i := m.selected
	for range n {
		i = (i + dir + n) % n
		if m.items[i].selectable() {
			m.selected = i
			return
		}
	}
}

// HandleKey handles a key while the menu is open: Up and Down (or Tab) move
// the selection, Enter or Space chooses, Escape closes, and a letter selects
// the next entry starting with it — and chooses it if it is the only one.
// Every key is consumed while the menu is open; HandleKey returns false only
// when it is closed.
func (m *MenuState) HandleKey(e KeyEvent) bool {
	if !m.open {
		return false
	}
	switch {
	case e.Key == KeyEscape || isInterruptKey(e):
		m.Close()
	case e.Key == KeyArrowUp || (e.Key == KeyTab && e.Shift):
		m.move(-1)
	case e.Key == KeyArrowDown || e.Key == KeyTab:
		m.move(1)
	case e.Key == KeyHome:
		m.selected = -1
		m.move(1)
	case e.Key == KeyEnd:
		m.selected = m.shown
		m.move(-1)
	case e.Key == KeyEnter || (e.Key == KeyUnknown && e.Rune == ' '):
		m.Choose(m.selected)
	case e.Key == KeyUnknown && e.Rune != 0 && !e.Ctrl && !e.Alt:
		m.jump(e.Rune)
	}
	return true
}

// jump selects the next entry whose label starts with r, choosing it when no
// other entry does.
func (m *MenuState) jump(r rune) {
	r = unicode.ToLower(r)
	var matches []int
	for i, it := range m.items[:m.shown] {
		if !it.selectable() {
			continue
		}
		first, _ := firstRuneOf(it.Label)
		if unicode.ToLower(first) == r {
			matches = append(matches, i)
		}
	}
	switch len(matches) {
	case 0:
	case 1:
		m.Choose(matches[0])
	default:
		next := matches[0]
		for _, i := range matches {
			if i > m.selected {
				next = i
				break
			}
		}
		m.selected = next
	}
}

func firstRuneOf(s string) (rune, bool) {
	for _, r := range strings.TrimSpace(s) {
		return r, true
	}
	return 0, false
}

// HandleMouse handles a mouse event while the menu is open. Moving over an
// entry selects it and clicking chooses it, with either button. A press
// outside the menu closes it, and the rest of that press, up to its release
// and click, is consumed as well, so it never reaches what was clicked.
// Every event is consumed while the menu is open; otherwise HandleMouse
// returns false.
func (m *MenuState) HandleMouse(e MouseEvent) bool {
	if !m.open {
		if m.dismissed {
			switch e.Type {
			case MouseDrag, MouseRelease, MouseClick, MouseDoubleClick, MouseTripleClick:
				return true
			}
			m.dismissed = false
		}
		return false
	}
	pt := image.Pt(e.X, e.Y)
	inside := pt.In(m.bounds)
	switch e.Type {
	case MouseMove, MouseDrag:
		if i := m.itemAt(e.Y); inside && i >= 0 && m.items[i].selectable() {
			m.selected = i
		}
	case MousePress:
		if !inside && e.Button != MouseButtonWheelUp && e.Button != MouseButtonWheelDown {
			m.Close()
			m.dismissed = true
		}
	case MouseClick:
		if inside {
			m.Choose(m.itemAt(e.Y))
		}
	}
	return true
}

func (m *MenuState) itemAt(y int) int {
	i := y - m.itemTop
	if i < 0 || i >= m.shown {
		return -1
	}
	return i
}

// MenuView draws a MenuState.
type MenuView struct {
	state         *MenuState
	style         Style
	selectedStyle Style
	disabledStyle Style
	dangerStyle   Style
	hintStyle     Style
	titleStyle    Style
	borderStyle   Style
	border        *BorderStyle
}

// Menu draws state's menu when it is open, and nothing when it is closed.
// It fills the space it is given so it can be laid over the rest of the
// screen in a ZStack; put it last so it is drawn on top.
func Menu(state *MenuState) *MenuView {
	return &MenuView{
		state:         state,
		style:         NewStyle(),
		selectedStyle: NewStyle().WithReverse(),
		disabledStyle: NewStyle().WithDim(),
		dangerStyle:   NewStyle().WithForeground(ColorRed),
		hintStyle:     NewStyle().WithDim(),
		titleStyle:    NewStyle().WithBold(),
		borderStyle:   NewStyle(),
		border:        &RoundedBorder,
	}
}

// Style sets the style of the menu's background and entries.
func (v *MenuView) Style(s Style) *MenuView { v.style = s; return v }

// SelectedStyle sets the style of the highlighted entry. The default
// reverses it.
func (v *MenuView) SelectedStyle(s Style) *MenuView { v.selectedStyle = s; return v }

// DisabledStyle sets the style of entries that cannot be chosen.
func (v *MenuView) DisabledStyle(s Style) *MenuView { v.disabledStyle = s; return v }

// DangerStyle sets the style of destructive entries. The default is red.
func (v *MenuView) DangerStyle(s Style) *MenuView { v.dangerStyle = s; return v }

// HintStyle sets the style of the hints after the labels.
func (v *MenuView) HintStyle(s Style) *MenuView { v.hintStyle = s; return v }

// TitleStyle sets the style of the title.
func (v *MenuView) TitleStyle(s Style) *MenuView { v.titleStyle = s; return v }

// BorderStyle sets the style of the border.
func (v *MenuView) BorderStyle(s Style) *MenuView { v.borderStyle = s; return v }

// Border sets the border characters. The default is RoundedBorder.
func (v *MenuView) Border(b *BorderStyle) *MenuView {
	if b != nil {
		v.border = b
	}
	return v
}

func (v *MenuView) flex() int { return 1 }

func (v *MenuView) size(maxWidth, maxHeight int) (int, int) {
	return maxWidth, maxHeight
}

func (v *MenuView) render(ctx *RenderContext) {
	m := v.state
	if !m.open {
		return
	}
	screen := ctx.AbsoluteBounds()

	// Size the box to its widest line: " label   hint ".
	labelW, hintW := 0, 0
	for _, it := range m.items {
		labelW = max(labelW, runewidth.StringWidth(it.Label))
		hintW = max(hintW, runewidth.StringWidth(it.Hint))
	}
	inner := labelW + 2
	if hintW > 0 {
		inner += hintW + 3
	}
	if m.title != "" {
		inner = max(inner, runewidth.StringWidth(m.title)+2)
	}
	header := 0
	if m.title != "" {
		header = 2 // the title and a rule under it
	}
	w := min(inner+2, screen.Dx())
	h := min(len(m.items)+header+2, screen.Dy())
	if w < 4 || h < 3 {
		return
	}

	// Open at the anchor, flipping to the other side of it when the box
	// would run off the screen, and clamping as a last resort.
	x, y := m.anchor.X, m.anchor.Y
	if x+w > screen.Max.X {
		x = max(screen.Min.X, min(m.anchor.X-w+1, screen.Max.X-w))
	}
	if y+h > screen.Max.Y {
		y = max(screen.Min.Y, min(m.anchor.Y-h+1, screen.Max.Y-h))
	}
	m.bounds = image.Rect(x, y, x+w, y+h)
	m.itemTop = y + 1 + header
	// A box cut short by the screen shows only the items that fit; the keys
	// and the mouse reach only those.
	m.shown = max(0, min(len(m.items), h-2-header))
	if m.selected >= m.shown {
		m.selected = -1
		m.move(1)
	}

	box := m.bounds.Sub(screen.Min)
	ctx.FillStyled(box.Min.X, box.Min.Y, w, h, ' ', v.style)
	v.drawBorder(ctx, box)

	row := box.Min.Y + 1
	if m.title != "" {
		ctx.PrintTruncated(box.Min.X+2, row, runewidth.Truncate(m.title, w-4, "…"), v.style.Merge(v.titleStyle))
		row++
		v.drawRule(ctx, box, row)
		row++
	}
	for i, it := range m.items[:m.shown] {
		if it.Separator {
			v.drawRule(ctx, box, row)
			row++
			continue
		}
		style := v.style
		switch {
		case it.Disabled:
			style = style.Merge(v.disabledStyle)
		case it.Danger:
			style = style.Merge(v.dangerStyle)
		}
		if i == m.selected {
			style = style.Merge(v.selectedStyle)
		}
		ctx.FillStyled(box.Min.X+1, row, w-2, 1, ' ', style)
		ctx.PrintTruncated(box.Min.X+2, row, runewidth.Truncate(it.Label, w-4, "…"), style)
		if it.Hint != "" {
			hw := runewidth.StringWidth(it.Hint)
			hx := box.Max.X - 2 - hw
			if hx > box.Min.X+2+runewidth.StringWidth(it.Label) {
				hint := style
				if i != m.selected {
					hint = style.Merge(v.hintStyle)
				}
				ctx.PrintTruncated(hx, row, it.Hint, hint)
			}
		}
		row++
	}
}

func (v *MenuView) drawBorder(ctx *RenderContext, r image.Rectangle) {
	b := v.border
	style := v.style.Merge(v.borderStyle)
	for x := r.Min.X + 1; x < r.Max.X-1; x++ {
		ctx.PrintTruncated(x, r.Min.Y, b.Horizontal, style)
		ctx.PrintTruncated(x, r.Max.Y-1, b.Horizontal, style)
	}
	for y := r.Min.Y + 1; y < r.Max.Y-1; y++ {
		ctx.PrintTruncated(r.Min.X, y, b.Vertical, style)
		ctx.PrintTruncated(r.Max.X-1, y, b.Vertical, style)
	}
	ctx.PrintTruncated(r.Min.X, r.Min.Y, b.TopLeft, style)
	ctx.PrintTruncated(r.Max.X-1, r.Min.Y, b.TopRight, style)
	ctx.PrintTruncated(r.Min.X, r.Max.Y-1, b.BottomLeft, style)
	ctx.PrintTruncated(r.Max.X-1, r.Max.Y-1, b.BottomRight, style)
}

func (v *MenuView) drawRule(ctx *RenderContext, r image.Rectangle, y int) {
	style := v.style.Merge(v.borderStyle)
	left, right := v.border.LeftJoin, v.border.RightJoin
	if left == "" {
		left, right = v.border.Vertical, v.border.Vertical
	}
	ctx.PrintTruncated(r.Min.X, y, left, style)
	for x := r.Min.X + 1; x < r.Max.X-1; x++ {
		ctx.PrintTruncated(x, y, v.border.Horizontal, style)
	}
	ctx.PrintTruncated(r.Max.X-1, y, right, style)
}
