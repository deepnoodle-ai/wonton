package terminal

import (
	"fmt"
	"strings"
)

// Hyperlink represents a clickable hyperlink in the terminal using the OSC 8 protocol.
//
// OSC 8 is a terminal escape sequence standard that makes text clickable.
// When clicked, the terminal opens the URL in the default browser.
//
// # Terminal Support
//
// OSC 8 is supported by many modern terminals:
//   - iTerm2 (macOS)
//   - WezTerm (cross-platform)
//   - kitty (Linux, macOS)
//   - Windows Terminal
//   - GNOME Terminal 3.26+
//   - Konsole 18.08+
//   - Hyper
//
// For unsupported terminals, the escape codes are ignored and only the text is shown.
//
// # Usage
//
// Create hyperlinks and render them using RenderFrame:
//
//	link := terminal.NewHyperlink("https://example.com", "Click here")
//	frame.PrintHyperlink(10, 5, link)
//
// Or use the fallback format for terminals without OSC 8 support:
//
//	frame.PrintHyperlinkFallback(10, 5, link) // Prints: Click here (https://example.com)
//
// # Styling
//
// Hyperlinks have a default style (blue, underlined) but can be customized:
//
//	link := terminal.NewHyperlink("https://example.com", "Click here")
//	link = link.WithStyle(terminal.NewStyle().WithForeground(terminal.ColorGreen))
type Hyperlink struct {
	URL   string // The target URL (e.g., "https://example.com")
	Text  string // The display text (e.g., "Click here")
	Style Style  // Styling for the link text (default: blue, underlined)
}

// NewHyperlink creates a new Hyperlink with the given URL and display text.
// The hyperlink is given a default style (blue foreground, underlined).
//
// Example:
//
//	link := terminal.NewHyperlink("https://golang.org", "Go Website")
//	frame.PrintHyperlink(x, y, link)
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
