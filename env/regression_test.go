package env

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

type upperStruct struct{ V string }

// BUG-032: value types that are structs (or pointers to them) are parsed from
// the environment, not treated as nested config structs.
func TestParseStructValueTypes(t *testing.T) {
	type Config struct {
		Location  time.Location `env:"TZ"`
		StartTime *time.Time    `env:"START_TIME"`
		Endpoint  *url.URL      `env:"ENDPOINT"`
		Custom    upperStruct   `env:"CUSTOM"`
		CustomPtr *upperStruct  `env:"CUSTOM_PTR"`
	}
	upper := WithParser(func(s string) (upperStruct, error) {
		return upperStruct{V: s + "!"}, nil
	})
	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"TZ":         "America/New_York",
		"START_TIME": "2026-01-02T03:04:05Z",
		"ENDPOINT":   "https://example.com/x",
		"CUSTOM":     "a",
		"CUSTOM_PTR": "b",
	}), upper)
	assert.NoError(t, err)
	assert.Equal(t, "America/New_York", cfg.Location.String())
	assert.NotNil(t, cfg.StartTime)
	assert.Equal(t, 2026, cfg.StartTime.Year())
	assert.NotNil(t, cfg.Endpoint)
	assert.Equal(t, "example.com", cfg.Endpoint.Host)
	assert.Equal(t, "a!", cfg.Custom.V)
	assert.NotNil(t, cfg.CustomPtr)
	assert.Equal(t, "b!", cfg.CustomPtr.V)
}

func TestParseStructValueTypesRequiredAndDefault(t *testing.T) {
	type Config struct {
		Location  time.Location `env:"TZ,required"`
		StartTime *time.Time    `env:"START_TIME,required"`
	}
	_, err := Parse[Config](WithEnvironment(map[string]string{}))
	assert.Error(t, err)
	var notSet *VarNotSetError
	assert.True(t, errors.As(err, &notSet), "got %v", err)

	type WithDefault struct {
		StartTime *time.Time `env:"START_TIME" envDefault:"2020-05-06T00:00:00Z"`
		Optional  *time.Time `env:"OPTIONAL"`
	}
	cfg, err := Parse[WithDefault](WithEnvironment(map[string]string{}))
	assert.NoError(t, err)
	assert.NotNil(t, cfg.StartTime)
	assert.Equal(t, 2020, cfg.StartTime.Year())
	assert.Nil(t, cfg.Optional)
}

// BUG-066: envDefault must not overwrite a value from JSON when the field has
// no env tag.
func TestEnvDefaultKeepsJSONValueWithoutEnvTag(t *testing.T) {
	type Config struct {
		Port int `json:"port" envDefault:"8080"`
	}
	path := filepath.Join(t.TempDir(), "config.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{"port": 9090}`), 0o644))
	cfg, err := Parse[Config](WithJSONFile(path), WithEnvironment(map[string]string{}))
	assert.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port)

	pre := Config{Port: 7070}
	assert.NoError(t, ParseInto(&pre, WithEnvironment(map[string]string{})))
	assert.Equal(t, 7070, pre.Port)

	var empty Config
	assert.NoError(t, ParseInto(&empty, WithEnvironment(map[string]string{})))
	assert.Equal(t, 8080, empty.Port)
}

// BUG-076: values containing a carriage return are quoted so they survive a
// write/read round trip.
func TestWriteEnvFileQuotesCarriageReturn(t *testing.T) {
	for _, value := range []string{"hello\rworld", "trailing\r", "\rleading"} {
		path := filepath.Join(t.TempDir(), ".env")
		assert.NoError(t, WriteEnvFile(map[string]string{"K": value}, path))
		got, err := ReadEnvFile(path)
		assert.NoError(t, err)
		assert.Equal(t, value, got["K"], fmt.Sprintf("round trip of %q", value))
	}
}

// H18: env:"-" skips nested struct and *struct fields too.
func TestIgnoreTagSkipsNestedStructs(t *testing.T) {
	type Internal struct {
		Secret string `env:"SECRET,required"`
	}
	type Config struct {
		Ptr    *Internal `env:"-"`
		Nested Internal  `env:"-"`
	}
	cfg, err := Parse[Config](WithEnvironment(map[string]string{}))
	assert.NoError(t, err)
	assert.Nil(t, cfg.Ptr)
}

// M33: WithOnSet reports the stage-prefixed variable that supplied the value.
func TestOnSetReportsStagedEnvVar(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
		Port string `env:"PORT"`
	}
	got := map[string]string{}
	_, err := Parse[Config](
		WithEnvironment(map[string]string{"PROD_HOST": "h", "PORT": "1"}),
		WithStage("PROD"),
		WithOnSet(func(field, envVar string, value any, isDefault bool) {
			got[field] = envVar
		}),
	)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"Host": "PROD_HOST", "Port": "PORT"}, got)
}
