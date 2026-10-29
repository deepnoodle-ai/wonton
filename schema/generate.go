package schema

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
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

	seen := map[reflect.Type]bool{}

	// For non-struct types, create a schema preserving all property details
	if t.Kind() != reflect.Struct && (t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct) {
		prop, err := reflectType(t, opt, seen)
		if err != nil {
			return nil, err
		}
		// Convert Property to Schema, preserving all relevant fields
		return propertyToSchema(prop), nil
	}

	// For struct types, generate a full object schema
	prop, err := reflectType(t, opt, seen)
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
//
// seen tracks the struct types, and the named slice, array, map, and pointer
// types, currently in the recursion path so that self-referential types
// (e.g. linked-list nodes, tree nodes, type Tree map[string]Tree) terminate
// with a placeholder schema instead of overflowing the stack. The map is
// keyed by reflect.Type and entries are removed as the recursion unwinds,
// so non-cyclic shared types still produce full schemas at each occurrence.
func reflectType(t reflect.Type, opt GenerateOptions, seen map[reflect.Type]bool) (*Property, error) {
	// Check for time.Time first (before struct handling)
	if t == reflect.TypeOf(time.Time{}) {
		format := "date-time"
		return &Property{Type: String, Format: &format}, nil
	}

	// json.RawMessage holds arbitrary pre-encoded JSON, so any value is valid.
	if t == reflect.TypeOf(json.RawMessage{}) {
		return &Property{}, nil
	}

	// Byte slices marshal to base64 strings in encoding/json, not arrays.
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
		return &Property{Type: String}, nil
	}

	// Types implementing encoding.TextMarshaler (and not json.Marshaler)
	// marshal as JSON strings in encoding/json.
	if t.Kind() != reflect.Ptr && implementsTextMarshaler(t) {
		return &Property{Type: String}, nil
	}

	// Named slice, array, map, and pointer types can refer to themselves
	// (type Tree map[string]Tree). Guard them like structs so the recursion
	// ends with a placeholder instead of overflowing the stack.
	switch t.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Ptr:
		if t.Name() != "" {
			if seen[t] {
				return recursivePlaceholder(t), nil
			}
			seen[t] = true
			defer delete(seen, t)
		}
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
		items, err := reflectType(t.Elem(), opt, seen)
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
		valueType, err := reflectType(t.Elem(), opt, seen)
		if err != nil {
			return nil, fmt.Errorf("failed to reflect map value type: %w", err)
		}
		// JSON Schema represents maps as objects with additionalProperties schema
		return &Property{
			Type:                       Object,
			AdditionalPropertiesSchema: valueType,
		}, nil

	case reflect.Struct:
		return reflectStruct(t, opt, seen)

	case reflect.Ptr:
		// Check for *time.Time
		if t.Elem() == reflect.TypeOf(time.Time{}) {
			format := "date-time"
			nullable := true
			return &Property{Type: String, Format: &format, Nullable: &nullable}, nil
		}
		// For pointer types, reflect the underlying type and mark as nullable
		underlying, err := reflectType(t.Elem(), opt, seen)
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
//
// If t is already present in seen, the type is part of the current
// recursion path (a self-referential or mutually recursive type). In that
// case reflectStruct returns a permissive placeholder rather than
// recursing further; without this guard the recursion would overflow the
// stack on types like linked-list nodes or tree nodes.
func reflectStruct(t reflect.Type, opt GenerateOptions, seen map[reflect.Type]bool) (*Property, error) {
	if seen[t] {
		return recursivePlaceholder(t), nil
	}
	seen[t] = true
	defer delete(seen, t)

	properties := make(map[string]*Property)
	var required []string
	requiredSet := make(map[string]bool)
	addRequired := func(name string) {
		if !requiredSet[name] {
			requiredSet[name] = true
			required = append(required, name)
		}
	}

	// Resolve direct and promoted embedded struct fields using encoding/json's
	// dominance rules: shallower fields shadow deeper ones, a single tagged
	// field at the shallowest depth wins over untagged siblings, and multiple
	// conflicting fields at the shallowest depth cancel each other out.
	embedSeen := map[reflect.Type]bool{t: true}
	candidates := collectStructFields(t, 0, false, nil, seen, embedSeen)
	winners := resolveDominantFields(candidates)

	for _, c := range winners {
		// Temporarily mark any enclosing embedded struct types as seen while
		// reflecting this field so self-referential fields on embedded structs
		// terminate with a placeholder.
		var addedSeen []reflect.Type
		for _, enc := range c.enclosing {
			if !seen[enc] {
				seen[enc] = true
				addedSeen = append(addedSeen, enc)
			}
		}
		prop, err := reflectType(c.field.Type, opt, seen)
		for _, enc := range addedSeen {
			delete(seen, enc)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to reflect field %s: %w", c.field.Name, err)
		}

		// encoding/json encodes string, numeric, and boolean fields (or
		// pointers to them) with the ",string" option as JSON strings.
		if c.stringOpt && supportsJSONStringOption(c.field.Type) {
			prop.Type = String
		}

		// Apply field tags to the property
		applyFieldTags(prop, c.field)

		// Pointer-embedded structs are optional, so their promoted fields are
		// not required on the enclosing struct.
		if !c.fromPtrEmbed && checkRequired(c.field, c.isRequired) {
			addRequired(c.jsonName)
		}

		properties[c.jsonName] = prop
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

// structFieldCandidate records a direct or promoted struct field discovered
// while walking a struct and its flattened embedded structs.
type structFieldCandidate struct {
	field        reflect.StructField
	jsonName     string
	isRequired   bool
	tagged       bool
	stringOpt    bool
	depth        int
	fromPtrEmbed bool
	enclosing    []reflect.Type
}

// collectStructFields walks t and any flattened anonymous struct fields in
// declaration order, recording each candidate field together with its embedding
// depth and tag metadata.
func collectStructFields(
	t reflect.Type,
	depth int,
	fromPtrEmbed bool,
	enclosing []reflect.Type,
	seen map[reflect.Type]bool,
	embedSeen map[reflect.Type]bool,
) []structFieldCandidate {
	var out []structFieldCandidate
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !isJSONField(field) || field.Tag.Get("json") == "-" {
			continue
		}

		if field.Anonymous && shouldFlattenEmbedded(field) {
			elem := field.Type
			isPtr := false
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
				isPtr = true
			}
			if seen[elem] || embedSeen[elem] {
				continue
			}
			embedSeen[elem] = true
			nextEnclosing := append(append([]reflect.Type(nil), enclosing...), elem)
			out = append(out, collectStructFields(elem, depth+1, fromPtrEmbed || isPtr, nextEnclosing, seen, embedSeen)...)
			delete(embedSeen, elem)
			continue
		}

		jsonName, isRequired := parseJSONTag(field)
		tagName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		out = append(out, structFieldCandidate{
			field:        field,
			jsonName:     jsonName,
			isRequired:   isRequired,
			tagged:       tagName != "",
			stringOpt:    hasJSONStringOption(field),
			depth:        depth,
			fromPtrEmbed: fromPtrEmbed,
			enclosing:    enclosing,
		})
	}
	return out
}

// resolveDominantFields filters candidates using encoding/json's field
// dominance rules while preserving the declaration order of surviving fields.
func resolveDominantFields(candidates []structFieldCandidate) []structFieldCandidate {
	byName := make(map[string][]int, len(candidates))
	for i, c := range candidates {
		byName[c.jsonName] = append(byName[c.jsonName], i)
	}

	winningIdx := make(map[int]bool, len(byName))
	for _, indices := range byName {
		if idx, ok := dominantCandidateIndex(candidates, indices); ok {
			winningIdx[idx] = true
		}
	}

	winners := make([]structFieldCandidate, 0, len(winningIdx))
	for i, c := range candidates {
		if winningIdx[i] {
			winners = append(winners, c)
		}
	}
	return winners
}

// dominantCandidateIndex implements encoding/json's dominantField rule for all
// candidates sharing the same JSON name.
func dominantCandidateIndex(candidates []structFieldCandidate, indices []int) (int, bool) {
	minDepth := candidates[indices[0]].depth
	for _, idx := range indices[1:] {
		if candidates[idx].depth < minDepth {
			minDepth = candidates[idx].depth
		}
	}

	var shallowest []int
	var tagged []int
	for _, idx := range indices {
		if candidates[idx].depth == minDepth {
			shallowest = append(shallowest, idx)
			if candidates[idx].tagged {
				tagged = append(tagged, idx)
			}
		}
	}

	if len(tagged) > 0 {
		if len(tagged) == 1 {
			return tagged[0], true
		}
		return 0, false
	}
	if len(shallowest) == 1 {
		return shallowest[0], true
	}
	return 0, false
}

// implementsTextMarshaler reports whether a concrete type t (or *t) implements
// encoding.TextMarshaler without implementing json.Marshaler. Interface types
// are excluded because encoding/json marshals their dynamic value, which may
// not be a JSON string.
func implementsTextMarshaler(t reflect.Type) bool {
	if t.Kind() == reflect.Interface {
		return false
	}
	if t.Implements(jsonMarshalerType) || reflect.PointerTo(t).Implements(jsonMarshalerType) {
		return false
	}
	return t.Implements(textMarshalerType) || reflect.PointerTo(t).Implements(textMarshalerType)
}

// hasJSONStringOption reports whether field's json tag includes the ",string" option.
func hasJSONStringOption(field reflect.StructField) bool {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		return false
	}
	parts := strings.Split(jsonTag, ",")
	for _, part := range parts[1:] {
		if part == "string" {
			return true
		}
	}
	return false
}

// supportsJSONStringOption reports whether encoding/json honors the ",string"
// struct tag option for a field of type t (string, integer, floating-point, or
// boolean types, or pointers to them, that do not implement json.Marshaler).
func supportsJSONStringOption(t reflect.Type) bool {
	if t.Implements(jsonMarshalerType) || reflect.PointerTo(t).Implements(jsonMarshalerType) {
		return false
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		if t.Implements(jsonMarshalerType) || reflect.PointerTo(t).Implements(jsonMarshalerType) {
			return false
		}
	}
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	default:
		return false
	}
}

// recursivePlaceholder is the permissive schema used where a type refers to
// itself, to break the cycle.
func recursivePlaceholder(t reflect.Type) *Property {
	desc := fmt.Sprintf("recursive reference to %s; schema truncated to break cycle", t.String())
	switch t.Kind() {
	case reflect.Struct, reflect.Map:
		additionalProps := true
		return &Property{Type: Object, AdditionalProperties: &additionalProps, Description: desc}
	case reflect.Slice, reflect.Array:
		// Arrays need an items schema; {} accepts any element, as for []any.
		return &Property{Type: Array, Items: &Property{}, Description: desc}
	default:
		return &Property{Description: desc}
	}
}

// isJSONField reports whether a struct field can carry JSON input.
// Unexported fields are ignored, except embedded structs of unexported type:
// their exported fields are still promoted, and with a json name tag they
// decode as a named field. An embedded pointer to an unexported struct is
// ignored, because encoding/json cannot allocate it when decoding.
func isJSONField(field reflect.StructField) bool {
	if field.IsExported() {
		return true
	}
	return field.Anonymous && field.Type.Kind() == reflect.Struct
}

// shouldFlattenEmbedded reports whether an anonymous struct field's
// properties should be merged into the parent schema. encoding/json only
// flattens embedded structs (or pointers to structs) that don't carry a
// json name tag; everything else marshals as a regular named field.
func shouldFlattenEmbedded(field reflect.StructField) bool {
	t := field.Type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	return name == ""
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
		Type:                       prop.Type,
		Description:                prop.Description,
		Properties:                 prop.Properties,
		Required:                   prop.Required,
		AdditionalProperties:       prop.AdditionalProperties,
		AdditionalPropertiesSchema: prop.AdditionalPropertiesSchema,
		Items:                      prop.Items,
		Format:                     prop.Format,
		Nullable:                   prop.Nullable,
	}
}
