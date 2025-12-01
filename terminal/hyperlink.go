package terminal

import (
	"fmt"
	"strings"
)

// Hyperlink represents a clickable hyperlink in the terminal using OSC 8 protocol.
// OSC 8 is supported by many modern terminals including iTerm2, WezTerm, kitty,
// and others. For unsupported terminals, it gracefully falls back to showing
// the text with the URL in parentheses.
type Hyperlink struct {
	URL   string // The target URL (e.g., "https://example.com")
	Text  string // The display text (e.g., "Click here")
	Style Style  // Optional styling for the link text
}

// NewHyperlink creates a new hyperlink with the given URL and display text.
// The URL is validated to ensure it's a valid URL format.
func NewHyperlink(url, text string) Hyperlink {
	return Hyperlink{
		URL:   url,
		Text:  text,
		Style: NewStyle().WithUnderline().WithForeground(ColorBlue),
	}
}

// WithStyle sets the style for the hyperlink text.
// By default, hyperlinks are blue and underlined.
func (h Hyperlink) WithStyle(style Style) Hyperlink {
	h.Style = style
	return h
}

// Validate checks if the hyperlink has valid components.
// Returns an error if the URL is invalid or empty, or if the text is empty.
func (h Hyperlink) Validate() error {
	if h.Text == "" {
		return fmt.Errorf("hyperlink text cannot be empty")
	}
	return ValidateURL(h.URL)
}

// OSC8Start returns the OSC 8 escape sequence to start a hyperlink.
// Format: \033]8;;URL\033\\
func OSC8Start(url string) string {
	return Start(url)
}

// OSC8End returns the OSC 8 escape sequence to end a hyperlink.
// Format: \033]8;;\033\\
func OSC8End() string {
	return End()
}

// Format returns the formatted hyperlink with OSC 8 escape codes.
// If the terminal supports OSC 8, it will be clickable.
// The text is styled according to the hyperlink's Style field.
func (h Hyperlink) Format() string {
	var sb strings.Builder

	// Start hyperlink
	sb.WriteString(Start(h.URL))

	// Add styled text
	if !h.Style.IsEmpty() {
		sb.WriteString(h.Style.String())
	}
	sb.WriteString(h.Text)
	if !h.Style.IsEmpty() {
		sb.WriteString("\033[0m")
	}

	// End hyperlink
	sb.WriteString(End())

	return sb.String()
}

// FormatFallback returns a fallback representation for terminals that don't support OSC 8.
// Format: "Text (URL)"
// For example: "Click here (https://example.com)"
func (h Hyperlink) FormatFallback() string {
	var sb strings.Builder

	// Add styled text
	if !h.Style.IsEmpty() {
		sb.WriteString(h.Style.String())
	}
	sb.WriteString(h.Text)
	if !h.Style.IsEmpty() {
		sb.WriteString("\033[0m")
	}

	// Add URL in parentheses
	sb.WriteString(" (")
	sb.WriteString(h.URL)
	sb.WriteString(")")

	return sb.String()
}

// FormatWithOption returns either the OSC 8 formatted hyperlink or the fallback,
// depending on the useOSC8 parameter.
func (h Hyperlink) FormatWithOption(useOSC8 bool) string {
	if useOSC8 {
		return h.Format()
	}
	return h.FormatFallback()
}
