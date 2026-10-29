package tui

import "image"

// HitMap records where tagged views were drawn, so an event handler can ask
// which of them is under the pointer. It answers the question a right-click
// menu needs answered — what was clicked? — for views that have no click
// handling of their own: rows of a sidebar, a tab bar, a status line.
//
// The application owns the HitMap, resets it at the start of View, and wraps
// the views it wants to find with Area:
//
//	func (a *app) View() tui.View {
//	    a.hits.Reset()
//	    var rows []tui.View
//	    for _, s := range a.sessions {
//	        rows = append(rows, a.hits.Area("session:"+s.ID, tui.Text("%s", s.Name)))
//	    }
//	    return tui.Stack(rows...)
//	}
//
//	func (a *app) HandleEvent(ev tui.Event) []tui.Cmd {
//	    if e, ok := ev.(tui.MouseEvent); ok && e.Type == tui.MouseClick {
//	        if key, ok := a.hits.At(e.X, e.Y); ok {
//	            a.clicked(key, e)
//	        }
//	    }
//	    return nil
//	}
//
// Areas may nest. At returns the area drawn last, which for nested areas is
// the innermost and for overlapping ones is the one on top.
type HitMap struct {
	areas []hitArea
}

type hitArea struct {
	key    string
	bounds image.Rectangle
}

// Reset forgets every area. Call it at the start of View so areas that are
// no longer drawn stop answering.
func (h *HitMap) Reset() { h.areas = h.areas[:0] }

// Area returns child unchanged in layout and appearance, recording under key
// where it is drawn.
func (h *HitMap) Area(key string, child View) View {
	return &hitAreaView{hits: h, key: key, child: child}
}

// At returns the key of the area drawn last that contains the screen
// position (x, y).
func (h *HitMap) At(x, y int) (string, bool) {
	pt := image.Pt(x, y)
	for i := len(h.areas) - 1; i >= 0; i-- {
		if pt.In(h.areas[i].bounds) {
			return h.areas[i].key, true
		}
	}
	return "", false
}

// Bounds returns where the area with this key was drawn, in screen
// coordinates. If several areas share the key, it returns the last.
func (h *HitMap) Bounds(key string) (image.Rectangle, bool) {
	for i := len(h.areas) - 1; i >= 0; i-- {
		if h.areas[i].key == key {
			return h.areas[i].bounds, true
		}
	}
	return image.Rectangle{}, false
}

type hitAreaView struct {
	hits  *HitMap
	key   string
	child View
}

func (v *hitAreaView) size(maxWidth, maxHeight int) (int, int) {
	return v.child.size(maxWidth, maxHeight)
}

func (v *hitAreaView) flex() int {
	if f, ok := v.child.(Flexible); ok {
		return f.flex()
	}
	return 0
}

func (v *hitAreaView) render(ctx *RenderContext) {
	if b := ctx.clickArea(ctx.AbsoluteBounds()); !b.Empty() {
		v.hits.areas = append(v.hits.areas, hitArea{key: v.key, bounds: b})
	}
	v.child.render(ctx)
}
