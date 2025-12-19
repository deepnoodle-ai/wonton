package terminal

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestRedactCredentials_Passwords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "password with colon",
			input:    "password: mysecretpass123",
			contains: "[REDACTED]",
			excludes: "mysecretpass123",
		},
		{
			name:     "password with equals",
			input:    "password=mysecretpass123",
			contains: "[REDACTED]",
			excludes: "mysecretpass123",
		},
		{
			name:     "PASSWORD uppercase",
			input:    "PASSWORD: SuperSecret",
			contains: "[REDACTED]",
			excludes: "SuperSecret",
		},
		{
			name:     "passwd variant",
			input:    "passwd=mypass",
			contains: "[REDACTED]",
			excludes: "mypass",
		},
		{
			name:     "pwd variant",
			input:    "pwd: shortpw",
			contains: "[REDACTED]",
			excludes: "shortpw",
		},
		{
			name:     "password with space separator",
			input:    "password secret123",
			contains: "[REDACTED]",
			excludes: "secret123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, tt.contains)
			assert.NotContains(t, result, tt.excludes)
		})
	}
}

func TestRedactCredentials_APIKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "api_key with underscore",
			input:    "api_key: ak_1234567890abcdef",
			contains: "[REDACTED]",
			excludes: "ak_1234567890abcdef",
		},
		{
			name:     "api-key with hyphen",
			input:    "api-key=sk-abc123def456",
			contains: "[REDACTED]",
			excludes: "sk-abc123def456",
		},
		{
			name:     "apikey no separator",
			input:    "apikey: myapikey123",
			contains: "[REDACTED]",
			excludes: "myapikey123",
		},
		{
			name:     "api_secret",
			input:    "api_secret=verysecretvalue",
			contains: "[REDACTED]",
			excludes: "verysecretvalue",
		},
		{
			name:     "API_KEY uppercase",
			input:    "API_KEY: UPPERCASE_KEY_VALUE",
			contains: "[REDACTED]",
			excludes: "UPPERCASE_KEY_VALUE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, tt.contains)
			assert.NotContains(t, result, tt.excludes)
		})
	}
}

func TestRedactCredentials_Tokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "generic token",
			input:    "token: abc123xyz789",
			contains: "[REDACTED]",
			excludes: "abc123xyz789",
		},
		{
			name:     "access_token",
			input:    "access_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			contains: "[REDACTED]",
			excludes: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:     "auth_token",
			input:    "auth_token: myauthtoken",
			contains: "[REDACTED]",
			excludes: "myauthtoken",
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature",
			contains: "[REDACTED]",
			excludes: "eyJhbGciOiJIUzI1NiJ9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, tt.contains)
			assert.NotContains(t, result, tt.excludes)
		})
	}
}

func TestRedactCredentials_AWSKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "AWS access key ID",
			input:    "aws_key: AKIAIOSFODNN7EXAMPLE",
			contains: "[REDACTED]",
			excludes: "AKIAIOSFODNN7EXAMPLE",
		},
		{
			name:     "AWS key in config",
			input:    "AWS_ACCESS_KEY_ID=AKIAI44QH8DHBEXAMPLE",
			contains: "[REDACTED]",
			excludes: "AKIAI44QH8DHBEXAMPLE",
		},
		{
			name:     "AWS key standalone",
			input:    "Found key: AKIAZ1234567890ABCDE in logs",
			contains: "[REDACTED]",
			excludes: "AKIAZ1234567890ABCDE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, tt.contains)
			assert.NotContains(t, result, tt.excludes)
		})
	}
}

func TestRedactCredentials_GitHubTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "GitHub personal access token ghp_",
			input:    "GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			contains: "[REDACTED]",
			excludes: "ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
		{
			name:     "GitHub OAuth access token gho_",
			input:    "token: gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			contains: "[REDACTED]",
			excludes: "gho_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
		{
			name:     "GitHub user-to-server token ghu_",
			input:    "Authorization: Bearer ghu_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			contains: "[REDACTED]",
			excludes: "ghu_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
		{
			name:     "GitHub server-to-server token ghs_",
			input:    "GH_TOKEN=ghs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			contains: "[REDACTED]",
			excludes: "ghs_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
		{
			name:     "GitHub refresh token ghr_",
			input:    "refresh_token: ghr_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			contains: "[REDACTED]",
			excludes: "ghr_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, tt.contains)
			assert.NotContains(t, result, tt.excludes)
		})
	}
}

func TestRedactCredentials_JWTTokens(t *testing.T) {
	// Real-looking JWT token (header.payload.signature format)
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "standalone JWT",
			input: jwt,
		},
		{
			name:  "JWT in authorization header",
			input: "Authorization: Bearer " + jwt,
		},
		{
			name:  "JWT in log line",
			input: "[INFO] User authenticated with token: " + jwt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, "[REDACTED]")
			// The full JWT should not appear in output
			assert.NotContains(t, result, jwt)
		})
	}
}

func TestRedactCredentials_HexStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "32-char hex string",
			input:    "secret: a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
			contains: "[REDACTED]",
			excludes: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
		},
		{
			name:     "64-char hex string (SHA256)",
			input:    "hash=a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			contains: "[REDACTED]",
			excludes: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name:     "uppercase hex",
			input:    "KEY=A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4",
			contains: "[REDACTED]",
			excludes: "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, tt.contains)
			assert.NotContains(t, result, tt.excludes)
		})
	}
}

func TestRedactCredentials_SecretsAndPrivateKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "generic secret",
			input:    "secret: mysupersecretsecret",
			contains: "[REDACTED]",
			excludes: "mysupersecretsecret",
		},
		{
			name:     "SECRET uppercase",
			input:    "SECRET=VERY_SECRET_VALUE",
			contains: "[REDACTED]",
			excludes: "VERY_SECRET_VALUE",
		},
		{
			name:     "private_key",
			input:    "private_key: -----BEGIN RSA PRIVATE KEY-----",
			contains: "[REDACTED]",
			excludes: "-----BEGIN RSA PRIVATE KEY-----",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, tt.contains)
			assert.NotContains(t, result, tt.excludes)
		})
	}
}

func TestRedactCredentials_NoFalsePositives(t *testing.T) {
	// These should NOT be redacted or should preserve important context
	tests := []struct {
		name   string
		input  string
		verify func(t *testing.T, result string)
	}{
		{
			name:  "regular text",
			input: "Hello, this is a normal message without secrets.",
			verify: func(t *testing.T, result string) {
				assert.Equal(t, "Hello, this is a normal message without secrets.", result)
			},
		},
		{
			name:  "short hex string (not long enough)",
			input: "commit abc123",
			verify: func(t *testing.T, result string) {
				assert.Equal(t, "commit abc123", result)
			},
		},
		{
			name:  "URL with normal query params",
			input: "https://example.com?page=1&sort=desc",
			verify: func(t *testing.T, result string) {
				assert.Equal(t, "https://example.com?page=1&sort=desc", result)
			},
		},
		{
			name:  "regular numbers",
			input: "User ID: 12345, Order: 67890",
			verify: func(t *testing.T, result string) {
				assert.Equal(t, "User ID: 12345, Order: 67890", result)
			},
		},
		{
			name:  "git commit hash (40 chars hex)",
			input: "commit a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0",
			verify: func(t *testing.T, result string) {
				// 40 char hex strings are redacted as they look like tokens
				assert.Contains(t, result, "[REDACTED]")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			tt.verify(t, result)
		})
	}
}

func TestRedactCredentials_PreservesContext(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldHave    []string
		shouldNotHave []string
	}{
		{
			name:          "preserves key name with colon",
			input:         "password: secret123",
			shouldHave:    []string{"password", "[REDACTED]"},
			shouldNotHave: []string{"secret123"},
		},
		{
			name:          "preserves key name with equals",
			input:         "API_KEY=sk_live_abc123",
			shouldHave:    []string{"API_KEY", "[REDACTED]"},
			shouldNotHave: []string{"sk_live_abc123"},
		},
		{
			name:          "preserves surrounding text",
			input:         "Error: Failed to authenticate. password: wrongpass. Please retry.",
			shouldHave:    []string{"Error:", "Failed to authenticate", "[REDACTED]", "Please retry"},
			shouldNotHave: []string{"wrongpass"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			for _, s := range tt.shouldHave {
				assert.Contains(t, result, s)
			}
			for _, s := range tt.shouldNotHave {
				assert.NotContains(t, result, s)
			}
		})
	}
}

func TestRedactCredentials_MultipleSecrets(t *testing.T) {
	input := `
DB_PASSWORD=secret123
API_KEY=sk_live_abcdefghijklmnop
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.sig
`
	result := RedactCredentials(input)

	// All secrets should be redacted
	assert.NotContains(t, result, "secret123")
	assert.NotContains(t, result, "sk_live_abcdefghijklmnop")
	assert.NotContains(t, result, "AKIAIOSFODNN7EXAMPLE")
	assert.NotContains(t, result, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")

	// Key names should be preserved
	assert.Contains(t, result, "DB_PASSWORD")
	assert.Contains(t, result, "API_KEY")

	// Should have multiple redacted markers
	count := strings.Count(result, "[REDACTED]")
	assert.True(t, count >= 4, "expected at least 4 redactions, got %d", count)
}

func TestRedactCredentials_EmptyAndWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   \t\n   ",
			want:  "   \t\n   ",
		},
		{
			name:  "newlines",
			input: "\n\n\n",
			want:  "\n\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRedactCredentials_Base64Secrets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "secret with base64 value",
			input:    "secret=SGVsbG8gV29ybGQhIFRoaXMgaXMgYSBsb25nIGJhc2U2NA==",
			contains: "[REDACTED]",
			excludes: "SGVsbG8gV29ybGQhIFRoaXMgaXMgYSBsb25nIGJhc2U2NA==",
		},
		{
			name:     "key with base64 value",
			input:    "key: YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=",
			contains: "[REDACTED]",
			excludes: "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXo=",
		},
		{
			name:     "token with base64 value",
			input:    "token=VGhpcyBpcyBhIHNlY3JldCB0b2tlbiB2YWx1ZQ==",
			contains: "[REDACTED]",
			excludes: "VGhpcyBpcyBhIHNlY3JldCB0b2tlbiB2YWx1ZQ==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactCredentials(tt.input)
			assert.Contains(t, result, tt.contains)
			assert.NotContains(t, result, tt.excludes)
		})
	}
}

func TestRedactCredentials_Performance(t *testing.T) {
	// Test with a large input containing many potential patterns
	var builder strings.Builder
	for i := 0; i < 1000; i++ {
		builder.WriteString("Line ")
		builder.WriteString(string(rune('0' + i%10)))
		builder.WriteString(": Some log output with password: secret")
		builder.WriteString(string(rune('0' + i%10)))
		builder.WriteString(" and api_key=key")
		builder.WriteString(string(rune('0' + i%10)))
		builder.WriteString("\n")
	}

	input := builder.String()
	result := RedactCredentials(input)

	// Should have redacted all secrets
	assert.NotContains(t, result, "secret0")
	assert.NotContains(t, result, "key0")

	// Should contain redaction markers
	assert.Contains(t, result, "[REDACTED]")
}

func TestRedactSecretPatterns_Internal(t *testing.T) {
	// Test the internal function directly
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "password pattern",
			input: "password: test",
		},
		{
			name:  "api key pattern",
			input: "api_key=test",
		},
		{
			name:  "bearer token",
			input: "bearer abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactSecretPatterns(tt.input)
			assert.Contains(t, result, "[REDACTED]")
		})
	}
}
