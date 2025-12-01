package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents hierarchical configuration.
// Resolution order: flag > env > project config > user config > default
type Config struct {
	paths  []string
	values map[string]any
}

// ConfigOption configures the config loader.
type ConfigOption func(*configLoader)

type configLoader struct {
	paths      []string
	envPrefix  string
	structType reflect.Type
}

// WithConfigPaths sets config file paths to search.
// Paths can include ~ for home directory.
func WithConfigPaths(paths ...string) ConfigOption {
	return func(l *configLoader) {
		l.paths = paths
	}
}

// WithEnvPrefix sets the environment variable prefix.
func WithEnvPrefix(prefix string) ConfigOption {
	return func(l *configLoader) {
		l.envPrefix = prefix
	}
}

// LoadConfig loads configuration from files and environment.
// Supports YAML config files with hierarchical resolution.
func LoadConfig[T any](opts ...ConfigOption) (*T, error) {
	var result T
	loader := &configLoader{
		structType: reflect.TypeOf(result),
	}

	for _, opt := range opts {
		opt(loader)
	}

	// Load config files in order (later files override earlier)
	values := make(map[string]any)
	for _, path := range loader.paths {
		expanded := expandPath(path)
		if data, err := os.ReadFile(expanded); err == nil {
			var fileValues map[string]any
			if err := yaml.Unmarshal(data, &fileValues); err == nil {
				mergeConfig(values, fileValues)
			}
		}
	}

	// Apply to struct
	rv := reflect.ValueOf(&result).Elem()
	if err := applyConfig(rv, values, loader.envPrefix); err != nil {
		return nil, err
	}

	return &result, nil
}

// BindConfig binds configuration to a struct using tags.
// Supports tags: yaml, env, default
//
// Example:
//
//	type Config struct {
//	    APIKey   string `yaml:"api_key" env:"API_KEY"`
//	    Model    string `yaml:"model" env:"MODEL" default:"claude-sonnet"`
//	    Endpoint string `yaml:"endpoint" env:"ENDPOINT"`
//	}
func BindConfig[T any](ctx *Context, configPaths ...string) (*T, error) {
	var result T

	// Load from config files
	values := make(map[string]any)
	for _, path := range configPaths {
		expanded := expandPath(path)
		if data, err := os.ReadFile(expanded); err == nil {
			var fileValues map[string]any
			if err := yaml.Unmarshal(data, &fileValues); err == nil {
				mergeConfig(values, fileValues)
			}
		}
	}

	// Apply values to struct
	rv := reflect.ValueOf(&result).Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fv := rv.Field(i)

		// Get value from config file (yaml tag)
		yamlKey := field.Tag.Get("yaml")
		if yamlKey == "" {
			yamlKey = strings.ToLower(field.Name)
		}

		// Check config value
		if v, ok := values[yamlKey]; ok {
			if err := setFieldFromAny(fv, v); err != nil {
				continue // Skip on error
			}
		}

		// Override with env var
		if envKey := field.Tag.Get("env"); envKey != "" {
			if v, ok := os.LookupEnv(envKey); ok {
				setFieldValue(fv, v)
			}
		}

		// Override with flag
		flagName := field.Tag.Get("flag")
		if flagName == "" {
			parts := strings.Split(field.Tag.Get("flag"), ",")
			if len(parts) > 0 {
				flagName = parts[0]
			}
		}
		if flagName != "" && ctx.IsSet(flagName) {
			setFieldFromAny(fv, ctx.flags[flagName])
		}

		// Apply default if still zero
		if fv.IsZero() {
			if def := field.Tag.Get("default"); def != "" {
				setFieldValue(fv, def)
			}
		}
	}

	return &result, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func mergeConfig(dst, src map[string]any) {
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			if dstMap, ok := dst[k].(map[string]any); ok {
				mergeConfig(dstMap, srcMap)
				continue
			}
		}
		dst[k] = v
	}
}

func applyConfig(v reflect.Value, values map[string]any, envPrefix string) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)

		// Get yaml key
		yamlKey := field.Tag.Get("yaml")
		if yamlKey == "" {
			yamlKey = strings.ToLower(field.Name)
		}

		// Get env var name
		envKey := field.Tag.Get("env")
		if envKey == "" && envPrefix != "" {
			envKey = envPrefix + "_" + strings.ToUpper(field.Name)
		}

		// Priority: env > config > default
		var value any
		var found bool

		// Check env first (highest priority)
		if envKey != "" {
			if v, ok := os.LookupEnv(envKey); ok {
				value = v
				found = true
			}
		}

		// Check config file
		if !found {
			if v, ok := values[yamlKey]; ok {
				value = v
				found = true
			}
		}

		// Apply default if not found
		if !found {
			if def := field.Tag.Get("default"); def != "" {
				value = def
				found = true
			}
		}

		if found && value != nil {
			setFieldFromAny(fv, value)
		}
	}

	return nil
}

// GetString gets a string value from config with fallback chain.
func (c *Config) GetString(key string, fallback ...string) string {
	if v, ok := c.values[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}

// GetInt gets an int value from config with fallback chain.
func (c *Config) GetInt(key string, fallback ...int) int {
	if v, ok := c.values[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return 0
}

// GetBool gets a bool value from config with fallback chain.
func (c *Config) GetBool(key string, fallback ...bool) bool {
	if v, ok := c.values[key]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case string:
			return val == "true" || val == "1" || val == "yes"
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return false
}
