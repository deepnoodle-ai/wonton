package terminal

import (
	"regexp"
	"strings"
)

// Common patterns for secrets that should be redacted
var secretPatterns = []*regexp.Regexp{
	// Password fields
	regexp.MustCompile(`(?i)password[:\s=]+\S+`),
	regexp.MustCompile(`(?i)passwd[:\s=]+\S+`),
	regexp.MustCompile(`(?i)pwd[:\s=]+\S+`),

	// API keys and tokens
	regexp.MustCompile(`(?i)api[_-]?key[:\s=]+\S+`),
	regexp.MustCompile(`(?i)api[_-]?secret[:\s=]+\S+`),
	regexp.MustCompile(`(?i)token[:\s=]+\S+`),
	regexp.MustCompile(`(?i)access[_-]?token[:\s=]+\S+`),
	regexp.MustCompile(`(?i)auth[_-]?token[:\s=]+\S+`),
	regexp.MustCompile(`(?i)bearer\s+\S+`),

	// Common secret environment variables
	regexp.MustCompile(`(?i)secret[:\s=]+\S+`),
	regexp.MustCompile(`(?i)private[_-]?key[:\s=]+\S+`),

	// Long hexadecimal strings (likely tokens/keys)
	regexp.MustCompile(`\b[a-fA-F0-9]{32,}\b`),

	// JWT tokens (xxx.yyy.zzz format)
	regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\b`),

	// AWS keys
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),

	// GitHub tokens
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{36,}`),

	// Generic secret-like strings (base64-ish)
	regexp.MustCompile(`(?i)(secret|key|token|password)[:\s=]+[a-zA-Z0-9+/]{20,}={0,2}`),
}

// redactSecretPatterns applies pattern matching to redact secrets
func redactSecretPatterns(data string) string {
	result := data

	for _, pattern := range secretPatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Try to preserve the key name for context
			parts := strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + ": [REDACTED]"
			}
			parts = strings.SplitN(match, "=", 2)
			if len(parts) == 2 {
				return parts[0] + "=[REDACTED]"
			}
			parts = strings.Fields(match)
			if len(parts) >= 2 {
				return parts[0] + " [REDACTED]"
			}
			return "[REDACTED]"
		})
	}

	return result
}

// RedactCredentials can be called manually to test redaction
func RedactCredentials(input string) string {
	return redactSecretPatterns(input)
}
