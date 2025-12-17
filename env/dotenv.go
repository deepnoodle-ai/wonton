// Package env provides .env file parsing and manipulation.
// This file contains functions for reading, parsing, and writing .env format files.
package env

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// LoadEnvFile loads environment variables from .env files into os.Environ().
// Variables that already exist in the environment are NOT overwritten.
// Use OverloadEnvFile to override existing values.
//
// If no filenames are provided, it defaults to loading ".env" from the current directory.
// Missing files return an error.
//
// Example:
//
//	err := env.LoadEnvFile(".env", ".env.local")
//	if err != nil {
//	    log.Fatal(err)
//	}
func LoadEnvFile(filenames ...string) error {
	if len(filenames) == 0 {
		filenames = []string{".env"}
	}

	for _, filename := range filenames {
		envMap, err := ReadEnvFile(filename)
		if err != nil {
			return err
		}

		for k, v := range envMap {
			if _, exists := os.LookupEnv(k); !exists {
				os.Setenv(k, v)
			}
		}
	}

	return nil
}

// OverloadEnvFile loads environment variables from .env files,
// overwriting any existing values in os.Environ().
//
// If no filenames are provided, it defaults to loading ".env" from the current directory.
// Missing files return an error.
//
// Example:
//
//	err := env.OverloadEnvFile(".env.production")
//	if err != nil {
//	    log.Fatal(err)
//	}
func OverloadEnvFile(filenames ...string) error {
	if len(filenames) == 0 {
		filenames = []string{".env"}
	}

	for _, filename := range filenames {
		envMap, err := ReadEnvFile(filename)
		if err != nil {
			return err
		}

		for k, v := range envMap {
			os.Setenv(k, v)
		}
	}

	return nil
}

// ReadEnvFile reads a .env file and returns a map of key-value pairs.
// Does not modify the environment. Returns an error if the file cannot be opened.
//
// Example:
//
//	envVars, err := env.ReadEnvFile(".env")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(envVars["DATABASE_URL"])
func ReadEnvFile(filename string) (map[string]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseEnvReader(f)
}

// ParseEnvReader parses .env format from a reader and returns a map of key-value pairs.
// Supports standard .env syntax including comments, quoted values, export statements,
// and both = and : separators.
//
// Example:
//
//	file, _ := os.Open("config.env")
//	defer file.Close()
//	envVars, err := env.ParseEnvReader(file)
func ParseEnvReader(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)
	// Increase buffer size to handle large values (e.g., base64-encoded certificates)
	// Default is 64KB; we allow up to 1MB per line
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments (# or //)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		// Parse line
		key, value, err := parseLine(line)
		if err != nil {
			// Skip malformed lines silently (matches godotenv behavior)
			continue
		}

		result[key] = value
	}

	return result, scanner.Err()
}

// ParseEnvString parses a .env format string and returns a map of key-value pairs.
// Useful for parsing inline .env configuration.
//
// Example:
//
//	envVars, err := env.ParseEnvString(`
//	    HOST=localhost
//	    PORT=8080
//	    DEBUG=true
//	`)
func ParseEnvString(s string) (map[string]string, error) {
	return ParseEnvReader(strings.NewReader(s))
}

// parseLine parses a single line from a .env file.
func parseLine(line string) (string, string, error) {
	// Remove 'export ' prefix if present
	line = strings.TrimPrefix(line, "export ")
	line = strings.TrimSpace(line)

	// Find the key-value separator (= or :)
	var key, value string
	if idx := strings.Index(line, "="); idx != -1 {
		key = strings.TrimSpace(line[:idx])
		value = strings.TrimSpace(line[idx+1:])
	} else if idx := strings.Index(line, ":"); idx != -1 {
		// YAML-style
		key = strings.TrimSpace(line[:idx])
		value = strings.TrimSpace(line[idx+1:])
	} else {
		return "", "", &ParseError{Err: errInvalidFormat}
	}

	// Validate key
	if key == "" {
		return "", "", &ParseError{Err: errEmptyKey}
	}

	// Handle quoted values: if value starts with a quote, extract the quoted portion
	// and ignore everything after the closing quote (including trailing comments)
	if len(value) > 0 && (value[0] == '"' || value[0] == '\'') {
		quote := value[0]
		// Find the closing quote, accounting for escapes in double-quoted strings
		endIdx := findClosingQuote(value, quote)
		if endIdx != -1 {
			// Extract just the quoted portion and unquote it
			value = unquote(value[:endIdx+1])
		}
		// If no closing quote found, unquote will handle it (returns as-is)
	} else {
		// Unquoted value: strip inline comments
		if idx := strings.Index(value, " #"); idx != -1 {
			value = strings.TrimSpace(value[:idx])
		}
	}

	return key, value, nil
}

// findClosingQuote finds the index of the closing quote in a quoted string.
// For double quotes, it handles escape sequences. Returns -1 if not found.
func findClosingQuote(s string, quote byte) int {
	if len(s) < 2 {
		return -1
	}
	for i := 1; i < len(s); i++ {
		if s[i] == quote {
			return i
		}
		// Skip escaped characters in double-quoted strings
		if quote == '"' && s[i] == '\\' && i+1 < len(s) {
			i++
		}
	}
	return -1
}

// unquote removes surrounding quotes and processes escape sequences.
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}

	// Check for matching quotes
	if (s[0] == '"' && s[len(s)-1] == '"') ||
		(s[0] == '\'' && s[len(s)-1] == '\'') {
		quote := s[0]
		s = s[1 : len(s)-1]

		// Process escape sequences for double quotes only
		if quote == '"' {
			s = processEscapes(s)
		}
	}

	return s
}

// processEscapes handles escape sequences in double-quoted strings.
func processEscapes(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			case '$':
				result.WriteByte('$')
			default:
				// Keep the backslash for unknown escapes
				result.WriteByte('\\')
				result.WriteByte(s[i+1])
			}
			i += 2
		} else {
			result.WriteByte(s[i])
			i++
		}
	}

	return result.String()
}

// WriteEnvFile writes a map of environment variables to a .env file.
// Values are automatically quoted when necessary (spaces, special characters, etc.).
// Escape sequences are applied to quoted values.
// Uses default file permissions (0666 before umask). For sensitive data,
// use WriteEnvFileWithPerm with restrictive permissions like 0600.
//
// Example:
//
//	envVars := map[string]string{
//	    "HOST": "localhost",
//	    "PORT": "8080",
//	    "MESSAGE": "Hello, World!",
//	}
//	err := env.WriteEnvFile(envVars, ".env.output")
func WriteEnvFile(envMap map[string]string, filename string) error {
	return WriteEnvFileWithPerm(envMap, filename, 0666)
}

// WriteEnvFileWithPerm writes a map of environment variables to a .env file
// with the specified file permissions. Use this for sensitive configuration
// files that should have restrictive permissions.
//
// Example:
//
//	envVars := map[string]string{
//	    "API_KEY": "secret123",
//	}
//	// Only owner can read/write
//	err := env.WriteEnvFileWithPerm(envVars, ".env.secrets", 0600)
func WriteEnvFileWithPerm(envMap map[string]string, filename string, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer f.Close()

	for k, v := range envMap {
		line := formatEnvLine(k, v)
		if _, err := f.WriteString(line + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// formatEnvLine formats a key-value pair for .env file output.
func formatEnvLine(key, value string) string {
	// Check if value needs quoting
	needsQuotes := strings.ContainsAny(value, " \t\n\"'\\#") ||
		value == "" ||
		strings.HasPrefix(value, " ") ||
		strings.HasSuffix(value, " ")

	if needsQuotes {
		// Escape special characters and quote
		escaped := strings.ReplaceAll(value, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		escaped = strings.ReplaceAll(escaped, "\n", "\\n")
		escaped = strings.ReplaceAll(escaped, "\t", "\\t")
		escaped = strings.ReplaceAll(escaped, "\r", "\\r")
		return key + "=\"" + escaped + "\""
	}

	return key + "=" + value
}

// sentinel errors
var (
	errInvalidFormat = &ParseError{Err: nil}
	errEmptyKey      = &ParseError{Err: nil}
)
