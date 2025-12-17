package env

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

// ExampleParseEnvString demonstrates parsing .env format from a string.
func ExampleParseEnvString() {
	envData := `
# Database configuration
HOST=localhost
PORT=5432
DB_NAME=myapp

# API Keys
API_KEY="secret-key-here"
`

	envVars, err := ParseEnvString(envData)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("HOST:", envVars["HOST"])
	fmt.Println("PORT:", envVars["PORT"])
	fmt.Println("DB_NAME:", envVars["DB_NAME"])
	fmt.Println("API_KEY:", envVars["API_KEY"])
	// Output:
	// HOST: localhost
	// PORT: 5432
	// DB_NAME: myapp
	// API_KEY: secret-key-here
}

// ExampleWriteEnvFile demonstrates writing environment variables to a .env file.
func ExampleWriteEnvFile() {
	envVars := map[string]string{
		"HOST":    "localhost",
		"PORT":    "8080",
		"DEBUG":   "true",
		"MESSAGE": "Hello, World!",
	}

	// In a real application, you would write to an actual file
	// For this example, we'll just demonstrate the function signature
	_ = WriteEnvFile(envVars, "/tmp/example.env")

	fmt.Println("Environment variables written to file")
	// Output: Environment variables written to file
}

func TestReadEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	content := `
# This is a comment
HOST=localhost
PORT=8080

# Quoted values
MESSAGE="Hello, World!"
SINGLE='single quoted'

# With export
export DEBUG=true

# Empty value
EMPTY=

# Inline comment
VALUE=test # this is ignored

# Escape sequences
MULTILINE="line1\nline2"

# YAML-style
API_KEY: secret123
`

	assert.NoError(t, os.WriteFile(envFile, []byte(content), 0644))

	env, err := ReadEnvFile(envFile)
	assert.NoError(t, err)

	assert.Equal(t, "localhost", env["HOST"])
	assert.Equal(t, "8080", env["PORT"])
	assert.Equal(t, "Hello, World!", env["MESSAGE"])
	assert.Equal(t, "single quoted", env["SINGLE"])
	assert.Equal(t, "true", env["DEBUG"])
	assert.Equal(t, "", env["EMPTY"])
	assert.Equal(t, "test", env["VALUE"])
	assert.Equal(t, "line1\nline2", env["MULTILINE"])
	assert.Equal(t, "secret123", env["API_KEY"])
}

func TestParseEnvString(t *testing.T) {
	env, err := ParseEnvString(`
FOO=bar
BAZ=qux
`)
	assert.NoError(t, err)
	assert.Equal(t, "bar", env["FOO"])
	assert.Equal(t, "qux", env["BAZ"])
}

func TestLoadEnvFile(t *testing.T) {
	// Save current env
	originalValue, hadValue := os.LookupEnv("TEST_LOAD_VAR")
	defer func() {
		if hadValue {
			os.Setenv("TEST_LOAD_VAR", originalValue)
		} else {
			os.Unsetenv("TEST_LOAD_VAR")
		}
	}()

	// Create temp .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	assert.NoError(t, os.WriteFile(envFile, []byte("TEST_LOAD_VAR=from_file\n"), 0644))

	// Test: Load does NOT override existing
	os.Setenv("TEST_LOAD_VAR", "existing")
	assert.NoError(t, LoadEnvFile(envFile))
	assert.Equal(t, "existing", os.Getenv("TEST_LOAD_VAR"))

	// Test: Load sets non-existing
	os.Unsetenv("TEST_LOAD_VAR")
	assert.NoError(t, LoadEnvFile(envFile))
	assert.Equal(t, "from_file", os.Getenv("TEST_LOAD_VAR"))
}

func TestOverloadEnvFile(t *testing.T) {
	// Save current env
	originalValue, hadValue := os.LookupEnv("TEST_OVERLOAD_VAR")
	defer func() {
		if hadValue {
			os.Setenv("TEST_OVERLOAD_VAR", originalValue)
		} else {
			os.Unsetenv("TEST_OVERLOAD_VAR")
		}
	}()

	// Create temp .env file
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	assert.NoError(t, os.WriteFile(envFile, []byte("TEST_OVERLOAD_VAR=from_file\n"), 0644))

	// Test: Overload DOES override existing
	os.Setenv("TEST_OVERLOAD_VAR", "existing")
	assert.NoError(t, OverloadEnvFile(envFile))
	assert.Equal(t, "from_file", os.Getenv("TEST_OVERLOAD_VAR"))
}

func TestWriteEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "out.env")

	input := map[string]string{
		"SIMPLE": "value",
		"QUOTED": "has spaces",
		"ESCAPE": "line1\nline2",
	}

	assert.NoError(t, WriteEnvFile(input, outFile))

	// Read back
	env, err := ReadEnvFile(outFile)
	assert.NoError(t, err)
	assert.Equal(t, "value", env["SIMPLE"])
	assert.Equal(t, "has spaces", env["QUOTED"])
	assert.Equal(t, "line1\nline2", env["ESCAPE"])
}

func TestParseEnvWithEscapes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`VAR="hello\nworld"`, "hello\nworld"},
		{`VAR="tab\there"`, "tab\there"},
		{`VAR="quote\"here"`, "quote\"here"},
		{`VAR="back\\slash"`, "back\\slash"},
		{`VAR="dollar\$sign"`, "dollar$sign"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			env, err := ParseEnvString(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, env["VAR"])
		})
	}
}

func TestParseEnv_SkipsMalformed(t *testing.T) {
	// Lines without = or : are silently skipped
	env, err := ParseEnvString(`
GOOD=value
this line has no separator
ALSO_GOOD=another
`)
	assert.NoError(t, err)
	assert.Len(t, env, 2)
	assert.Equal(t, "value", env["GOOD"])
	assert.Equal(t, "another", env["ALSO_GOOD"])
}

func TestParseEnv_Comments(t *testing.T) {
	env, err := ParseEnvString(`
# Hash comment
HOST=localhost

// C-style comment
PORT=8080

  # Indented hash comment
  // Indented C-style comment

VALUE=test
`)
	assert.NoError(t, err)
	assert.Len(t, env, 3)
	assert.Equal(t, "localhost", env["HOST"])
	assert.Equal(t, "8080", env["PORT"])
	assert.Equal(t, "test", env["VALUE"])
}

func TestReadEnvFile_NotFound(t *testing.T) {
	_, err := ReadEnvFile("/nonexistent/path/.env")
	assert.Error(t, err)
}

func TestLoadEnvFile_DefaultFilename(t *testing.T) {
	// Create .env in temp dir
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	envFile := filepath.Join(tmpDir, ".env")
	assert.NoError(t, os.WriteFile(envFile, []byte("TEST_DEFAULT_LOAD=value\n"), 0644))

	// Clean up after test
	defer os.Unsetenv("TEST_DEFAULT_LOAD")

	err := LoadEnvFile() // No filename - uses default ".env"
	assert.NoError(t, err)
	assert.Equal(t, "value", os.Getenv("TEST_DEFAULT_LOAD"))
}

func TestOverloadEnvFile_DefaultFilename(t *testing.T) {
	// Create .env in temp dir
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	envFile := filepath.Join(tmpDir, ".env")
	assert.NoError(t, os.WriteFile(envFile, []byte("TEST_DEFAULT_OVERLOAD=value\n"), 0644))

	// Clean up after test
	defer os.Unsetenv("TEST_DEFAULT_OVERLOAD")

	err := OverloadEnvFile() // No filename - uses default ".env"
	assert.NoError(t, err)
	assert.Equal(t, "value", os.Getenv("TEST_DEFAULT_OVERLOAD"))
}

func TestLoadEnvFile_MissingFile(t *testing.T) {
	err := LoadEnvFile("/nonexistent/.env")
	assert.Error(t, err)
}

func TestOverloadEnvFile_MissingFile(t *testing.T) {
	err := OverloadEnvFile("/nonexistent/.env")
	assert.Error(t, err)
}

func TestParseEnv_SingleQuotesNoEscape(t *testing.T) {
	// Single quotes should NOT process escape sequences
	env, err := ParseEnvString(`VAR='hello\nworld'`)
	assert.NoError(t, err)
	assert.Equal(t, "hello\\nworld", env["VAR"])
}

func TestParseEnv_EmptyQuotedValue(t *testing.T) {
	env, err := ParseEnvString(`
DOUBLE=""
SINGLE=''
`)
	assert.NoError(t, err)
	assert.Equal(t, "", env["DOUBLE"])
	assert.Equal(t, "", env["SINGLE"])
}

func TestParseEnv_UnknownEscape(t *testing.T) {
	// Unknown escape sequences should keep the backslash
	env, err := ParseEnvString(`VAR="hello\xworld"`)
	assert.NoError(t, err)
	assert.Equal(t, "hello\\xworld", env["VAR"])
}

func TestParseEnv_CarriageReturn(t *testing.T) {
	env, err := ParseEnvString(`VAR="line1\rline2"`)
	assert.NoError(t, err)
	assert.Equal(t, "line1\rline2", env["VAR"])
}

func TestWriteEnvFile_EmptyValue(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "out.env")

	input := map[string]string{
		"EMPTY": "",
	}

	assert.NoError(t, WriteEnvFile(input, outFile))

	// Read back
	env, err := ReadEnvFile(outFile)
	assert.NoError(t, err)
	assert.Equal(t, "", env["EMPTY"])
}

func TestWriteEnvFile_SpecialChars(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "out.env")

	input := map[string]string{
		"HASH":        "value#with#hash",
		"LEADING":     " leading space",
		"TRAILING":    "trailing space ",
		"TAB":         "has\ttab",
		"QUOTE":       `has"quote`,
		"BACKSLASH":   `has\backslash`,
		"SINGLEQUOTE": "has'quote",
	}

	assert.NoError(t, WriteEnvFile(input, outFile))

	// Read back
	env, err := ReadEnvFile(outFile)
	assert.NoError(t, err)
	assert.Equal(t, "value#with#hash", env["HASH"])
	assert.Equal(t, " leading space", env["LEADING"])
	assert.Equal(t, "trailing space ", env["TRAILING"])
	assert.Equal(t, "has\ttab", env["TAB"])
	assert.Equal(t, `has"quote`, env["QUOTE"])
	assert.Equal(t, `has\backslash`, env["BACKSLASH"])
	assert.Equal(t, "has'quote", env["SINGLEQUOTE"])
}

func TestFormatEnvLine_Simple(t *testing.T) {
	line := formatEnvLine("KEY", "value")
	assert.Equal(t, "KEY=value", line)
}

func TestFormatEnvLine_NeedsQuotes(t *testing.T) {
	line := formatEnvLine("KEY", "has space")
	assert.Contains(t, line, `"`)
}

func TestUnquote_ShortString(t *testing.T) {
	// Strings less than 2 chars should be returned as-is
	result := unquote("x")
	assert.Equal(t, "x", result)
}

func TestUnquote_MismatchedQuotes(t *testing.T) {
	// Mismatched quotes should return the string as-is
	result := unquote(`"value'`)
	assert.Equal(t, `"value'`, result)
}

func TestLoadEnvFile_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "first.env")
	file2 := filepath.Join(tmpDir, "second.env")

	assert.NoError(t, os.WriteFile(file1, []byte("MULTI_A=from_first\nMULTI_B=from_first\n"), 0644))
	assert.NoError(t, os.WriteFile(file2, []byte("MULTI_B=from_second\nMULTI_C=from_second\n"), 0644))

	defer os.Unsetenv("MULTI_A")
	defer os.Unsetenv("MULTI_B")
	defer os.Unsetenv("MULTI_C")

	err := LoadEnvFile(file1, file2)
	assert.NoError(t, err)

	assert.Equal(t, "from_first", os.Getenv("MULTI_A"))
	assert.Equal(t, "from_first", os.Getenv("MULTI_B")) // First file wins
	assert.Equal(t, "from_second", os.Getenv("MULTI_C"))
}

func TestOverloadEnvFile_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "first.env")
	file2 := filepath.Join(tmpDir, "second.env")

	assert.NoError(t, os.WriteFile(file1, []byte("OVERLOAD_A=from_first\nOVERLOAD_B=from_first\n"), 0644))
	assert.NoError(t, os.WriteFile(file2, []byte("OVERLOAD_B=from_second\nOVERLOAD_C=from_second\n"), 0644))

	defer os.Unsetenv("OVERLOAD_A")
	defer os.Unsetenv("OVERLOAD_B")
	defer os.Unsetenv("OVERLOAD_C")

	err := OverloadEnvFile(file1, file2)
	assert.NoError(t, err)

	assert.Equal(t, "from_first", os.Getenv("OVERLOAD_A"))
	assert.Equal(t, "from_second", os.Getenv("OVERLOAD_B")) // Second file overwrites
	assert.Equal(t, "from_second", os.Getenv("OVERLOAD_C"))
}
