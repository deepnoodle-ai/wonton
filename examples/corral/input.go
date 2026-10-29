package main

import (
	"image"
	"strings"

	"github.com/deepnoodle-ai/wonton/tui"
)

// isPrefix reports whether e is Ctrl+B, which however the terminal encodes
// it arrives either as its own key or as a Ctrl-modified 'b'.
func isPrefix(e tui.KeyEvent) bool {
	return e.Key == tui.KeyCtrlB || (e.Key == tui.KeyUnknown && e.Ctrl && (e.Rune == 'b' || e.Rune == 'B'))
}

func (a *app) handleKey(e tui.KeyEvent) []tui.Cmd {
	if !a.selecting {
		a.selPane = "" // typing dismisses a selection
	}
	if a.menu.HandleKey(e) {
		return nil
	}
	if a.dialog != nil {
		// The input field has the keys it wants already; Escape cancels.
		if e.Key == tui.KeyEscape {
			a.dialog = nil
		}
		return nil
	}
	if a.prefix {
		a.prefix = false
		if isPrefix(e) {
			if p := a.focused(); p != nil {
				_ = p.term.SendKey(e)
			}
			return nil
		}
		return a.command(e)
	}
	if isPrefix(e) {
		a.prefix = true
		return nil
	}
	if p := a.focused(); p != nil {
		_ = p.term.SendKey(e)
	}
	return nil
}

// command runs the corral command bound to the key typed after the prefix.
func (a *app) command(e tui.KeyEvent) []tui.Cmd {
	sp := a.space()
	focus := sp.focus
	switch e.Key {
	case tui.KeyArrowLeft:
		a.moveFocus(-1, 0)
		return nil
	case tui.KeyArrowRight:
		a.moveFocus(1, 0)
		return nil
	case tui.KeyArrowUp:
		a.moveFocus(0, -1)
		return nil
	case tui.KeyArrowDown:
		a.moveFocus(0, 1)
		return nil
	}
	switch e.Rune {
	case '|', '%', '\\':
		return a.splitPane(focus, tui.SplitColumns)
	case '-', '"':
		return a.splitPane(focus, tui.SplitRows)
	case 'c':
		return a.newColumn(sp)
	case 'x':
		return a.closePane(focus)
	case 'z':
		a.toggleZoom(focus)
	case 'h':
		a.moveFocus(-1, 0)
	case 'l':
		a.moveFocus(1, 0)
	case 'k':
		a.moveFocus(0, -1)
	case 'j':
		a.moveFocus(0, 1)
	case 'o':
		a.cycleFocus(1)
	case 'n':
		a.switchSpace((a.current + 1) % len(a.spaces))
	case 'p':
		a.switchSpace((a.current - 1 + len(a.spaces)) % len(a.spaces))
	case 's':
		return a.newSpace(a.startDir(sp, a.focused()))
	case ',':
		return a.renamePane(focus)
	case '=':
		sp.split.Equalize()
		return a.save()
	case 'm':
		if b, ok := a.hits.Bounds("pane:" + focus); ok {
			a.openMenu("pane:"+focus, b.Min.X+2, b.Min.Y+1)
		}
	case '?':
		a.openHelp()
	case 'd', 'q':
		return a.quit()
	}
	return nil
}

func (a *app) handleMouse(e tui.MouseEvent) []tui.Cmd {
	if a.menu.HandleMouse(e) {
		return nil
	}
	if a.dialog != nil {
		// Modal: a press outside the prompt cancels it.
		if e.Type == tui.MousePress {
			if b, ok := a.hits.Bounds("dialog"); !ok || !inRect(e.X, e.Y, b) {
				a.dialog = nil
				a.swallow = true
			}
		}
		return nil
	}
	if a.swallow {
		// The release and click of the press that cancelled the prompt.
		switch e.Type {
		case tui.MouseDrag, tui.MouseRelease, tui.MouseClick, tui.MouseDoubleClick, tui.MouseTripleClick:
			return nil
		}
		a.swallow = false
	}

	// Dragging the sidebar's edge.
	if a.navDrag {
		switch e.Type {
		case tui.MouseDrag, tui.MouseMove:
			a.navWidth = max(16, min(e.X, a.screenWidth()/2))
		case tui.MouseRelease:
			a.navDrag = false
		}
		return nil
	}

	// A drag that started in a pane whose program reads the mouse stays
	// with that pane until the button is released, wherever it goes.
	if a.capture != "" {
		if p := a.panes[a.capture]; p != nil {
			if x, y, ok := a.paneLocal(p.id, e.X, e.Y, true); ok {
				p.term.SendMouse(e, x, y)
			}
		}
		if e.Type == tui.MouseRelease {
			a.capture = ""
		}
		return nil
	}

	// Selecting text: the drag stays with its pane, wherever it goes.
	if a.selecting {
		if x, y, ok := a.paneLocal(a.selPane, e.X, e.Y, true); ok {
			a.sel.Cursor = image.Pt(x, y)
		}
		if e.Type == tui.MouseRelease {
			a.selecting = false
			a.copySelection()
		}
		return nil
	}

	sp := a.space()
	if sp.zoom == "" && sp.split.HandleMouse(e) {
		if e.Type == tui.MouseRelease {
			return a.save()
		}
		return nil
	}

	key, _ := a.hits.At(e.X, e.Y)
	kind, id, _ := strings.Cut(key, ":")

	switch e.Type {
	case tui.MouseScroll:
		if kind == "pane" {
			a.scrollPane(id, e)
		}
	case tui.MousePress:
		a.selPane = "" // any press dismisses a selection
		if key == "nav-gutter" && e.Button == tui.MouseButtonLeft {
			a.navDrag = true
			return nil
		}
		if kind == "pane" {
			a.focusPane(id)
			if e.Button == tui.MouseButtonRight {
				return nil
			}
			x, y, ok := a.paneLocal(id, e.X, e.Y, false)
			switch {
			case !ok:
			case a.panes[id].term.SendMouse(e, x, y):
				// The pane's program reads the mouse: the drag is its.
				a.capture = id
			case e.Button == tui.MouseButtonLeft:
				pt := image.Pt(x, y)
				a.selPane, a.sel, a.selecting = id, tui.GridSelection{Anchor: pt, Cursor: pt}, true
			}
		}
	case tui.MouseClick:
		if e.Button == tui.MouseButtonRight {
			a.openMenu(key, e.X, e.Y)
			return nil
		}
		if e.Button == tui.MouseButtonLeft {
			return a.click(kind, id)
		}
	}
	return nil
}

// click acts on a left click on the area kind:id.
func (a *app) click(kind, id string) []tui.Cmd {
	switch kind {
	case "space":
		a.switchSpace(a.spaceIndex(id))
	case "item":
		a.focusPane(id)
	case "btn":
		sp := a.space()
		switch id {
		case "split-right":
			return a.splitPane(sp.focus, tui.SplitColumns)
		case "split-down":
			return a.splitPane(sp.focus, tui.SplitRows)
		case "column":
			return a.newColumn(sp)
		case "new-space":
			return a.newSpace(a.startDir(sp, a.focused()))
		case "help":
			a.openHelp()
		case "unzoom":
			sp.zoom = ""
		}
	}
	return nil
}

// scrollPane gives the wheel to the pane's program if it reads the mouse,
// and otherwise scrolls the pane's history.
func (a *app) scrollPane(id string, e tui.MouseEvent) {
	p := a.panes[id]
	if p == nil {
		return
	}
	if x, y, ok := a.paneLocal(id, e.X, e.Y, false); ok && p.term.SendMouse(e, x, y) {
		return
	}
	switch e.Button {
	case tui.MouseButtonWheelUp:
		p.term.Scroll(-3)
	case tui.MouseButtonWheelDown:
		p.term.Scroll(3)
	}
}

// paneLocal converts a screen position to a cell inside a pane's border.
// clamp pins positions outside the pane to its edge, for drags that leave it.
func (a *app) paneLocal(id string, x, y int, clamp bool) (int, int, bool) {
	b, ok := a.hits.Bounds("pane:" + id)
	if !ok {
		return 0, 0, false
	}
	lx, ly := x-b.Min.X-1, y-b.Min.Y-1
	w, h := b.Dx()-2, b.Dy()-2
	if clamp {
		lx, ly = max(0, min(lx, w-1)), max(0, min(ly, h-1))
	}
	if lx < 0 || ly < 0 || lx >= w || ly >= h {
		return 0, 0, false
	}
	return lx, ly, true
}

func (a *app) spaceIndex(id string) int {
	for i, sp := range a.spaces {
		if sp.id == id {
			return i
		}
	}
	return -1
}

func (a *app) screenWidth() int {
	if b, ok := a.hits.Bounds("root"); ok {
		return b.Dx()
	}
	return 160
}
