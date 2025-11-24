# Secure Password Input Demo

This example demonstrates Gooey's secure password input capabilities.

## Features Demonstrated

1. **Standard Secure Password** (no echo)
   - Traditional password input with no visual feedback
   - Characters are not displayed as you type

2. **Masked Characters**
   - Password displayed as asterisks (*)
   - Provides visual feedback while maintaining security

3. **Custom Mask Character**
   - Use bullet points (•) or any character
   - Customizable visual appearance

4. **Max Length Enforcement**
   - Limit password length (e.g., 16 characters)
   - Beep when limit reached

5. **Placeholder Text**
   - Show hint text before typing begins
   - Only visible in no-echo mode

## Security Features

### Terminal-Specific Protection

- **iTerm2**: Automatically enables secure input mode
  - Prevents system-wide keylogging
  - Disables third-party keyboard monitoring
  - Sends escape code: `\033]1337;SetUserVar=PasswordInput=1\007`

- **VS Code Terminal**: Detected but uses generic mode

- **Other Terminals**: Standard secure input with no-echo

### Memory Security

- Passwords stored in `[]byte` not `string`
- Automatic memory zeroing via `defer password.Clear()`
- `SecureString` wrapper prevents accidental exposure

### Clipboard Protection

- Optional clipboard disable during input
- Paste confirmation available
- Prevents accidental paste of sensitive data

## Running the Demo

```bash
go run examples/password_demo/main.go
```

## Terminal Detection

The demo automatically detects your terminal and shows which security features are active:

```
Detected terminal: iTerm.app
✓ iTerm2 secure input mode will be enabled
```

## Code Example

```go
// Create secure password input
pwdInput := gooey.NewPasswordInput(terminal)
pwdInput.WithPrompt("Enter password: ", gooey.NewStyle().WithForeground(gooey.ColorYellow))
pwdInput.ShowCharacters(true)        // Show asterisks
pwdInput.WithMaskChar('•')           // Use bullets
pwdInput.WithMaxLength(64)           // Limit length
pwdInput.EnableSecureMode(true)      // iTerm2 protection
pwdInput.DisableClipboard(true)      // No clipboard
pwdInput.ConfirmPaste(true)          // Confirm pastes

// Read password
password, err := pwdInput.Read()
if err != nil {
    return err
}
defer password.Clear() // Zero memory when done

// Use password
authenticate(password.Bytes())
```

## Best Practices

1. **Always Clear Passwords**
   ```go
   password, _ := pwdInput.Read()
   defer password.Clear() // Do this immediately
   ```

2. **Use Bytes, Not Strings**
   ```go
   // Good: No string copy
   hashPassword(password.Bytes())

   // Less secure: Creates string in memory
   passwordStr := password.String()
   ```

3. **Don't Store Passwords**
   ```go
   // Bad
   type User struct {
       Password *gooey.SecureString
   }

   // Good
   password, _ := pwdInput.Read()
   hashedPassword := hashPassword(password.Bytes())
   password.Clear()
   ```

## Documentation

See `documentation/password_input.md` for complete API documentation.

## Related Examples

- `examples/input_forms/` - Form with password input
- `examples/input_api_demo/` - General input API demo
