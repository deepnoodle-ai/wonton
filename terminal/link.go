// Package terminal provides OSC 8 hyperlink support for terminals.
//
// OSC 8 is a terminal escape sequence that enables clickable hyperlinks.
// It is supported by many modern terminals including:
//   - iTerm2
//   - WezTerm
//   - kitty
//   - Hyper
//   - Windows Terminal
//   - GNOME Terminal (3.26+)
//   - Konsole (18.08+)
//
// For unsupported terminals, the escape codes are simply ignored,
// so it's safe to use unconditionally.
//
// Example usage:
//
//	// Simple hyperlink
//	fmt.Print(terminal.Format("https://example.com", "Click here"))
//
//	// Hyperlink with custom ID (for grouping)
//	fmt.Print(terminal.FormatWithID("https://example.com", "Click here", "link1"))
//
//	// Check if URL is valid first
//	if err := terminal.ValidateURL("https://example.com"); err != nil {
//	    log.Printf("Invalid URL: %v", err)
//	}
package terminal

import (
	"fmt"
	"net/url"
	"strings"
)

// Start returns the OSC 8 escape sequence to start a hyperlink.
// Format: ESC ] 8 ; ; URL ST
// Where ST (String Terminator) is ESC \
func Start(targetURL string) string {
	return fmt.Sprintf("\033]8;;%s\033\\", targetURL)
}

// StartWithID returns the OSC 8 escape sequence to start a hyperlink with an ID.
// The ID can be used to group multiple hyperlink segments together.
// Format: ESC ] 8 ; id=ID ; URL ST
func StartWithID(targetURL, id string) string {
	return fmt.Sprintf("\033]8;id=%s;%s\033\\", id, targetURL)
}

// End returns the OSC 8 escape sequence to end a hyperlink.
// Format: ESC ] 8 ; ; ST
func End() string {
	return "\033]8;;\033\\"
}

// Format returns a complete hyperlink with the given URL and display text.
// This wraps the text in OSC 8 start and end sequences.
func Format(targetURL, text string) string {
	return Start(targetURL) + text + End()
}

// FormatWithID returns a complete hyperlink with a custom ID.
// The ID can be used to group multiple hyperlink segments as a single link.
func FormatWithID(targetURL, text, id string) string {
	return StartWithID(targetURL, id) + text + End()
}

// ValidateURL checks if a URL is valid for use in a hyperlink.
// Returns nil if valid, or an error describing the issue.
// Note: This is lenient and accepts relative URLs and URLs without schemes.
func ValidateURL(targetURL string) error {
	if targetURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	_, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return nil
}

// ValidateAbsoluteURL checks if a URL is a valid absolute URL with a scheme.
// Use this for stricter validation when you want to ensure the URL is absolute.
func ValidateAbsoluteURL(targetURL string) error {
	if targetURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsed, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Check for scheme
	if parsed.Scheme == "" {
		return fmt.Errorf("URL must have a scheme (e.g., https://)")
	}

	return nil
}

// Fallback returns a fallback representation for terminals that don't support OSC 8.
// Format: "text (URL)"
// For example: "Click here (https://example.com)"
func Fallback(targetURL, text string) string {
	return text + " (" + targetURL + ")"
}

// StripOSC8 removes OSC 8 hyperlink escape sequences from text.
// This is useful for getting plain text from hyperlink-formatted strings.
func StripOSC8(text string) string {
	result := text

	// Remove OSC 8 start sequences: ESC ] 8 ; ... ; URL ESC \
	for {
		start := strings.Index(result, "\033]8;")
		if start == -1 {
			break
		}
		// Find the ST (String Terminator): ESC \
		end := strings.Index(result[start:], "\033\\")
		if end == -1 {
			break
		}
		// Remove the escape sequence
		result = result[:start] + result[start+end+2:]
	}

	return result
}
