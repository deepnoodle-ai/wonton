package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/require"
)

func TestParse_BasicTypes(t *testing.T) {
	type Config struct {
		Host     string        `env:"HOST"`
		Port     int           `env:"PORT"`
		Debug    bool          `env:"DEBUG"`
		Timeout  time.Duration `env:"TIMEOUT"`
		Rate     float64       `env:"RATE"`
		MaxConns uint          `env:"MAX_CONNS"`
	}

	env := map[string]string{
		"HOST":      "localhost",
		"PORT":      "8080",
		"DEBUG":     "true",
		"TIMEOUT":   "30s",
		"RATE":      "0.5",
		"MAX_CONNS": "100",
	}

	cfg, err := Parse[Config](WithEnvironment(env))
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Host)
	require.Equal(t, 8080, cfg.Port)
	require.True(t, cfg.Debug)
	require.Equal(t, 30*time.Second, cfg.Timeout)
	require.Equal(t, 0.5, cfg.Rate)
	require.Equal(t, uint(100), cfg.MaxConns)
}

func TestParse_Defaults(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" default:"localhost"`
		Port int    `env:"PORT" default:"3000"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{}))
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Host)
	require.Equal(t, 3000, cfg.Port)
}

func TestParse_Required(t *testing.T) {
	type Config struct {
		APIKey string `env:"API_KEY,required"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{}))
	require.Error(t, err)
	require.True(t, HasError[*VarNotSetError](err))
}

func TestParse_NotEmpty(t *testing.T) {
	type Config struct {
		APIKey string `env:"API_KEY,notEmpty"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"API_KEY": "",
	}))
	require.Error(t, err)
	require.True(t, HasError[*EmptyVarError](err))
}

func TestParse_Prefix(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	env := map[string]string{
		"MYAPP_HOST": "example.com",
		"MYAPP_PORT": "9000",
	}

	cfg, err := Parse[Config](
		WithEnvironment(env),
		WithPrefix("MYAPP"),
	)
	require.NoError(t, err)
	require.Equal(t, "example.com", cfg.Host)
	require.Equal(t, 9000, cfg.Port)
}

func TestParse_Stage(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	env := map[string]string{
		"HOST":      "localhost",
		"PORT":      "3000",
		"PROD_HOST": "prod.example.com",
		"PROD_PORT": "443",
	}

	// Without stage - use default vars
	cfg, err := Parse[Config](WithEnvironment(env))
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Host)
	require.Equal(t, 3000, cfg.Port)

	// With stage - use stage-prefixed vars
	cfg, err = Parse[Config](
		WithEnvironment(env),
		WithStage("PROD"),
	)
	require.NoError(t, err)
	require.Equal(t, "prod.example.com", cfg.Host)
	require.Equal(t, 443, cfg.Port)
}

func TestParse_Slices(t *testing.T) {
	type Config struct {
		Hosts []string `env:"HOSTS"`
		Ports []int    `env:"PORTS"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HOSTS": "a.com,b.com,c.com",
		"PORTS": "80, 443, 8080",
	}))
	require.NoError(t, err)
	require.Equal(t, []string{"a.com", "b.com", "c.com"}, cfg.Hosts)
	require.Equal(t, []int{80, 443, 8080}, cfg.Ports)
}

func TestParse_Maps(t *testing.T) {
	type Config struct {
		Headers map[string]string `env:"HEADERS"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HEADERS": "Content-Type:application/json, Accept:*/*",
	}))
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "*/*",
	}, cfg.Headers)
}

func TestParse_NestedStructs(t *testing.T) {
	type Database struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	type Config struct {
		Database Database `envPrefix:"DB_"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"DB_HOST": "dbhost",
		"DB_PORT": "5432",
	}))
	require.NoError(t, err)
	require.Equal(t, "dbhost", cfg.Database.Host)
	require.Equal(t, 5432, cfg.Database.Port)
}

func TestParse_FileLoading(t *testing.T) {
	// Create temp file with content
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.txt")
	require.NoError(t, os.WriteFile(secretFile, []byte("super-secret"), 0600))

	type Config struct {
		Secret string `env:"SECRET_FILE,file"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"SECRET_FILE": secretFile,
	}))
	require.NoError(t, err)
	require.Equal(t, "super-secret", cfg.Secret)
}

func TestParse_Expand(t *testing.T) {
	type Config struct {
		Path string `env:"PATH_TEMPLATE,expand"`
	}

	os.Setenv("HOME", "/home/user")
	defer os.Unsetenv("HOME")

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"PATH_TEMPLATE": "$HOME/app/data",
	}))
	require.NoError(t, err)
	require.Equal(t, "/home/user/app/data", cfg.Path)
}

func TestParse_WithOnSet(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" default:"localhost"`
		Port int    `env:"PORT"`
	}

	envVars := map[string]string{
		"PORT": "8080",
	}

	seen := map[string]struct {
		envVar    string
		value     any
		isDefault bool
	}{}

	_, err := Parse[Config](
		WithEnvironment(envVars),
		WithOnSet(func(fieldName, envVar string, value any, isDefault bool) {
			seen[fieldName] = struct {
				envVar    string
				value     any
				isDefault bool
			}{envVar: envVar, value: value, isDefault: isDefault}
		}),
	)
	require.NoError(t, err)

	hostCall, ok := seen["Host"]
	require.True(t, ok, "Host should trigger OnSet")
	require.Equal(t, "localhost", hostCall.value)
	require.True(t, hostCall.isDefault, "Host should be marked as default")
	require.True(t, strings.HasSuffix(hostCall.envVar, "HOST"), "env var should end with HOST")

	portCall, ok := seen["Port"]
	require.True(t, ok, "Port should trigger OnSet")
	require.Equal(t, 8080, portCall.value)
	require.False(t, portCall.isDefault, "Port should come from env vars")
}

func TestParse_WithUseFieldName(t *testing.T) {
	type Config struct {
		APIKey  string
		Timeout time.Duration
	}

	envVars := map[string]string{
		"A_P_I_KEY": "secret",
		"TIMEOUT": "45s",
	}

	cfg, err := Parse[Config](
		WithEnvironment(envVars),
		WithUseFieldName(),
	)
	require.NoError(t, err)
	require.Equal(t, "secret", cfg.APIKey)
	require.Equal(t, 45*time.Second, cfg.Timeout)
}

func TestParse_CustomParserOption(t *testing.T) {
	type Endpoint string

	type Config struct {
		Base Endpoint `env:"BASE"`
	}

	envVars := map[string]string{
		"BASE": "https://api.example.com",
	}

	cfg, err := Parse[Config](
		WithEnvironment(envVars),
		WithParser(func(value string) (Endpoint, error) {
			if value == "" {
				return "", fmt.Errorf("empty endpoint")
			}
			return Endpoint(value), nil
		}),
	)
	require.NoError(t, err)
	require.Equal(t, Endpoint("https://api.example.com"), cfg.Base)
}

func TestParse_WithEnvFileOverridesOrder(t *testing.T) {
	tmpDir := t.TempDir()

	first := filepath.Join(tmpDir, "first.env")
	second := filepath.Join(tmpDir, "second.env")

	require.NoError(t, os.WriteFile(first, []byte("VALUE=one\n"), 0644))
	require.NoError(t, os.WriteFile(second, []byte("VALUE=two\nEXTRA=from_second\n"), 0644))

	type Config struct {
		Value string `env:"VALUE"`
		Extra string `env:"EXTRA"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithEnvFile(first, second),
	)
	require.NoError(t, err)
	require.Equal(t, "one", cfg.Value, "first env file should win when key already set")
	require.Equal(t, "from_second", cfg.Extra)
}

func TestParse_WithJSONFileOverrideOrder(t *testing.T) {
	tmpDir := t.TempDir()

	first := filepath.Join(tmpDir, "first.json")
	second := filepath.Join(tmpDir, "second.json")

	require.NoError(t, os.WriteFile(first, []byte(`{"host":"first","port":8080}`), 0644))
	require.NoError(t, os.WriteFile(second, []byte(`{"host":"second"}`), 0644))

	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithJSONFile(first, second),
	)
	require.NoError(t, err)
	require.Equal(t, "second", cfg.Host, "later JSON file should override")
	require.Equal(t, 8080, cfg.Port, "unchanged field should come from the first file")
}

func TestParseInto_UpdatesExistingStruct(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT" default:"80"`
	}

	cfg := Config{Host: "initial"}
	err := ParseInto(&cfg, WithEnvironment(map[string]string{
		"HOST": "parsed",
	}))
	require.NoError(t, err)
	require.Equal(t, "parsed", cfg.Host)
	require.Equal(t, 80, cfg.Port, "default should populate missing field")
}

func TestParse_RequiredIfNoDefault(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	_, err := Parse[Config](
		WithEnvironment(map[string]string{
			"HOST": "localhost",
		}),
		WithRequiredIfNoDefault(),
	)
	require.Error(t, err)
	require.True(t, HasError[*VarNotSetError](err))
}

func TestParse_UseFieldName(t *testing.T) {
	type Config struct {
		ServerHost string
		ServerPort int
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{
			"SERVER_HOST": "myhost",
			"SERVER_PORT": "9999",
		}),
		WithUseFieldName(),
	)
	require.NoError(t, err)
	require.Equal(t, "myhost", cfg.ServerHost)
	require.Equal(t, 9999, cfg.ServerPort)
}

func TestParse_CustomParser(t *testing.T) {
	type Level int
	const (
		LevelDebug Level = iota
		LevelInfo
		LevelWarn
		LevelError
	)

	type Config struct {
		LogLevel Level `env:"LOG_LEVEL"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{
			"LOG_LEVEL": "warn",
		}),
		WithParser(func(s string) (Level, error) {
			switch s {
			case "debug":
				return LevelDebug, nil
			case "info":
				return LevelInfo, nil
			case "warn":
				return LevelWarn, nil
			case "error":
				return LevelError, nil
			default:
				return 0, &ParseError{Err: nil}
			}
		}),
	)
	require.NoError(t, err)
	require.Equal(t, LevelWarn, cfg.LogLevel)
}

func TestParse_OnSet(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" default:"localhost"`
		Port int    `env:"PORT"`
	}

	var setCalls []string
	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{
			"PORT": "8080",
		}),
		WithOnSet(func(field, envVar string, value any, isDefault bool) {
			setCalls = append(setCalls, field)
		}),
	)
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Host)
	require.Equal(t, 8080, cfg.Port)
	require.Contains(t, setCalls, "Host")
	require.Contains(t, setCalls, "Port")
}

func TestParse_IgnoredField(t *testing.T) {
	type Config struct {
		Host   string `env:"HOST"`
		Secret string `env:"-"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HOST":   "localhost",
		"SECRET": "should-be-ignored",
	}))
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Host)
	require.Empty(t, cfg.Secret)
}

func TestParse_BooleanVariants(t *testing.T) {
	type Config struct {
		A bool `env:"A"`
		B bool `env:"B"`
		C bool `env:"C"`
		D bool `env:"D"`
		E bool `env:"E"`
		F bool `env:"F"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"A": "true",
		"B": "1",
		"C": "yes",
		"D": "false",
		"E": "0",
		"F": "no",
	}))
	require.NoError(t, err)
	require.True(t, cfg.A)
	require.True(t, cfg.B)
	require.True(t, cfg.C)
	require.False(t, cfg.D)
	require.False(t, cfg.E)
	require.False(t, cfg.F)
}

func TestParse_Pointers(t *testing.T) {
	type Config struct {
		Host *string `env:"HOST"`
		Port *int    `env:"PORT"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HOST": "example.com",
		"PORT": "8080",
	}))
	require.NoError(t, err)
	require.NotNil(t, cfg.Host)
	require.NotNil(t, cfg.Port)
	require.Equal(t, "example.com", *cfg.Host)
	require.Equal(t, 8080, *cfg.Port)
}

func TestParse_EmbeddedStruct(t *testing.T) {
	type Common struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	type Config struct {
		Common
		Name string `env:"NAME"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HOST": "localhost",
		"PORT": "8080",
		"NAME": "myapp",
	}))
	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.Host)
	require.Equal(t, 8080, cfg.Port)
	require.Equal(t, "myapp", cfg.Name)
}

func TestParse_AggregateErrors(t *testing.T) {
	type Config struct {
		A string `env:"A,required"`
		B string `env:"B,required"`
		C int    `env:"C"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"C": "not-a-number",
	}))
	require.Error(t, err)

	// Should have multiple errors
	aggErr, ok := err.(*AggregateError)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(aggErr.Errors), 2) // A missing, B missing, C parse error
}

func TestMust_Panics(t *testing.T) {
	type Config struct {
		Required string `env:"REQUIRED,required"`
	}

	require.Panics(t, func() {
		Must[Config](WithEnvironment(map[string]string{}))
	})
}

func TestMust_Success(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" default:"localhost"`
	}

	require.NotPanics(t, func() {
		cfg := Must[Config](WithEnvironment(map[string]string{}))
		require.Equal(t, "localhost", cfg.Host)
	})
}

func TestToUpperSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Host", "HOST"},
		{"ServerHost", "SERVER_HOST"},
		{"HTTPServer", "H_T_T_P_SERVER"},
		{"DB", "D_B"},
		{"APIKey", "A_P_I_KEY"},
		{"MaxConnections", "MAX_CONNECTIONS"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.expected, toUpperSnakeCase(tt.input))
		})
	}
}
