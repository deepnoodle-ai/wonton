package tui

import (
	"testing"

	"github.com/deepnoodle-ai/gooey/require"
)

func TestNewPasswordInput(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	require.NotNil(t, pwdInput)
	require.Equal(t, terminal, pwdInput.terminal)
	require.Equal(t, "Password: ", pwdInput.prompt)
	require.True(t, pwdInput.showCharacters) // Default is true for visual feedback
	require.Equal(t, '*', pwdInput.maskChar)
	require.True(t, pwdInput.enableSecureMode)
	require.True(t, pwdInput.disableClipboard)
	require.True(t, pwdInput.confirmPaste)
}

func TestPasswordInput_WithPrompt(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	style := NewStyle().WithForeground(ColorRed)
	pwdInput.WithPrompt("Enter PIN: ", style)

	require.Equal(t, "Enter PIN: ", pwdInput.prompt)
	require.Equal(t, style, pwdInput.promptStyle)
}

func TestPasswordInput_WithPlaceholder(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.WithPlaceholder("(hidden)")

	require.Equal(t, "(hidden)", pwdInput.placeholder)
}

func TestPasswordInput_WithMaxLength(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.WithMaxLength(64)

	require.Equal(t, 64, pwdInput.maxLength)
}

func TestPasswordInput_WithMaskChar(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.WithMaskChar('•')

	require.Equal(t, '•', pwdInput.maskChar)
}

func TestPasswordInput_ShowCharacters(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.ShowCharacters(true)
	require.True(t, pwdInput.showCharacters)

	pwdInput.ShowCharacters(false)
	require.False(t, pwdInput.showCharacters)
}

func TestPasswordInput_EnableSecureMode(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.EnableSecureMode(false)
	require.False(t, pwdInput.enableSecureMode)

	pwdInput.EnableSecureMode(true)
	require.True(t, pwdInput.enableSecureMode)
}

func TestPasswordInput_DisableClipboard(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.DisableClipboard(false)
	require.False(t, pwdInput.disableClipboard)

	pwdInput.DisableClipboard(true)
	require.True(t, pwdInput.disableClipboard)
}

func TestPasswordInput_ConfirmPaste(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.ConfirmPaste(false)
	require.False(t, pwdInput.confirmPaste)

	pwdInput.ConfirmPaste(true)
	require.True(t, pwdInput.confirmPaste)
}

func TestPasswordInput_ChainedConfiguration(t *testing.T) {
	terminal := &Terminal{}

	pwdInput := NewPasswordInput(terminal).
		WithPrompt("PIN: ", NewStyle()).
		WithPlaceholder("1234").
		WithMaxLength(4).
		WithMaskChar('•').
		ShowCharacters(true).
		EnableSecureMode(true).
		DisableClipboard(true).
		ConfirmPaste(false)

	require.Equal(t, "PIN: ", pwdInput.prompt)
	require.Equal(t, "1234", pwdInput.placeholder)
	require.Equal(t, 4, pwdInput.maxLength)
	require.Equal(t, '•', pwdInput.maskChar)
	require.True(t, pwdInput.showCharacters)
	require.True(t, pwdInput.enableSecureMode)
	require.True(t, pwdInput.disableClipboard)
	require.False(t, pwdInput.confirmPaste)
}

// SecureString tests

func TestNewSecureString(t *testing.T) {
	data := []byte("password123")
	ss := NewSecureString(data)

	require.NotNil(t, ss)
	require.Equal(t, data, ss.data)
}

func TestSecureString_String(t *testing.T) {
	data := []byte("password123")
	ss := NewSecureString(data)

	require.Equal(t, "password123", ss.String())

	// Test nil
	var nilSS *SecureString
	require.Equal(t, "", nilSS.String())

	// Test nil data
	ss2 := &SecureString{data: nil}
	require.Equal(t, "", ss2.String())
}

func TestSecureString_Bytes(t *testing.T) {
	data := []byte("password123")
	ss := NewSecureString(data)

	require.Equal(t, data, ss.Bytes())

	// Test nil
	var nilSS *SecureString
	require.Nil(t, nilSS.Bytes())
}

func TestSecureString_Len(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{"empty", []byte(""), 0},
		{"short", []byte("abc"), 3},
		{"long", []byte("verylongpassword"), 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := NewSecureString(tt.data)
			require.Equal(t, tt.expected, ss.Len())
		})
	}

	// Test nil
	var nilSS *SecureString
	require.Equal(t, 0, nilSS.Len())

	// Test nil data
	ss := &SecureString{data: nil}
	require.Equal(t, 0, ss.Len())
}

func TestSecureString_Clear(t *testing.T) {
	data := []byte("password123")
	ss := NewSecureString(data)

	require.Equal(t, "password123", ss.String())
	require.Equal(t, 11, ss.Len())

	ss.Clear()

	// Data should be zeroed
	for _, b := range data {
		require.Equal(t, byte(0), b)
	}

	// SecureString should be empty
	require.Nil(t, ss.data)
	require.Equal(t, 0, ss.Len())
	require.True(t, ss.IsEmpty())

	// Calling Clear again should be safe
	ss.Clear()
	require.Nil(t, ss.data)

	// Test nil SecureString
	var nilSS *SecureString
	nilSS.Clear() // Should not panic
}

func TestSecureString_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		ss       *SecureString
		expected bool
	}{
		{"nil", nil, true},
		{"nil data", &SecureString{data: nil}, true},
		{"empty data", &SecureString{data: []byte("")}, true},
		{"with data", &SecureString{data: []byte("password")}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.ss.IsEmpty())
		})
	}
}

func TestSecureString_MemoryZeroing(t *testing.T) {
	// Create password
	password := []byte("sensitive_data_12345")
	original := make([]byte, len(password))
	copy(original, password)

	ss := NewSecureString(password)

	// Verify password is stored correctly
	require.Equal(t, string(original), ss.String())

	// Clear the password
	ss.Clear()

	// Verify original buffer was zeroed
	for i, b := range password {
		require.Equal(t, byte(0), b, "byte at index %d not zeroed", i)
	}

	// Verify SecureString is empty
	require.True(t, ss.IsEmpty())
	require.Equal(t, "", ss.String())
}

func TestSecureString_DeferredClear(t *testing.T) {
	func() {
		data := []byte("password123")
		ss := NewSecureString(data)
		defer ss.Clear()

		// Use password
		require.Equal(t, "password123", ss.String())
		require.Equal(t, 11, ss.Len())
	}()

	// After function returns, Clear should have been called
	// (We can't directly test this, but it validates the pattern)
}

func TestSecureString_MultipleReferences(t *testing.T) {
	// Test that clearing one SecureString doesn't affect another
	data1 := []byte("password1")
	data2 := []byte("password2")

	ss1 := NewSecureString(data1)
	ss2 := NewSecureString(data2)

	// Clear first password
	ss1.Clear()

	// Verify first is cleared
	require.True(t, ss1.IsEmpty())
	for _, b := range data1 {
		require.Equal(t, byte(0), b)
	}

	// Verify second is unchanged
	require.False(t, ss2.IsEmpty())
	require.Equal(t, "password2", ss2.String())
}

// Integration-style tests (require user interaction - commented out by default)

/*
func TestPasswordInput_Read_Interactive(t *testing.T) {
	// This test requires user interaction
	// Uncomment to test interactively

	terminal, err := NewTerminal()
	require.NoError(t, err)
	defer terminal.Restore()

	pwdInput := NewPasswordInput(terminal)
	pwdInput.WithPrompt("Test Password: ", NewStyle())
	pwdInput.ShowCharacters(true)

	password, err := pwdInput.Read()
	require.NoError(t, err)
	defer password.Clear()

	t.Logf("Password length: %d", password.Len())
	require.Greater(t, password.Len(), 0)
}
*/

// Benchmark tests

func BenchmarkSecureString_String(b *testing.B) {
	data := []byte("password123")
	ss := NewSecureString(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ss.String()
	}
}

func BenchmarkSecureString_Bytes(b *testing.B) {
	data := []byte("password123")
	ss := NewSecureString(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ss.Bytes()
	}
}

func BenchmarkSecureString_Clear(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data := []byte("password123")
		ss := NewSecureString(data)
		ss.Clear()
	}
}

func BenchmarkNewSecureString(b *testing.B) {
	data := []byte("password123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewSecureString(data)
	}
}
