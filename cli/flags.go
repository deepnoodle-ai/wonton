package cli

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// ParseFlags parses a struct with flag tags and populates a command's flags.
// Supports tags: flag, default, help, env, enum, required
//
// Example:
//
//	type Flags struct {
//	    Model string  `flag:"model,m" default:"claude-sonnet" help:"Model to use" env:"MODEL"`
//	    Temp  float64 `flag:"temperature,t" default:"0.7" help:"Temperature"`
//	    Debug bool    `flag:"debug,d" help:"Enable debug mode"`
//	    Format string `flag:"format,f" enum:"text,json,markdown" default:"text"`
//	}
func ParseFlags[T any](c *Command) *T {
	var result T
	parseStructFlags(c, reflect.TypeOf(result))
	return &result
}

// BindFlags binds parsed flag values to a struct.
func BindFlags[T any](ctx *Context) (*T, error) {
	var result T
	rv := reflect.ValueOf(&result).Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("flag")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")
		name := parts[0]

		if !ctx.IsSet(name) {
			// Try default from tag
			if def := field.Tag.Get("default"); def != "" {
				if err := setFieldValue(rv.Field(i), def); err != nil {
					return nil, fmt.Errorf("invalid default for %s: %w", name, err)
				}
			}
			continue
		}

		value := ctx.flags[name]
		if err := setFieldFromAny(rv.Field(i), value); err != nil {
			return nil, fmt.Errorf("invalid value for --%s: %w", name, err)
		}
	}

	return &result, nil
}

func parseStructFlags(c *Command, t reflect.Type) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("flag")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")
		flag := &Flag{
			Name:        parts[0],
			Description: field.Tag.Get("help"),
			EnvVar:      field.Tag.Get("env"),
		}

		// Short flag
		if len(parts) > 1 {
			flag.Short = parts[1]
		}

		// Default value
		if def := field.Tag.Get("default"); def != "" {
			flag.Default = parseDefaultValue(field.Type, def)
		} else {
			// Set type-appropriate zero default
			switch field.Type.Kind() {
			case reflect.Bool:
				flag.Default = false
			case reflect.String:
				flag.Default = ""
			case reflect.Int, reflect.Int64:
				flag.Default = 0
			case reflect.Float64:
				flag.Default = 0.0
			}
		}

		// Enum values
		if enum := field.Tag.Get("enum"); enum != "" {
			flag.Enum = strings.Split(enum, ",")
		}

		// Required
		if _, ok := field.Tag.Lookup("required"); ok {
			flag.Required = true
		}

		// Hidden
		if _, ok := field.Tag.Lookup("hidden"); ok {
			flag.Hidden = true
		}

		c.flags = append(c.flags, flag)
	}
}

func parseDefaultValue(t reflect.Type, s string) any {
	switch t.Kind() {
	case reflect.Bool:
		return s == "true" || s == "1"
	case reflect.Int:
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
		return 0
	case reflect.Int64:
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
		return int64(0)
	case reflect.Float64:
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
		return 0.0
	default:
		return s
	}
}

func setFieldValue(v reflect.Value, s string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Bool:
		v.SetBool(s == "true" || s == "1" || s == "yes")
	case reflect.Int, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported type: %s", v.Kind())
	}
	return nil
}

func setFieldFromAny(v reflect.Value, val any) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(fmt.Sprint(val))
	case reflect.Bool:
		switch b := val.(type) {
		case bool:
			v.SetBool(b)
		case string:
			v.SetBool(b == "true" || b == "1" || b == "yes")
		}
	case reflect.Int, reflect.Int64:
		switch i := val.(type) {
		case int:
			v.SetInt(int64(i))
		case int64:
			v.SetInt(i)
		case float64:
			v.SetInt(int64(i))
		case string:
			n, err := strconv.ParseInt(i, 10, 64)
			if err != nil {
				return err
			}
			v.SetInt(n)
		}
	case reflect.Float64:
		switch f := val.(type) {
		case float64:
			v.SetFloat(f)
		case int:
			v.SetFloat(float64(f))
		case int64:
			v.SetFloat(float64(f))
		case string:
			n, err := strconv.ParseFloat(f, 64)
			if err != nil {
				return err
			}
			v.SetFloat(n)
		}
	default:
		return fmt.Errorf("unsupported type: %s", v.Kind())
	}
	return nil
}

// lookupEnv is a helper that returns env var value if set.
func lookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
