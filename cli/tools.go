package cli

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// ToolSchema represents a tool definition for AI agents (MCP-compatible).
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]ParamSchema `json:"parameters,omitempty"`
	Required    []string               `json:"required,omitempty"`
}

// ParamSchema describes a tool parameter.
type ParamSchema struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Default     any      `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// GetToolSchemas returns JSON schemas for all commands marked as tools.
func (a *App) GetToolSchemas() []ToolSchema {
	var schemas []ToolSchema

	// Collect from direct commands
	for _, cmd := range a.commands {
		if cmd.isTool {
			schemas = append(schemas, cmd.toToolSchema())
		}
	}

	// Collect from groups
	for _, group := range a.groups {
		for _, cmd := range group.commands {
			if cmd.isTool {
				schema := cmd.toToolSchema()
				// Prefix with group name
				schema.Name = group.name + ":" + schema.Name
				schemas = append(schemas, schema)
			}
		}
	}

	return schemas
}

// ToolsJSON returns a JSON string of all tool schemas.
func (a *App) ToolsJSON() (string, error) {
	schemas := a.GetToolSchemas()
	data, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Command) toToolSchema() ToolSchema {
	schema := ToolSchema{
		Name:        c.name,
		Description: c.description,
		Parameters:  make(map[string]ParamSchema),
	}

	// Add flags as parameters
	for _, f := range c.flags {
		if f.IsHidden() {
			continue
		}
		param := ParamSchema{
			Description: f.GetHelp(),
			Enum:        f.GetEnum(),
		}

		// Determine type from default value
		switch f.GetDefault().(type) {
		case bool:
			param.Type = "boolean"
		case int, int64:
			param.Type = "integer"
		case float64:
			param.Type = "number"
		default:
			param.Type = "string"
		}

		def := f.GetDefault()
		if def != nil && def != "" && def != false && def != 0 {
			param.Default = def
		}

		schema.Parameters[f.GetName()] = param

		if f.IsRequired() {
			schema.Required = append(schema.Required, f.GetName())
		}
	}

	// Add positional args as parameters
	for _, arg := range c.args {
		param := ParamSchema{
			Type:        "string",
			Description: arg.Description,
		}
		if arg.Default != nil {
			param.Default = arg.Default
		}

		schema.Parameters[arg.Name] = param

		if arg.Required {
			schema.Required = append(schema.Required, arg.Name)
		}
	}

	return schema
}

// GenerateToolSchemaFromStruct creates a ToolSchema from a flags struct.
func GenerateToolSchemaFromStruct[T any](name, description string) ToolSchema {
	var t T
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	schema := ToolSchema{
		Name:        name,
		Description: description,
		Parameters:  make(map[string]ParamSchema),
	}

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("flag")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")
		paramName := parts[0]

		param := ParamSchema{
			Description: field.Tag.Get("help"),
		}

		// Enum
		if enum := field.Tag.Get("enum"); enum != "" {
			param.Enum = strings.Split(enum, ",")
		}

		// Type
		switch field.Type.Kind() {
		case reflect.Bool:
			param.Type = "boolean"
		case reflect.Int, reflect.Int64:
			param.Type = "integer"
		case reflect.Float64:
			param.Type = "number"
		default:
			param.Type = "string"
		}

		// Default
		if def := field.Tag.Get("default"); def != "" {
			param.Default = def
		}

		schema.Parameters[paramName] = param

		// Required
		if _, ok := field.Tag.Lookup("required"); ok {
			schema.Required = append(schema.Required, paramName)
		}
	}

	return schema
}

// PrintToolsJSON is a convenience handler to output tool schemas.
func PrintToolsJSON(ctx *Context) error {
	schemas := ctx.App().GetToolSchemas()
	data, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.Stdout(), string(data))
	return nil
}
