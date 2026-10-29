package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/deepnoodle-ai/wonton/tmux"
	"github.com/deepnoodle-ai/wonton/tui"
)

// space is a named group of panes with its own layout, typically one per
// project. The sidebar lists them; one is shown at a time.
type space struct {
	id    string
	name  string
	dir   string         // where its new panes start
	split tui.SplitState // Tight: the panes' own borders separate them
	focus string         // the pane that has focus while the space is shown
	zoom  string         // the pane filling the space, or ""
}

// pane is one terminal: a tmux session mirrored into the layout. Its ID is
// also its session's name.
type pane struct {
	id    string
	name  string // set by the user; empty means the running command's name
	space *space
	term  *tmux.Terminal
}

// label is what the pane is called in the sidebar and its border.
func (p *pane) label() string {
	if p.name != "" {
		return p.name
	}
	if cmd := p.term.Screen().Command; cmd != "" {
		return cmd
	}
	return p.id
}

// busy reports whether the pane's program wrote something recently.
func (p *pane) busy() bool {
	return time.Since(p.term.LastOutput()) < 1500*time.Millisecond
}

// optionName is the tmux user option the layout is kept in. It lives in the
// server alongside the sessions, so the two cannot drift apart: when the
// server goes away, so does the layout.
const optionName = "@corral-layout"

type savedState struct {
	Seq     int          `json:"seq"`
	Current int          `json:"current"`
	Spaces  []savedSpace `json:"spaces"`
}

type savedSpace struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Dir    string            `json:"dir"`
	Focus  string            `json:"focus,omitempty"`
	Zoom   string            `json:"zoom,omitempty"`
	Layout *tui.SplitNode    `json:"layout,omitempty"`
	Names  map[string]string `json:"names,omitempty"`
}

func (a *app) snapshot() savedState {
	st := savedState{Seq: a.seq, Current: a.current}
	for _, sp := range a.spaces {
		ss := savedSpace{
			ID:     sp.id,
			Name:   sp.name,
			Dir:    sp.dir,
			Focus:  sp.focus,
			Zoom:   sp.zoom,
			Layout: sp.split.Root,
			Names:  map[string]string{},
		}
		for _, id := range sp.split.Leaves() {
			if p := a.panes[id]; p != nil && p.name != "" {
				ss.Names[id] = p.name
			}
		}
		st.Spaces = append(st.Spaces, ss)
	}
	return st
}

// save writes the layout to the tmux server in the background.
func (a *app) save() {
	if data, err := json.Marshal(a.snapshot()); err == nil {
		a.saver.save(string(data))
	}
}

// saveNow writes the layout and waits until it is written.
func (a *app) saveNow() {
	a.save()
	a.saver.wait()
}

// saver writes layouts one at a time, in the order they were made. A layout
// saved while another is being written replaces any still waiting, so the
// last one saved is always the one stored.
type saver struct {
	write func(data string)

	mu      sync.Mutex
	idle    *sync.Cond
	next    string
	pending bool // next is waiting to be written
	running bool // the writer goroutine is going
}

func newSaver(write func(data string)) *saver {
	s := &saver{write: write}
	s.idle = sync.NewCond(&s.mu)
	return s
}

func (s *saver) save(data string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next, s.pending = data, true
	if !s.running {
		s.running = true
		go s.run()
	}
}

func (s *saver) run() {
	s.mu.Lock()
	for s.pending {
		data := s.next
		s.pending = false
		s.mu.Unlock()
		s.write(data)
		s.mu.Lock()
	}
	s.running = false
	s.idle.Broadcast()
	s.mu.Unlock()
}

// wait returns once every layout saved so far is written.
func (s *saver) wait() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for s.running {
		s.idle.Wait()
	}
}

// restore rebuilds the spaces from the layout saved in the server, keeping
// only the panes whose sessions are still running. Sessions the layout does
// not know about are added to the current space, so nothing running is ever
// hidden.
func (a *app) restore() error {
	ctx := context.Background()
	sessions, err := a.server.Sessions(ctx)
	if err != nil {
		return err
	}
	alive := map[string]bool{}
	for _, s := range sessions {
		alive[s.Name] = true
	}

	var st savedState
	if raw, _ := a.server.GetOption(ctx, optionName); raw != "" {
		_ = json.Unmarshal([]byte(raw), &st)
	}
	a.seq = st.Seq

	used := map[string]bool{}
	for _, ss := range st.Spaces {
		sp := &space{id: ss.ID, name: ss.Name, dir: ss.Dir, focus: ss.Focus, zoom: ss.Zoom}
		sp.split = tui.SplitState{Root: ss.Layout, Tight: true}
		for _, id := range sp.split.Leaves() {
			if !alive[id] || used[id] {
				sp.split.Remove(id)
				continue
			}
			// No size: the session keeps the one it has until the first
			// frame lays it out, rather than redrawing at a guess.
			if _, err := a.attach(sp, id, ss.Names[id], "", 0, 0); err != nil {
				sp.split.Remove(id)
				continue
			}
			used[id] = true
		}
		a.spaces = append(a.spaces, sp)
	}
	a.current = st.Current

	if len(a.spaces) == 0 {
		a.spaces = append(a.spaces, a.newSpaceValue(""))
		a.current = 0
	}
	a.current = max(0, min(a.current, len(a.spaces)-1))
	sp := a.space()
	for _, s := range sessions {
		if used[s.Name] {
			continue
		}
		if _, err := a.attach(sp, s.Name, "", "", 0, 0); err == nil {
			sp.split.Append(s.Name, tui.SplitColumns)
			a.bumpSeq(s.Name)
		}
	}
	if len(a.panes) == 0 {
		p, err := a.openPane(sp, sp.dir, 80, 24)
		if err != nil {
			return err
		}
		sp.split.Append(p.id, tui.SplitColumns)
	}
	for _, sp := range a.spaces {
		a.fixFocus(sp)
	}
	return nil
}

// bumpSeq keeps generated IDs clear of a session named like one.
func (a *app) bumpSeq(id string) {
	var n int
	if _, err := fmt.Sscanf(id, "p%d", &n); err == nil && n > a.seq {
		a.seq = n
	}
}

func (a *app) newSpaceValue(dir string) *space {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	a.seq++
	return &space{
		id:    fmt.Sprintf("s%d", a.seq),
		name:  filepath.Base(dir),
		dir:   dir,
		split: tui.SplitState{Tight: true},
	}
}

// attach mirrors an existing or new session into a pane of sp. It does not
// place the pane in the layout. A zero size leaves an existing session's
// size as it is.
func (a *app) attach(sp *space, id, name, dir string, width, height int) (*pane, error) {
	term, err := a.server.Open(context.Background(), tmux.TerminalOptions{
		Session:  id,
		Dir:      dir,
		Width:    size(width, 10),
		Height:   size(height, 3),
		OnUpdate: a.wake,
	})
	if err != nil {
		return nil, err
	}
	p := &pane{id: id, name: name, space: sp, term: term}
	a.panes[id] = p
	return p, nil
}

// size returns at least least, or zero to leave a size alone.
func size(n, least int) int {
	if n <= 0 {
		return 0
	}
	return max(n, least)
}

// openPane starts a new session for sp. The caller places it in the layout.
func (a *app) openPane(sp *space, dir string, width, height int) (*pane, error) {
	for {
		a.seq++
		id := fmt.Sprintf("p%d", a.seq)
		if _, taken := a.panes[id]; !taken {
			return a.attach(sp, id, "", dir, width, height)
		}
	}
}

// panesIn returns a space's panes in layout order.
func (a *app) panesIn(sp *space) []*pane {
	var out []*pane
	for _, id := range sp.split.Leaves() {
		if p := a.panes[id]; p != nil {
			out = append(out, p)
		}
	}
	return out
}

// fixFocus points a space's focus and zoom at panes it still has.
func (a *app) fixFocus(sp *space) {
	leaves := sp.split.Leaves()
	if !slices.Contains(leaves, sp.zoom) {
		sp.zoom = ""
	}
	if !slices.Contains(leaves, sp.focus) {
		sp.focus = ""
		if len(leaves) > 0 {
			sp.focus = leaves[0]
		}
	}
}
