package tui

import (
	"unicode/utf8"
)

// AnimatedText represents text with color animation
type AnimatedText struct {
	x, y         int
	text         string
	animation    TextAnimation
	currentFrame uint64
}

// NewAnimatedText creates a new animated text element
func NewAnimatedText(x, y int, text string, animation TextAnimation) *AnimatedText {
	return &AnimatedText{
		x:         x,
		y:         y,
		text:      text,
		animation: animation,
	}
}

// Update updates the animation frame
func (at *AnimatedText) Update(frame uint64) {
	at.currentFrame = frame
}

// Draw renders the animated text
func (at *AnimatedText) Draw(frame RenderFrame) {
	runes := []rune(at.text)
	totalChars := len(runes)

	for i, r := range runes {
		currentX := at.x + i
		var style Style
		if at.animation != nil {
			// Use the animation's GetStyle method - consolidated rendering
			style = at.animation.GetStyle(at.currentFrame, i, totalChars)
		} else {
			style = NewStyle()
		}
		frame.SetCell(currentX, at.y, r, style)
	}
}

// Position returns the element's position
func (at *AnimatedText) Position() (x, y int) {
	return at.x, at.y
}

// Dimensions returns the element's dimensions
func (at *AnimatedText) Dimensions() (width, height int) {
	return utf8.RuneCountInString(at.text), 1
}

// SetText updates the text content
func (at *AnimatedText) SetText(text string) {
	at.text = text
}

// SetPosition updates the position
func (at *AnimatedText) SetPosition(x, y int) {
	at.x = x
	at.y = y
}

// AnimatedMultiLine represents multiple lines of animated content
type AnimatedMultiLine struct {
	x, y         int
	width        int
	lines        []string
	animations   []TextAnimation
	currentFrame uint64
}

// NewAnimatedMultiLine creates a new multi-line animated element
func NewAnimatedMultiLine(x, y, width int) *AnimatedMultiLine {
	return &AnimatedMultiLine{
		x:          x,
		y:          y,
		width:      width,
		lines:      make([]string, 0),
		animations: make([]TextAnimation, 0),
	}
}

// SetLine sets the content and animation for a specific line
func (aml *AnimatedMultiLine) SetLine(index int, text string, animation TextAnimation) {
	// Ensure slices are large enough
	for len(aml.lines) <= index {
		aml.lines = append(aml.lines, "")
		aml.animations = append(aml.animations, nil)
	}

	aml.lines[index] = text
	aml.animations[index] = animation
}

// AddLine adds a new line with animation
func (aml *AnimatedMultiLine) AddLine(text string, animation TextAnimation) {
	aml.lines = append(aml.lines, text)
	aml.animations = append(aml.animations, animation)
}

// ClearLines removes all lines
func (aml *AnimatedMultiLine) ClearLines() {
	aml.lines = aml.lines[:0]
	aml.animations = aml.animations[:0]
}

// Update updates the animation frame
func (aml *AnimatedMultiLine) Update(frame uint64) {
	aml.currentFrame = frame
}

// Draw renders all animated lines
func (aml *AnimatedMultiLine) Draw(frame RenderFrame) {
	frameW, _ := frame.Size()

	for i, line := range aml.lines {
		// Clear line first
		frame.FillStyled(0, aml.y+i, frameW, 1, ' ', NewStyle())

		if i < len(aml.animations) && aml.animations[i] != nil {
			// Draw animated line
			runes := []rune(line)
			totalChars := len(runes)

			for j, r := range runes {
				if j >= aml.width {
					break // Respect width limit
				}

				currentX := aml.x + j
				// Use the animation's GetStyle method - consolidated rendering
				style := aml.animations[i].GetStyle(aml.currentFrame, j, totalChars)
				frame.SetCell(currentX, aml.y+i, r, style)
			}
		} else {
			// Draw static line
			frame.PrintStyled(aml.x, aml.y+i, line, NewStyle())
		}
	}
}

// Position returns the element's position
func (aml *AnimatedMultiLine) Position() (x, y int) {
	return aml.x, aml.y
}

// Dimensions returns the element's dimensions
func (aml *AnimatedMultiLine) Dimensions() (width, height int) {
	return aml.width, len(aml.lines)
}

// SetPosition updates the element's position
func (aml *AnimatedMultiLine) SetPosition(x, y int) {
	aml.x = x
	aml.y = y
}

// SetWidth updates the element's width
func (aml *AnimatedMultiLine) SetWidth(width int) {
	aml.width = width
}

// AnimatedStatusBar represents an animated status bar
type AnimatedStatusBar struct {
	x, y         int
	width        int
	items        []*AnimatedStatusItem
	separator    string
	background   RGB
	currentFrame uint64
}

// AnimatedStatusItem represents a single status item with animation
type AnimatedStatusItem struct {
	Key       string
	Value     string
	Icon      string
	Animation TextAnimation
	Style     Style
}

// NewAnimatedStatusBar creates a new animated status bar
func NewAnimatedStatusBar(x, y, width int) *AnimatedStatusBar {
	return &AnimatedStatusBar{
		x:          x,
		y:          y,
		width:      width,
		items:      make([]*AnimatedStatusItem, 0),
		separator:  " │ ",
		background: NewRGB(40, 40, 40),
	}
}

// AddItem adds a status item with optional animation
func (asb *AnimatedStatusBar) AddItem(key, value, icon string, animation TextAnimation, style Style) {
	item := &AnimatedStatusItem{
		Key:       key,
		Value:     value,
		Icon:      icon,
		Animation: animation,
		Style:     style,
	}

	asb.items = append(asb.items, item)
}

// UpdateItem updates an existing status item
func (asb *AnimatedStatusBar) UpdateItem(index int, key, value string) {
	if index >= 0 && index < len(asb.items) {
		asb.items[index].Key = key
		asb.items[index].Value = value
	}
}

// Update updates the animation frame
func (asb *AnimatedStatusBar) Update(frame uint64) {
	asb.currentFrame = frame
}

// Draw renders the animated status bar
func (asb *AnimatedStatusBar) Draw(frame RenderFrame) {
	// Draw background
	bgStyle := NewStyle().WithBgRGB(asb.background)
	frame.FillStyled(asb.x, asb.y, asb.width, 1, ' ', bgStyle)

	currentX := asb.x
	for i, item := range asb.items {
		if currentX >= asb.x+asb.width {
			break
		}

		// Add separator if not first item
		if i > 0 {
			sepStyle := NewStyle().WithForeground(ColorBrightBlack).WithBgRGB(asb.background)
			frame.PrintStyled(currentX, asb.y, asb.separator, sepStyle)
			currentX += utf8.RuneCountInString(asb.separator)
		}

		// Draw icon if present
		if item.Icon != "" {
			iconText := item.Icon + " "
			if item.Animation != nil {
				// Apply animation to icon
				runes := []rune(iconText)
				for j, r := range runes {
					style := item.Animation.GetStyle(asb.currentFrame, j, len(runes))
					// Combine with background
					style = style.WithBgRGB(asb.background)
					frame.SetCell(currentX+j, asb.y, r, style)
				}
			} else {
				iconStyle := item.Style.WithBgRGB(asb.background)
				frame.PrintStyled(currentX, asb.y, iconText, iconStyle)
			}
			currentX += utf8.RuneCountInString(iconText)
		}

		// Draw key
		if item.Key != "" {
			keyText := item.Key + ": "
			keyStyle := item.Style
			if keyStyle.IsEmpty() {
				keyStyle = NewStyle().WithBold()
			}
			keyStyle = keyStyle.WithBgRGB(asb.background)
			frame.PrintStyled(currentX, asb.y, keyText, keyStyle)
			currentX += utf8.RuneCountInString(keyText)
		}

		// Draw value
		if item.Value != "" {
			if item.Animation != nil {
				// Apply animation to value
				runes := []rune(item.Value)
				for j, r := range runes {
					switch anim := item.Animation.(type) {
					case *RainbowAnimation:
						colors := SmoothRainbow(anim.Length)
						offset := int(asb.currentFrame) / anim.Speed
						if anim.Reversed {
							offset = -offset
						}
						rainbowPos := (j + offset) % len(colors)
						if rainbowPos < 0 {
							rainbowPos += len(colors)
						}
						rgb := colors[rainbowPos]
						// Combine with background
						style := NewStyle().WithFgRGB(rgb).WithBgRGB(asb.background)
						frame.SetCell(currentX+j, asb.y, r, style)
					default:
						style := item.Animation.GetStyle(asb.currentFrame, j, len(runes))
						style = style.WithBgRGB(asb.background)
						frame.SetCell(currentX+j, asb.y, r, style)
					}
				}
			} else {
				valueStyle := NewStyle().WithForeground(ColorWhite).WithBgRGB(asb.background)
				frame.PrintStyled(currentX, asb.y, item.Value, valueStyle)
			}
			currentX += utf8.RuneCountInString(item.Value)
		}
	}
}

// Position returns the status bar position
func (asb *AnimatedStatusBar) Position() (x, y int) {
	return asb.x, asb.y
}

// Dimensions returns the status bar dimensions
func (asb *AnimatedStatusBar) Dimensions() (width, height int) {
	return asb.width, 1
}

// SetPosition updates the status bar position
func (asb *AnimatedStatusBar) SetPosition(x, y int) {
	asb.x = x
	asb.y = y
}
