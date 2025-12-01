package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deepnoodle-ai/gooey/require"
)

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

	require.NoError(t, os.WriteFile(envFile, []byte(content), 0644))

	env, err := ReadEnvFile(envFile)
	require.NoError(t, err)

	require.Equal(t, "localhost", env["HOST"])
	require.Equal(t, "8080", env["PORT"])
	require.Equal(t, "Hello, World!", env["MESSAGE"])
	require.Equal(t, "single quoted", env["SINGLE"])
	require.Equal(t, "true", env["DEBUG"])
	require.Equal(t, "", env["EMPTY"])
	require.Equal(t, "test", env["VALUE"])
	require.Equal(t, "line1\nline2", env["MULTILINE"])
	require.Equal(t, "secret123", env["API_KEY"])
}

func TestParseEnvString(t *testing.T) {
	env, err := ParseEnvString(`
FOO=bar
BAZ=qux
`)
	require.NoError(t, err)
	require.Equal(t, "bar", env["FOO"])
	require.Equal(t, "qux", env["BAZ"])
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
	require.NoError(t, os.WriteFile(envFile, []byte("TEST_LOAD_VAR=from_file\n"), 0644))

	// Test: Load does NOT override existing
	os.Setenv("TEST_LOAD_VAR", "existing")
	require.NoError(t, LoadEnvFile(envFile))
	require.Equal(t, "existing", os.Getenv("TEST_LOAD_VAR"))

	// Test: Load sets non-existing
	os.Unsetenv("TEST_LOAD_VAR")
	require.NoError(t, LoadEnvFile(envFile))
	require.Equal(t, "from_file", os.Getenv("TEST_LOAD_VAR"))
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
	require.NoError(t, os.WriteFile(envFile, []byte("TEST_OVERLOAD_VAR=from_file\n"), 0644))

	// Test: Overload DOES override existing
	os.Setenv("TEST_OVERLOAD_VAR", "existing")
	require.NoError(t, OverloadEnvFile(envFile))
	require.Equal(t, "from_file", os.Getenv("TEST_OVERLOAD_VAR"))
}

func TestWriteEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "out.env")

	input := map[string]string{
		"SIMPLE": "value",
		"QUOTED": "has spaces",
		"ESCAPE": "line1\nline2",
	}

	require.NoError(t, WriteEnvFile(input, outFile))

	// Read back
	env, err := ReadEnvFile(outFile)
	require.NoError(t, err)
	require.Equal(t, "value", env["SIMPLE"])
	require.Equal(t, "has spaces", env["QUOTED"])
	require.Equal(t, "line1\nline2", env["ESCAPE"])
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
			require.NoError(t, err)
			require.Equal(t, tt.expected, env["VAR"])
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
	require.NoError(t, err)
	require.Len(t, env, 2)
	require.Equal(t, "value", env["GOOD"])
	require.Equal(t, "another", env["ALSO_GOOD"])
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
	require.NoError(t, err)
	require.Len(t, env, 3)
	require.Equal(t, "localhost", env["HOST"])
	require.Equal(t, "8080", env["PORT"])
	require.Equal(t, "test", env["VALUE"])
}
