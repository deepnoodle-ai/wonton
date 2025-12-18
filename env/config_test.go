package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

// Example demonstrates basic configuration parsing with default values.
func Example() {
	type Config struct {
		Host string `env:"HOST" envDefault:"localhost"`
		Port int    `env:"PORT" envDefault:"8080"`
	}

	// Parse with defaults (no environment variables set)
	cfg, err := Parse[Config](WithEnvironment(map[string]string{}))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Host: %s, Port: %d\n", cfg.Host, cfg.Port)
	// Output: Host: localhost, Port: 8080
}

// Example_withPrefix demonstrates using a prefix for all environment variables.
func Example_withPrefix() {
	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	env := map[string]string{
		"MYAPP_HOST": "api.example.com",
		"MYAPP_PORT": "443",
	}

	cfg, err := Parse[Config](
		WithEnvironment(env),
		WithPrefix("MYAPP"),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Host: %s, Port: %d\n", cfg.Host, cfg.Port)
	// Output: Host: api.example.com, Port: 443
}

// Example_nestedStructs demonstrates configuration with nested structs.
func Example_nestedStructs() {
	type Database struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	type Config struct {
		Database Database `envPrefix:"DB_"`
	}

	env := map[string]string{
		"DB_HOST": "postgres.example.com",
		"DB_PORT": "5432",
	}

	cfg, err := Parse[Config](WithEnvironment(env))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Database: %s:%d\n", cfg.Database.Host, cfg.Database.Port)
	// Output: Database: postgres.example.com:5432
}

// Example_slicesAndMaps demonstrates parsing slices and maps from environment variables.
func Example_slicesAndMaps() {
	type Config struct {
		Hosts  []string          `env:"HOSTS"`
		Labels map[string]string `env:"LABELS"`
	}

	env := map[string]string{
		"HOSTS":  "host1,host2,host3",
		"LABELS": "env:prod,region:us-west",
	}

	cfg, err := Parse[Config](WithEnvironment(env))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Hosts: %v\n", cfg.Hosts)
	fmt.Printf("Labels: %v\n", cfg.Labels)
	// Output:
	// Hosts: [host1 host2 host3]
	// Labels: map[env:prod region:us-west]
}

// Example_customParser demonstrates using a custom parser for a user-defined type.
func Example_customParser() {
	type LogLevel int
	const (
		Debug LogLevel = iota
		Info
		Warn
		Error
	)

	type Config struct {
		Level LogLevel `env:"LOG_LEVEL"`
	}

	env := map[string]string{
		"LOG_LEVEL": "warn",
	}

	cfg, err := Parse[Config](
		WithEnvironment(env),
		WithParser(func(s string) (LogLevel, error) {
			switch strings.ToLower(s) {
			case "debug":
				return Debug, nil
			case "info":
				return Info, nil
			case "warn":
				return Warn, nil
			case "error":
				return Error, nil
			default:
				return 0, fmt.Errorf("invalid log level: %s", s)
			}
		}),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Log Level: %d (Warn)\n", cfg.Level)
	// Output: Log Level: 2 (Warn)
}

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
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
	assert.True(t, cfg.Debug)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, 0.5, cfg.Rate)
	assert.Equal(t, uint(100), cfg.MaxConns)
}

func TestParse_Defaults(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" envDefault:"localhost"`
		Port int    `env:"PORT" envDefault:"3000"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{}))
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 3000, cfg.Port)
}

func TestParse_Required(t *testing.T) {
	type Config struct {
		APIKey string `env:"API_KEY,required"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{}))
	assert.Error(t, err)
	assert.True(t, HasError[*VarNotSetError](err))
}

func TestParse_NotEmpty(t *testing.T) {
	type Config struct {
		APIKey string `env:"API_KEY,notEmpty"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"API_KEY": "",
	}))
	assert.Error(t, err)
	assert.True(t, HasError[*EmptyVarError](err))
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
	assert.NoError(t, err)
	assert.Equal(t, "example.com", cfg.Host)
	assert.Equal(t, 9000, cfg.Port)
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
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 3000, cfg.Port)

	// With stage - use stage-prefixed vars
	cfg, err = Parse[Config](
		WithEnvironment(env),
		WithStage("PROD"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "prod.example.com", cfg.Host)
	assert.Equal(t, 443, cfg.Port)
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
	assert.NoError(t, err)
	assert.Equal(t, []string{"a.com", "b.com", "c.com"}, cfg.Hosts)
	assert.Equal(t, []int{80, 443, 8080}, cfg.Ports)
}

func TestParse_Maps(t *testing.T) {
	type Config struct {
		Headers map[string]string `env:"HEADERS"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HEADERS": "Content-Type:application/json, Accept:*/*",
	}))
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{
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
	assert.NoError(t, err)
	assert.Equal(t, "dbhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
}

func TestParse_FileLoading(t *testing.T) {
	// Create temp file with content
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "secret.txt")
	assert.NoError(t, os.WriteFile(secretFile, []byte("super-secret"), 0600))

	type Config struct {
		Secret string `env:"SECRET_FILE,file"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"SECRET_FILE": secretFile,
	}))
	assert.NoError(t, err)
	assert.Equal(t, "super-secret", cfg.Secret)
}

func TestParse_Expand(t *testing.T) {
	type Config struct {
		Path string `env:"PATH_TEMPLATE,expand"`
	}

	// When using WithEnvironment, expansion uses the custom environment map
	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HOME":          "/home/user",
		"PATH_TEMPLATE": "$HOME/app/data",
	}))
	assert.NoError(t, err)
	assert.Equal(t, "/home/user/app/data", cfg.Path)
}

func TestParse_WithOnSet(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" envDefault:"localhost"`
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
	assert.NoError(t, err)

	hostCall, ok := seen["Host"]
	assert.True(t, ok, "Host should trigger OnSet")
	assert.Equal(t, "localhost", hostCall.value)
	assert.True(t, hostCall.isDefault, "Host should be marked as default")
	assert.True(t, strings.HasSuffix(hostCall.envVar, "HOST"), "env var should end with HOST")

	portCall, ok := seen["Port"]
	assert.True(t, ok, "Port should trigger OnSet")
	assert.Equal(t, 8080, portCall.value)
	assert.False(t, portCall.isDefault, "Port should come from env vars")
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
	assert.NoError(t, err)
	assert.Equal(t, "secret", cfg.APIKey)
	assert.Equal(t, 45*time.Second, cfg.Timeout)
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
	assert.NoError(t, err)
	assert.Equal(t, Endpoint("https://api.example.com"), cfg.Base)
}

func TestParse_WithEnvFileOverridesOrder(t *testing.T) {
	tmpDir := t.TempDir()

	first := filepath.Join(tmpDir, "first.env")
	second := filepath.Join(tmpDir, "second.env")

	assert.NoError(t, os.WriteFile(first, []byte("VALUE=one\nFIRST_ONLY=yes\n"), 0644))
	assert.NoError(t, os.WriteFile(second, []byte("VALUE=two\nEXTRA=from_second\n"), 0644))

	type Config struct {
		Value     string `env:"VALUE"`
		Extra     string `env:"EXTRA"`
		FirstOnly string `env:"FIRST_ONLY"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithEnvFile(first, second),
	)
	assert.NoError(t, err)
	assert.Equal(t, "two", cfg.Value, "later env file should override earlier ones")
	assert.Equal(t, "from_second", cfg.Extra)
	assert.Equal(t, "yes", cfg.FirstOnly, "keys only in first file should be preserved")
}

func TestParse_WithJSONFileOverrideOrder(t *testing.T) {
	tmpDir := t.TempDir()

	first := filepath.Join(tmpDir, "first.json")
	second := filepath.Join(tmpDir, "second.json")

	assert.NoError(t, os.WriteFile(first, []byte(`{"host":"first","port":8080}`), 0644))
	assert.NoError(t, os.WriteFile(second, []byte(`{"host":"second"}`), 0644))

	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithJSONFile(first, second),
	)
	assert.NoError(t, err)
	assert.Equal(t, "second", cfg.Host, "later JSON file should override")
	assert.Equal(t, 8080, cfg.Port, "unchanged field should come from the first file")
}

func TestParseInto_UpdatesExistingStruct(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT" envDefault:"80"`
	}

	cfg := Config{Host: "initial"}
	err := ParseInto(&cfg, WithEnvironment(map[string]string{
		"HOST": "parsed",
	}))
	assert.NoError(t, err)
	assert.Equal(t, "parsed", cfg.Host)
	assert.Equal(t, 80, cfg.Port, "default should populate missing field")
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
	assert.Error(t, err)
	assert.True(t, HasError[*VarNotSetError](err))
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
	assert.NoError(t, err)
	assert.Equal(t, "myhost", cfg.ServerHost)
	assert.Equal(t, 9999, cfg.ServerPort)
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
	assert.NoError(t, err)
	assert.Equal(t, LevelWarn, cfg.LogLevel)
}

func TestParse_OnSet(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" envDefault:"localhost"`
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
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
	assert.Contains(t, setCalls, "Host")
	assert.Contains(t, setCalls, "Port")
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
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Empty(t, cfg.Secret)
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
	assert.NoError(t, err)
	assert.True(t, cfg.A)
	assert.True(t, cfg.B)
	assert.True(t, cfg.C)
	assert.False(t, cfg.D)
	assert.False(t, cfg.E)
	assert.False(t, cfg.F)
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
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Host)
	assert.NotNil(t, cfg.Port)
	assert.Equal(t, "example.com", *cfg.Host)
	assert.Equal(t, 8080, *cfg.Port)
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
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, "myapp", cfg.Name)
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
	assert.Error(t, err)

	// Should have multiple errors
	aggErr, ok := err.(*AggregateError)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(aggErr.Errors), 2) // A missing, B missing, C parse error
}

func TestMust_Panics(t *testing.T) {
	type Config struct {
		Required string `env:"REQUIRED,required"`
	}

	assert.Panics(t, func() {
		Must[Config](WithEnvironment(map[string]string{}))
	})
}

func TestMust_Success(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" envDefault:"localhost"`
	}

	assert.NotPanics(t, func() {
		cfg := Must[Config](WithEnvironment(map[string]string{}))
		assert.Equal(t, "localhost", cfg.Host)
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
			assert.Equal(t, tt.expected, toUpperSnakeCase(tt.input))
		})
	}
}

func TestParseInto_NonPointer(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
	}

	var cfg Config
	err := ParseInto(cfg, WithEnvironment(map[string]string{"HOST": "localhost"}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected non-nil pointer to struct")
}

func TestParseInto_NilPointer(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
	}

	var cfg *Config
	err := ParseInto(cfg, WithEnvironment(map[string]string{"HOST": "localhost"}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected non-nil pointer to struct")
}

func TestParseInto_PointerToNonStruct(t *testing.T) {
	var s string
	err := ParseInto(&s, WithEnvironment(map[string]string{}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected pointer to struct")
}

func TestParse_NestedStructWithoutEnvPrefix(t *testing.T) {
	type Database struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	type Config struct {
		Database Database // No envPrefix tag - uses DATABASE_ prefix
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"DATABASE_HOST": "dbhost",
		"DATABASE_PORT": "5432",
	}))
	assert.NoError(t, err)
	assert.Equal(t, "dbhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
}

func TestParse_PointerToNestedStruct(t *testing.T) {
	type Database struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	type Config struct {
		Database *Database `envPrefix:"DB_"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"DB_HOST": "dbhost",
		"DB_PORT": "5432",
	}))
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Database)
	assert.Equal(t, "dbhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
}

func TestParse_InvalidBool(t *testing.T) {
	type Config struct {
		Debug bool `env:"DEBUG"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"DEBUG": "not-a-bool",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid boolean value")
}

func TestParse_InvalidInt(t *testing.T) {
	type Config struct {
		Port int `env:"PORT"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"PORT": "not-a-number",
	}))
	assert.Error(t, err)
}

func TestParse_InvalidUint(t *testing.T) {
	type Config struct {
		Count uint `env:"COUNT"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"COUNT": "-1",
	}))
	assert.Error(t, err)
}

func TestParse_InvalidFloat(t *testing.T) {
	type Config struct {
		Rate float64 `env:"RATE"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"RATE": "not-a-float",
	}))
	assert.Error(t, err)
}

func TestParse_InvalidDuration(t *testing.T) {
	type Config struct {
		Timeout time.Duration `env:"TIMEOUT"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"TIMEOUT": "invalid",
	}))
	assert.Error(t, err)
}

func TestParse_InvalidMapFormat(t *testing.T) {
	type Config struct {
		Headers map[string]string `env:"HEADERS"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"HEADERS": "invalid-no-colon",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid map entry")
}

func TestParse_EmptySlice(t *testing.T) {
	type Config struct {
		Hosts []string `env:"HOSTS"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HOSTS": "",
	}))
	assert.NoError(t, err)
	assert.Nil(t, cfg.Hosts)
}

func TestParse_EmptyMap(t *testing.T) {
	type Config struct {
		Headers map[string]string `env:"HEADERS"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HEADERS": "",
	}))
	assert.NoError(t, err)
	assert.Nil(t, cfg.Headers)
}

func TestParse_InvalidSliceElement(t *testing.T) {
	type Config struct {
		Ports []int `env:"PORTS"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"PORTS": "80, not-a-number, 443",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "element 1")
}

func TestParse_FileLoadError(t *testing.T) {
	type Config struct {
		Secret string `env:"SECRET_FILE,file"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"SECRET_FILE": "/nonexistent/path/to/file",
	}))
	assert.Error(t, err)
	assert.True(t, HasError[*FileLoadError](err))
}

func TestParse_WithTagName(t *testing.T) {
	type Config struct {
		Host string `config:"HOST" fallback:"localhost"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithTagName("config", "fallback"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)

	cfg, err = Parse[Config](
		WithEnvironment(map[string]string{"HOST": "myhost"}),
		WithTagName("config", "fallback"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "myhost", cfg.Host)
}

func TestParse_DefaultWithoutEnvTag(t *testing.T) {
	type Config struct {
		Host string `envDefault:"localhost"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{}))
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
}

func TestParse_DefaultParseError(t *testing.T) {
	type Config struct {
		Port int `envDefault:"not-a-number"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{}))
	assert.Error(t, err)
}

func TestParse_IntegerTypes(t *testing.T) {
	type Config struct {
		Int8Val  int8  `env:"INT8"`
		Int16Val int16 `env:"INT16"`
		Int32Val int32 `env:"INT32"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"INT8":  "127",
		"INT16": "32767",
		"INT32": "2147483647",
	}))
	assert.NoError(t, err)
	assert.Equal(t, int8(127), cfg.Int8Val)
	assert.Equal(t, int16(32767), cfg.Int16Val)
	assert.Equal(t, int32(2147483647), cfg.Int32Val)
}

func TestParse_UintTypes(t *testing.T) {
	type Config struct {
		Uint8Val  uint8  `env:"UINT8"`
		Uint16Val uint16 `env:"UINT16"`
		Uint32Val uint32 `env:"UINT32"`
		Uint64Val uint64 `env:"UINT64"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"UINT8":  "255",
		"UINT16": "65535",
		"UINT32": "4294967295",
		"UINT64": "18446744073709551615",
	}))
	assert.NoError(t, err)
	assert.Equal(t, uint8(255), cfg.Uint8Val)
	assert.Equal(t, uint16(65535), cfg.Uint16Val)
	assert.Equal(t, uint32(4294967295), cfg.Uint32Val)
	assert.Equal(t, uint64(18446744073709551615), cfg.Uint64Val)
}

func TestParse_Float32(t *testing.T) {
	type Config struct {
		Rate float32 `env:"RATE"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"RATE": "3.14",
	}))
	assert.NoError(t, err)
	assert.InDelta(t, 3.14, float64(cfg.Rate), 0.001)
}

func TestParse_HexInt(t *testing.T) {
	type Config struct {
		Value int `env:"VALUE"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"VALUE": "0xFF",
	}))
	assert.NoError(t, err)
	assert.Equal(t, 255, cfg.Value)
}

func TestParse_MoreBooleanVariants(t *testing.T) {
	type Config struct {
		A bool `env:"A"`
		B bool `env:"B"`
		C bool `env:"C"`
		D bool `env:"D"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"A": "on",
		"B": "off",
		"C": "t",
		"D": "f",
	}))
	assert.NoError(t, err)
	assert.True(t, cfg.A)
	assert.False(t, cfg.B)
	assert.True(t, cfg.C)
	assert.False(t, cfg.D)
}

func TestParse_EmptyBool(t *testing.T) {
	type Config struct {
		Debug bool `env:"DEBUG"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"DEBUG": "",
	}))
	assert.NoError(t, err)
	assert.False(t, cfg.Debug)
}

func TestParse_YBooleanVariants(t *testing.T) {
	type Config struct {
		A bool `env:"A"`
		B bool `env:"B"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"A": "y",
		"B": "n",
	}))
	assert.NoError(t, err)
	assert.True(t, cfg.A)
	assert.False(t, cfg.B)
}

func TestParse_UnexportedField(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
		port int    `env:"PORT"` //nolint:unused // Testing unexported field handling
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"HOST": "localhost",
		"PORT": "8080",
	}))
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
	// port should remain zero since it's unexported
}

func TestParse_CustomParserError(t *testing.T) {
	type Custom string

	type Config struct {
		Value Custom `env:"VALUE"`
	}

	_, err := Parse[Config](
		WithEnvironment(map[string]string{
			"VALUE": "invalid",
		}),
		WithParser(func(s string) (Custom, error) {
			if s == "invalid" {
				return "", fmt.Errorf("custom parser error")
			}
			return Custom(s), nil
		}),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom parser error")
}

func TestParse_StageWithPrefix(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{
			"PROD_MYAPP_HOST": "prod-host",
			"MYAPP_HOST":      "default-host",
		}),
		WithPrefix("MYAPP"),
		WithStage("PROD"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "prod-host", cfg.Host)
}

func TestParse_StagePartialOverride(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
		Port int    `env:"PORT"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{
			"PROD_HOST": "prod-host",
			"HOST":      "default-host",
			"PORT":      "8080",
		}),
		WithStage("PROD"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "prod-host", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
}

func TestParse_UnsupportedType(t *testing.T) {
	type Config struct {
		Ch chan int `env:"CH"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"CH": "something",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestParse_MapWithIntKeys(t *testing.T) {
	type Config struct {
		Ports map[int]string `env:"PORTS"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"PORTS": "80:http, 443:https",
	}))
	assert.NoError(t, err)
	assert.Equal(t, map[int]string{
		80:  "http",
		443: "https",
	}, cfg.Ports)
}

func TestParse_MapWithIntValues(t *testing.T) {
	type Config struct {
		Priorities map[string]int `env:"PRIORITIES"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"PRIORITIES": "high:1, low:10",
	}))
	assert.NoError(t, err)
	assert.Equal(t, map[string]int{
		"high": 1,
		"low":  10,
	}, cfg.Priorities)
}

func TestParse_MapKeyError(t *testing.T) {
	type Config struct {
		Ports map[int]string `env:"PORTS"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"PORTS": "invalid:http",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "map key")
}

func TestParse_MapValueError(t *testing.T) {
	type Config struct {
		Priorities map[string]int `env:"PRIORITIES"`
	}

	_, err := Parse[Config](WithEnvironment(map[string]string{
		"PRIORITIES": "high:invalid",
	}))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "map value")
}

func TestAggregateError_Empty(t *testing.T) {
	err := &AggregateError{}
	assert.Equal(t, "no errors", err.Error())
	assert.Empty(t, err.Unwrap())
}

func TestAggregateError_Single(t *testing.T) {
	inner := fmt.Errorf("single error")
	err := &AggregateError{Errors: []error{inner}}
	assert.Equal(t, "single error", err.Error())
	assert.Equal(t, []error{inner}, err.Unwrap())
}

func TestAggregateError_Multiple(t *testing.T) {
	err := &AggregateError{
		Errors: []error{
			fmt.Errorf("error 1"),
			fmt.Errorf("error 2"),
		},
	}
	assert.Contains(t, err.Error(), "2 errors")
	assert.Contains(t, err.Error(), "error 1")
	assert.Contains(t, err.Error(), "error 2")
}

func TestAggregateError_Is(t *testing.T) {
	target := fmt.Errorf("target error")
	err := &AggregateError{
		Errors: []error{
			fmt.Errorf("other error"),
			fmt.Errorf("wrapped: %w", target),
		},
	}
	assert.True(t, err.Is(target))

	unrelated := fmt.Errorf("unrelated")
	assert.False(t, err.Is(unrelated))
}

func TestParseError_WithWrapped(t *testing.T) {
	inner := fmt.Errorf("inner error")
	err := &ParseError{Err: inner}
	assert.Contains(t, err.Error(), "inner error")
	assert.Equal(t, inner, err.Unwrap())
}

func TestParseError_NoWrapped(t *testing.T) {
	err := &ParseError{}
	assert.Equal(t, "parse error", err.Error())
	assert.Nil(t, err.Unwrap())
}

func TestFieldError_AllFields(t *testing.T) {
	err := &FieldError{
		Field:  "Port",
		EnvVar: "APP_PORT",
		Value:  "invalid",
		Err:    fmt.Errorf("not a number"),
	}
	msg := err.Error()
	assert.Contains(t, msg, "Port")
	assert.Contains(t, msg, "APP_PORT")
	assert.Contains(t, msg, "not a number")
	// Note: Value is intentionally excluded from Error() string to prevent
	// leaking secrets into logs. Access err.Value directly if needed.
	assert.NotContains(t, msg, "invalid", "values should not appear in error messages")
	assert.Equal(t, "invalid", err.Value, "value should still be accessible directly")
}

func TestFieldError_NoValue(t *testing.T) {
	err := &FieldError{
		Field:  "Port",
		EnvVar: "APP_PORT",
		Err:    fmt.Errorf("not found"),
	}
	msg := err.Error()
	assert.Contains(t, msg, "Port")
	assert.Contains(t, msg, "APP_PORT")
	assert.NotContains(t, msg, "cannot parse")
}

func TestFieldError_NoEnvVar(t *testing.T) {
	err := &FieldError{
		Field: "Port",
		Err:   fmt.Errorf("some error"),
	}
	msg := err.Error()
	assert.Contains(t, msg, "Port")
	assert.Contains(t, msg, "some error")
}

func TestVarNotSetError(t *testing.T) {
	err := &VarNotSetError{
		Field:  "APIKey",
		EnvVar: "API_KEY",
	}
	msg := err.Error()
	assert.Contains(t, msg, "API_KEY")
	assert.Contains(t, msg, "APIKey")
	assert.Contains(t, msg, "required")
}

func TestEmptyVarError(t *testing.T) {
	err := &EmptyVarError{
		Field:  "APIKey",
		EnvVar: "API_KEY",
	}
	msg := err.Error()
	assert.Contains(t, msg, "API_KEY")
	assert.Contains(t, msg, "APIKey")
	assert.Contains(t, msg, "empty")
}

func TestFileLoadError(t *testing.T) {
	inner := fmt.Errorf("permission denied")
	err := &FileLoadError{
		Field:    "Secret",
		EnvVar:   "SECRET_FILE",
		Filename: "/path/to/secret",
		Err:      inner,
	}
	msg := err.Error()
	assert.Contains(t, msg, "Secret")
	assert.Contains(t, msg, "SECRET_FILE")
	assert.Contains(t, msg, "/path/to/secret")
	assert.Contains(t, msg, "permission denied")
	assert.Equal(t, inner, err.Unwrap())
}

func TestGetErrors(t *testing.T) {
	err := &AggregateError{
		Errors: []error{
			&VarNotSetError{Field: "A", EnvVar: "A"},
			&FieldError{Field: "B", Err: fmt.Errorf("b error")},
			&VarNotSetError{Field: "C", EnvVar: "C"},
		},
	}

	varNotSetErrs := GetErrors[*VarNotSetError](err)
	assert.Len(t, varNotSetErrs, 2)
	assert.Equal(t, "A", varNotSetErrs[0].Field)
	assert.Equal(t, "C", varNotSetErrs[1].Field)

	fieldErrs := GetErrors[*FieldError](err)
	assert.Len(t, fieldErrs, 1)
	assert.Equal(t, "B", fieldErrs[0].Field)
}

func TestGetErrors_NotAggregate(t *testing.T) {
	err := fmt.Errorf("plain error")
	varNotSetErrs := GetErrors[*VarNotSetError](err)
	assert.Empty(t, varNotSetErrs)
}

func TestHasError_WithWrapping(t *testing.T) {
	inner := &VarNotSetError{Field: "A", EnvVar: "A"}
	wrapped := fmt.Errorf("wrapped: %w", inner)
	assert.True(t, HasError[*VarNotSetError](wrapped))
}

func TestParse_TextUnmarshaler(t *testing.T) {
	type Config struct {
		Time time.Time `env:"TIME"`
	}

	cfg, err := Parse[Config](WithEnvironment(map[string]string{
		"TIME": "2023-06-15T10:30:00Z",
	}))
	assert.NoError(t, err)
	expected, _ := time.Parse(time.RFC3339, "2023-06-15T10:30:00Z")
	assert.Equal(t, expected, cfg.Time)
}

func TestParse_JSONWithNonStringValue(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	content := `{
		"port": 8080,
		"debug": true,
		"rate": 0.5
	}`
	assert.NoError(t, os.WriteFile(jsonFile, []byte(content), 0644))

	type Config struct {
		Port  int     `env:"PORT"`
		Debug bool    `env:"DEBUG"`
		Rate  float64 `env:"RATE"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithJSONFile(jsonFile),
	)
	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
	assert.True(t, cfg.Debug)
	assert.Equal(t, 0.5, cfg.Rate)
}

func TestParse_JSONTypeMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	// Boolean can't be unmarshalled into string - standard JSON behavior
	content := `{"host": true}`
	assert.NoError(t, os.WriteFile(jsonFile, []byte(content), 0644))

	type Config struct {
		Host string `json:"host" env:"HOST"`
	}

	_, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithJSONFile(jsonFile),
	)
	// Standard json.Unmarshal returns error on type mismatch
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot unmarshal")
}

func TestParse_OnSetWithStageVar(t *testing.T) {
	type Config struct {
		Host string `env:"HOST"`
	}

	var setCalls []struct {
		field     string
		isDefault bool
	}

	_, err := Parse[Config](
		WithEnvironment(map[string]string{
			"PROD_HOST": "prod-host",
		}),
		WithStage("PROD"),
		WithOnSet(func(field, envVar string, value any, isDefault bool) {
			setCalls = append(setCalls, struct {
				field     string
				isDefault bool
			}{field, isDefault})
		}),
	)
	assert.NoError(t, err)
	assert.Len(t, setCalls, 1)
	assert.Equal(t, "Host", setCalls[0].field)
	assert.False(t, setCalls[0].isDefault)
}

func TestParse_MissingEnvFileSkipped(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" envDefault:"localhost"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithEnvFile("/nonexistent/.env"),
	)
	// Should succeed - missing .env files are silently skipped
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
}

func TestParse_MissingJSONFileSkipped(t *testing.T) {
	type Config struct {
		Host string `env:"HOST" envDefault:"localhost"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithJSONFile("/nonexistent/config.json"),
	)
	// Should succeed - missing JSON files are silently skipped
	assert.NoError(t, err)
	assert.Equal(t, "localhost", cfg.Host)
}

func TestParse_ConvertibleTypes(t *testing.T) {
	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "config.json")

	// JSON numbers are float64, need to convert to int
	content := `{"count": 42}`
	assert.NoError(t, os.WriteFile(jsonFile, []byte(content), 0644))

	type Config struct {
		Count int `env:"COUNT"`
	}

	cfg, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithJSONFile(jsonFile),
	)
	assert.NoError(t, err)
	assert.Equal(t, 42, cfg.Count)
}

func TestParse_OnSetWithDefaultNoEnvTag(t *testing.T) {
	type Config struct {
		Host string `envDefault:"localhost"`
	}

	var setCalls []string

	_, err := Parse[Config](
		WithEnvironment(map[string]string{}),
		WithOnSet(func(field, envVar string, value any, isDefault bool) {
			setCalls = append(setCalls, field)
		}),
	)
	assert.NoError(t, err)
	assert.Contains(t, setCalls, "Host")
}

func TestParse_UnsetWithStage(t *testing.T) {
	// Regression test: when using stage + unset, the staged variable (e.g., PROD_SECRET)
	// should be unset, not just the base variable (SECRET)
	type Config struct {
		Secret string `env:"SECRET,unset"`
	}

	// Use custom environment map to verify unset behavior
	env := map[string]string{
		"PROD_SECRET": "staged-secret-value",
		"SECRET":      "base-secret-value",
	}

	cfg, err := Parse[Config](
		WithEnvironment(env),
		WithStage("PROD"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "staged-secret-value", cfg.Secret)

	// The staged variable should be unset from the environ map
	_, hasProdSecret := env["PROD_SECRET"]
	assert.False(t, hasProdSecret, "PROD_SECRET should be unset")

	// The base variable should still exist (we only unset the one that was used)
	_, hasSecret := env["SECRET"]
	assert.True(t, hasSecret, "SECRET should still exist (wasn't used)")
}

func TestParse_UnsetWithStage_FallbackToBase(t *testing.T) {
	// When stage var doesn't exist, base var should be used and unset
	type Config struct {
		Secret string `env:"SECRET,unset"`
	}

	env := map[string]string{
		"SECRET": "base-secret-value",
	}

	cfg, err := Parse[Config](
		WithEnvironment(env),
		WithStage("PROD"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "base-secret-value", cfg.Secret)

	// The base variable should be unset (it was the one used)
	_, hasSecret := env["SECRET"]
	assert.False(t, hasSecret, "SECRET should be unset")
}
