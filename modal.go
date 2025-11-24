package gooey

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Modal represents a popup dialog
type Modal struct {
	Title        string
	Content      string
	Buttons      []string
	ActiveButton int
	Callback     func(buttonIndex int)
	Style        Style
	BorderStyle  BorderStyle
	Width        int
	Height       int
}

// NewModal creates a new modal dialog
func NewModal(title, content string, buttons []string) *Modal {
	return &Modal{
		Title:        title,
		Content:      content,
		Buttons:      buttons,
		ActiveButton: 0,
		Style:        NewStyle().WithForeground(ColorWhite).WithBackground(ColorBlack),
		BorderStyle:  DoubleBorder,
		Width:        50,
		Height:       10,
	}
}

// SetWidth sets the width of the modal
func (m *Modal) SetWidth(width int) *Modal {
	m.Width = width
	return m
}

// SetHeight sets the height of the modal
func (m *Modal) SetHeight(height int) *Modal {
	m.Height = height
	return m
}

// SetCallback sets the callback function when a button is pressed
func (m *Modal) SetCallback(callback func(buttonIndex int)) *Modal {
	m.Callback = callback
	return m
}

// Draw renders the modal centered on the screen
func (m *Modal) Draw(frame RenderFrame) {
	screenWidth, screenHeight := frame.Size()

	// Center the modal
	x := (screenWidth - m.Width) / 2
	y := (screenHeight - m.Height) / 2

	// Draw shadow (optional, simple offset)
	shadowStyle := NewStyle().WithBackground(ColorBlack).WithDim()
	frame.FillStyled(x+1, y+1, m.Width, m.Height, ' ', shadowStyle)

	// Draw background and border
	f := NewFrame(x, y, m.Width, m.Height).
		WithBorderStyle(m.BorderStyle).
		WithColor(m.Style)
	f.Draw(frame)

	// Draw Title
	if m.Title != "" {
		titleLen := runewidth.StringWidth(m.Title)
		titleX := x + (m.Width-titleLen)/2
		// Ensure title fits
		if titleX < x+1 {
			titleX = x + 1
		}
		frame.PrintStyled(titleX, y+1, m.Title, m.Style.WithBold())

		// Draw separator below title
		// Using the border style for the separator line
		// We need to handle this manually or use the Frame utility if it supported internal lines
		// For now, just draw a line
		// frame.FillStyled(x+1, y+2, m.Width-2, 1, '─', m.Style)
	}

	// Draw Content
	// Simple text wrapping for now
	contentLines := splitLines(m.Content, m.Width-4)
	contentY := y + 3 // Start below title and separator space
	for i, line := range contentLines {
		if contentY+i >= y+m.Height-2 {
			break // Prevent overflow
		}
		frame.PrintStyled(x+2, contentY+i, line, m.Style)
	}

	// Draw Buttons
	if len(m.Buttons) > 0 {
		m.drawButtons(frame, x, y)
	}
}

func (m *Modal) drawButtons(frame RenderFrame, modalX, modalY int) {
	totalBtnWidth := 0
	gap := 2
	for _, btn := range m.Buttons {
		totalBtnWidth += runewidth.StringWidth(btn) + 4 // +4 for padding/brackets
	}
	totalBtnWidth += (len(m.Buttons) - 1) * gap

	startX := modalX + (m.Width-totalBtnWidth)/2
	buttonY := modalY + m.Height - 2

	currentX := startX
	for i, btn := range m.Buttons {
		style := m.Style
		label := fmt.Sprintf("[ %s ]", btn)

		if i == m.ActiveButton {
			style = style.WithReverse()
		}

		frame.PrintStyled(currentX, buttonY, label, style)
		currentX += runewidth.StringWidth(label) + gap
	}
}

// HandleKey handles key events for the modal
func (m *Modal) HandleKey(event KeyEvent) bool {
	switch event.Key {
	case KeyArrowLeft:
		if m.ActiveButton > 0 {
			m.ActiveButton--
			return true
		}
	case KeyArrowRight:
		if m.ActiveButton < len(m.Buttons)-1 {
			m.ActiveButton++
			return true
		}
	case KeyEnter:
		if m.Callback != nil {
			m.Callback(m.ActiveButton)
		}
		return true
	}
	return false // Consume all keys? Or just navigation?
	// Modals usually trap focus, so we should probably return true for most keys to prevent background interaction,
	// or at least specific ones.
}

// Helper for simple wrapping
func splitLines(text string, maxWidth int) []string {
	var lines []string
	// This is a very basic wrapper. A better one (Feature #10) would use a proper algorithm.
	// For now, split by newlines and then length

	// First, normalize newlines
	text = strings.ReplaceAll(text, "\r\n", "\n")

	rawLines := strings.Split(text, "\n")
	for _, rawLine := range rawLines {
		if len(rawLine) <= maxWidth {
			lines = append(lines, rawLine)
		} else {
			// chunks
			runes := []rune(rawLine)
			for len(runes) > 0 {
				chunkSize := maxWidth
				if len(runes) < chunkSize {
					chunkSize = len(runes)
				}
				lines = append(lines, string(runes[:chunkSize]))
				runes = runes[chunkSize:]
			}
		}
	}
	return lines
}
