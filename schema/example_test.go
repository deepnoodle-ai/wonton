package schema_test

import (
	"encoding/json"
	"fmt"

	"github.com/deepnoodle-ai/wonton/schema"
)

// ExampleGenerate demonstrates automatic schema generation from a Go struct.
func ExampleGenerate() {
	type SearchParams struct {
		Query   string   `json:"query" description:"Search query string"`
		Limit   int      `json:"limit,omitempty" description:"Max results" default:"10"`
		Filters []string `json:"filters,omitempty" description:"Filter expressions"`
	}

	s, err := schema.Generate(SearchParams{})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	data, _ := json.MarshalIndent(s, "", "  ")
	fmt.Println(string(data))
	// Output:
	// {
	//   "type": "object",
	//   "required": [
	//     "query"
	//   ],
	//   "properties": {
	//     "filters": {
	//       "type": "array",
	//       "description": "Filter expressions",
	//       "items": {
	//         "type": "string"
	//       }
	//     },
	//     "limit": {
	//       "type": "integer",
	//       "description": "Max results",
	//       "default": 10
	//     },
	//     "query": {
	//       "type": "string",
	//       "description": "Search query string"
	//     }
	//   }
	// }
}

// ExampleNewSchema demonstrates programmatic schema construction.
func ExampleNewSchema() {
	s := schema.NewSchema(map[string]*schema.Property{
		"name": schema.StringProp("User's name"),
		"age":  schema.IntegerProp("User's age"),
	}, "name")

	data, _ := json.MarshalIndent(s, "", "  ")
	fmt.Println(string(data))
	// Output:
	// {
	//   "type": "object",
	//   "required": [
	//     "name"
	//   ],
	//   "additionalProperties": false,
	//   "properties": {
	//     "age": {
	//       "type": "integer",
	//       "description": "User's age"
	//     },
	//     "name": {
	//       "type": "string",
	//       "description": "User's name"
	//     }
	//   }
	// }
}

// ExampleEnumProp demonstrates creating enum properties.
func ExampleEnumProp() {
	s := schema.NewSchema(map[string]*schema.Property{
		"status": schema.EnumProp("Current status", "pending", "active", "completed"),
	}, "status")

	data, _ := json.MarshalIndent(s, "", "  ")
	fmt.Println(string(data))
	// Output:
	// {
	//   "type": "object",
	//   "required": [
	//     "status"
	//   ],
	//   "additionalProperties": false,
	//   "properties": {
	//     "status": {
	//       "type": "string",
	//       "description": "Current status",
	//       "enum": [
	//         "pending",
	//         "active",
	//         "completed"
	//       ]
	//     }
	//   }
	// }
}
