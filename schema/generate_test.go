package schema_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/schema"
)

// jsonEqual compares two JSON strings for equality by unmarshaling and comparing
func jsonEqual(t testing.TB, expected, actual string) {
	t.Helper()
	var expectedMap, actualMap any
	if err := json.Unmarshal([]byte(expected), &expectedMap); err != nil {
		t.Fatalf("failed to unmarshal expected JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(actual), &actualMap); err != nil {
		t.Fatalf("failed to unmarshal actual JSON: %v", err)
	}
	assert.Equal(t, actualMap, expectedMap)
}

// Test types for schema generation
type TestUser struct {
	Name     string   `json:"name" description:"User's full name"`
	Age      int      `json:"age,omitempty" description:"User's age in years" minimum:"0" maximum:"150"`
	Email    string   `json:"email" description:"User's email address" format:"email" required:"true"`
	Tags     []string `json:"tags,omitempty" description:"User tags" maxItems:"10"`
	Active   bool     `json:"active" description:"Whether the user is active"`
	Metadata *string  `json:"metadata,omitempty" description:"Optional metadata"`
}

type TestProduct struct {
	ID          string  `json:"id" pattern:"^[A-Z0-9]+$"`
	Name        string  `json:"name" minLength:"1" maxLength:"100"`
	Price       float64 `json:"price" minimum:"0"`
	Category    string  `json:"category" enum:"electronics,books,clothing"`
	InStock     bool    `json:"in_stock"`
	Description *string `json:"description,omitempty" nullable:"true"`
}

func TestGenerate_SimpleTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected schema.SchemaType
	}{
		{"string", "", schema.String},
		{"int", 0, schema.Integer},
		{"int64", int64(0), schema.Integer},
		{"float64", 0.0, schema.Number},
		{"bool", false, schema.Boolean},
		{"slice", []string{}, schema.Array},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := schema.Generate(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, s.Type)
		})
	}
}

func TestGenerate_Struct(t *testing.T) {
	s, err := schema.Generate(TestUser{})
	assert.NoError(t, err)

	// Check basic schema properties
	assert.Equal(t, schema.Object, s.Type)
	// By default, additionalProperties is not set (Anthropic-compatible)
	assert.Nil(t, s.AdditionalProperties)

	// Check properties exist
	expectedProps := []string{"name", "age", "email", "tags", "active", "metadata"}
	assert.Len(t, s.Properties, len(expectedProps))

	for _, prop := range expectedProps {
		_, exists := s.Properties[prop]
		assert.True(t, exists, "Property %s not found", prop)
	}

	// Check required fields
	expectedRequired := []string{"name", "email", "active"}
	assert.Len(t, s.Required, len(expectedRequired))

	for _, req := range expectedRequired {
		found := false
		for _, r := range s.Required {
			if r == req {
				found = true
				break
			}
		}
		assert.True(t, found, "Required field %s not found", req)
	}
}

func TestGenerate_PropertyTypes(t *testing.T) {
	s, err := schema.Generate(TestUser{})
	assert.NoError(t, err)

	tests := []struct {
		field    string
		expected schema.SchemaType
	}{
		{"name", schema.String},
		{"age", schema.Integer},
		{"email", schema.String},
		{"tags", schema.Array},
		{"active", schema.Boolean},
		{"metadata", schema.String},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			prop, exists := s.Properties[tt.field]
			assert.True(t, exists, "Property %s not found", tt.field)
			assert.Equal(t, tt.expected, prop.Type)
		})
	}
}

func TestGenerate_PropertyDescriptions(t *testing.T) {
	s, err := schema.Generate(TestUser{})
	assert.NoError(t, err)

	tests := []struct {
		field       string
		description string
	}{
		{"name", "User's full name"},
		{"age", "User's age in years"},
		{"email", "User's email address"},
		{"tags", "User tags"},
		{"active", "Whether the user is active"},
		{"metadata", "Optional metadata"},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			prop, exists := s.Properties[tt.field]
			assert.True(t, exists, "Property %s not found", tt.field)
			assert.Equal(t, tt.description, prop.Description)
		})
	}
}

func TestGenerate_PropertyConstraints(t *testing.T) {
	s, err := schema.Generate(TestProduct{})
	assert.NoError(t, err)

	// Test pattern constraint
	idProp := s.Properties["id"]
	assert.NotNil(t, idProp.Pattern)
	assert.Equal(t, "^[A-Z0-9]+$", *idProp.Pattern)

	// Test length constraints
	nameProp := s.Properties["name"]
	assert.NotNil(t, nameProp.MinLength)
	assert.Equal(t, 1, *nameProp.MinLength)
	assert.NotNil(t, nameProp.MaxLength)
	assert.Equal(t, 100, *nameProp.MaxLength)

	// Test numeric constraints
	priceProp := s.Properties["price"]
	assert.NotNil(t, priceProp.Minimum)
	assert.Equal(t, 0.0, *priceProp.Minimum)

	// Test enum constraint
	categoryProp := s.Properties["category"]
	expected := []any{"electronics", "books", "clothing"}
	assert.Equal(t, expected, categoryProp.Enum)

	// Test nullable constraint
	descProp := s.Properties["description"]
	assert.NotNil(t, descProp.Nullable)
	assert.True(t, *descProp.Nullable)
}

func TestGenerate_ArrayType(t *testing.T) {
	s, err := schema.Generate(TestUser{})
	assert.NoError(t, err)

	tagsProp := s.Properties["tags"]
	assert.Equal(t, schema.Array, tagsProp.Type)
	assert.NotNil(t, tagsProp.Items)
	assert.Equal(t, schema.String, tagsProp.Items.Type)
	assert.NotNil(t, tagsProp.MaxItems)
	assert.Equal(t, 10, *tagsProp.MaxItems)
}

func TestGenerate_PointerType(t *testing.T) {
	s, err := schema.Generate(TestUser{})
	assert.NoError(t, err)

	metadataProp := s.Properties["metadata"]
	assert.Equal(t, schema.String, metadataProp.Type)
	assert.NotNil(t, metadataProp.Nullable)
	assert.True(t, *metadataProp.Nullable)
}

func TestGenerate_JSONSerialization(t *testing.T) {
	s, err := schema.Generate(TestUser{})
	assert.NoError(t, err)

	// Test that the schema can be serialized to JSON
	jsonData, err := json.MarshalIndent(s, "", "  ")
	assert.NoError(t, err)

	// Test that it can be deserialized back
	var deserializedSchema schema.Schema
	err = json.Unmarshal(jsonData, &deserializedSchema)
	assert.NoError(t, err)

	// Basic validation that the deserialized schema matches
	assert.Equal(t, s.Type, deserializedSchema.Type)
	assert.Len(t, deserializedSchema.Properties, len(s.Properties))
}

func TestGenerate_NilInput(t *testing.T) {
	_, err := schema.Generate(nil)
	assert.Error(t, err)
}

func TestGenerate_UnsupportedType(t *testing.T) {
	unsupported := make(chan int)
	_, err := schema.Generate(unsupported)
	assert.Error(t, err)
}

type NestedStruct struct {
	Inner struct {
		Value string `json:"value" description:"Inner value"`
		Count int    `json:"count,omitempty"`
	} `json:"inner" description:"Nested inner struct"`
}

func TestGenerate_NestedStruct(t *testing.T) {
	s, err := schema.Generate(NestedStruct{})
	assert.NoError(t, err)

	innerProp := s.Properties["inner"]
	assert.Equal(t, schema.Object, innerProp.Type)
	assert.NotNil(t, innerProp.Properties)

	valueProp := innerProp.Properties["value"]
	assert.Equal(t, schema.String, valueProp.Type)

	// Check that "value" is required but "count" is not
	valueRequired := false
	countRequired := false
	for _, req := range innerProp.Required {
		if req == "value" {
			valueRequired = true
		}
		if req == "count" {
			countRequired = true
		}
	}

	assert.True(t, valueRequired, "Inner value should be required")
	assert.False(t, countRequired, "Inner count should not be required")
}

type SimpleTestStruct struct {
	Name   string `json:"name" description:"A name field"`
	Age    int    `json:"age,omitempty" description:"Age in years"`
	Active bool   `json:"active" description:"Whether active"`
}

func TestGenerate_SimpleJSONSerialization(t *testing.T) {
	s, err := schema.Generate(SimpleTestStruct{})
	assert.NoError(t, err)

	jsonData, err := json.MarshalIndent(s, "", "  ")
	assert.NoError(t, err)

	// Default: Anthropic-compatible (no additionalProperties)
	expectedJSON := `{
  "type": "object",
  "properties": {
    "active": {
      "type": "boolean",
      "description": "Whether active"
    },
    "age": {
      "type": "integer",
      "description": "Age in years"
    },
    "name": {
      "type": "string",
      "description": "A name field"
    }
  },
  "required": [
    "name",
    "active"
  ]
}`

	jsonEqual(t, expectedJSON, string(jsonData))
}

type ComplexTestStruct struct {
	ID     string `json:"id" description:"Unique identifier"`
	Config struct {
		MaxRetries int    `json:"max_retries" description:"Maximum retry attempts"`
		Timeout    string `json:"timeout,omitempty" description:"Timeout duration"`
	} `json:"config" description:"Configuration settings"`
	Tags []string `json:"tags,omitempty" description:"List of tags" maxItems:"5"`
}

func TestGenerate_ComplexJSONSerialization(t *testing.T) {
	s, err := schema.Generate(ComplexTestStruct{})
	assert.NoError(t, err)

	jsonData, err := json.MarshalIndent(s, "", "  ")
	assert.NoError(t, err)

	// Default: Anthropic-compatible (no additionalProperties)
	expectedJSON := `{
  "type": "object",
  "properties": {
    "config": {
      "type": "object",
      "properties": {
        "max_retries": {
          "type": "integer",
          "description": "Maximum retry attempts"
        },
        "timeout": {
          "type": "string",
          "description": "Timeout duration"
        }
      },
      "required": [
        "max_retries"
      ],
      "description": "Configuration settings"
    },
    "id": {
      "type": "string",
      "description": "Unique identifier"
    },
    "tags": {
      "type": "array",
      "items": {
        "type": "string"
      },
      "description": "List of tags",
      "maxItems": 5
    }
  },
  "required": [
    "id",
    "config"
  ]
}`

	jsonEqual(t, expectedJSON, string(jsonData))
}

// Test OpenAI strict mode (requires additionalProperties: false)
func TestGenerate_OpenAIStrictMode(t *testing.T) {
	type Location struct {
		City    string `json:"city" description:"City name"`
		Country string `json:"country,omitempty" description:"Country name"`
	}

	s, err := schema.Generate(Location{}, schema.GenerateOptions{DisallowAdditionalProperties: true})
	assert.NoError(t, err)

	// Should have additionalProperties: false at top level
	assert.NotNil(t, s.AdditionalProperties)
	assert.False(t, *s.AdditionalProperties)

	jsonData, err := json.MarshalIndent(s, "", "  ")
	assert.NoError(t, err)

	expectedJSON := `{
  "type": "object",
  "properties": {
    "city": {
      "type": "string",
      "description": "City name"
    },
    "country": {
      "type": "string",
      "description": "Country name"
    }
  },
  "required": ["city"],
  "additionalProperties": false
}`
	jsonEqual(t, expectedJSON, string(jsonData))
}

// Test nested structs with strict mode
func TestGenerate_OpenAIStrictModeNested(t *testing.T) {
	type Inner struct {
		Value string `json:"value"`
	}
	type Outer struct {
		Inner Inner `json:"inner"`
	}

	s, err := schema.Generate(Outer{}, schema.GenerateOptions{DisallowAdditionalProperties: true})
	assert.NoError(t, err)

	// Top level should have additionalProperties: false
	assert.NotNil(t, s.AdditionalProperties)
	assert.False(t, *s.AdditionalProperties)

	// Nested object should also have additionalProperties: false
	innerProp := s.Properties["inner"]
	assert.NotNil(t, innerProp.AdditionalProperties)
	assert.False(t, *innerProp.AdditionalProperties)
}

// Test time.Time support
type TimeStruct struct {
	CreatedAt time.Time  `json:"created_at" description:"Creation timestamp"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" description:"Last update timestamp"`
}

func TestGenerate_TimeType(t *testing.T) {
	s, err := schema.Generate(TimeStruct{})
	assert.NoError(t, err)

	// time.Time should be string with date-time format
	createdProp := s.Properties["created_at"]
	assert.Equal(t, schema.String, createdProp.Type)
	assert.NotNil(t, createdProp.Format)
	assert.Equal(t, "date-time", *createdProp.Format)

	// *time.Time should also be string with date-time format, but nullable
	updatedProp := s.Properties["updated_at"]
	assert.Equal(t, schema.String, updatedProp.Type)
	assert.NotNil(t, updatedProp.Format)
	assert.Equal(t, "date-time", *updatedProp.Format)
	assert.NotNil(t, updatedProp.Nullable)
	assert.True(t, *updatedProp.Nullable)
}

// Test map support
type MapStruct struct {
	Labels   map[string]string `json:"labels,omitempty" description:"Key-value labels"`
	Metadata map[string]any    `json:"metadata,omitempty" description:"Arbitrary metadata"`
}

func TestGenerate_MapType(t *testing.T) {
	s, err := schema.Generate(MapStruct{})
	assert.NoError(t, err)

	// map[string]string should be object with additionalProperties schema
	labelsProp := s.Properties["labels"]
	assert.Equal(t, schema.Object, labelsProp.Type)
	assert.NotNil(t, labelsProp.AdditionalPropertiesSchema)
	assert.Equal(t, schema.String, labelsProp.AdditionalPropertiesSchema.Type)

	// map[string]any should also work
	metaProp := s.Properties["metadata"]
	assert.Equal(t, schema.Object, metaProp.Type)
	assert.NotNil(t, metaProp.AdditionalPropertiesSchema)
}

func TestGenerate_MapWithInvalidKey(t *testing.T) {
	type InvalidMap struct {
		Data map[int]string `json:"data"`
	}
	_, err := schema.Generate(InvalidMap{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "map key type must be string")
}

// Test default and example values
type DefaultsStruct struct {
	Name    string  `json:"name" default:"unnamed"`
	Count   int     `json:"count,omitempty" default:"10"`
	Enabled bool    `json:"enabled,omitempty" default:"true"`
	Score   float64 `json:"score,omitempty" default:"0.5"`
}

func TestGenerate_DefaultValues(t *testing.T) {
	s, err := schema.Generate(DefaultsStruct{})
	assert.NoError(t, err)

	// String default
	nameProp := s.Properties["name"]
	assert.Equal(t, "unnamed", nameProp.Default)

	// Integer default
	countProp := s.Properties["count"]
	assert.Equal(t, int64(10), countProp.Default)

	// Boolean default
	enabledProp := s.Properties["enabled"]
	assert.Equal(t, true, enabledProp.Default)

	// Float default
	scoreProp := s.Properties["score"]
	assert.Equal(t, 0.5, scoreProp.Default)
}

type ExamplesStruct struct {
	Email string `json:"email" example:"user@example.com"`
	Age   int    `json:"age,omitempty" example:"25"`
}

func TestGenerate_ExampleValues(t *testing.T) {
	s, err := schema.Generate(ExamplesStruct{})
	assert.NoError(t, err)

	emailProp := s.Properties["email"]
	assert.Equal(t, "user@example.com", emailProp.Example)

	ageProp := s.Properties["age"]
	assert.Equal(t, int64(25), ageProp.Example)
}

// Test embedded structs
type BaseFields struct {
	ID        string    `json:"id" description:"Unique identifier"`
	CreatedAt time.Time `json:"created_at" description:"Creation time"`
}

type EntityWithEmbedded struct {
	BaseFields
	Name string `json:"name" description:"Entity name"`
}

func TestGenerate_EmbeddedStruct(t *testing.T) {
	s, err := schema.Generate(EntityWithEmbedded{})
	assert.NoError(t, err)

	// Should have all fields from both embedded and parent
	assert.Contains(t, s.Properties, "id")
	assert.Contains(t, s.Properties, "created_at")
	assert.Contains(t, s.Properties, "name")

	// Check embedded field types
	idProp := s.Properties["id"]
	assert.Equal(t, schema.String, idProp.Type)

	createdProp := s.Properties["created_at"]
	assert.Equal(t, schema.String, createdProp.Type)
	assert.NotNil(t, createdProp.Format)
	assert.Equal(t, "date-time", *createdProp.Format)
}

// Test json:"-" tag
type SkipFieldStruct struct {
	Public  string `json:"public"`
	Private string `json:"-"`
}

func TestGenerate_SkipField(t *testing.T) {
	s, err := schema.Generate(SkipFieldStruct{})
	assert.NoError(t, err)

	assert.Contains(t, s.Properties, "public")
	assert.NotContains(t, s.Properties, "Private")
	assert.NotContains(t, s.Properties, "-")
}

// Test interface{} / any type
type AnyStruct struct {
	Data any `json:"data,omitempty" description:"Arbitrary data"`
}

func TestGenerate_AnyType(t *testing.T) {
	s, err := schema.Generate(AnyStruct{})
	assert.NoError(t, err)

	dataProp := s.Properties["data"]
	// any type should have no type constraint
	assert.Empty(t, dataProp.Type)
}

// Test nested arrays
type NestedArrayStruct struct {
	Matrix [][]int `json:"matrix" description:"2D integer matrix"`
}

func TestGenerate_NestedArray(t *testing.T) {
	s, err := schema.Generate(NestedArrayStruct{})
	assert.NoError(t, err)

	matrixProp := s.Properties["matrix"]
	assert.Equal(t, schema.Array, matrixProp.Type)
	assert.NotNil(t, matrixProp.Items)
	assert.Equal(t, schema.Array, matrixProp.Items.Type)
	assert.NotNil(t, matrixProp.Items.Items)
	assert.Equal(t, schema.Integer, matrixProp.Items.Items.Type)
}

// Test that map serializes correctly with additionalProperties as schema
func TestGenerate_MapSerialization(t *testing.T) {
	type MapExample struct {
		Labels map[string]string `json:"labels" description:"Key-value labels"`
	}

	s, err := schema.Generate(MapExample{})
	assert.NoError(t, err)

	// Serialize to JSON
	jsonData, err := json.Marshal(s)
	assert.NoError(t, err)

	// Should contain additionalProperties as an object, not items
	assert.Contains(t, string(jsonData), `"additionalProperties":{`)
	assert.NotContains(t, string(jsonData), `"items"`)

	// The additionalProperties should have type: string
	assert.Contains(t, string(jsonData), `"type":"string"`)
}

// Test that Generate for non-struct types preserves all schema details
func TestGenerate_NonStructPreservesDetails(t *testing.T) {
	// Test array type
	s, err := schema.Generate([]string{})
	assert.NoError(t, err)
	assert.Equal(t, schema.Array, s.Type)
	assert.NotNil(t, s.Items, "Array schema should have Items")
	assert.Equal(t, schema.String, s.Items.Type)

	// Test time.Time type
	s, err = schema.Generate(time.Time{})
	assert.NoError(t, err)
	assert.Equal(t, schema.String, s.Type)
	assert.NotNil(t, s.Format, "time.Time schema should have Format")
	assert.Equal(t, "date-time", *s.Format)

	// Test pointer type
	var ptr *string
	s, err = schema.Generate(ptr)
	assert.NoError(t, err)
	assert.Equal(t, schema.String, s.Type)
	assert.NotNil(t, s.Nullable, "Pointer schema should have Nullable")
	assert.True(t, *s.Nullable)
}

// Test that embedded pointer structs don't force required fields
type EmbeddedBase struct {
	ID   string `json:"id" description:"Base ID"`
	Name string `json:"name" description:"Base name"`
}

type EmbeddedPointerStruct struct {
	*EmbeddedBase
	Extra string `json:"extra" description:"Extra field"`
}

type EmbeddedNonPointerStruct struct {
	EmbeddedBase
	Extra string `json:"extra" description:"Extra field"`
}

func TestGenerate_EmbeddedPointerNotRequired(t *testing.T) {
	// Embedded pointer struct: fields from embedded should NOT be required
	s, err := schema.Generate(EmbeddedPointerStruct{})
	assert.NoError(t, err)

	// Should have all fields
	assert.Contains(t, s.Properties, "id")
	assert.Contains(t, s.Properties, "name")
	assert.Contains(t, s.Properties, "extra")

	// Only "extra" should be required (embedded pointer fields are optional)
	assert.Equal(t, []string{"extra"}, s.Required)
}

func TestGenerate_EmbeddedNonPointerRequired(t *testing.T) {
	// Embedded non-pointer struct: fields from embedded SHOULD be required
	s, err := schema.Generate(EmbeddedNonPointerStruct{})
	assert.NoError(t, err)

	// Should have all fields as required (id, name from embedded, extra from parent)
	assert.Len(t, s.Required, 3)
	assert.Contains(t, s.Required, "id")
	assert.Contains(t, s.Required, "name")
	assert.Contains(t, s.Required, "extra")
}

// Test that enum values are typed correctly for non-string fields
type EnumTypedStruct struct {
	Status  string `json:"status" enum:"active,inactive"`
	Count   int    `json:"count" enum:"1,2,3"`
	Enabled bool   `json:"enabled" enum:"true,false"`
}

func TestGenerate_EnumTypedValues(t *testing.T) {
	s, err := schema.Generate(EnumTypedStruct{})
	assert.NoError(t, err)

	// String enum should have string values
	statusProp := s.Properties["status"]
	assert.Equal(t, []any{"active", "inactive"}, statusProp.Enum)

	// Integer enum should have integer values
	countProp := s.Properties["count"]
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, countProp.Enum)

	// Boolean enum should have boolean values
	enabledProp := s.Properties["enabled"]
	assert.Equal(t, []any{true, false}, enabledProp.Enum)
}
