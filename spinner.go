package gooey

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
)

// Spinner represents an animated loading spinner
type Spinner struct {
	frames      []string
	interval    time.Duration
	style       Style
	message     string
	mu          sync.Mutex
	active      bool
	done        chan struct{}
	terminal    *Terminal
	startX      int // Absolute cursor position in buffer
	startY      int // Absolute cursor position in buffer
	initialized bool
}

// SpinnerStyle defines different spinner animations
type SpinnerStyle struct {
	Frames   []string
	Interval time.Duration
}

// Predefined spinner styles
var (
	SpinnerDots = SpinnerStyle{
		Frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		Interval: 80 * time.Millisecond,
	}

	SpinnerLine = SpinnerStyle{
		Frames:   []string{"-", "\\", "|", "/"},
		Interval: 100 * time.Millisecond,
	}

	SpinnerArrows = SpinnerStyle{
		Frames:   []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
		Interval: 100 * time.Millisecond,
	}

	SpinnerCircle = SpinnerStyle{
		Frames:   []string{"◐", "◓", "◑", "◒"},
		Interval: 120 * time.Millisecond,
	}

	SpinnerSquare = SpinnerStyle{
		Frames:   []string{"◰", "◳", "◲", "◱"},
		Interval: 100 * time.Millisecond,
	}

	SpinnerBounce = SpinnerStyle{
		Frames:   []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
		Interval: 80 * time.Millisecond,
	}

	SpinnerBar = SpinnerStyle{
		Frames: []string{
			"[    ]",
			"[=   ]",
			"[==  ]",
			"[=== ]",
			"[====]",
			"[ ===]",
			"[  ==]",
			"[   =]",
		},
		Interval: 80 * time.Millisecond,
	}

	SpinnerStars = SpinnerStyle{
		Frames: []string{
			"✶",
			"✸",
			"✹",
			"✺",
			"✹",
			"✷",
		},
		Interval: 120 * time.Millisecond,
	}

	SpinnerStarField = SpinnerStyle{
		Frames: []string{
			"·   ",
			"*·  ",
			"✦*· ",
			"✧✦*·",
			"·✧✦*",
			" ·✧✦",
			"  ·✧",
			"   ·",
		},
		Interval: 100 * time.Millisecond,
	}

	SpinnerAsterisk = SpinnerStyle{
		Frames: []string{
			"*",
			"⁎",
			"✱",
			"✲",
			"✳",
			"✴",
			"✵",
			"✶",
			"✷",
			"✸",
			"✹",
			"✺",
		},
		Interval: 100 * time.Millisecond,
	}

	SpinnerSparkle = SpinnerStyle{
		Frames: []string{
			"✨",
			"💫",
			"⭐",
			"🌟",
			"✨",
			"💫",
		},
		Interval: 150 * time.Millisecond,
	}
)

// NewSpinner creates a new spinner
func NewSpinner(terminal *Terminal, style SpinnerStyle) *Spinner {
	return &Spinner{
		frames:   style.Frames,
		interval: style.Interval,
		style:    NewStyle(),
		terminal: terminal,
		done:     make(chan struct{}),
	}
}

// WithStyle sets the spinner style
func (s *Spinner) WithStyle(style Style) *Spinner {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.style = style
	return s
}

// WithMessage sets the spinner message
func (s *Spinner) WithMessage(message string) *Spinner {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
	return s
}

// ensurePosition makes sure we have a valid start position
func (s *Spinner) ensurePosition() {
	if !s.initialized {
		x, y := s.terminal.CursorPosition()
		s.startX = x
		s.startY = y
		s.initialized = true
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	// Allow reuse after Stop
	s.done = make(chan struct{})
	s.ensurePosition()
	s.active = true
	s.mu.Unlock()

	go s.animate()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	close(s.done)
	s.mu.Unlock()

	// Clear the spinner line using atomic frame
	if frame, err := s.terminal.BeginFrame(); err == nil {
		// Clear line by printing spaces or using FillStyled?
		// RenderFrame doesn't have ClearToEndOfLine.
		// We'll simulate it or just print spaces if we knew the width.
		// For now, let's assume a reasonable width or use terminal.Width
		w, _ := frame.Size()
		frame.FillStyled(s.startX, s.startY, w-s.startX, 1, ' ', NewStyle())
		s.terminal.EndFrame(frame)
	}
}

// Success stops the spinner with a success message
func (s *Spinner) Success(message string) {
	s.Stop()

	if frame, err := s.terminal.BeginFrame(); err == nil {
		w, _ := frame.Size()
		// Clear first
		frame.FillStyled(s.startX, s.startY, w-s.startX, 1, ' ', NewStyle())

		successStyle := NewStyle().WithForeground(ColorGreen)
		frame.PrintStyled(s.startX, s.startY, "✓ "+message, successStyle)
		s.terminal.EndFrame(frame)
	}
	// We don't automatically advance line here to keep it clean,
	// or we can let the user handle layout.
}

// Error stops the spinner with an error message
func (s *Spinner) Error(message string) {
	s.Stop()

	if frame, err := s.terminal.BeginFrame(); err == nil {
		w, _ := frame.Size()
		frame.FillStyled(s.startX, s.startY, w-s.startX, 1, ' ', NewStyle())

		errorStyle := NewStyle().WithForeground(ColorRed)
		frame.PrintStyled(s.startX, s.startY, "✗ "+message, errorStyle)
		s.terminal.EndFrame(frame)
	}
}

func (s *Spinner) animate() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	frameIdx := 0
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			message := s.message
			style := s.style
			startX, startY := s.startX, s.startY
			s.mu.Unlock()

			if frame, err := s.terminal.BeginFrame(); err == nil {
				w, _ := frame.Size()
				// Clear line
				frame.FillStyled(startX, startY, w-startX, 1, ' ', NewStyle())

				// Draw frame
				frame.PrintStyled(startX, startY, s.frames[frameIdx], style)

				// Draw message
				if message != "" {
					// Length of frame + space
					offset := len(s.frames[frameIdx]) + 1
					frame.PrintStyled(startX+offset, startY, message, NewStyle())
				}

				s.terminal.EndFrame(frame)
			}

			frameIdx = (frameIdx + 1) % len(s.frames)
		}
	}
}

// ProgressBar represents a progress bar
type ProgressBar struct {
	total       int
	current     int
	width       int
	style       Style
	fillChar    string
	emptyChar   string
	showPercent bool
	showNumbers bool
	message     string
	terminal    *Terminal
	mu          sync.Mutex
	startX      int // Absolute cursor position in buffer
	startY      int // Absolute cursor position in buffer
	initialized bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(terminal *Terminal, total int) *ProgressBar {
	return &ProgressBar{
		total:       total,
		width:       40,
		style:       NewStyle().WithForeground(ColorCyan),
		fillChar:    "█",
		emptyChar:   "░",
		showPercent: true,
		showNumbers: false,
		terminal:    terminal,
	}
}

// WithWidth sets the progress bar width
func (p *ProgressBar) WithWidth(width int) *ProgressBar {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.width = width
	return p
}

// WithStyle sets the progress bar style
func (p *ProgressBar) WithStyle(style Style) *ProgressBar {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.style = style
	return p
}

// WithChars sets the fill and empty characters
func (p *ProgressBar) WithChars(fill, empty string) *ProgressBar {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fillChar = fill
	p.emptyChar = empty
	return p
}

// ShowNumbers enables showing current/total numbers
func (p *ProgressBar) ShowNumbers() *ProgressBar {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.showNumbers = true
	return p
}

// Update updates the progress bar
func (p *ProgressBar) Update(current int, message string) {
	p.mu.Lock()
	p.current = current
	if current > p.total {
		p.current = p.total
	}
	p.message = message

	// Ensure position is set if this is the first update
	if !p.initialized {
		// We can't easily get virtualX/Y without locking terminal,
		// but BeginFrame will give us a frame.
		// However, for standalone usage, we might rely on external positioning?
		// Or just grab it now (but thread unsafe without lock).
		// Safe way: NewProgressBar should probably not assume position until drawn/started.
		// For now, let's assume if not initialized, we default to 0,0 or caller handles it.
		// But the old code used terminal.virtualX which is internal.
		// We'll rely on MultiProgress setting it, or if standalone, it might break.
		// Standalone usage should probably use terminal.Print* directly or we need a 'Start' for ProgressBar too.
		p.initialized = true
	}
	p.mu.Unlock()

	p.draw()
}

// Increment increments the progress by 1
func (p *ProgressBar) Increment(message string) {
	p.mu.Lock()
	p.current++
	if p.current > p.total {
		p.current = p.total
	}
	p.message = message

	if !p.initialized {
		p.initialized = true
	}
	p.mu.Unlock()

	p.draw()
}

// Complete marks the progress as complete
func (p *ProgressBar) Complete(message string) {
	p.Update(p.total, message)
	// We don't force newline here anymore to stay atomic
}

func (p *ProgressBar) draw() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if frame, err := p.terminal.BeginFrame(); err == nil {
		w, _ := frame.Size()
		// Clear line
		frame.FillStyled(p.startX, p.startY, w-p.startX, 1, ' ', NewStyle())

		// Calculate fill
		percent := 0.0
		if p.total > 0 {
			percent = float64(p.current) / float64(p.total)
		}
		filled := int(percent * float64(p.width))

		// Build bar
		currentX := p.startX

		frame.PrintStyled(currentX, p.startY, "[", NewStyle())
		currentX++

		// Fill part
		fillStr := strings.Repeat(p.fillChar, filled)
		frame.PrintStyled(currentX, p.startY, fillStr, p.style)
		// runewidth for multi-byte chars? Let's assume width 1 or rely on terminal logic
		// Actually frame.PrintStyled handles width. But we need to advance X correctly if we do piecemeal.
		// Simpler to build string? No, style changes.
		// We need to know width of fillStr.
		currentX += runewidth.StringWidth(fillStr)

		// Empty part
		emptyWidth := p.width - filled
		if emptyWidth > 0 {
			emptyStr := strings.Repeat(p.emptyChar, emptyWidth)
			frame.PrintStyled(currentX, p.startY, emptyStr, NewStyle())
			currentX += runewidth.StringWidth(emptyStr)
		}

		frame.PrintStyled(currentX, p.startY, "]", NewStyle())
		currentX++

		// Add percentage
		if p.showPercent {
			pctStr := fmt.Sprintf(" %3.0f%%", percent*100)
			frame.PrintStyled(currentX, p.startY, pctStr, NewStyle())
			currentX += len(pctStr)
		}

		// Add numbers
		if p.showNumbers {
			numStr := fmt.Sprintf(" (%d/%d)", p.current, p.total)
			frame.PrintStyled(currentX, p.startY, numStr, NewStyle())
			currentX += len(numStr)
		}

		// Add message
		if p.message != "" {
			frame.PrintStyled(currentX, p.startY, " "+p.message, NewStyle())
		}

		p.terminal.EndFrame(frame)
	}
}

// MultiProgress manages multiple progress items
type MultiProgress struct {
	items    []*ProgressItem
	terminal *Terminal
	mu       sync.Mutex
}

// ProgressItem represents a single progress item in MultiProgress
type ProgressItem struct {
	ID          string
	Message     string
	Current     int
	Total       int
	Style       Style
	SpinnerOnly bool
	Spinner     *Spinner
	Bar         *ProgressBar
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress(terminal *Terminal) *MultiProgress {
	return &MultiProgress{
		terminal: terminal,
		items:    make([]*ProgressItem, 0),
	}
}

// Add adds a new progress item
func (m *MultiProgress) Add(id string, total int, spinnerOnly bool) *ProgressItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	item := &ProgressItem{
		ID:          id,
		Total:       total,
		Style:       NewStyle(),
		SpinnerOnly: spinnerOnly,
	}

	if spinnerOnly {
		item.Spinner = NewSpinner(m.terminal, SpinnerDots)
	} else {
		item.Bar = NewProgressBar(m.terminal, total)
	}

	m.items = append(m.items, item)
	return item
}

// Update updates a progress item
func (m *MultiProgress) Update(id string, current int, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, item := range m.items {
		if item.ID == id {
			item.Current = current
			item.Message = message
			m.drawItem(i, item)
			return
		}
	}
}

// Start begins rendering all progress items
func (m *MultiProgress) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reserve space and track positions for each item
	for _, item := range m.items {
		// Check if adding a new line will cause scrolling (at bottom of screen)
		_, y := m.terminal.CursorPosition()
		_, h := m.terminal.Size()

		// If we are at the last line, printing a newline will scroll
		if y >= h-1 {
			// Shift all previous items up by 1
			for _, prevItem := range m.items {
				if prevItem.Bar != nil && prevItem.Bar.initialized {
					prevItem.Bar.startY--
				}
				if prevItem.Spinner != nil && prevItem.Spinner.initialized {
					prevItem.Spinner.startY--
				}
			}
		}

		// Capture current position for this item
		// Note: We call CursorPosition again because logic above might have changed understanding,
		// but actually Println will trigger the scroll.
		// The item will be placed at 'currentY'.
		currentX, currentY := m.terminal.CursorPosition()

		if item.Bar != nil {
			item.Bar.startY = currentY
			item.Bar.startX = currentX
			item.Bar.initialized = true
		}
		if item.Spinner != nil {
			item.Spinner.startY = currentY
			item.Spinner.startX = currentX
			item.Spinner.initialized = true
		}

		// Reserve the line (moves cursor down)
		m.terminal.Println("")
	}

	// Ensure changes are flushed to screen
	m.terminal.Flush()
}

// Stop stops all progress items
func (m *MultiProgress) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop all spinners
	for _, item := range m.items {
		if item.Spinner != nil {
			item.Spinner.Stop()
		}
	}
}

func (m *MultiProgress) drawItem(index int, item *ProgressItem) {
	// The progress bar/spinner will draw at its own startY position
	// We just need to trigger the update
	if item.SpinnerOnly {
		// Draw spinner - it will use its own startY position
		if item.Spinner != nil {
			item.Spinner.WithMessage(item.Message)
			// Note: Spinner animates in its own goroutine, but WithMessage updates it safely
			// If we want to force a redraw here (e.g. if message changed), the spinner loop handles it
		}
	} else {
		// Draw progress bar - it will use its own startY position
		if item.Bar != nil {
			// We call Update which triggers draw()
			// But we need to be careful not to double-lock or call Update recursively if we are called from Update
			// m.Update calls this.
			// item.Bar.Update locks bar.mu. That's fine.
			item.Bar.Update(item.Current, item.Message)
		}
	}
}
