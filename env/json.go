package env

import (
	"encoding/json"
	"os"
)

// ReadJSONFile reads a JSON config file and returns a flat map of values.
// Nested objects are flattened with underscore separators.
// Example: {"database": {"host": "localhost"}} becomes {"database_host": "localhost"}
func ReadJSONFile(filename string) (map[string]any, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return ParseJSON(data)
}

// ParseJSON parses JSON data and returns a flat map of values.
func ParseJSON(data []byte) (map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	result := make(map[string]any)
	flattenJSON(raw, "", result)
	return result, nil
}

// flattenJSON recursively flattens a nested map.
func flattenJSON(data map[string]any, prefix string, result map[string]any) {
	for k, v := range data {
		key := k
		if prefix != "" {
			key = prefix + "_" + k
		}

		switch val := v.(type) {
		case map[string]any:
			// Recursively flatten nested objects
			flattenJSON(val, key, result)
		default:
			// Store the value with the flattened key
			result[key] = v
		}
	}
}

// WriteJSONFile writes a map to a JSON file.
func WriteJSONFile(values map[string]any, filename string) error {
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, append(data, '\n'), 0644)
}
