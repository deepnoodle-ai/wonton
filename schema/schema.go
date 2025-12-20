// Package schema provides JSON Schema types and generation utilities for defining
// tool parameters in LLM API integrations (OpenAI, Anthropic, etc).
//
// The package supports two workflows:
//
//  1. Programmatic schema construction using Schema and Property types
//  2. Automatic schema generation from Go struct types using Generate
//
// # Programmatic Construction
//
// Build schemas directly when you need fine-grained control:
//
//	schema := &Schema{
//	    Type: schema.Object,
//	    Properties: map[string]*Property{
//	        "name": {Type: schema.String, Description: "User's name"},
//	        "age":  {Type: schema.Integer, Description: "User's age"},
//	    },
//	    Required: []string{"name"},
//	}
//
// # Generation from Go Types
//
// Generate schemas from annotated Go structs for rapid development:
//
//	type User struct {
//	    Name  string `json:"name" description:"User's name"`
//	    Age   int    `json:"age,omitempty" description:"User's age" minimum:"0"`
//	    Email string `json:"email" format:"email"`
//	}
//
//	schema, err := schema.Generate(User{})
//
// # Supported Tags
//
// The following struct tags are recognized during generation:
//
//   - json: Field name and omitempty (affects required status)
//   - description: Human-readable description
//   - enum: Comma-separated allowed values
//   - format: String format (email, date-time, uri, etc.)
//   - pattern: Regex pattern for string validation
//   - default: Default value (parsed by type: strings as-is, numbers/bools parsed, arrays/objects as JSON)
//   - example: Example value (parsed by type: strings as-is, numbers/bools parsed, arrays/objects as JSON)
//   - minimum, maximum: Numeric bounds
//   - minLength, maxLength: String length bounds
//   - minItems, maxItems: Array length bounds
//   - nullable: Whether null is allowed
//   - required: Override required status (true/false)
package schema

import (
	"encoding/json"
	"fmt"
)

// SchemaType represents JSON Schema type values.
type SchemaType string

const (
	Array   SchemaType = "array"
	Boolean SchemaType = "boolean"
	Integer SchemaType = "integer"
	Null    SchemaType = "null"
	Number  SchemaType = "number"
	Object  SchemaType = "object"
	String  SchemaType = "string"
)

// Schema describes the structure of a JSON object, typically used as the root
// schema for tool parameters. This type is designed to be compatible with
// JSON Schema draft-07 and the subset used by OpenAI and Anthropic APIs.
type Schema struct {
	// Type is the JSON Schema type (typically "object" for tool parameters).
	Type SchemaType `json:"type"`

	// Description provides a human-readable explanation of the schema.
	Description string `json:"description,omitempty"`

	// Title is an optional short name for the schema.
	Title string `json:"title,omitempty"`

	// Properties maps property names to their definitions.
	Properties map[string]*Property `json:"properties,omitempty"`

	// Required lists property names that must be present.
	Required []string `json:"required,omitempty"`

	// AdditionalProperties controls whether extra properties are allowed.
	// Set to false (via pointer) to disallow additional properties.
	AdditionalProperties *bool `json:"additionalProperties,omitempty"`

	// AdditionalPropertiesSchema defines the schema for additional properties.
	// Used to represent map[string]T types where T is the value schema.
	// When set, this takes precedence over AdditionalProperties in JSON output.
	AdditionalPropertiesSchema *Property `json:"-"`

	// Items defines the schema for array elements (when Type is "array").
	Items *Property `json:"items,omitempty"`

	// Format specifies the string format (when Type is "string").
	Format *string `json:"format,omitempty"`

	// Nullable indicates whether null is an allowed value.
	Nullable *bool `json:"nullable,omitempty"`
}

// MarshalJSON implements json.Marshaler to handle AdditionalPropertiesSchema
// and ensure Properties is marshaled as an empty object {} for object types
// with fixed properties. LLM APIs require tools with no parameters to have
// "properties": {} not null.
func (s *Schema) MarshalJSON() ([]byte, error) {
	type schemaAlias Schema

	// Handle AdditionalPropertiesSchema (map schemas)
	if s.AdditionalPropertiesSchema != nil {
		data, err := json.Marshal((*schemaAlias)(s))
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		// Replace additionalProperties with the schema
		m["additionalProperties"] = s.AdditionalPropertiesSchema
		// For pure map schemas (no fixed properties), don't include empty properties
		if len(s.Properties) == 0 {
			delete(m, "properties")
		}
		return json.Marshal(m)
	}

	// For object types, ensure properties is included (even if empty) and
	// comes after other fields for consistent output
	if s.Type == Object {
		props := s.Properties
		if props == nil {
			props = map[string]*Property{}
		}
		return json.Marshal(&struct {
			*schemaAlias
			Properties map[string]*Property `json:"properties"`
		}{
			schemaAlias: (*schemaAlias)(s),
			Properties:  props,
		})
	}

	return json.Marshal((*schemaAlias)(s))
}

// UnmarshalJSON implements json.Unmarshaler to handle additionalProperties
// which can be either a boolean or a schema object.
func (s *Schema) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a map to extract additionalProperties separately
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Handle additionalProperties specially - remove it before alias unmarshal
	var apRaw json.RawMessage
	var hasAP bool
	if ap, ok := raw["additionalProperties"]; ok {
		apRaw = ap
		hasAP = true
		delete(raw, "additionalProperties")
	}

	// Re-marshal without additionalProperties and unmarshal into alias
	modified, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	type schemaAlias Schema
	if err := json.Unmarshal(modified, (*schemaAlias)(s)); err != nil {
		return err
	}

	// Now handle additionalProperties
	if hasAP {
		// Clear both fields first to avoid stale state on reused structs
		s.AdditionalProperties = nil
		s.AdditionalPropertiesSchema = nil

		// Try to unmarshal as boolean first
		var boolVal bool
		if err := json.Unmarshal(apRaw, &boolVal); err == nil {
			s.AdditionalProperties = &boolVal
			return nil
		}

		// Try to unmarshal as schema object
		var schemaProp Property
		if err := json.Unmarshal(apRaw, &schemaProp); err == nil {
			s.AdditionalPropertiesSchema = &schemaProp
			return nil
		}

		// Neither bool nor object - invalid additionalProperties value
		return fmt.Errorf("additionalProperties must be boolean or object, got: %s", string(apRaw))
	}

	return nil
}

// AsMap converts the schema to a map[string]any, useful for APIs that
// accept schema definitions as generic maps.
func (s *Schema) AsMap() map[string]any {
	var result map[string]any
	data, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// Property defines a single property within a schema. It can represent
// any JSON Schema type including nested objects and arrays.
type Property struct {
	// Type is the JSON Schema type for this property.
	Type SchemaType `json:"type,omitempty"`

	// Description provides a human-readable explanation of the property.
	Description string `json:"description,omitempty"`

	// Enum restricts the value to a fixed set of options.
	// Values should match the property type (strings for string fields,
	// integers for integer fields, etc.).
	Enum []any `json:"enum,omitempty"`

	// Items defines the schema for array elements (required when Type is "array").
	Items *Property `json:"items,omitempty"`

	// Required lists required properties (for object types).
	Required []string `json:"required,omitempty"`

	// Properties maps property names to definitions (for object types).
	Properties map[string]*Property `json:"properties,omitempty"`

	// AdditionalProperties controls whether extra properties are allowed.
	// When false (via Ptr(false)), no additional properties are allowed.
	// When nil, additional properties are implicitly allowed.
	// For maps (object with typed values), use AdditionalPropertiesSchema instead.
	AdditionalProperties *bool `json:"additionalProperties,omitempty"`

	// AdditionalPropertiesSchema defines the schema for additional properties.
	// Used to represent map[string]T types where T is the value schema.
	// When set, this takes precedence over AdditionalProperties in JSON output.
	AdditionalPropertiesSchema *Property `json:"-"`

	// Nullable indicates whether null is an allowed value.
	Nullable *bool `json:"nullable,omitempty"`

	// Pattern is a regex pattern for string validation.
	Pattern *string `json:"pattern,omitempty"`

	// Format specifies the string format (email, date-time, uri, etc.).
	Format *string `json:"format,omitempty"`

	// Default is the default value if not provided (JSON-encoded).
	Default any `json:"default,omitempty"`

	// Example provides a sample value (JSON-encoded).
	Example any `json:"example,omitempty"`

	// MinItems is the minimum array length.
	MinItems *int `json:"minItems,omitempty"`

	// MaxItems is the maximum array length.
	MaxItems *int `json:"maxItems,omitempty"`

	// MinLength is the minimum string length.
	MinLength *int `json:"minLength,omitempty"`

	// MaxLength is the maximum string length.
	MaxLength *int `json:"maxLength,omitempty"`

	// Minimum is the minimum numeric value (inclusive).
	Minimum *float64 `json:"minimum,omitempty"`

	// Maximum is the maximum numeric value (inclusive).
	Maximum *float64 `json:"maximum,omitempty"`
}

// MarshalJSON implements json.Marshaler to handle AdditionalPropertiesSchema.
// When AdditionalPropertiesSchema is set, it serializes as:
//
//	"additionalProperties": { ...schema... }
//
// Otherwise, AdditionalProperties (*bool) is used if set.
func (p *Property) MarshalJSON() ([]byte, error) {
	type propertyAlias Property
	if p.AdditionalPropertiesSchema != nil {
		// Create a map representation to properly merge additionalProperties
		data, err := json.Marshal((*propertyAlias)(p))
		if err != nil {
			return nil, err
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		// Replace additionalProperties with the schema
		m["additionalProperties"] = p.AdditionalPropertiesSchema
		return json.Marshal(m)
	}
	return json.Marshal((*propertyAlias)(p))
}

// UnmarshalJSON implements json.Unmarshaler to handle additionalProperties
// which can be either a boolean or a schema object.
func (p *Property) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a map to extract additionalProperties separately
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Handle additionalProperties specially - remove it before alias unmarshal
	var apRaw json.RawMessage
	var hasAP bool
	if ap, ok := raw["additionalProperties"]; ok {
		apRaw = ap
		hasAP = true
		delete(raw, "additionalProperties")
	}

	// Re-marshal without additionalProperties and unmarshal into alias
	modified, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	type propertyAlias Property
	if err := json.Unmarshal(modified, (*propertyAlias)(p)); err != nil {
		return err
	}

	// Now handle additionalProperties
	if hasAP {
		// Clear both fields first to avoid stale state on reused structs
		p.AdditionalProperties = nil
		p.AdditionalPropertiesSchema = nil

		// Try to unmarshal as boolean first
		var boolVal bool
		if err := json.Unmarshal(apRaw, &boolVal); err == nil {
			p.AdditionalProperties = &boolVal
			return nil
		}

		// Try to unmarshal as schema object
		var schemaProp Property
		if err := json.Unmarshal(apRaw, &schemaProp); err == nil {
			p.AdditionalPropertiesSchema = &schemaProp
			return nil
		}

		// Neither bool nor object - invalid additionalProperties value
		return fmt.Errorf("additionalProperties must be boolean or object, got: %s", string(apRaw))
	}

	return nil
}

// Ptr returns a pointer to the value, useful for setting optional fields.
func Ptr[T any](v T) *T {
	return &v
}

// NewSchema creates a new object schema with the given properties.
func NewSchema(properties map[string]*Property, required ...string) *Schema {
	additionalProps := false
	return &Schema{
		Type:                 Object,
		Properties:           properties,
		Required:             required,
		AdditionalProperties: &additionalProps,
	}
}

// StringProp creates a string property with the given description.
func StringProp(description string) *Property {
	return &Property{Type: String, Description: description}
}

// IntegerProp creates an integer property with the given description.
func IntegerProp(description string) *Property {
	return &Property{Type: Integer, Description: description}
}

// NumberProp creates a number (float) property with the given description.
func NumberProp(description string) *Property {
	return &Property{Type: Number, Description: description}
}

// BooleanProp creates a boolean property with the given description.
func BooleanProp(description string) *Property {
	return &Property{Type: Boolean, Description: description}
}

// ArrayProp creates an array property with the given item schema and description.
func ArrayProp(items *Property, description string) *Property {
	return &Property{Type: Array, Items: items, Description: description}
}

// ObjectProp creates an object property with the given properties and description.
func ObjectProp(properties map[string]*Property, description string, required ...string) *Property {
	additionalProps := false
	return &Property{
		Type:                 Object,
		Properties:           properties,
		Required:             required,
		AdditionalProperties: &additionalProps,
		Description:          description,
	}
}

// EnumProp creates a string property constrained to specific values.
func EnumProp(description string, values ...string) *Property {
	enum := make([]any, len(values))
	for i, v := range values {
		enum[i] = v
	}
	return &Property{Type: String, Description: description, Enum: enum}
}
