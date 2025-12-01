# Secure Password Input

This document describes Gooey's secure password input system, which provides enhanced security features for password collection in terminal applications.

## Overview

Gooey provides a `PasswordInput` component that implements security best practices for password input:

- **Terminal Secure Mode**: Sends escape codes to iTerm2 to enable secure input mode, preventing keylogging
- **Memory Security**: Stores passwords in byte slices with automatic zeroing after use
- **Visual Feedback Options**: Support for no-echo or masked character display
- **Clipboard Control**: Option to disable clipboard access during password input
- **Paste Confirmation**: Optional confirmation before accepting pasted content
- **Length Validation**: Configurable maximum password length

## Quick Start

### Basic Usage (Default: Masked Characters)

```go
terminal, _ := tui.NewTerminal()
defer terminal.Reset()

pwdInput := tui.NewPasswordInput(terminal)
pwdInput.WithPrompt("Password: ", tui.NewStyle().WithForeground(tui.ColorYellow))

password, err := pwdInput.Read()
if err != nil {
    return err
}
defer password.Clear() // Important: zero memory when done

// Use password
validatePassword(password.String())
```

By default, `PasswordInput` shows masked characters (asterisks) as you type, providing visual feedback while maintaining security.

### No Echo Mode

To disable visual feedback entirely (traditional no-echo password input):

```go
pwdInput := tui.NewPasswordInput(terminal)
pwdInput.ShowCharacters(false) // Disable visual feedback (no echo)
pwdInput.WithPrompt("Password: ", tui.NewStyle())

password, err := pwdInput.Read()
if err != nil {
    return err
}
defer password.Clear()
```

### Custom Mask Character

```go
pwdInput := tui.NewPasswordInput(terminal)
pwdInput.ShowCharacters(true)
pwdInput.WithMaskChar('•') // Use bullet points instead of asterisks

password, err := pwdInput.Read()
defer password.Clear()
```

## Configuration Options

### Prompt and Style

```go
pwdInput.WithPrompt("Enter your password: ",
    tui.NewStyle().WithForeground(tui.ColorRed).WithBold())
```

### Placeholder Text

```go
// Only shown in no-echo mode
pwdInput.WithPlaceholder("(type to begin)")
```

### Maximum Length

```go
pwdInput.WithMaxLength(64) // Enforce max 64 characters
```

### Secure Mode

```go
// Enable/disable terminal secure input mode (default: enabled)
pwdInput.EnableSecureMode(true)
```

### Clipboard Control

```go
// Disable clipboard during password input (default: enabled)
pwdInput.DisableClipboard(true)
```

### Paste Confirmation

```go
// Require confirmation before accepting pasted content (default: enabled)
pwdInput.ConfirmPaste(true)
```

## SecureString API

The `SecureString` type provides safe handling of password data:

```go
type SecureString struct {
    // ... internal fields
}

// Methods
func (s *SecureString) String() string          // Convert to string (creates copy)
func (s *SecureString) Bytes() []byte           // Get byte slice (no copy)
func (s *SecureString) Len() int                // Get length
func (s *SecureString) Clear()                  // Zero memory
func (s *SecureString) IsEmpty() bool           // Check if empty
```

### Important: Memory Safety

Always call `Clear()` when done with the password to zero the memory:

```go
password, err := pwdInput.Read()
if err != nil {
    return err
}
defer password.Clear() // Ensures memory is zeroed

// Use password...
authenticateUser(password.Bytes())
```

## Terminal-Specific Features

### iTerm2 Secure Input Mode

When running in iTerm2, Gooey automatically enables secure input mode:

```
\033]1337;SetUserVar=PasswordInput=1\007
```

This prevents:
- Keylogging attacks
- System-wide keyboard monitoring
- Third-party apps from reading keystrokes

The mode is automatically disabled after password input:

```
\033]1337;SetUserVar=PasswordInput=0\007
```

### VS Code Terminal

VS Code terminal is detected but currently has no specific protocol. Falls back to generic secure mode.

### Generic Terminals

For other terminals, the secure input mode simply uses standard no-echo input or masked characters.

## Terminal Detection

Gooey detects the terminal type using the `TERM_PROGRAM` environment variable:

| Value | Terminal | Support |
|-------|----------|---------|
| `iTerm.app` | iTerm2 | Secure input mode enabled |
| `vscode` | VS Code | Detected, generic mode |
| Other | Generic | Standard no-echo input |

## Security Best Practices

### 1. Always Clear Passwords

```go
password, _ := pwdInput.Read()
defer password.Clear() // Do this immediately after Read()
```

### 2. Use Bytes() Instead of String()

```go
// Good: No string copy in memory
passwordBytes := password.Bytes()
hashPassword(passwordBytes)

// Less secure: Creates string copy in memory
passwordStr := password.String()
```

### 3. Don't Store Passwords Long-Term

```go
// Bad: Password stays in memory
type User struct {
    Password *tui.SecureString
}

// Good: Hash immediately, clear original
password, _ := pwdInput.Read()
hashedPassword := hashPassword(password.Bytes())
password.Clear()
```

### 4. Enable All Security Features

```go
pwdInput := tui.NewPasswordInput(terminal)
pwdInput.EnableSecureMode(true)      // iTerm2 protection
pwdInput.DisableClipboard(true)      // Prevent accidental paste
pwdInput.ConfirmPaste(true)          // Require confirmation
pwdInput.WithMaxLength(128)          // Prevent buffer attacks
```

## Complete Example

```go
package main

import (
    "fmt"
    "github.com/deepnoodle-ai/gooey/tui"
)

func main() {
    terminal, err := tui.NewTerminal()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    defer terminal.Restore()

    // Create password input with all security features
    pwdInput := tui.NewPasswordInput(terminal)
    pwdInput.WithPrompt("Password: ", tui.NewStyle().WithForeground(tui.ColorYellow))
    pwdInput.ShowCharacters(true)        // Visual feedback
    pwdInput.WithMaskChar('•')           // Custom mask
    pwdInput.WithMaxLength(64)           // Length limit
    pwdInput.EnableSecureMode(true)      // iTerm2 protection
    pwdInput.DisableClipboard(true)      // No clipboard
    pwdInput.ConfirmPaste(true)          // Confirm pastes

    password, err := pwdInput.Read()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    defer password.Clear() // Zero memory

    // Use password
    if authenticateUser(password.Bytes()) {
        fmt.Println("✓ Authentication successful")
    } else {
        fmt.Println("✗ Authentication failed")
    }
}

func authenticateUser(password []byte) bool {
    // Your authentication logic here
    return true
}
```

## Comparison with Standard Input

### Standard Input.ReadPassword()

```go
input := tui.NewInput(terminal)
password, err := input.ReadPassword()
// Returns: string (not zeroed)
```

**Features:**
- No-echo input
- Simple API
- Returns string

**Limitations:**
- No terminal secure mode
- String stays in memory
- No visual feedback option
- No paste control

### Enhanced PasswordInput.Read()

```go
pwdInput := tui.NewPasswordInput(terminal)
password, err := pwdInput.Read()
defer password.Clear()
// Returns: *SecureString (zeroed on Clear)
```

**Features:**
- Terminal secure mode (iTerm2)
- Memory zeroing
- Masked character option
- Paste confirmation
- Max length enforcement
- Clipboard control

**Use when:**
- Handling sensitive passwords
- Need extra security
- Running in iTerm2
- Want visual feedback

## Keyboard Controls

When using masked character mode (`ShowCharacters(true)`):

| Key | Action |
|-----|--------|
| Enter | Submit password |
| Backspace | Delete previous character |
| Escape | Cancel input (zeros memory) |
| Ctrl+C | Interrupt (zeros memory) |
| Paste | Triggers confirmation (if enabled) |

## Error Handling

```go
password, err := pwdInput.Read()
if err != nil {
    switch err.Error() {
    case "password input cancelled":
        // User pressed Escape
    case "interrupted":
        // User pressed Ctrl+C
    default:
        if strings.Contains(err.Error(), "maximum length") {
            // Password too long
        }
    }
    return err
}
defer password.Clear()
```

## Advanced Usage

### Password Strength Indicator

```go
password, _ := pwdInput.Read()
defer password.Clear()

strength := calculatePasswordStrength(password.Bytes())
if strength < 3 {
    fmt.Println("⚠ Weak password")
} else if strength < 4 {
    fmt.Println("✓ Good password")
} else {
    fmt.Println("✓✓ Strong password")
}
```

### Confirmation Prompt

```go
func readPasswordWithConfirmation(terminal *tui.Terminal) (*tui.SecureString, error) {
    pwdInput := tui.NewPasswordInput(terminal)
    pwdInput.WithPrompt("New password: ", tui.NewStyle())
    pwdInput.ShowCharacters(true)

    password1, err := pwdInput.Read()
    if err != nil {
        return nil, err
    }
    defer password1.Clear()

    pwdInput.WithPrompt("Confirm password: ", tui.NewStyle())
    password2, err := pwdInput.Read()
    if err != nil {
        return nil, err
    }

    if string(password1.Bytes()) != string(password2.Bytes()) {
        password2.Clear()
        return nil, fmt.Errorf("passwords do not match")
    }

    // Return password2, clear password1
    return password2, nil
}
```

## Testing

Since password input requires terminal interaction, testing can be challenging. For unit tests, use the standard `Input.ReadPassword()` method or mock the terminal.

For integration tests, you can simulate user input:

```go
// This is complex - see examples/password_demo for interactive testing
```

## Future Enhancements

Potential future additions (see `documentation/world_class_features_analysis.md`):

- [ ] Password strength meter visualization
- [ ] Show/hide toggle (eye icon)
- [ ] Timeout-based auto-hide
- [ ] Integration with system password managers
- [ ] Additional terminal protocols (WezTerm, Kitty)

## Related Documentation

- [Input Guide](INPUT_GUIDE.md) - General input handling
- [World-Class Features Analysis](world_class_features_analysis.md) - Feature roadmap
- [Security Considerations](../SECURITY.md) - If available

## References

- [iTerm2 Proprietary Escape Codes](https://iterm2.com/documentation-escape-codes.html)
- [Terminal Input Security](https://en.wikipedia.org/wiki/Terminal_emulator)
- [OWASP Password Storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
