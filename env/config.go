// Package env provides flexible configuration loading from environment variables,
// .env files, and JSON files. It supports struct tags for automatic binding,
// variable prefixes, stage-based configuration, and aggregate error handling.
//
// Basic usage:
//
//	type Config struct {
//	    Host string `env:"HOST" default:"localhost"`
//	    Port int    `env:"PORT" default:"8080"`
//	}
//
//	cfg, err := env.Parse[Config]()
//
// With options:
//
//	cfg, err := env.Parse[Config](
//	    env.WithPrefix("MYAPP"),
//	    env.WithEnvFile(".env"),
//	    env.WithJSONFile("config.json"),
//	)
package env

import (
	"encoding"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Options configures the parsing behavior.
type Options struct {
	// Environment is a custom map of environment variables.
	// If nil, os.Environ() is used.
	Environment map[string]string

	// Prefix is prepended to all env var lookups.
	// Example: Prefix="MYAPP" means HOST becomes MYAPP_HOST
	Prefix string

	// Stage enables stage-based variable resolution.
	// If set, looks for STAGE_VARNAME before VARNAME.
	// Example: Stage="PROD" means PORT looks for PROD_PORT first, then PORT.
	Stage string

	// EnvFiles is a list of .env files to load.
	// Later files override earlier ones.
	EnvFiles []string

	// JSONFiles is a list of JSON config files to load.
	// Later files override earlier ones.
	JSONFiles []string

	// TagName is the struct tag to use for env var names (default: "env").
	TagName string

	// DefaultTagName is the tag for default values (default: "default").
	DefaultTagName string

	// RequiredIfNoDefault makes fields without defaults required.
	RequiredIfNoDefault bool

	// UseFieldNameByDefault uses the field name (converted to UPPER_SNAKE_CASE)
	// if no env tag is specified.
	UseFieldNameByDefault bool

	// FuncMap provides custom parsers for specific types.
	FuncMap map[reflect.Type]ParserFunc

	// OnSet is called whenever a field value is set.
	OnSet OnSetFunc
}

// ParserFunc parses a string value into a typed value.
type ParserFunc func(value string) (any, error)

// OnSetFunc is called when a field is set.
type OnSetFunc func(fieldName, envVar string, value any, isDefault bool)

// Option is a functional option for configuring parsing.
type Option func(*Options)

// WithPrefix sets the environment variable prefix.
func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

// WithStage enables stage-based variable resolution.
// Example: WithStage("PROD") causes PORT to look for PROD_PORT first.
func WithStage(stage string) Option {
	return func(o *Options) {
		o.Stage = stage
	}
}

// WithEnvFile adds .env files to load (in order, later overrides earlier).
func WithEnvFile(files ...string) Option {
	return func(o *Options) {
		o.EnvFiles = append(o.EnvFiles, files...)
	}
}

// WithJSONFile adds JSON config files to load (in order, later overrides earlier).
func WithJSONFile(files ...string) Option {
	return func(o *Options) {
		o.JSONFiles = append(o.JSONFiles, files...)
	}
}

// WithEnvironment uses a custom environment map instead of os.Environ().
func WithEnvironment(env map[string]string) Option {
	return func(o *Options) {
		o.Environment = env
	}
}

// WithTagName sets custom tag names.
func WithTagName(envTag, defaultTag string) Option {
	return func(o *Options) {
		if envTag != "" {
			o.TagName = envTag
		}
		if defaultTag != "" {
			o.DefaultTagName = defaultTag
		}
	}
}

// WithRequiredIfNoDefault makes fields without defaults required.
func WithRequiredIfNoDefault() Option {
	return func(o *Options) {
		o.RequiredIfNoDefault = true
	}
}

// WithUseFieldName uses field names as env var names when no tag is present.
func WithUseFieldName() Option {
	return func(o *Options) {
		o.UseFieldNameByDefault = true
	}
}

// WithParser adds a custom parser for a specific type.
func WithParser[T any](parser func(string) (T, error)) Option {
	return func(o *Options) {
		if o.FuncMap == nil {
			o.FuncMap = make(map[reflect.Type]ParserFunc)
		}
		var zero T
		o.FuncMap[reflect.TypeOf(zero)] = func(s string) (any, error) {
			return parser(s)
		}
	}
}

// WithOnSet registers a callback for when fields are set.
func WithOnSet(fn OnSetFunc) Option {
	return func(o *Options) {
		o.OnSet = fn
	}
}

// Parse parses environment variables into a struct of type T.
func Parse[T any](opts ...Option) (T, error) {
	var result T
	if err := ParseInto(&result, opts...); err != nil {
		return result, err
	}
	return result, nil
}

// Must wraps Parse and panics on error.
func Must[T any](opts ...Option) T {
	result, err := Parse[T](opts...)
	if err != nil {
		panic(fmt.Sprintf("env: %v", err))
	}
	return result
}

// ParseInto parses environment variables into an existing struct pointer.
func ParseInto(v any, opts ...Option) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return &ParseError{Err: fmt.Errorf("expected non-nil pointer to struct, got %T", v)}
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return &ParseError{Err: fmt.Errorf("expected pointer to struct, got pointer to %s", rv.Kind())}
	}

	// Build options
	options := &Options{
		TagName:        "env",
		DefaultTagName: "default",
	}
	for _, opt := range opts {
		opt(options)
	}

	// Load environment
	environ := options.Environment
	if environ == nil {
		environ = make(map[string]string)
		for _, e := range os.Environ() {
			if k, v, ok := strings.Cut(e, "="); ok {
				environ[k] = v
			}
		}
	}

	// Load .env files
	for _, file := range options.EnvFiles {
		if envVars, err := ReadEnvFile(file); err == nil {
			for k, v := range envVars {
				if _, exists := environ[k]; !exists {
					environ[k] = v
				}
			}
		}
		// Silently skip missing .env files
	}

	// Load JSON files
	jsonValues := make(map[string]any)
	for _, file := range options.JSONFiles {
		if values, err := ReadJSONFile(file); err == nil {
			for k, v := range values {
				jsonValues[k] = v
			}
		}
		// Silently skip missing JSON files
	}

	// Parse struct
	parser := &structParser{
		options:    options,
		environ:    environ,
		jsonValues: jsonValues,
		errors:     &AggregateError{},
	}

	parser.parseStruct(rv, "")

	if len(parser.errors.Errors) > 0 {
		return parser.errors
	}
	return nil
}

// structParser handles the reflection-based parsing.
type structParser struct {
	options    *Options
	environ    map[string]string
	jsonValues map[string]any
	errors     *AggregateError
}

func (p *structParser) parseStruct(rv reflect.Value, prefix string) {
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fv := rv.Field(i)

		if !fv.CanSet() {
			continue
		}

		// Handle embedded structs
		if field.Anonymous && fv.Kind() == reflect.Struct {
			p.parseStruct(fv, prefix)
			continue
		}

		// Handle nested structs with envPrefix tag
		if fv.Kind() == reflect.Struct && !isSpecialType(fv.Type()) {
			nestedPrefix := prefix
			if prefixTag := field.Tag.Get("envPrefix"); prefixTag != "" {
				nestedPrefix = prefix + prefixTag
			} else {
				nestedPrefix = prefix + toUpperSnakeCase(field.Name) + "_"
			}
			p.parseStruct(fv, nestedPrefix)
			continue
		}

		// Handle pointer to struct
		if fv.Kind() == reflect.Ptr && fv.Type().Elem().Kind() == reflect.Struct {
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			nestedPrefix := prefix
			if prefixTag := field.Tag.Get("envPrefix"); prefixTag != "" {
				nestedPrefix = prefix + prefixTag
			} else {
				nestedPrefix = prefix + toUpperSnakeCase(field.Name) + "_"
			}
			p.parseStruct(fv.Elem(), nestedPrefix)
			continue
		}

		p.parseField(field, fv, prefix)
	}
}

func (p *structParser) parseField(field reflect.StructField, fv reflect.Value, prefix string) {
	// Get tag values
	envTag := field.Tag.Get(p.options.TagName)
	defaultValue := field.Tag.Get(p.options.DefaultTagName)

	// Parse tag options (env:"VAR,required,file")
	var envVar string
	var required, notEmpty, loadFile, expand bool

	if envTag == "-" {
		return // Ignored field
	}

	if envTag != "" {
		parts := strings.Split(envTag, ",")
		envVar = parts[0]
		for _, opt := range parts[1:] {
			switch opt {
			case "required":
				required = true
			case "notEmpty":
				notEmpty = true
			case "file":
				loadFile = true
			case "expand":
				expand = true
			}
		}
	}

	// Determine env var name
	if envVar == "" {
		if p.options.UseFieldNameByDefault {
			envVar = toUpperSnakeCase(field.Name)
		} else {
			// No env var, just apply default if present
			if defaultValue != "" {
				if err := p.setFieldValue(fv, defaultValue, field.Type); err != nil {
					p.errors.Errors = append(p.errors.Errors, &FieldError{
						Field: field.Name,
						Err:   err,
					})
				} else if p.options.OnSet != nil {
					p.options.OnSet(field.Name, "", fv.Interface(), true)
				}
			}
			return
		}
	}

	// Build full env var name with prefix
	fullEnvVar := prefix + p.options.Prefix
	if fullEnvVar != "" && !strings.HasSuffix(fullEnvVar, "_") {
		fullEnvVar += "_"
	}
	fullEnvVar += envVar

	// Try stage-prefixed var first
	var value string
	var found bool

	if p.options.Stage != "" {
		stageVar := p.options.Stage + "_" + fullEnvVar
		if v, ok := p.environ[stageVar]; ok {
			value = v
			found = true
		}
	}

	// Try regular var
	if !found {
		if v, ok := p.environ[fullEnvVar]; ok {
			value = v
			found = true
		}
	}

	// Try JSON values (lowercase key matching)
	if !found {
		jsonKey := strings.ToLower(envVar)
		if v, ok := p.jsonValues[jsonKey]; ok {
			if s, ok := v.(string); ok {
				value = s
				found = true
			} else {
				// Handle non-string JSON values
				if err := p.setFieldFromAny(fv, v); err != nil {
					p.errors.Errors = append(p.errors.Errors, &FieldError{
						Field:  field.Name,
						EnvVar: fullEnvVar,
						Err:    err,
					})
				} else if p.options.OnSet != nil {
					p.options.OnSet(field.Name, fullEnvVar, fv.Interface(), false)
				}
				return
			}
		}
	}

	// Apply default if not found
	if !found {
		if defaultValue != "" {
			value = defaultValue
			found = true
		} else if required || (p.options.RequiredIfNoDefault && defaultValue == "") {
			p.errors.Errors = append(p.errors.Errors, &VarNotSetError{
				Field:  field.Name,
				EnvVar: fullEnvVar,
			})
			return
		} else {
			return
		}
	}

	// Check notEmpty
	if found && value == "" && notEmpty {
		p.errors.Errors = append(p.errors.Errors, &EmptyVarError{
			Field:  field.Name,
			EnvVar: fullEnvVar,
		})
		return
	}

	// Handle file loading
	if loadFile && value != "" {
		content, err := os.ReadFile(value)
		if err != nil {
			p.errors.Errors = append(p.errors.Errors, &FileLoadError{
				Field:    field.Name,
				EnvVar:   fullEnvVar,
				Filename: value,
				Err:      err,
			})
			return
		}
		value = string(content)
	}

	// Handle variable expansion
	if expand {
		value = os.ExpandEnv(value)
	}

	// Set the field value
	if err := p.setFieldValue(fv, value, field.Type); err != nil {
		p.errors.Errors = append(p.errors.Errors, &FieldError{
			Field:  field.Name,
			EnvVar: fullEnvVar,
			Value:  value,
			Err:    err,
		})
		return
	}

	if p.options.OnSet != nil {
		isDefault := !p.hasEnvVar(fullEnvVar)
		p.options.OnSet(field.Name, fullEnvVar, fv.Interface(), isDefault)
	}
}

func (p *structParser) hasEnvVar(name string) bool {
	if p.options.Stage != "" {
		if _, ok := p.environ[p.options.Stage+"_"+name]; ok {
			return true
		}
	}
	_, ok := p.environ[name]
	return ok
}

func (p *structParser) setFieldValue(fv reflect.Value, value string, t reflect.Type) error {
	// Check for custom parser
	if p.options.FuncMap != nil {
		if parser, ok := p.options.FuncMap[t]; ok {
			parsed, err := parser(value)
			if err != nil {
				return err
			}
			fv.Set(reflect.ValueOf(parsed))
			return nil
		}
	}

	// Handle pointer types
	if fv.Kind() == reflect.Ptr {
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return p.setFieldValue(fv.Elem(), value, fv.Type().Elem())
	}

	// Check for TextUnmarshaler
	if fv.CanAddr() {
		if tu, ok := fv.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return tu.UnmarshalText([]byte(value))
		}
	}

	// Built-in types
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(value)

	case reflect.Bool:
		b, err := parseBool(value)
		if err != nil {
			return err
		}
		fv.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special case for time.Duration
		if fv.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			fv.SetInt(int64(d))
			return nil
		}
		i, err := strconv.ParseInt(value, 0, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetInt(i)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 0, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetUint(u)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, fv.Type().Bits())
		if err != nil {
			return err
		}
		fv.SetFloat(f)

	case reflect.Slice:
		return p.setSliceValue(fv, value)

	case reflect.Map:
		return p.setMapValue(fv, value)

	default:
		return fmt.Errorf("unsupported type: %s", fv.Kind())
	}

	return nil
}

func (p *structParser) setSliceValue(fv reflect.Value, value string) error {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	slice := reflect.MakeSlice(fv.Type(), len(parts), len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if err := p.setFieldValue(slice.Index(i), part, fv.Type().Elem()); err != nil {
			return fmt.Errorf("element %d: %w", i, err)
		}
	}

	fv.Set(slice)
	return nil
}

func (p *structParser) setMapValue(fv reflect.Value, value string) error {
	if value == "" {
		return nil
	}

	m := reflect.MakeMap(fv.Type())
	pairs := strings.Split(value, ",")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		k, v, ok := strings.Cut(pair, ":")
		if !ok {
			return fmt.Errorf("invalid map entry: %q (expected key:value)", pair)
		}

		key := reflect.New(fv.Type().Key()).Elem()
		val := reflect.New(fv.Type().Elem()).Elem()

		if err := p.setFieldValue(key, strings.TrimSpace(k), fv.Type().Key()); err != nil {
			return fmt.Errorf("map key %q: %w", k, err)
		}
		if err := p.setFieldValue(val, strings.TrimSpace(v), fv.Type().Elem()); err != nil {
			return fmt.Errorf("map value for %q: %w", k, err)
		}

		m.SetMapIndex(key, val)
	}

	fv.Set(m)
	return nil
}

func (p *structParser) setFieldFromAny(fv reflect.Value, val any) error {
	rv := reflect.ValueOf(val)
	if rv.Type().AssignableTo(fv.Type()) {
		fv.Set(rv)
		return nil
	}

	// Try to convert
	if rv.Type().ConvertibleTo(fv.Type()) {
		fv.Set(rv.Convert(fv.Type()))
		return nil
	}

	// Fall back to string conversion
	return p.setFieldValue(fv, fmt.Sprint(val), fv.Type())
}

// isSpecialType returns true for types that should be parsed as values, not structs.
func isSpecialType(t reflect.Type) bool {
	// time.Time, url.URL, etc.
	return t.Implements(reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()) ||
		reflect.PointerTo(t).Implements(reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem())
}

// toUpperSnakeCase converts CamelCase to UPPER_SNAKE_CASE.
func toUpperSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		if r >= 'a' && r <= 'z' {
			result.WriteByte(byte(r - 'a' + 'A'))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// parseBool parses various boolean representations.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes", "on", "t", "y":
		return true, nil
	case "false", "0", "no", "off", "f", "n", "":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %q", s)
	}
}
