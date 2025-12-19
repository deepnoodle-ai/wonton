package cli

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ParseFlags parses a struct with flag tags and adds flags to a command.
//
// This provides a declarative way to define flags using struct tags.
// Supports tags: flag, default, help, env, enum, required, hidden
//
// Example:
//
//	type Flags struct {
//	    Model string  `flag:"model,m" default:"claude-sonnet" help:"Model to use" env:"MODEL"`
//	    Temp  float64 `flag:"temperature,t" default:"0.7" help:"Temperature"`
//	    Debug bool    `flag:"debug,d" help:"Enable debug mode"`
//	    Format string `flag:"format,f" enum:"text,json,markdown" default:"text"`
//	}
//
//	cmd := app.Command("chat")
//	ParseFlags[Flags](cmd)
//
// Use BindFlags in the handler to populate the struct with parsed values.
func ParseFlags[T any](c *Command) *T {
	var result T
	parseStructFlags(c, reflect.TypeOf(result))
	return &result
}

// BindFlags binds parsed flag values to a struct.
//
// This populates a struct with values from the context based on flag tags:
//
//	type Flags struct {
//	    Name string `flag:"name,n"`
//	    Port int    `flag:"port,p" default:"8080"`
//	}
//
//	func handler(ctx *cli.Context) error {
//	    flags, err := cli.BindFlags[Flags](ctx)
//	    if err != nil {
//	        return err
//	    }
//	    // Use flags.Name, flags.Port
//	    return nil
//	}
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
		name := parts[0]
		short := ""
		if len(parts) > 1 {
			short = parts[1]
		}

		help := field.Tag.Get("help")
		envVar := field.Tag.Get("env")
		enumStr := field.Tag.Get("enum")
		var enum []string
		if enumStr != "" {
			enum = strings.Split(enumStr, ",")
		}
		_, required := field.Tag.Lookup("required")
		_, hidden := field.Tag.Lookup("hidden")

		// Create typed flag based on field type
		switch field.Type.Kind() {
		case reflect.Bool:
			defVal := false
			if def := field.Tag.Get("default"); def != "" {
				defVal = def == "true" || def == "1"
			}
			c.flags = append(c.flags, &BoolFlag{
				Name:     name,
				Short:    short,
				Help:     help,
				Value:    defVal,
				EnvVar:   envVar,
				Required: required,
				Hidden:   hidden,
			})
		case reflect.String:
			defVal := field.Tag.Get("default")
			c.flags = append(c.flags, &StringFlag{
				Name:     name,
				Short:    short,
				Help:     help,
				Value:    defVal,
				EnvVar:   envVar,
				Required: required,
				Hidden:   hidden,
				Enum:     enum,
			})
		case reflect.Int, reflect.Int64:
			defVal := 0
			if def := field.Tag.Get("default"); def != "" {
				if v, err := strconv.Atoi(def); err == nil {
					defVal = v
				}
			}
			c.flags = append(c.flags, &IntFlag{
				Name:     name,
				Short:    short,
				Help:     help,
				Value:    defVal,
				EnvVar:   envVar,
				Required: required,
				Hidden:   hidden,
			})
		case reflect.Float64:
			defVal := 0.0
			if def := field.Tag.Get("default"); def != "" {
				if v, err := strconv.ParseFloat(def, 64); err == nil {
					defVal = v
				}
			}
			c.flags = append(c.flags, &Float64Flag{
				Name:     name,
				Short:    short,
				Help:     help,
				Value:    defVal,
				EnvVar:   envVar,
				Required: required,
				Hidden:   hidden,
			})
		}
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

// -----------------------------------------------------------------------------
// Flag Builders
//
// The flag builder functions provide a fluent API for defining typed flags.
//
// Example:
//
//	app.Command("deploy").
//	    Flags(
//	        cli.String("env", "e").Default("staging").Help("Environment"),
//	        cli.Bool("force", "f").Help("Force deployment"),
//	        cli.Int("replicas", "r").Default(3).Help("Number of replicas"),
//	    )
//
// All flag builders support these methods:
//   - Default(v): Set the default value
//   - Help(s): Set the help text
//   - Env(s): Bind to an environment variable
//   - Required(): Mark as required
//   - Hidden(): Hide from help output
//
// String flags also support:
//   - Enum(values...): Restrict to a set of valid values
//   - ValidateWith(fn): Add custom validation
// -----------------------------------------------------------------------------

// String creates a new string flag builder.
//
// The first parameter is the long flag name, the second is the optional short name:
//
//	cli.String("output", "o").Default("output.txt").Help("Output file")
func String(name, short string) *stringBuilder {
	return &stringBuilder{name: name, short: short}
}

type stringBuilder struct {
	name, short, help, envVar string
	value                     string
	enum                      []string
	required, hidden          bool
	validator                 func(string) error
}

func (b *stringBuilder) Default(v string) *stringBuilder { b.value = v; return b }
func (b *stringBuilder) Help(v string) *stringBuilder    { b.help = v; return b }
func (b *stringBuilder) Env(v string) *stringBuilder     { b.envVar = v; return b }
func (b *stringBuilder) Enum(v ...string) *stringBuilder { b.enum = v; return b }
func (b *stringBuilder) Required() *stringBuilder        { b.required = true; return b }
func (b *stringBuilder) Hidden() *stringBuilder          { b.hidden = true; return b }
func (b *stringBuilder) ValidateWith(f func(string) error) *stringBuilder {
	b.validator = f
	return b
}

func (b *stringBuilder) GetName() string   { return b.name }
func (b *stringBuilder) GetShort() string  { return b.short }
func (b *stringBuilder) GetHelp() string   { return b.help }
func (b *stringBuilder) GetEnvVar() string { return b.envVar }
func (b *stringBuilder) GetDefault() any   { return b.value }
func (b *stringBuilder) IsRequired() bool  { return b.required }
func (b *stringBuilder) IsHidden() bool    { return b.hidden }
func (b *stringBuilder) GetEnum() []string { return b.enum }
func (b *stringBuilder) Validate(value string) error {
	if b.validator != nil {
		return b.validator(value)
	}
	return nil
}

// Bool creates a new boolean flag builder.
//
//	cli.Bool("verbose", "v").Help("Enable verbose output")
func Bool(name, short string) *boolBuilder {
	return &boolBuilder{name: name, short: short}
}

type boolBuilder struct {
	name, short, help, envVar string
	value                     bool
	required, hidden          bool
}

func (b *boolBuilder) Default(v bool) *boolBuilder { b.value = v; return b }
func (b *boolBuilder) Help(v string) *boolBuilder  { b.help = v; return b }
func (b *boolBuilder) Env(v string) *boolBuilder   { b.envVar = v; return b }
func (b *boolBuilder) Required() *boolBuilder      { b.required = true; return b }
func (b *boolBuilder) Hidden() *boolBuilder        { b.hidden = true; return b }

func (b *boolBuilder) GetName() string       { return b.name }
func (b *boolBuilder) GetShort() string      { return b.short }
func (b *boolBuilder) GetHelp() string       { return b.help }
func (b *boolBuilder) GetEnvVar() string     { return b.envVar }
func (b *boolBuilder) GetDefault() any       { return b.value }
func (b *boolBuilder) IsRequired() bool      { return b.required }
func (b *boolBuilder) IsHidden() bool        { return b.hidden }
func (b *boolBuilder) GetEnum() []string     { return nil }
func (b *boolBuilder) Validate(string) error { return nil }

// Int creates a new integer flag builder.
//
//	cli.Int("port", "p").Default(8080).Help("Server port")
func Int(name, short string) *intBuilder {
	return &intBuilder{name: name, short: short}
}

type intBuilder struct {
	name, short, help, envVar string
	value                     int
	required, hidden          bool
	validator                 func(int) error
}

func (b *intBuilder) Default(v int) *intBuilder { b.value = v; return b }
func (b *intBuilder) Help(v string) *intBuilder { b.help = v; return b }
func (b *intBuilder) Env(v string) *intBuilder  { b.envVar = v; return b }
func (b *intBuilder) Required() *intBuilder     { b.required = true; return b }
func (b *intBuilder) Hidden() *intBuilder       { b.hidden = true; return b }
func (b *intBuilder) ValidateWith(f func(int) error) *intBuilder {
	b.validator = f
	return b
}

func (b *intBuilder) GetName() string       { return b.name }
func (b *intBuilder) GetShort() string      { return b.short }
func (b *intBuilder) GetHelp() string       { return b.help }
func (b *intBuilder) GetEnvVar() string     { return b.envVar }
func (b *intBuilder) GetDefault() any       { return b.value }
func (b *intBuilder) IsRequired() bool      { return b.required }
func (b *intBuilder) IsHidden() bool        { return b.hidden }
func (b *intBuilder) GetEnum() []string     { return nil }
func (b *intBuilder) Validate(string) error { return nil }

// Float creates a new float64 flag builder.
//
//	cli.Float("ratio", "r").Default(0.5).Help("Compression ratio")
func Float(name, short string) *floatBuilder {
	return &floatBuilder{name: name, short: short}
}

type floatBuilder struct {
	name, short, help, envVar string
	value                     float64
	required, hidden          bool
	validator                 func(float64) error
}

func (b *floatBuilder) Default(v float64) *floatBuilder { b.value = v; return b }
func (b *floatBuilder) Help(v string) *floatBuilder     { b.help = v; return b }
func (b *floatBuilder) Env(v string) *floatBuilder      { b.envVar = v; return b }
func (b *floatBuilder) Required() *floatBuilder         { b.required = true; return b }
func (b *floatBuilder) Hidden() *floatBuilder           { b.hidden = true; return b }
func (b *floatBuilder) ValidateWith(f func(float64) error) *floatBuilder {
	b.validator = f
	return b
}

func (b *floatBuilder) GetName() string       { return b.name }
func (b *floatBuilder) GetShort() string      { return b.short }
func (b *floatBuilder) GetHelp() string       { return b.help }
func (b *floatBuilder) GetEnvVar() string     { return b.envVar }
func (b *floatBuilder) GetDefault() any       { return b.value }
func (b *floatBuilder) IsRequired() bool      { return b.required }
func (b *floatBuilder) IsHidden() bool        { return b.hidden }
func (b *floatBuilder) GetEnum() []string     { return nil }
func (b *floatBuilder) Validate(string) error { return nil }

// Duration creates a new duration flag builder.
//
//	cli.Duration("timeout", "t").Default(30*time.Second).Help("Request timeout")
func Duration(name, short string) *durationBuilder {
	return &durationBuilder{name: name, short: short}
}

type durationBuilder struct {
	name, short, help, envVar string
	value                     time.Duration
	required, hidden          bool
}

func (b *durationBuilder) Default(v time.Duration) *durationBuilder { b.value = v; return b }
func (b *durationBuilder) Help(v string) *durationBuilder           { b.help = v; return b }
func (b *durationBuilder) Env(v string) *durationBuilder            { b.envVar = v; return b }
func (b *durationBuilder) Required() *durationBuilder               { b.required = true; return b }
func (b *durationBuilder) Hidden() *durationBuilder                 { b.hidden = true; return b }

func (b *durationBuilder) GetName() string       { return b.name }
func (b *durationBuilder) GetShort() string      { return b.short }
func (b *durationBuilder) GetHelp() string       { return b.help }
func (b *durationBuilder) GetEnvVar() string     { return b.envVar }
func (b *durationBuilder) GetDefault() any       { return b.value }
func (b *durationBuilder) IsRequired() bool      { return b.required }
func (b *durationBuilder) IsHidden() bool        { return b.hidden }
func (b *durationBuilder) GetEnum() []string     { return nil }
func (b *durationBuilder) Validate(string) error { return nil }

// Strings creates a new string slice flag builder.
//
// Can be specified multiple times on the command line:
//
//	cli.Strings("include", "i").Help("Paths to include")
//	// Usage: --include /foo --include /bar
func Strings(name, short string) *stringsBuilder {
	return &stringsBuilder{name: name, short: short}
}

type stringsBuilder struct {
	name, short, help, envVar string
	value                     []string
	required, hidden          bool
}

func (b *stringsBuilder) Default(v ...string) *stringsBuilder { b.value = v; return b }
func (b *stringsBuilder) Help(v string) *stringsBuilder       { b.help = v; return b }
func (b *stringsBuilder) Env(v string) *stringsBuilder        { b.envVar = v; return b }
func (b *stringsBuilder) Required() *stringsBuilder           { b.required = true; return b }
func (b *stringsBuilder) Hidden() *stringsBuilder             { b.hidden = true; return b }

func (b *stringsBuilder) GetName() string       { return b.name }
func (b *stringsBuilder) GetShort() string      { return b.short }
func (b *stringsBuilder) GetHelp() string       { return b.help }
func (b *stringsBuilder) GetEnvVar() string     { return b.envVar }
func (b *stringsBuilder) GetDefault() any       { return b.value }
func (b *stringsBuilder) IsRequired() bool      { return b.required }
func (b *stringsBuilder) IsHidden() bool        { return b.hidden }
func (b *stringsBuilder) GetEnum() []string     { return nil }
func (b *stringsBuilder) Validate(string) error { return nil }

// Ints creates a new int slice flag builder.
//
// Can be specified multiple times on the command line:
//
//	cli.Ints("port", "p").Help("Ports to listen on")
//	// Usage: --port 8080 --port 8081
func Ints(name, short string) *intsBuilder {
	return &intsBuilder{name: name, short: short}
}

type intsBuilder struct {
	name, short, help, envVar string
	value                     []int
	required, hidden          bool
}

func (b *intsBuilder) Default(v ...int) *intsBuilder { b.value = v; return b }
func (b *intsBuilder) Help(v string) *intsBuilder    { b.help = v; return b }
func (b *intsBuilder) Env(v string) *intsBuilder     { b.envVar = v; return b }
func (b *intsBuilder) Required() *intsBuilder        { b.required = true; return b }
func (b *intsBuilder) Hidden() *intsBuilder          { b.hidden = true; return b }

func (b *intsBuilder) GetName() string       { return b.name }
func (b *intsBuilder) GetShort() string      { return b.short }
func (b *intsBuilder) GetHelp() string       { return b.help }
func (b *intsBuilder) GetEnvVar() string     { return b.envVar }
func (b *intsBuilder) GetDefault() any       { return b.value }
func (b *intsBuilder) IsRequired() bool      { return b.required }
func (b *intsBuilder) IsHidden() bool        { return b.hidden }
func (b *intsBuilder) GetEnum() []string     { return nil }
func (b *intsBuilder) Validate(string) error { return nil }
