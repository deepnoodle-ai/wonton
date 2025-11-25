package gooey

import (
	"time"

	"github.com/mattn/go-runewidth"
)

// Layout manages the overall terminal layout with header, footer, and content area
type Layout struct {
	terminal         *Terminal
	header           *Header
	footer           *Footer
	contentY         int
	contentHeight    int
	autoRefresh      bool
	refreshTicker    *time.Ticker
	stopRefresh      chan bool
	unregisterResize func() // Cleanup function for resize callback
}

// Header represents the top section of the terminal
type Header struct {
	Left        string
	Center      string
	Right       string
	Style       Style
	Background  Style
	Border      bool
	BorderStyle BorderStyle
	Height      int
}

// Footer represents the bottom section of the terminal
type Footer struct {
	Left        string
	Center      string
	Right       string
	Style       Style
	Background  Style
	Border      bool
	BorderStyle BorderStyle
	Height      int
	StatusBar   bool
	StatusItems []StatusItem
}

// StatusItem represents an item in the status bar
type StatusItem struct {
	Key   string
	Value string
	Style Style
	Icon  string
}

// NewLayout creates a new layout manager
func NewLayout(terminal *Terminal) *Layout {
	_, height := terminal.Size()
	l := &Layout{
		terminal:      terminal,
		contentY:      0,
		contentHeight: height,
	}

	// Register for resize events
	l.unregisterResize = terminal.OnResize(func(width, height int) {
		l.handleResize(width, height)
	})

	return l
}

// handleResize updates the layout when the terminal is resized
func (l *Layout) handleResize(width, height int) {
	l.updateContentArea()
}

// SetHeader configures the header
func (l *Layout) SetHeader(header *Header) *Layout {
	l.header = header
	if header != nil {
		// Automatically adjust height based on border setting
		// If border is enabled, we need 3 rows (top border, content, bottom border)
		// If no border, we need 1 row for content
		if header.Border && header.Height < 3 {
			header.Height = 3
		} else if !header.Border && header.Height == 0 {
			header.Height = 1
		}
	}
	l.updateContentArea()
	return l
}

// SetFooter configures the footer
func (l *Layout) SetFooter(footer *Footer) *Layout {
	l.footer = footer
	if footer != nil {
		// Automatically adjust height based on border setting
		// If border is enabled, we need 3 rows (top border, content, bottom border)
		// If no border, we need 1 row for content
		if footer.Border && footer.Height < 3 {
			footer.Height = 3
		} else if !footer.Border && footer.Height == 0 {
			footer.Height = 1
		}
	}
	l.updateContentArea()
	return l
}

// EnableAutoRefresh enables automatic refresh of header/footer
func (l *Layout) EnableAutoRefresh(interval time.Duration) *Layout {
	if l.autoRefresh {
		return l
	}

	l.autoRefresh = true
	l.refreshTicker = time.NewTicker(interval)
	l.stopRefresh = make(chan bool)

	go func() {
		for {
			select {
			case <-l.refreshTicker.C:
				l.Refresh()
			case <-l.stopRefresh:
				return
			}
		}
	}()

	return l
}

// DisableAutoRefresh stops automatic refresh
func (l *Layout) DisableAutoRefresh() {
	if l.autoRefresh && l.refreshTicker != nil {
		l.refreshTicker.Stop()
		close(l.stopRefresh)
		l.autoRefresh = false
	}
}

// Close stops any background refresh goroutines and releases resources
func (l *Layout) Close() error {
	l.DisableAutoRefresh()

	// Unregister resize callback
	if l.unregisterResize != nil {
		l.unregisterResize()
		l.unregisterResize = nil
	}

	return nil
}

// Draw renders the entire layout
func (l *Layout) Draw() {
	frame, err := l.terminal.BeginFrame()
	if err != nil {
		return
	}

	l.DrawTo(frame)

	l.terminal.EndFrame(frame)

	// Position cursor in content area (best effort, as EndFrame flushes)
	l.terminal.MoveCursor(0, l.contentY)
}

// DrawTo renders the layout to the provided frame
func (l *Layout) DrawTo(frame RenderFrame) {
	// Clear screen
	w, h := frame.Size()
	frame.FillStyled(0, 0, w, h, ' ', NewStyle())

	if l.header != nil {
		l.drawHeader(frame)
	}

	if l.footer != nil {
		l.drawFooter(frame)
	}
}

// Refresh updates the header and footer
func (l *Layout) Refresh() {
	frame, err := l.terminal.BeginFrame()
	if err != nil {
		return
	}

	// Save current cursor position?
	// BeginFrame/EndFrame might not preserve virtual cursor if we rely on terminal state?
	// Terminal state (virtualX, virtualY) is in Terminal struct.
	// Frame operations use internal methods.
	// If we want to preserve cursor, we must save/restore around the update.
	l.terminal.SaveCursor() // Locks, saves to t.savedCursor

	if l.header != nil {
		l.drawHeader(frame)
	}

	if l.footer != nil {
		l.drawFooter(frame)
	}

	l.terminal.EndFrame(frame)

	l.terminal.RestoreCursor()
}

// ContentArea returns the Y position and height of the content area
func (l *Layout) ContentArea() (y, height int) {
	return l.contentY, l.contentHeight
}

// PrintInContent prints text in the content area
func (l *Layout) PrintInContent(text string) {
	l.terminal.MoveCursor(0, l.contentY)
	l.terminal.Print(text)
}

func (l *Layout) updateContentArea() {
	_, height := l.terminal.Size()

	contentY := 0
	contentHeight := height

	if l.header != nil {
		contentY += l.header.Height
		contentHeight -= l.header.Height
	}

	if l.footer != nil {
		contentHeight -= l.footer.Height
	}

	l.contentY = contentY
	l.contentHeight = contentHeight
}

func (l *Layout) drawHeader(frame RenderFrame) {
	width, _ := frame.Size()
	// Header starts at 0,0

	if l.header.Border {
		// Draw bordered header
		f := NewFrame(0, 0, width, l.header.Height).
			WithBorderStyle(l.header.BorderStyle).
			WithColor(l.header.Style)
		f.Draw(frame)

		// Draw content inside border
		l.drawHeaderContent(frame, 1, 1, width-2)
	} else {
		// Draw header without border
		l.drawHeaderContent(frame, 0, 0, width)
	}
}

func (l *Layout) drawHeaderContent(frame RenderFrame, x, y, width int) {
	// Apply background if set
	if !l.header.Background.IsEmpty() {
		frame.FillStyled(x, y, width, 1, ' ', l.header.Background)
	}

	// Calculate sections
	leftLen := runewidth.StringWidth(l.header.Left)
	centerLen := runewidth.StringWidth(l.header.Center)
	rightLen := runewidth.StringWidth(l.header.Right)

	// Draw left section
	if l.header.Left != "" {
		frame.PrintStyled(x, y, l.header.Left, l.header.Style)
	}

	// Draw center section
	if l.header.Center != "" {
		centerPos := (width - centerLen) / 2
		if centerPos > leftLen {
			frame.PrintStyled(x+centerPos, y, l.header.Center, l.header.Style)
		}
	}

	// Draw right section
	if l.header.Right != "" {
		rightPos := width - rightLen
		frame.PrintStyled(x+rightPos, y, l.header.Right, l.header.Style)
	}
}

func (l *Layout) drawFooter(frame RenderFrame) {
	width, height := frame.Size()
	footerY := height - l.footer.Height

	if l.footer.Border {
		// Draw bordered footer
		f := NewFrame(0, footerY, width, l.footer.Height).
			WithBorderStyle(l.footer.BorderStyle).
			WithColor(l.footer.Style)
		f.Draw(frame)

		// Draw content inside border
		if l.footer.StatusBar {
			l.drawStatusBar(frame, 1, footerY+1, width-2)
		} else {
			l.drawFooterContent(frame, 1, footerY+1, width-2)
		}
	} else {
		// Draw footer without border
		if l.footer.StatusBar {
			l.drawStatusBar(frame, 0, footerY, width)
		} else {
			l.drawFooterContent(frame, 0, footerY, width)
		}
	}
}

func (l *Layout) drawFooterContent(frame RenderFrame, x, y, width int) {
	// Apply background if set
	if !l.footer.Background.IsEmpty() {
		frame.FillStyled(x, y, width, 1, ' ', l.footer.Background)
	}

	// Calculate sections
	leftLen := runewidth.StringWidth(l.footer.Left)
	centerLen := runewidth.StringWidth(l.footer.Center)
	rightLen := runewidth.StringWidth(l.footer.Right)

	// Draw left section
	if l.footer.Left != "" {
		frame.PrintStyled(x, y, l.footer.Left, l.footer.Style)
	}

	// Draw center section
	if l.footer.Center != "" {
		centerPos := (width - centerLen) / 2
		if centerPos > leftLen {
			frame.PrintStyled(x+centerPos, y, l.footer.Center, l.footer.Style)
		}
	}

	// Draw right section
	if l.footer.Right != "" {
		rightPos := width - rightLen
		frame.PrintStyled(x+rightPos, y, l.footer.Right, l.footer.Style)
	}
}

func (l *Layout) drawStatusBar(frame RenderFrame, x, y, width int) {
	// Apply background
	bg := l.footer.Background
	if bg.IsEmpty() {
		bg = NewStyle().WithBackground(ColorBrightBlack)
	}

	// Clear the line with background
	frame.FillStyled(x, y, width, 1, ' ', bg)

	// Draw status items
	currentX := x
	for i, item := range l.footer.StatusItems {
		if currentX >= x+width {
			break
		}

		// Add separator if not first item
		if i > 0 {
			sep := " │ "
			sepStyle := NewStyle().WithForeground(ColorBrightBlack)
			// Apply bg to separator style
			sepStyle.Background = bg.Background
			if bg.BgRGB != nil {
				sepStyle.BgRGB = bg.BgRGB
			}

			frame.PrintStyled(currentX, y, sep, sepStyle)
			currentX += runewidth.StringWidth(sep)
		}

		// Draw icon if present
		if item.Icon != "" {
			iconStyle := item.Style
			iconStyle.Background = bg.Background // Inherit bg
			if bg.BgRGB != nil {
				iconStyle.BgRGB = bg.BgRGB
			}

			iconText := item.Icon + " "
			frame.PrintStyled(currentX, y, iconText, iconStyle)
			currentX += runewidth.StringWidth(iconText)
		}

		// Draw key
		if item.Key != "" {
			keyStyle := item.Style
			if keyStyle.IsEmpty() {
				keyStyle = NewStyle().WithBold()
			}
			keyStyle.Background = bg.Background // Inherit bg
			if bg.BgRGB != nil {
				keyStyle.BgRGB = bg.BgRGB
			}

			keyText := item.Key + ": "
			frame.PrintStyled(currentX, y, keyText, keyStyle)
			currentX += runewidth.StringWidth(keyText)
		}

		// Draw value
		valueStyle := NewStyle().WithForeground(ColorWhite)
		valueStyle.Background = bg.Background // Inherit bg
		if bg.BgRGB != nil {
			valueStyle.BgRGB = bg.BgRGB
		}

		frame.PrintStyled(currentX, y, item.Value, valueStyle)
		currentX += runewidth.StringWidth(item.Value)
	}
}

// SimpleHeader creates a simple header with title
func SimpleHeader(title string, style Style) *Header {
	return &Header{
		Center: title,
		Style:  style,
		Height: 1,
	}
}

// BorderedHeader creates a bordered header
func BorderedHeader(title string, style Style) *Header {
	return &Header{
		Center:      title,
		Style:       style,
		Border:      true,
		BorderStyle: SingleBorder,
		Height:      3,
	}
}

// SimpleFooter creates a simple footer
func SimpleFooter(left, center, right string, style Style) *Footer {
	return &Footer{
		Left:   left,
		Center: center,
		Right:  right,
		Style:  style,
		Height: 1,
	}
}

// StatusBarFooter creates a status bar footer
func StatusBarFooter(items []StatusItem) *Footer {
	return &Footer{
		StatusBar:   true,
		StatusItems: items,
		Height:      1,
	}
}