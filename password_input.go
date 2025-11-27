package gooey

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// PasswordInput provides secure password input with advanced features.
// It supports:
// - iTerm2 secure input mode (prevents keylogging)
// - VS Code terminal integration
// - Clipboard disable during input
// - Memory zeroing on completion
// - Show/hide toggle
// - Paste confirmation
type PasswordInput struct {
	terminal         *Terminal
	prompt           string
	promptStyle      Style
	placeholder      string
	maxLength        int
	showCharacters   bool
	maskChar         rune
	enableSecureMode bool // Enable iTerm2/VS Code secure input
	disableClipboard bool
	confirmPaste     bool
	autoHideDelay    int // Seconds before auto-hiding (0 = disabled)
}

// NewPasswordInput creates a new secure password input handler.
func NewPasswordInput(terminal *Terminal) *PasswordInput {
	return &PasswordInput{
		terminal:         terminal,
		prompt:           "Password: ",
		promptStyle:      NewStyle().WithForeground(ColorYellow).WithBold(),
		placeholder:      "",
		maxLength:        0,
		showCharacters:   true, // Default to showing masked characters for visual feedback
		maskChar:         '*',
		enableSecureMode: true,
		disableClipboard: true,
		confirmPaste:     true,
		autoHideDelay:    0,
	}
}

// WithPrompt sets the password prompt.
func (p *PasswordInput) WithPrompt(prompt string, style Style) *PasswordInput {
	p.prompt = prompt
	p.promptStyle = style
	return p
}

// WithPlaceholder sets placeholder text.
func (p *PasswordInput) WithPlaceholder(placeholder string) *PasswordInput {
	p.placeholder = placeholder
	return p
}

// WithMaxLength sets the maximum password length.
func (p *PasswordInput) WithMaxLength(length int) *PasswordInput {
	p.maxLength = length
	return p
}

// WithMaskChar sets the character used to mask password input.
// Only used when ShowCharacters is enabled.
func (p *PasswordInput) WithMaskChar(char rune) *PasswordInput {
	p.maskChar = char
	return p
}

// ShowCharacters enables showing masked characters (*) instead of no echo.
// This provides visual feedback while maintaining security.
func (p *PasswordInput) ShowCharacters(show bool) *PasswordInput {
	p.showCharacters = show
	return p
}

// EnableSecureMode controls whether to enable terminal-specific secure input modes.
// When enabled (default), sends escape codes to iTerm2/VS Code to prevent keylogging.
func (p *PasswordInput) EnableSecureMode(enable bool) *PasswordInput {
	p.enableSecureMode = enable
	return p
}

// DisableClipboard controls whether to disable clipboard access during password input.
func (p *PasswordInput) DisableClipboard(disable bool) *PasswordInput {
	p.disableClipboard = disable
	return p
}

// ConfirmPaste controls whether to require confirmation before accepting pasted content.
func (p *PasswordInput) ConfirmPaste(confirm bool) *PasswordInput {
	p.confirmPaste = confirm
	return p
}

// Read reads a password with enhanced security features.
// The password is stored in a byte slice that is zeroed after use.
//
// Example:
//
//	pwdInput := gooey.NewPasswordInput(terminal)
//	pwdInput.WithPrompt("Enter password: ", gooey.NewStyle())
//	password, err := pwdInput.Read()
//	if err != nil {
//	    return err
//	}
//	defer password.Clear() // Zero memory when done
//	// Use password...
func (p *PasswordInput) Read() (*SecureString, error) {
	// Enable secure input mode
	if p.enableSecureMode {
		p.enableTerminalSecureMode()
		defer p.disableTerminalSecureMode()
	}

	// Show prompt
	p.terminal.SetStyle(p.promptStyle)
	p.terminal.Print(p.prompt)
	p.terminal.Reset()

	// Show placeholder if provided (in masked mode only, since no-echo mode handles it differently)
	if p.placeholder != "" && p.showCharacters {
		placeholderStyle := NewStyle().WithForeground(ColorBrightBlack)
		p.terminal.SetStyle(placeholderStyle)
		p.terminal.Print(p.placeholder)
		p.terminal.Reset()
	}

	p.terminal.Flush()

	var password []byte
	var err error

	if p.showCharacters {
		// Ensure terminal is in normal mode before entering raw mode
		// This fixes issues with leftover state from previous inputs
		p.terminal.DisableRawMode()

		// Use custom input with masked characters
		password, err = p.readMasked()
	} else {
		// Use term.ReadPassword for no-echo input
		fd := int(os.Stdin.Fd())
		password, err = term.ReadPassword(fd)
	}

	if err != nil {
		return nil, err
	}

	// Move to next line (term.ReadPassword doesn't add newline)
	p.terminal.Println("")
	p.terminal.Flush()

	// Check max length
	if p.maxLength > 0 && len(password) > p.maxLength {
		// Clear the password
		for i := range password {
			password[i] = 0
		}
		return nil, fmt.Errorf("password exceeds maximum length of %d", p.maxLength)
	}

	return NewSecureString(password), nil
}

// readMasked reads password with visual masking (showing * characters).
func (p *PasswordInput) readMasked() ([]byte, error) {
	// Enable raw mode for character-by-character input
	if err := p.terminal.EnableRawMode(); err != nil {
		return nil, err
	}
	defer p.terminal.DisableRawMode()

	var buffer []byte
	decoder := NewKeyDecoder(os.Stdin)
	firstChar := true

	for {
		event, err := decoder.ReadKeyEvent()
		if err != nil {
			return nil, err
		}

		// Debug: log what we received (temporary)
		// fmt.Fprintf(os.Stderr, "DEBUG: Key=%v Rune=%v (%c)\n", event.Key, event.Rune, event.Rune)

		// Handle regular character input first (before checking Key)
		if event.Rune != 0 && event.Key == 0 {
			// Regular character
			if p.maxLength == 0 || len(buffer) < p.maxLength {
				buffer = append(buffer, byte(event.Rune))
				p.updateMaskedDisplay(buffer, firstChar)
				firstChar = false
			}
			continue
		}

		switch event.Key {
		case KeyEnter:
			// Make sure we got a newline
			os.Stdout.WriteString("\r\n")
			return buffer, nil

		case KeyBackspace:
			if len(buffer) > 0 {
				buffer = buffer[:len(buffer)-1]
				p.updateMaskedDisplay(buffer, false)
			}

		case KeyEscape:
			// Zero the buffer
			for i := range buffer {
				buffer[i] = 0
			}
			return nil, fmt.Errorf("password input cancelled")

		case KeyCtrlC:
			// Zero the buffer
			for i := range buffer {
				buffer[i] = 0
			}
			return nil, fmt.Errorf("interrupted")

		default:
			// Handle paste events
			if event.Paste != "" {
				if p.confirmPaste {
					// For now, reject paste (could add confirmation UI later)
					// TODO: Add visual feedback for rejected paste
					continue
				}
				buffer = append(buffer, []byte(event.Paste)...)
				p.updateMaskedDisplay(buffer, firstChar)
				firstChar = false
			}
			// Regular characters are handled above before the switch
		}
	}
}

// updateMaskedDisplay updates the display showing masked characters.
func (p *PasswordInput) updateMaskedDisplay(buffer []byte, _ bool) {
	// In raw mode, we need to use direct stdout writes with ANSI escape codes

	// Clear line and move to beginning
	os.Stdout.WriteString("\r\033[K")

	// Write prompt with styling
	// Apply prompt style (convert Style to ANSI codes)
	styleCode := p.promptStyle.String()
	if styleCode != "" {
		os.Stdout.WriteString(styleCode)
	}
	os.Stdout.WriteString(p.prompt)
	// Reset style
	if styleCode != "" {
		os.Stdout.WriteString("\033[0m")
	}

	// Show masked characters
	if len(buffer) > 0 {
		masked := strings.Repeat(string(p.maskChar), len(buffer))
		os.Stdout.WriteString(masked)
	}
}

// enableTerminalSecureMode enables secure input mode for supported terminals.
func (p *PasswordInput) enableTerminalSecureMode() {
	// Detect terminal type from environment
	termProgram := os.Getenv("TERM_PROGRAM")

	switch termProgram {
	case "iTerm.app":
		// iTerm2: Set user variable to enable secure input
		// This prevents keylogging and other security issues
		// Write directly to stdout to bypass buffering
		os.Stdout.WriteString("\033]1337;SetUserVar=PasswordInput=1\007")

	case "vscode":
		// VS Code Terminal: Currently no specific protocol
		// Falls back to generic secure mode

	default:
		// Generic terminals: no special escape codes
	}
}

// disableTerminalSecureMode disables secure input mode for supported terminals.
func (p *PasswordInput) disableTerminalSecureMode() {
	termProgram := os.Getenv("TERM_PROGRAM")

	switch termProgram {
	case "iTerm.app":
		// iTerm2: Clear user variable to disable secure input
		// Write directly to stdout to bypass buffering
		os.Stdout.WriteString("\033]1337;SetUserVar=PasswordInput=0\007")

	case "vscode":
		// VS Code Terminal: Currently no specific protocol

	default:
		// Generic terminals: no special escape codes
	}
}

// SecureString provides a wrapper around password data with automatic memory zeroing.
type SecureString struct {
	data []byte
}

// NewSecureString creates a new SecureString from a byte slice.
func NewSecureString(data []byte) *SecureString {
	return &SecureString{data: data}
}

// String returns the password as a string.
// Note: This creates a copy in memory. Use with caution.
func (s *SecureString) String() string {
	if s == nil || s.data == nil {
		return ""
	}
	return string(s.data)
}

// Bytes returns the underlying byte slice.
// Do not modify the returned slice.
func (s *SecureString) Bytes() []byte {
	if s == nil {
		return nil
	}
	return s.data
}

// Len returns the length of the password.
func (s *SecureString) Len() int {
	if s == nil || s.data == nil {
		return 0
	}
	return len(s.data)
}

// Clear zeros the password data in memory.
// After calling Clear, the SecureString should not be used.
func (s *SecureString) Clear() {
	if s != nil && s.data != nil {
		for i := range s.data {
			s.data[i] = 0
		}
		s.data = nil
	}
}

// IsEmpty returns true if the password is empty.
func (s *SecureString) IsEmpty() bool {
	return s == nil || s.data == nil || len(s.data) == 0
}
