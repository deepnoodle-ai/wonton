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
// overwriting any existing values.
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
// Does not modify the environment.
func ReadEnvFile(filename string) (map[string]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseEnvReader(f)
}

// ParseEnvReader parses .env format from a reader.
func ParseEnvReader(r io.Reader) (map[string]string, error) {
	result := make(map[string]string)
	scanner := bufio.NewScanner(r)
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

// ParseEnvString parses a .env format string.
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

	// Handle quoted values
	value = unquote(value)

	// Remove inline comments (only for unquoted values)
	if !strings.HasPrefix(value, "'") && !strings.HasPrefix(value, "\"") {
		if idx := strings.Index(value, " #"); idx != -1 {
			value = strings.TrimSpace(value[:idx])
		}
	}

	return key, value, nil
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
func WriteEnvFile(envMap map[string]string, filename string) error {
	f, err := os.Create(filename)
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
