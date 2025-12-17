package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestReadJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	content := `{
		"host": "localhost",
		"port": 8080,
		"debug": true,
		"database": {
			"host": "dbhost",
			"port": 5432
		}
	}`

	assert.NoError(t, os.WriteFile(jsonFile, []byte(content), 0644))

	values, err := ReadJSONFile(jsonFile)
	assert.NoError(t, err)

	assert.Equal(t, "localhost", values["host"])
	assert.Equal(t, float64(8080), values["port"])
	assert.Equal(t, true, values["debug"])
	assert.Equal(t, "dbhost", values["database_host"])
	assert.Equal(t, float64(5432), values["database_port"])
}

func TestParseJSON(t *testing.T) {
	data := []byte(`{"key": "value", "nested": {"a": 1, "b": 2}}`)

	values, err := ParseJSON(data)
	assert.NoError(t, err)

	assert.Equal(t, "value", values["key"])
	assert.Equal(t, float64(1), values["nested_a"])
	assert.Equal(t, float64(2), values["nested_b"])
}

func TestParseJSON_DeeplyNested(t *testing.T) {
	data := []byte(`{
		"level1": {
			"level2": {
				"level3": {
					"value": "deep"
				}
			}
		}
	}`)

	values, err := ParseJSON(data)
	assert.NoError(t, err)

	assert.Equal(t, "deep", values["level1_level2_level3_value"])
}

func TestWriteJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "out.json")

	input := map[string]any{
		"host": "localhost",
		"port": 8080,
	}

	assert.NoError(t, WriteJSONFile(input, outFile))

	// Read back
	values, err := ReadJSONFile(outFile)
	assert.NoError(t, err)
	assert.Equal(t, "localhost", values["host"])
	assert.Equal(t, float64(8080), values["port"])
}

func TestParse_WithJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	content := `{
		"host": "json-host",
		"port": 9000
	}`
	assert.NoError(t, os.WriteFile(jsonFile, []byte(content), 0644))

	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	// JSON file provides values when env vars are missing
	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithJSONFile(jsonFile),
	)
	assert.NoError(t, err)
	assert.Equal(t, "json-host", cfg.Host)
	assert.Equal(t, 9000, cfg.Port)
}

func TestParse_EnvOverridesJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	content := `{
		"host": "json-host",
		"port": 9000
	}`
	assert.NoError(t, os.WriteFile(jsonFile, []byte(content), 0644))

	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	// Env vars take precedence over JSON
	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{
			"HOST": "env-host",
		}),
		WithJSONFile(jsonFile),
	)
	assert.NoError(t, err)
	assert.Equal(t, "env-host", cfg.Host) // From env
	assert.Equal(t, 9000, cfg.Port)       // From JSON
}

func TestReadJSONFile_NotFound(t *testing.T) {
	_, err := ReadJSONFile("/nonexistent/config.json")
	assert.Error(t, err)
}

func TestParseJSON_Invalid(t *testing.T) {
	_, err := ParseJSON([]byte(`{invalid json}`))
	assert.Error(t, err)
}

func TestParseJSON_EmptyObject(t *testing.T) {
	values, err := ParseJSON([]byte(`{}`))
	assert.NoError(t, err)
	assert.Empty(t, values)
}

func TestParseJSON_ArrayValue(t *testing.T) {
	// Arrays should be stored as-is (not flattened)
	data := []byte(`{"items": [1, 2, 3]}`)
	values, err := ParseJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, []any{float64(1), float64(2), float64(3)}, values["items"])
}

func TestParseJSON_NullValue(t *testing.T) {
	data := []byte(`{"value": null}`)
	values, err := ParseJSON(data)
	assert.NoError(t, err)
	assert.Nil(t, values["value"])
}

func TestWriteJSONFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	outFile := filepath.Join(tmpDir, "out.json")

	input := map[string]any{
		"string": "value",
		"number": 42,
		"bool":   true,
	}

	assert.NoError(t, WriteJSONFile(input, outFile))

	// Read back raw content to verify format
	content, err := os.ReadFile(outFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "\"string\": \"value\"")
}

func TestFlattenJSON_MultiLevel(t *testing.T) {
	data := []byte(`{
		"a": {
			"b": {
				"c": {
					"d": "deep"
				}
			}
		},
		"x": {
			"y": "shallow"
		}
	}`)

	values, err := ParseJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, "deep", values["a_b_c_d"])
	assert.Equal(t, "shallow", values["x_y"])
}

func TestParseJSON_MixedTypes(t *testing.T) {
	data := []byte(`{
		"string": "text",
		"int": 42,
		"float": 3.14,
		"bool": true,
		"nested": {
			"value": "nested-value"
		}
	}`)

	values, err := ParseJSON(data)
	assert.NoError(t, err)
	assert.Equal(t, "text", values["string"])
	assert.Equal(t, float64(42), values["int"])
	assert.InDelta(t, 3.14, values["float"].(float64), 0.001)
	assert.Equal(t, true, values["bool"])
	assert.Equal(t, "nested-value", values["nested_value"])
}
