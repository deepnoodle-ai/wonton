package terminal

import (
	"regexp"
)

// Common patterns for secrets that should be redacted. When a pattern has a
// capture group, the group is the key and delimiter that precede the secret
// (such as "password=" or "Bearer "). It is kept for context and only the rest
// of the match is redacted.
var secretPatterns = []*regexp.Regexp{
	// Password fields
	regexp.MustCompile(`(?i)(password[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(passwd[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(pwd[:\s=]+)\S+`),

	// API keys and tokens
	regexp.MustCompile(`(?i)(api[_-]?key[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(api[_-]?secret[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(token[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(access[_-]?token[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(auth[_-]?token[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(bearer\s+)\S+`),

	// Common secret environment variables
	regexp.MustCompile(`(?i)(secret[:\s=]+)\S+`),
	regexp.MustCompile(`(?i)(private[_-]?key[:\s=]+)\S+`),

	// Long hexadecimal strings (likely tokens/keys)
	regexp.MustCompile(`\b[a-fA-F0-9]{32,}\b`),

	// JWT tokens (xxx.yyy.zzz format)
	regexp.MustCompile(`\beyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\b`),

	// AWS keys
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),

	// GitHub tokens
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{36,}`),

	// Generic secret-like strings (base64-ish)
	regexp.MustCompile(`(?i)((?:secret|key|token|password)[:\s=]+)[a-zA-Z0-9+/]{20,}={0,2}`),
}

// redactSecretPatterns applies pattern matching to redact secrets. The key
// is found by the pattern's capture group, not by splitting the match on ':'
// or '=': the secret itself can contain those characters (base64 padding,
// URLs with credentials), and splitting there would leak part of it.
func redactSecretPatterns(data string) string {
	result := data

	for _, pattern := range secretPatterns {
		if pattern.NumSubexp() > 0 {
			result = pattern.ReplaceAllString(result, "${1}[REDACTED]")
		} else {
			result = pattern.ReplaceAllLiteralString(result, "[REDACTED]")
		}
	}

	return result
}

// RedactCredentials can be called manually to test redaction
func RedactCredentials(input string) string {
	return redactSecretPatterns(input)
}
