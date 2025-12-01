package tui

// import (
// 	"fmt"
// 	"strings"

// 	"github.com/mattn/go-runewidth"
// )

// // Button represents a clickable button component
// type Button struct {
// 	X, Y       int
// 	Width      int
// 	Label      string
// 	Style      Style
// 	HoverStyle Style
// 	Focused    bool
// 	Hovered    bool
// 	OnClick    func()
// }

// // NewButton creates a new button
// func NewButton(x, y int, label string, onClick func()) *Button {
// 	return &Button{
// 		X:          x,
// 		Y:          y,
// 		Width:      len(label) + 4, // Add padding
// 		Label:      label,
// 		Style:      NewStyle().WithBackground(ColorBlue).WithForeground(ColorWhite),
// 		HoverStyle: NewStyle().WithBackground(ColorCyan).WithForeground(ColorBlack).WithBold(),
// 		OnClick:    onClick,
// 	}
// }

// // Draw renders the button
// func (b *Button) Draw(frame RenderFrame) {
// 	style := b.Style
// 	if b.Hovered {
// 		style = b.HoverStyle
// 	}
// 	if b.Focused {
// 		style = style.WithUnderline()
// 	}

// 	// Draw button with borders
// 	buttonText := fmt.Sprintf(" %s ", b.Label)
// 	if len(buttonText) < b.Width {
// 		padding := b.Width - len(buttonText)
// 		buttonText += strings.Repeat(" ", padding)
// 	}

// 	// Coordinate handling for compatibility with composition system
// 	// If we are in a SubFrame sized exactly for this button, we should draw at 0,0
// 	drawX, drawY := b.X, b.Y
// 	frameWidth, frameHeight := frame.Size()
// 	if frameWidth == b.Width && frameHeight == 1 {
// 		drawX, drawY = 0, 0
// 	}

// 	frame.PrintStyled(drawX, drawY, "["+buttonText+"]", style)
// }

// // GetRegion returns the mouse region for this button
// func (b *Button) GetRegion() *MouseRegion {
// 	return &MouseRegion{
// 		X:      b.X,
// 		Y:      b.Y,
// 		Width:  b.Width + 2, // Include brackets
// 		Height: 1,
// 		Label:  b.Label,
// 		OnClick: func(event *MouseEvent) {
// 			if b.OnClick != nil {
// 				b.OnClick()
// 			}
// 		},
// 		OnEnter: func(event *MouseEvent) {
// 			b.Hovered = true
// 		},
// 		OnLeave: func(event *MouseEvent) {
// 			b.Hovered = false
// 		},
// 	}
// }

// // TabCompleter handles tab completion functionality
// type TabCompleter struct {
// 	suggestions    []string
// 	currentIndex   int
// 	Visible        bool
// 	maxVisible     int
// 	prefix         string
// 	X, Y           int
// 	Width          int
// 	selectedStyle  Style
// 	normalStyle    Style
// 	OnSelect       func(string)
// 	clearDropdown  bool
// 	lastDrawnLines int
// 	savedContent   []string // Store content that dropdown overlays
// }

// // NewTabCompleter creates a new tab completer
// func NewTabCompleter() *TabCompleter {
// 	return &TabCompleter{
// 		suggestions:   make([]string, 0),
// 		maxVisible:    5,
// 		currentIndex:  0,
// 		selectedStyle: NewStyle().WithBackground(ColorBlue).WithForeground(ColorWhite),
// 		normalStyle:   NewStyle().WithBackground(ColorBlack).WithForeground(ColorWhite),
// 	}
// }

// // SetSuggestions updates the list of suggestions
// func (tc *TabCompleter) SetSuggestions(suggestions []string, prefix string) {
// 	tc.suggestions = suggestions
// 	tc.prefix = prefix
// 	tc.currentIndex = 0
// 	tc.Visible = len(suggestions) > 0
// }

// // Show displays the completion dropdown
// func (tc *TabCompleter) Show(x, y int, width int) {
// 	tc.X = x
// 	tc.Y = y
// 	tc.Width = width
// 	tc.Visible = true
// }

// // Hide hides the completion dropdown
// func (tc *TabCompleter) Hide() {
// 	tc.Visible = false
// 	tc.clearDropdown = true
// }

// // SelectNext moves to the next suggestion
// func (tc *TabCompleter) SelectNext() {
// 	if len(tc.suggestions) > 0 {
// 		tc.currentIndex = (tc.currentIndex + 1) % len(tc.suggestions)
// 	}
// }

// // SelectPrev moves to the previous suggestion
// func (tc *TabCompleter) SelectPrev() {
// 	if len(tc.suggestions) > 0 {
// 		tc.currentIndex--
// 		if tc.currentIndex < 0 {
// 			tc.currentIndex = len(tc.suggestions) - 1
// 		}
// 	}
// }

// // GetSelected returns the currently selected suggestion
// func (tc *TabCompleter) GetSelected() string {
// 	if tc.currentIndex < len(tc.suggestions) {
// 		return tc.suggestions[tc.currentIndex]
// 	}
// 	return ""
// }

// // Draw renders the tab completion dropdown
// func (tc *TabCompleter) Draw(frame RenderFrame) {
// 	termWidth, _ := frame.Size()

// 	// Handle clearing if dropdown was hidden
// 	if tc.clearDropdown && tc.lastDrawnLines > 0 {
// 		for i := 0; i <= tc.lastDrawnLines; i++ {
// 			clearWidth := tc.Width
// 			if tc.X+clearWidth > termWidth {
// 				clearWidth = termWidth - tc.X
// 			}
// 			if clearWidth > 0 {
// 				frame.FillStyled(tc.X, tc.Y+1+i, clearWidth, 1, ' ', NewStyle())
// 			}
// 		}
// 		tc.clearDropdown = false
// 		tc.lastDrawnLines = 0
// 		tc.savedContent = nil
// 		return
// 	}

// 	if !tc.Visible || len(tc.suggestions) == 0 || tc.Width < 4 {
// 		return
// 	}

// 	// Use a background color to ensure dropdown is visible over other content
// 	dropdownBg := ColorBlack

// 	// Calculate visible range
// 	start := 0
// 	end := len(tc.suggestions)
// 	if end > tc.maxVisible {
// 		// Show window around current selection
// 		start = tc.currentIndex - tc.maxVisible/2
// 		if start < 0 {
// 			start = 0
// 		}
// 		end = start + tc.maxVisible
// 		if end > len(tc.suggestions) {
// 			end = len(tc.suggestions)
// 			start = end - tc.maxVisible
// 		}
// 	}

// 	// Track how many lines we're drawing for cleanup later
// 	tc.lastDrawnLines = (end - start) + 1 // +1 for bottom border

// 	// Save content that will be overwritten (for proper layering)
// 	// Note: RenderFrame doesn't support reading content back easily.
// 	// We assume redraw will handle restoration if we clear.
// 	if tc.savedContent == nil {
// 		tc.savedContent = make([]string, tc.lastDrawnLines)
// 	}

// 	// Draw dropdown box with proper clearing
// 	for i := start; i < end; i++ {
// 		y := tc.Y + 1 + (i - start)

// 		// Clear the entire line properly first (conceptually)
// 		clearWidth := tc.Width
// 		if tc.X+clearWidth > termWidth {
// 			clearWidth = termWidth - tc.X
// 		}
// 		if clearWidth > 0 {
// 			frame.FillStyled(tc.X, y, clearWidth, 1, ' ', NewStyle())
// 		}

// 		// Format suggestion
// 		suggestion := tc.suggestions[i]
// 		displayWidth := runewidth.StringWidth(suggestion)
// 		if displayWidth > tc.Width-4 { // account for indicator and borders
// 			// Truncate to fit width
// 			truncated := runewidth.Truncate(suggestion, tc.Width-7, "...")
// 			suggestion = truncated
// 		}

// 		// Apply style with solid background for visibility
// 		style := tc.normalStyle.WithBackground(dropdownBg)
// 		indicator := "  "
// 		if i == tc.currentIndex {
// 			style = tc.selectedStyle.WithBackground(dropdownBg)
// 			indicator = "▶ "
// 		}

// 		// Draw suggestion with border
// 		line := fmt.Sprintf("│%s%s", indicator, suggestion)
// 		// Calculate actual display width for proper padding
// 		lineWidth := runewidth.StringWidth(line)
// 		padding := tc.Width - lineWidth - 1 // -1 for the closing border
// 		if padding > 0 {
// 			line += strings.Repeat(" ", padding)
// 		}
// 		line += "│"

// 		// Print the line, ensuring full width is covered with background
// 		frame.PrintStyled(tc.X, y, line, style)
// 	}

// 	// Draw bottom border
// 	y := tc.Y + 1 + (end - start)
// 	clearWidth := tc.Width
// 	if tc.X+clearWidth > termWidth {
// 		clearWidth = termWidth - tc.X
// 	}
// 	if clearWidth > 0 {
// 		frame.FillStyled(tc.X, y, clearWidth, 1, ' ', NewStyle()) // Clear line
// 	}

// 	borderWidth := tc.Width - 2
// 	if borderWidth < 0 {
// 		borderWidth = 0
// 	}
// 	border := "└" + strings.Repeat("─", borderWidth) + "┘"
// 	// Ensure border has background for visibility
// 	borderStyle := tc.normalStyle.WithBackground(dropdownBg)
// 	frame.PrintStyled(tc.X, y, border, borderStyle)

// 	// Show scroll indicators if needed
// 	if start > 0 {
// 		frame.PrintStyled(tc.X+tc.Width-3, tc.Y+1, "↑", NewStyle())
// 	}
// 	if end < len(tc.suggestions) {
// 		frame.PrintStyled(tc.X+tc.Width-3, y-1, "↓", NewStyle())
// 	}
// }

// // GetRegions returns mouse regions for clickable suggestions
// func (tc *TabCompleter) GetRegions() []*MouseRegion {
// 	if !tc.Visible || len(tc.suggestions) == 0 {
// 		return nil
// 	}

// 	regions := make([]*MouseRegion, 0)

// 	// Calculate visible range
// 	start := 0
// 	end := len(tc.suggestions)
// 	if end > tc.maxVisible {
// 		start = tc.currentIndex - tc.maxVisible/2
// 		if start < 0 {
// 			start = 0
// 		}
// 		end = start + tc.maxVisible
// 		if end > len(tc.suggestions) {
// 			end = len(tc.suggestions)
// 			start = end - tc.maxVisible
// 		}
// 	}

// 	// Create regions for each visible suggestion
// 	for i := start; i < end; i++ {
// 		idx := i // Capture for closure
// 		y := tc.Y + 1 + (i - start)

// 		region := &MouseRegion{
// 			X:      tc.X,
// 			Y:      y,
// 			Width:  tc.Width,
// 			Height: 1,
// 			Label:  tc.suggestions[idx],
// 			OnClick: func(event *MouseEvent) {
// 				tc.currentIndex = idx
// 				if tc.OnSelect != nil {
// 					tc.OnSelect(tc.suggestions[idx])
// 				}
// 			},
// 			OnEnter: func(event *MouseEvent) {
// 				tc.currentIndex = idx
// 			},
// 		}
// 		regions = append(regions, region)
// 	}

// 	return regions
// }

// // RadioGroup represents a group of radio buttons
// type RadioGroup struct {
// 	X, Y          int
// 	Options       []string
// 	Selected      int
// 	Style         Style
// 	SelectedStyle Style
// 	OnChange      func(int, string)
// }

// // NewRadioGroup creates a new radio button group
// func NewRadioGroup(x, y int, options []string) *RadioGroup {
// 	return &RadioGroup{
// 		X:             x,
// 		Y:             y,
// 		Options:       options,
// 		Selected:      0,
// 		Style:         NewStyle(),
// 		SelectedStyle: NewStyle().WithForeground(ColorGreen).WithBold(),
// 	}
// }

// // Draw renders the radio group
// func (rg *RadioGroup) Draw(frame RenderFrame) {
// 	// Coordinate handling for compatibility with composition system
// 	drawX, drawY := rg.X, rg.Y
// 	_, frameHeight := frame.Size()
// 	// If frame height matches our content height exactly, assume we are in a SubFrame
// 	// and should draw relative to it (at 0,0) instead of absolute coordinates.
// 	if frameHeight == len(rg.Options) {
// 		drawX, drawY = 0, 0
// 	}

// 	for i, option := range rg.Options {
// 		style := rg.Style
// 		indicator := "○"
// 		if i == rg.Selected {
// 			style = rg.SelectedStyle
// 			indicator = "●"
// 		}

// 		frame.PrintStyled(drawX, drawY+i, fmt.Sprintf(" %s %s", indicator, option), style)
// 	}
// }

// // GetRegions returns mouse regions for the radio buttons
// func (rg *RadioGroup) GetRegions() []*MouseRegion {
// 	regions := make([]*MouseRegion, 0)

// 	for i, option := range rg.Options {
// 		idx := i // Capture for closure

// 		region := &MouseRegion{
// 			X:      rg.X,
// 			Y:      rg.Y + i,
// 			Width:  len(option) + 4,
// 			Height: 1,
// 			Label:  option,
// 			OnClick: func(event *MouseEvent) {
// 				rg.Selected = idx
// 				if rg.OnChange != nil {
// 					rg.OnChange(idx, rg.Options[idx])
// 				}
// 			},
// 		}
// 		regions = append(regions, region)
// 	}

// 	return regions
// }
