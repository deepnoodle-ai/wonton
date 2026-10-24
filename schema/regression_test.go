package schema

import (
	"encoding/json"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

type jsonTree map[string]jsonTree

type nestedList []nestedList

type holdsTree struct {
	Tree  jsonTree   `json:"tree"`
	Items nestedList `json:"items,omitempty"`
}

// BUG-007: recursive named map and slice types must terminate.
func TestGenerateRecursiveMapAndSlice(t *testing.T) {
	s, err := Generate(jsonTree{})
	assert.NoError(t, err)
	assert.Equal(t, Object, s.Type)

	s, err = Generate(nestedList{})
	assert.NoError(t, err)
	assert.Equal(t, Array, s.Type)

	s, err = Generate(holdsTree{})
	assert.NoError(t, err)
	assert.Equal(t, Object, s.Properties["tree"].Type)
	assert.Equal(t, Array, s.Properties["items"].Type)
	_, err = json.Marshal(s)
	assert.NoError(t, err)
}

type EmbeddedBase struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type outerShadowsBefore struct {
	ID int `json:"id"`
	EmbeddedBase
}

type outerShadowsOptional struct {
	EmbeddedBase
	ID string `json:"id,omitempty"`
}

// BUG-067: an outer field shadows an embedded field regardless of order, and
// required names are unique and follow the winning field.
func TestGenerateEmbeddedShadowing(t *testing.T) {
	s, err := Generate(outerShadowsBefore{})
	assert.NoError(t, err)
	assert.Equal(t, Integer, s.Properties["id"].Type)
	assert.Equal(t, []string{"id", "name"}, s.Required)

	s, err = Generate(outerShadowsOptional{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"name"}, s.Required)

	type outerSame struct {
		EmbeddedBase
		ID string `json:"id"`
	}
	s, err = Generate(outerSame{})
	assert.NoError(t, err)
	assert.Equal(t, []string{"name", "id"}, s.Required)
}

// BUG-077: json:"-," names a field "-"; only json:"-" skips it.
func TestGenerateDashCommaTag(t *testing.T) {
	type dash struct {
		Dash string `json:"-,"`
		Skip string `json:"-"`
	}
	s, err := Generate(dash{})
	assert.NoError(t, err)
	_, ok := s.Properties["-"]
	assert.True(t, ok, "field tagged json:\"-,\" missing")
	assert.Equal(t, 1, len(s.Properties))
	assert.Equal(t, []string{"-"}, s.Required)

	type dashOptional struct {
		DashOpt string `json:"-,omitempty"`
	}
	s, err = Generate(dashOptional{})
	assert.NoError(t, err)
	_, ok = s.Properties["-"]
	assert.True(t, ok, "field tagged json:\"-,omitempty\" missing")
	assert.Equal(t, 0, len(s.Required))
}

// BUG-077: "additionalProperties": null means unset, not false.
func TestUnmarshalAdditionalPropertiesNull(t *testing.T) {
	var s Schema
	assert.NoError(t, json.Unmarshal([]byte(`{"type":"object","additionalProperties":null}`), &s))
	assert.Nil(t, s.AdditionalProperties)
	assert.Nil(t, s.AdditionalPropertiesSchema)

	var p Property
	assert.NoError(t, json.Unmarshal([]byte(`{"type":"object","additionalProperties":null}`), &p))
	assert.Nil(t, p.AdditionalProperties)
	assert.Nil(t, p.AdditionalPropertiesSchema)
}
