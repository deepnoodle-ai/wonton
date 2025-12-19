package schema_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/schema"
)

func TestSchemaMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input schema.Schema
	}{
		{
			name: "simple schema",
			input: schema.Schema{
				Type: "object",
				Properties: map[string]*schema.Property{
					"name": {
						Type:        "string",
						Description: "The name of the object",
					},
					"age": {
						Type:        "integer",
						Description: "The age of the object",
					},
				},
				Required: []string{"name"},
			},
		},
		{
			name: "nested properties",
			input: schema.Schema{
				Type: "object",
				Properties: map[string]*schema.Property{
					"user": {
						Type:        "object",
						Description: "User information",
						Properties: map[string]*schema.Property{
							"id": {
								Type:        "string",
								Description: "User ID",
							},
							"settings": {
								Type:        "object",
								Description: "User settings",
								Properties: map[string]*schema.Property{
									"theme": {
										Type:        "string",
										Description: "UI theme",
									},
								},
							},
						},
						Required: []string{"id"},
					},
				},
			},
		},
		{
			name: "array property",
			input: schema.Schema{
				Type: "object",
				Properties: map[string]*schema.Property{
					"tags": {
						Type:        "array",
						Description: "List of tags",
						Items: &schema.Property{
							Type:        "string",
							Description: "Tag value",
						},
					},
				},
			},
		},
		{
			name: "enum property",
			input: schema.Schema{
				Type: "object",
				Properties: map[string]*schema.Property{
					"status": {
						Type:        "string",
						Description: "Status of the item",
						Enum:        []any{"pending", "active", "completed"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal schema: %v", err)
			}

			// Unmarshal back to schema.Schema
			var result schema.Schema
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal schema: %v", err)
			}

			// Compare input and result
			if !reflect.DeepEqual(tt.input, result) {
				t.Errorf("schema.Schema after marshal/unmarshal doesn't match original:\nOriginal: %+v\nResult:   %+v", tt.input, result)
			}
		})
	}
}

func TestPropertyMarshalUnmarshal(t *testing.T) {
	tests := []struct {
		name  string
		input schema.Property
	}{
		{
			name: "simple property",
			input: schema.Property{
				Type:        "string",
				Description: "A simple string property",
			},
		},
		{
			name: "object property with nested fields",
			input: schema.Property{
				Type:        "object",
				Description: "A complex object",
				Properties: map[string]*schema.Property{
					"field1": {
						Type:        "string",
						Description: "First field",
					},
					"field2": {
						Type:        "number",
						Description: "Second field",
					},
				},
				Required: []string{"field1"},
			},
		},
		{
			name: "array property",
			input: schema.Property{
				Type:        "array",
				Description: "An array of objects",
				Items: &schema.Property{
					Type:        "object",
					Description: "Array item",
					Properties: map[string]*schema.Property{
						"id": {
							Type:        "string",
							Description: "Item ID",
						},
					},
				},
			},
		},
		{
			name: "property with enum",
			input: schema.Property{
				Type:        "string",
				Description: "schema.Property with enum values",
				Enum:        []any{"option1", "option2", "option3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal property: %v", err)
			}

			// Unmarshal back to schema.Property
			var result schema.Property
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("Failed to unmarshal property: %v", err)
			}

			// Compare input and result
			if !reflect.DeepEqual(tt.input, result) {
				t.Errorf("schema.Property after marshal/unmarshal doesn't match original:\nOriginal: %+v\nResult:   %+v", tt.input, result)
			}
		})
	}
}

func TestPtr(t *testing.T) {
	intVal := schema.Ptr(42)
	assert.NotNil(t, intVal)
	assert.Equal(t, 42, *intVal)

	strVal := schema.Ptr("hello")
	assert.NotNil(t, strVal)
	assert.Equal(t, "hello", *strVal)

	boolVal := schema.Ptr(true)
	assert.NotNil(t, boolVal)
	assert.True(t, *boolVal)
}

func TestNewSchema(t *testing.T) {
	props := map[string]*schema.Property{
		"name": {Type: schema.String, Description: "User name"},
		"age":  {Type: schema.Integer, Description: "User age"},
	}
	s := schema.NewSchema(props, "name")

	assert.Equal(t, schema.Object, s.Type)
	assert.Len(t, s.Properties, 2)
	assert.Equal(t, []string{"name"}, s.Required)
	assert.NotNil(t, s.AdditionalProperties)
	assert.False(t, s.AdditionalProperties != nil && *s.AdditionalProperties)
}

func TestStringProp(t *testing.T) {
	prop := schema.StringProp("A string field")
	assert.Equal(t, schema.String, prop.Type)
	assert.Equal(t, "A string field", prop.Description)
}

func TestIntegerProp(t *testing.T) {
	prop := schema.IntegerProp("An integer field")
	assert.Equal(t, schema.Integer, prop.Type)
	assert.Equal(t, "An integer field", prop.Description)
}

func TestNumberProp(t *testing.T) {
	prop := schema.NumberProp("A number field")
	assert.Equal(t, schema.Number, prop.Type)
	assert.Equal(t, "A number field", prop.Description)
}

func TestBooleanProp(t *testing.T) {
	prop := schema.BooleanProp("A boolean field")
	assert.Equal(t, schema.Boolean, prop.Type)
	assert.Equal(t, "A boolean field", prop.Description)
}

func TestArrayProp(t *testing.T) {
	items := schema.StringProp("Item")
	prop := schema.ArrayProp(items, "A list of strings")

	assert.Equal(t, schema.Array, prop.Type)
	assert.Equal(t, "A list of strings", prop.Description)
	assert.NotNil(t, prop.Items)
	assert.Equal(t, schema.String, prop.Items.Type)
}

func TestObjectProp(t *testing.T) {
	props := map[string]*schema.Property{
		"field1": schema.StringProp("First field"),
		"field2": schema.IntegerProp("Second field"),
	}
	prop := schema.ObjectProp(props, "Nested object", "field1")

	assert.Equal(t, schema.Object, prop.Type)
	assert.Equal(t, "Nested object", prop.Description)
	assert.Len(t, prop.Properties, 2)
	assert.Equal(t, []string{"field1"}, prop.Required)
	assert.NotNil(t, prop.AdditionalProperties)
	assert.False(t, *prop.AdditionalProperties)
}

func TestEnumProp(t *testing.T) {
	prop := schema.EnumProp("Status field", "pending", "active", "completed")

	assert.Equal(t, schema.String, prop.Type)
	assert.Equal(t, "Status field", prop.Description)
	assert.Equal(t, []any{"pending", "active", "completed"}, prop.Enum)
}

func TestSchemaAsMap(t *testing.T) {
	schema := schema.NewSchema(map[string]*schema.Property{
		"name": schema.StringProp("User name"),
	}, "name")

	m := schema.AsMap()
	assert.NotNil(t, m)
	assert.Equal(t, "object", m["type"])
	assert.NotNil(t, m["properties"])
}

func TestSchemaNilPropertiesMarshal(t *testing.T) {
	// schema.Schema with nil properties should marshal with empty object
	schema := &schema.Schema{Type: schema.Object}
	data, err := json.Marshal(schema)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"properties":{}`)
}

func TestJSONRoundTrip(t *testing.T) {
	// Create a schema with various property types
	original := schema.Schema{
		Type: "object",
		Properties: map[string]*schema.Property{
			"id": {
				Type:        "string",
				Description: "Unique identifier",
			},
			"details": {
				Type:        "object",
				Description: "Detailed information",
				Properties: map[string]*schema.Property{
					"name": {
						Type:        "string",
						Description: "Full name",
					},
					"age": {
						Type:        "integer",
						Description: "Age in years",
					},
					"preferences": {
						Type:        "object",
						Description: "User preferences",
						Properties: map[string]*schema.Property{
							"theme": {
								Type:        "string",
								Description: "UI theme",
								Enum:        []any{"light", "dark", "system"},
							},
						},
					},
				},
			},
			"tags": {
				Type:        "array",
				Description: "Associated tags",
				Items: &schema.Property{
					Type:        "string",
					Description: "Tag value",
				},
			},
			"status": {
				Type:        "string",
				Description: "Current status",
				Enum:        []any{"active", "inactive", "pending"},
			},
		},
		Required: []string{"id", "details"},
	}

	// Convert to JSON string representation
	jsonData, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal schema: %v", err)
	}

	// Output JSON for debugging if needed
	// t.Logf("JSON:\n%s", string(jsonData))

	// Parse JSON back to schema.Schema
	var parsed schema.Schema
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	// Ensure the round-trip preserves all data
	if !reflect.DeepEqual(original, parsed) {
		t.Errorf("schema.Schema after JSON round-trip doesn't match original")
	}
}
