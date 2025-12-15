package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestNewPasswordInput(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	assert.NotNil(t, pwdInput)
	assert.Equal(t, terminal, pwdInput.terminal)
	assert.Equal(t, "Password: ", pwdInput.prompt)
	assert.True(t, pwdInput.showCharacters) // Default is true for visual feedback
	assert.Equal(t, '*', pwdInput.maskChar)
	assert.True(t, pwdInput.enableSecureMode)
	assert.True(t, pwdInput.disableClipboard)
	assert.True(t, pwdInput.confirmPaste)
}

func TestPasswordInput_WithPrompt(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	style := NewStyle().WithForeground(ColorRed)
	pwdInput.WithPrompt("Enter PIN: ", style)

	assert.Equal(t, "Enter PIN: ", pwdInput.prompt)
	assert.Equal(t, style, pwdInput.promptStyle)
}

func TestPasswordInput_WithPlaceholder(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.WithPlaceholder("(hidden)")

	assert.Equal(t, "(hidden)", pwdInput.placeholder)
}

func TestPasswordInput_WithMaxLength(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.WithMaxLength(64)

	assert.Equal(t, 64, pwdInput.maxLength)
}

func TestPasswordInput_WithMaskChar(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.WithMaskChar('•')

	assert.Equal(t, '•', pwdInput.maskChar)
}

func TestPasswordInput_ShowCharacters(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.ShowCharacters(true)
	assert.True(t, pwdInput.showCharacters)

	pwdInput.ShowCharacters(false)
	assert.False(t, pwdInput.showCharacters)
}

func TestPasswordInput_EnableSecureMode(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.EnableSecureMode(false)
	assert.False(t, pwdInput.enableSecureMode)

	pwdInput.EnableSecureMode(true)
	assert.True(t, pwdInput.enableSecureMode)
}

func TestPasswordInput_DisableClipboard(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.DisableClipboard(false)
	assert.False(t, pwdInput.disableClipboard)

	pwdInput.DisableClipboard(true)
	assert.True(t, pwdInput.disableClipboard)
}

func TestPasswordInput_ConfirmPaste(t *testing.T) {
	terminal := &Terminal{}
	pwdInput := NewPasswordInput(terminal)

	pwdInput.ConfirmPaste(false)
	assert.False(t, pwdInput.confirmPaste)

	pwdInput.ConfirmPaste(true)
	assert.True(t, pwdInput.confirmPaste)
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

	assert.Equal(t, "PIN: ", pwdInput.prompt)
	assert.Equal(t, "1234", pwdInput.placeholder)
	assert.Equal(t, 4, pwdInput.maxLength)
	assert.Equal(t, '•', pwdInput.maskChar)
	assert.True(t, pwdInput.showCharacters)
	assert.True(t, pwdInput.enableSecureMode)
	assert.True(t, pwdInput.disableClipboard)
	assert.False(t, pwdInput.confirmPaste)
}

// SecureString tests

func TestNewSecureString(t *testing.T) {
	data := []byte("password123")
	ss := NewSecureString(data)

	assert.NotNil(t, ss)
	assert.Equal(t, data, ss.data)
}

func TestSecureString_String(t *testing.T) {
	data := []byte("password123")
	ss := NewSecureString(data)

	assert.Equal(t, "password123", ss.String())

	// Test nil
	var nilSS *SecureString
	assert.Equal(t, "", nilSS.String())

	// Test nil data
	ss2 := &SecureString{data: nil}
	assert.Equal(t, "", ss2.String())
}

func TestSecureString_Bytes(t *testing.T) {
	data := []byte("password123")
	ss := NewSecureString(data)

	assert.Equal(t, data, ss.Bytes())

	// Test nil
	var nilSS *SecureString
	assert.Nil(t, nilSS.Bytes())
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
			assert.Equal(t, tt.expected, ss.Len())
		})
	}

	// Test nil
	var nilSS *SecureString
	assert.Equal(t, 0, nilSS.Len())

	// Test nil data
	ss := &SecureString{data: nil}
	assert.Equal(t, 0, ss.Len())
}

func TestSecureString_Clear(t *testing.T) {
	data := []byte("password123")
	ss := NewSecureString(data)

	assert.Equal(t, "password123", ss.String())
	assert.Equal(t, 11, ss.Len())

	ss.Clear()

	// Data should be zeroed
	for _, b := range data {
		assert.Equal(t, byte(0), b)
	}

	// SecureString should be empty
	assert.Nil(t, ss.data)
	assert.Equal(t, 0, ss.Len())
	assert.True(t, ss.IsEmpty())

	// Calling Clear again should be safe
	ss.Clear()
	assert.Nil(t, ss.data)

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
			assert.Equal(t, tt.expected, tt.ss.IsEmpty())
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
	assert.Equal(t, string(original), ss.String())

	// Clear the password
	ss.Clear()

	// Verify original buffer was zeroed
	for i, b := range password {
		assert.Equal(t, byte(0), b, "byte at index %d not zeroed", i)
	}

	// Verify SecureString is empty
	assert.True(t, ss.IsEmpty())
	assert.Equal(t, "", ss.String())
}

func TestSecureString_DeferredClear(t *testing.T) {
	func() {
		data := []byte("password123")
		ss := NewSecureString(data)
		defer ss.Clear()

		// Use password
		assert.Equal(t, "password123", ss.String())
		assert.Equal(t, 11, ss.Len())
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
	assert.True(t, ss1.IsEmpty())
	for _, b := range data1 {
		assert.Equal(t, byte(0), b)
	}

	// Verify second is unchanged
	assert.False(t, ss2.IsEmpty())
	assert.Equal(t, "password2", ss2.String())
}

// Integration-style tests (require user interaction - commented out by default)

/*
func TestPasswordInput_Read_Interactive(t *testing.T) {
	// This test requires user interaction
	// Uncomment to test interactively

	terminal, err := NewTerminal()
	assert.NoError(t, err)
	defer terminal.Restore()

	pwdInput := NewPasswordInput(terminal)
	pwdInput.WithPrompt("Test Password: ", NewStyle())
	pwdInput.ShowCharacters(true)

	password, err := pwdInput.Read()
	assert.NoError(t, err)
	defer password.Clear()

	t.Logf("Password length: %d", password.Len())
	assert.Greater(t, password.Len(), 0)
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
