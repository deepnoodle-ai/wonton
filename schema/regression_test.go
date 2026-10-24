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
	assert.NotNil(t, s.Properties["items"].Items)
	assert.NotNil(t, s.Properties["items"].Items.Items, "recursive placeholder needs items")
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

type unexportedBase struct {
	Label string `json:"label"`
	Note  string `json:"note,omitempty"`
	id    string //nolint:unused // unexported fields stay hidden
}

type unexportedNumber int

type outerUnexportedEmbed struct {
	unexportedBase
	Count int `json:"count"`
}

type outerUnexportedPtrEmbed struct {
	*unexportedBase
	Count int `json:"count"`
}

type outerUnexportedNonStruct struct {
	unexportedNumber
	Count int `json:"count"`
}

type outerShadowsUnexported struct {
	unexportedBase
	Label int `json:"label"`
}

type outerTaggedUnexported struct {
	unexportedBase `json:"base"`
}

// Issue #103: encoding/json promotes the exported fields of an embedded
// struct whose type is unexported, so Generate must too.
func TestGenerateUnexportedEmbedded(t *testing.T) {
	s, err := Generate(outerUnexportedEmbed{}, GenerateOptions{DisallowAdditionalProperties: true})
	assert.NoError(t, err)
	assert.Contains(t, s.Properties, "label")
	assert.Equal(t, String, s.Properties["label"].Type)
	assert.Contains(t, s.Properties, "note")
	assert.Equal(t, Integer, s.Properties["count"].Type)
	assert.Equal(t, 3, len(s.Properties))
	assert.Equal(t, []string{"label", "count"}, s.Required)

	// A pointer to an unexported struct is walked too; its fields are optional.
	s, err = Generate(outerUnexportedPtrEmbed{})
	assert.NoError(t, err)
	assert.Contains(t, s.Properties, "label")
	assert.Equal(t, String, s.Properties["label"].Type)
	assert.Contains(t, s.Properties, "note")
	assert.Equal(t, []string{"count"}, s.Required)

	// An embedded non-struct of unexported type is ignored.
	s, err = Generate(outerUnexportedNonStruct{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(s.Properties))
	assert.Equal(t, []string{"count"}, s.Required)

	// A direct field shadows a field promoted from an unexported struct.
	s, err = Generate(outerShadowsUnexported{})
	assert.NoError(t, err)
	assert.Equal(t, Integer, s.Properties["label"].Type)
	assert.Equal(t, []string{"label"}, s.Required)

	// With a json name tag, an unexported embedded struct is a named field.
	s, err = Generate(outerTaggedUnexported{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(s.Properties))
	assert.Equal(t, Object, s.Properties["base"].Type)
	assert.Contains(t, s.Properties["base"].Properties, "label")
	assert.Equal(t, []string{"base"}, s.Required)
}
