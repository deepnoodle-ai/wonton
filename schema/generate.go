package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// GenerateOptions configures schema generation behavior.
type GenerateOptions struct {
	// DisallowAdditionalProperties sets additionalProperties: false on objects.
	// This is required by OpenAI's strict mode but optional for Anthropic.
	DisallowAdditionalProperties bool
}

// Generate creates a JSON Schema from a Go type using reflection.
// It analyzes the type structure and returns a Schema suitable for
// LLM tool definitions or JSON validation.
//
// For struct types, Generate examines struct tags to determine property
// names, descriptions, constraints, and required status. See package
// documentation for supported tags.
//
// Example:
//
//	type SearchParams struct {
//	    Query   string   `json:"query" description:"Search query string"`
//	    Limit   int      `json:"limit,omitempty" description:"Max results" default:"10"`
//	    Filters []string `json:"filters,omitempty" description:"Filter expressions"`
//	}
//
//	schema, err := Generate(SearchParams{})
func Generate(v any, opts ...GenerateOptions) (*Schema, error) {
	var opt GenerateOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	t := reflect.TypeOf(v)
	if t == nil {
		return nil, fmt.Errorf("cannot generate schema for nil value")
	}

	// For non-struct types, create a schema preserving all property details
	if t.Kind() != reflect.Struct && (t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct) {
		prop, err := reflectType(t, opt)
		if err != nil {
			return nil, err
		}
		// Convert Property to Schema, preserving all relevant fields
		return propertyToSchema(prop), nil
	}

	// For struct types, generate a full object schema
	prop, err := reflectType(t, opt)
	if err != nil {
		return nil, err
	}

	// Special case: some struct types (like time.Time) don't produce object schemas
	if prop.Type != Object {
		return propertyToSchema(prop), nil
	}

	schema := &Schema{
		Type:       prop.Type,
		Properties: prop.Properties,
		Required:   prop.Required,
	}
	if opt.DisallowAdditionalProperties {
		additionalProps := false
		schema.AdditionalProperties = &additionalProps
	}
	return schema, nil
}

// reflectType recursively analyzes a reflect.Type and returns a Property
// that describes its JSON Schema representation.
func reflectType(t reflect.Type, opt GenerateOptions) (*Property, error) {
	// Check for time.Time first (before struct handling)
	if t == reflect.TypeOf(time.Time{}) {
		format := "date-time"
		return &Property{Type: String, Format: &format}, nil
	}

	switch t.Kind() {
	case reflect.String:
		return &Property{Type: String}, nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Property{Type: Integer}, nil

	case reflect.Float32, reflect.Float64:
		return &Property{Type: Number}, nil

	case reflect.Bool:
		return &Property{Type: Boolean}, nil

	case reflect.Slice, reflect.Array:
		items, err := reflectType(t.Elem(), opt)
		if err != nil {
			return nil, fmt.Errorf("failed to reflect array/slice element type: %w", err)
		}
		return &Property{
			Type:  Array,
			Items: items,
		}, nil

	case reflect.Map:
		// Only support map[string]T
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key type must be string, got %s", t.Key().Kind())
		}
		valueType, err := reflectType(t.Elem(), opt)
		if err != nil {
			return nil, fmt.Errorf("failed to reflect map value type: %w", err)
		}
		// JSON Schema represents maps as objects with additionalProperties schema
		return &Property{
			Type:                       Object,
			AdditionalPropertiesSchema: valueType,
		}, nil

	case reflect.Struct:
		return reflectStruct(t, opt)

	case reflect.Ptr:
		// Check for *time.Time
		if t.Elem() == reflect.TypeOf(time.Time{}) {
			format := "date-time"
			nullable := true
			return &Property{Type: String, Format: &format, Nullable: &nullable}, nil
		}
		// For pointer types, reflect the underlying type and mark as nullable
		underlying, err := reflectType(t.Elem(), opt)
		if err != nil {
			return nil, fmt.Errorf("failed to reflect pointer underlying type: %w", err)
		}
		nullable := true
		underlying.Nullable = &nullable
		return underlying, nil

	case reflect.Interface:
		// For interface{} or any, don't specify a type to allow any JSON value
		return &Property{}, nil

	default:
		return nil, fmt.Errorf("unsupported type: %s", t.Kind().String())
	}
}

// reflectStruct analyzes a struct type and returns a Property representing
// an object schema with properties, required fields, and other constraints.
func reflectStruct(t reflect.Type, opt GenerateOptions) (*Property, error) {
	properties := make(map[string]*Property)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Handle embedded structs
		if field.Anonymous {
			embedded, err := reflectType(field.Type, opt)
			if err != nil {
				return nil, fmt.Errorf("failed to reflect embedded field %s: %w", field.Name, err)
			}
			// Merge embedded struct properties
			for name, prop := range embedded.Properties {
				properties[name] = prop
			}
			// Only merge required fields if the embedded struct is not a pointer.
			// Pointer-embedded structs are optional, so their fields shouldn't be required.
			if field.Type.Kind() != reflect.Ptr {
				required = append(required, embedded.Required...)
			}
			continue
		}

		// Parse JSON tag to get field name and options
		jsonName, isRequired := parseJSONTag(field)
		if jsonName == "-" {
			continue
		}

		// Generate property schema for the field type
		prop, err := reflectType(field.Type, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to reflect field %s: %w", field.Name, err)
		}

		// Apply field tags to the property
		applyFieldTags(prop, field)

		// Check if field is required (from tag or default behavior)
		if checkRequired(field, isRequired) {
			required = append(required, jsonName)
		}

		properties[jsonName] = prop
	}

	prop := &Property{
		Type:       Object,
		Properties: properties,
		Required:   required,
	}
	if opt.DisallowAdditionalProperties {
		additionalProps := false
		prop.AdditionalProperties = &additionalProps
	}
	return prop, nil
}

// parseJSONTag extracts the JSON field name and omitempty flag from a struct field's json tag.
// Returns the field name and whether the field is required (not omitempty).
func parseJSONTag(field reflect.StructField) (name string, required bool) {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return field.Name, true
	}

	parts := strings.Split(jsonTag, ",")
	name = parts[0]
	if name == "" {
		name = field.Name
	}

	// Check for omitempty flag
	required = true
	for _, part := range parts[1:] {
		if part == "omitempty" {
			required = false
			break
		}
	}

	return name, required
}

// applyFieldTags applies various struct field tags to a Property.
func applyFieldTags(prop *Property, field reflect.StructField) {
	// Description tag
	if desc := field.Tag.Get("description"); desc != "" {
		prop.Description = desc
	}

	// Enum tag (comma-separated values, parsed according to property type)
	if enum := field.Tag.Get("enum"); enum != "" {
		values := strings.Split(enum, ",")
		prop.Enum = make([]any, len(values))
		for i, v := range values {
			prop.Enum[i] = parseJSONValue(v, prop.Type)
		}
	}

	// Nullable tag
	if nullable := field.Tag.Get("nullable"); nullable != "" {
		if val, err := strconv.ParseBool(nullable); err == nil {
			prop.Nullable = &val
		}
	}

	// Pattern tag (for string validation)
	if pattern := field.Tag.Get("pattern"); pattern != "" {
		prop.Pattern = &pattern
	}

	// Format tag (e.g., "email", "date-time")
	if format := field.Tag.Get("format"); format != "" {
		prop.Format = &format
	}

	// Default tag - parse as JSON
	if defaultVal := field.Tag.Get("default"); defaultVal != "" {
		prop.Default = parseJSONValue(defaultVal, prop.Type)
	}

	// Example tag - parse as JSON
	if example := field.Tag.Get("example"); example != "" {
		prop.Example = parseJSONValue(example, prop.Type)
	}

	// Min/max length for strings
	if minLen := field.Tag.Get("minLength"); minLen != "" {
		if val, err := strconv.Atoi(minLen); err == nil {
			prop.MinLength = &val
		}
	}
	if maxLen := field.Tag.Get("maxLength"); maxLen != "" {
		if val, err := strconv.Atoi(maxLen); err == nil {
			prop.MaxLength = &val
		}
	}

	// Min/max for numbers
	if min := field.Tag.Get("minimum"); min != "" {
		if val, err := strconv.ParseFloat(min, 64); err == nil {
			prop.Minimum = &val
		}
	}
	if max := field.Tag.Get("maximum"); max != "" {
		if val, err := strconv.ParseFloat(max, 64); err == nil {
			prop.Maximum = &val
		}
	}

	// Min/max items for arrays
	if minItems := field.Tag.Get("minItems"); minItems != "" {
		if val, err := strconv.Atoi(minItems); err == nil {
			prop.MinItems = &val
		}
	}
	if maxItems := field.Tag.Get("maxItems"); maxItems != "" {
		if val, err := strconv.Atoi(maxItems); err == nil {
			prop.MaxItems = &val
		}
	}
}

// parseJSONValue parses a string value into the appropriate Go type based on schema type.
func parseJSONValue(s string, schemaType SchemaType) any {
	switch schemaType {
	case Integer:
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	case Number:
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	case Boolean:
		if v, err := strconv.ParseBool(s); err == nil {
			return v
		}
	case Array, Object:
		// Try to parse as JSON
		var v any
		if err := json.Unmarshal([]byte(s), &v); err == nil {
			return v
		}
	}
	// Default: return as string
	return s
}

// checkRequired determines if a field should be marked as required.
// It considers both the JSON tag omitempty flag and an explicit required tag.
func checkRequired(field reflect.StructField, jsonRequired bool) bool {
	// Explicit required tag takes precedence
	if req := field.Tag.Get("required"); req != "" {
		if val, err := strconv.ParseBool(req); err == nil {
			return val
		}
	}

	// Otherwise, use the result from JSON tag parsing
	return jsonRequired
}

// propertyToSchema converts a Property to a Schema, used for non-struct types
// like arrays, primitives, and maps when they are the root type.
func propertyToSchema(prop *Property) *Schema {
	return &Schema{
		Type:                 prop.Type,
		Description:          prop.Description,
		Properties:           prop.Properties,
		Required:             prop.Required,
		AdditionalProperties: prop.AdditionalProperties,
		Items:                prop.Items,
		Format:               prop.Format,
		Nullable:             prop.Nullable,
	}
}
