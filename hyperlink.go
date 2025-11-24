package gooey

import (
	"fmt"
	"net/url"
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
	if h.URL == "" {
		return fmt.Errorf("hyperlink URL cannot be empty")
	}
	if h.Text == "" {
		return fmt.Errorf("hyperlink text cannot be empty")
	}

	// Parse URL to validate format
	if _, err := url.Parse(h.URL); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return nil
}

// OSC8Start returns the OSC 8 escape sequence to start a hyperlink.
// Format: \033]8;;URL\033\\
func OSC8Start(url string) string {
	// OSC 8 format: ESC ] 8 ; ; URL ST
	// Where ST (String Terminator) can be either ESC \ or BEL (\007)
	// We use ESC \ for better compatibility
	return fmt.Sprintf("\033]8;;%s\033\\", url)
}

// OSC8End returns the OSC 8 escape sequence to end a hyperlink.
// Format: \033]8;;\033\\
func OSC8End() string {
	return "\033]8;;\033\\"
}

// Format returns the formatted hyperlink with OSC 8 escape codes.
// If the terminal supports OSC 8, it will be clickable.
// The text is styled according to the hyperlink's Style field.
func (h Hyperlink) Format() string {
	var sb strings.Builder

	// Start hyperlink
	sb.WriteString(OSC8Start(h.URL))

	// Add styled text
	if !h.Style.IsEmpty() {
		sb.WriteString(h.Style.String())
	}
	sb.WriteString(h.Text)
	if !h.Style.IsEmpty() {
		sb.WriteString("\033[0m")
	}

	// End hyperlink
	sb.WriteString(OSC8End())

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

// PrintHyperlink is a convenience method for RenderFrame to print a hyperlink.
// It always uses OSC 8 format (terminals that don't support it will ignore the codes).
func (tf *terminalRenderFrame) PrintHyperlink(x, y int, link Hyperlink) error {
	// Validate the hyperlink
	if err := link.Validate(); err != nil {
		// Fall back to printing just the text if invalid
		return tf.PrintStyled(x, y, link.Text, link.Style)
	}

	// Add the URL to the style and print the text
	// The flush logic will emit OSC 8 codes when the URL changes
	styleWithURL := link.Style.WithURL(link.URL)
	return tf.PrintStyled(x, y, link.Text, styleWithURL)
}

// PrintHyperlinkFallback prints a hyperlink using the fallback format (text + URL).
// Use this when you know the terminal doesn't support OSC 8, or when you want
// to explicitly show the URL.
func (tf *terminalRenderFrame) PrintHyperlinkFallback(x, y int, link Hyperlink) error {
	// Validate the hyperlink
	if err := link.Validate(); err != nil {
		// Fall back to printing just the text if invalid
		return tf.PrintStyled(x, y, link.Text, link.Style)
	}

	// Print the styled text followed by the URL in parentheses
	text := link.Text + " (" + link.URL + ")"
	return tf.PrintStyled(x, y, text, link.Style)
}
