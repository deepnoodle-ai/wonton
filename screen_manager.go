package gooey

import (
	"sync"
	"time"
)

// ScreenRegion represents a rectangular area of the screen
type ScreenRegion struct {
	mu            sync.Mutex
	X, Y          int
	Width, Height int
	Content       []string
	Animations    []TextAnimation
	Protected     bool                                             // If true, this region cannot be overwritten by animations
	DrawCallback  func(frame RenderFrame, x, y, width, height int) // Optional custom draw function
}

// ScreenManager coordinates all screen updates to prevent race conditions
type ScreenManager struct {
	terminal         *Terminal
	regions          map[string]*ScreenRegion
	regionOrder      []string // Deterministic drawing order
	mu               sync.RWMutex
	drawMutex        sync.Mutex // Ensures only one draw operation at a time
	cursorX, cursorY int        // Where the cursor should be positioned
	cursorVisible    bool
	running          bool
	updateChan       chan struct{}
	stopChan         chan struct{}
	frameCounter     uint64
	fps              int
	unregisterResize func() // Cleanup function for resize callback
}

// NewScreenManager creates a new screen manager
func NewScreenManager(terminal *Terminal, fps int) *ScreenManager {
	if fps <= 0 {
		fps = 30
	}
	sm := &ScreenManager{
		terminal:      terminal,
		regions:       make(map[string]*ScreenRegion),
		regionOrder:   make([]string, 0),
		updateChan:    make(chan struct{}, 100), // Buffered to avoid blocking
		stopChan:      make(chan struct{}),
		cursorVisible: true,
		fps:           fps,
	}

	// Register for resize events
	sm.unregisterResize = terminal.OnResize(func(width, height int) {
		sm.handleResize(width, height)
	})

	return sm
}

// handleResize adjusts regions when the terminal is resized
func (sm *ScreenManager) handleResize(width, height int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Update regions that depend on terminal dimensions
	for _, region := range sm.regions {
		region.mu.Lock()
		// If region extends to the edge, adjust its size
		if region.X+region.Width > width {
			region.Width = width - region.X
			if region.Width < 0 {
				region.Width = 0
			}
		}
		if region.Y+region.Height > height {
			region.Height = height - region.Y
			if region.Height < 0 {
				region.Height = 0
			}
		}
		region.mu.Unlock()
	}

	// Trigger a redraw
	select {
	case sm.updateChan <- struct{}{}:
	default:
	}
}

// DefineRegion creates or updates a screen region
func (sm *ScreenManager) DefineRegion(name string, x, y, width, height int, protected bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.regions[name]; !exists {
		sm.regionOrder = append(sm.regionOrder, name)
	}

	sm.regions[name] = &ScreenRegion{
		X:         x,
		Y:         y,
		Width:     width,
		Height:    height,
		Protected: protected,
		Content:   make([]string, height),
	}
}

// SetRegionDrawCallback sets a custom draw function for a region.
// This allows rendering custom widgets or content that isn't just text/animations.
func (sm *ScreenManager) SetRegionDrawCallback(name string, callback func(frame RenderFrame, x, y, width, height int)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if region, exists := sm.regions[name]; exists {
		region.mu.Lock()
		region.DrawCallback = callback
		region.mu.Unlock()
	}
}

// UpdateRegion updates the content of a region
func (sm *ScreenManager) UpdateRegion(name string, lineIndex int, content string, animation TextAnimation) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if region, exists := sm.regions[name]; exists {
		region.mu.Lock()
		defer region.mu.Unlock()

		if lineIndex >= 0 && lineIndex < len(region.Content) {
			region.Content[lineIndex] = content

			// Ensure animations slice is large enough
			for len(region.Animations) <= lineIndex {
				region.Animations = append(region.Animations, nil)
			}
			region.Animations[lineIndex] = animation
		}
	}

	// Trigger a redraw
	select {
	case sm.updateChan <- struct{}{}:
	default:
		// Channel is full, skip this update signal
	}
}

// SetCursorPosition sets where the cursor should be positioned
func (sm *ScreenManager) SetCursorPosition(x, y int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.cursorX = x
	sm.cursorY = y
}

// Start begins the managed drawing loop
func (sm *ScreenManager) Start() {
	sm.mu.Lock()
	if sm.running {
		sm.mu.Unlock()
		return
	}
	sm.running = true
	sm.stopChan = make(chan struct{})
	sm.mu.Unlock()

	go sm.drawLoop()
}

// Stop stops the drawing loop
func (sm *ScreenManager) Stop() {
	sm.mu.Lock()
	if !sm.running {
		sm.mu.Unlock()
		return
	}
	sm.running = false
	close(sm.stopChan)
	sm.mu.Unlock()
}

// Close stops the drawing loop and releases all resources
func (sm *ScreenManager) Close() error {
	sm.Stop()

	// Unregister resize callback
	if sm.unregisterResize != nil {
		sm.unregisterResize()
		sm.unregisterResize = nil
	}

	// Clear regions to release references
	sm.mu.Lock()
	sm.regions = make(map[string]*ScreenRegion)
	sm.regionOrder = make([]string, 0)
	sm.mu.Unlock()

	return nil
}

// drawLoop is the single drawing loop that coordinates all screen updates
func (sm *ScreenManager) drawLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(sm.fps))
	defer ticker.Stop()

	lastDraw := time.Now()
	minDrawInterval := time.Millisecond * 50 // Don't redraw more than 20 times per second

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.draw()
			lastDraw = time.Now()
		case <-sm.updateChan:
			// Batch rapid updates - only draw if enough time has passed
			if time.Since(lastDraw) > minDrawInterval {
				sm.draw()
				lastDraw = time.Now()
			}
			// Drain any additional updates in the channel
			for len(sm.updateChan) > 0 {
				select {
				case <-sm.updateChan:
				default:
				}
			}
		}
	}
}

// draw performs a single coordinated draw of all regions
func (sm *ScreenManager) draw() {
	// Lock drawing to prevent concurrent draws
	sm.drawMutex.Lock()
	defer sm.drawMutex.Unlock()

	// Get current frame counter
	sm.frameCounter++
	frameNum := sm.frameCounter

	// Draw all regions in order
	sm.mu.RLock()
	// Copy regions and order to avoid holding lock during draw
	regions := make(map[string]*ScreenRegion)
	for k, v := range sm.regions {
		regions[k] = v
	}
	order := make([]string, len(sm.regionOrder))
	copy(order, sm.regionOrder)
	cursorX, cursorY := sm.cursorX, sm.cursorY
	sm.mu.RUnlock()

	// Begin atomic frame
	frame, err := sm.terminal.BeginFrame()
	if err != nil {
		return
	}

	// Draw each region
	for _, name := range order {
		if region, ok := regions[name]; ok {
			sm.drawRegion(frame, region, frameNum)
		}
	}

	sm.terminal.EndFrame(frame)

	// Restore cursor position
	sm.terminal.MoveCursor(cursorX, cursorY)
}

// drawRegion draws a single region efficiently
func (sm *ScreenManager) drawRegion(frame RenderFrame, region *ScreenRegion, frameNum uint64) {
	region.mu.Lock()
	defer region.mu.Unlock()

	// If a custom draw callback is defined, use it exclusively
	if region.DrawCallback != nil {
		region.DrawCallback(frame, region.X, region.Y, region.Width, region.Height)
		return
	}

	for i, content := range region.Content {
		y := region.Y + i

		// Always clear the region line to handle content that may have changed or been removed
		frame.FillStyled(region.X, y, region.Width, 1, ' ', NewStyle())

		if content == "" {
			continue
		}

		// Draw the content
		if i < len(region.Animations) && region.Animations[i] != nil {
			// Draw with animation
			sm.drawAnimatedText(frame, region.X, y, content, region.Animations[i], frameNum)
		} else {
			// Draw static text
			frame.PrintStyled(region.X, y, content, NewStyle())
		}
	}
}

// drawAnimatedText draws text with animation using the consolidated GetStyle API
func (sm *ScreenManager) drawAnimatedText(frame RenderFrame, x, y int, text string, animation TextAnimation, frameNum uint64) {
	runes := []rune(text)
	totalChars := len(runes)

	for i, r := range runes {
		currentX := x + i
		// Use the animation's GetStyle method - consolidated rendering
		style := animation.GetStyle(frameNum, i, totalChars)
		frame.SetCell(currentX, y, r, style)
	}
}
