package schema

import (
	"encoding"
	"encoding/json"
	"regexp"
	"strconv"
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

type outerTaggedUnexportedPtr struct {
	*unexportedBase `json:"base"`
	Count           int `json:"count"`
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

	// A pointer to an unexported struct is ignored: encoding/json cannot
	// allocate it, so input for its fields would fail to decode.
	s, err = Generate(outerUnexportedPtrEmbed{})
	assert.NoError(t, err)
	assert.NotContains(t, s.Properties, "label")
	assert.NotContains(t, s.Properties, "note")
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

	// A json-tagged pointer to an unexported struct is ignored too.
	s, err = Generate(outerTaggedUnexportedPtr{})
	assert.NoError(t, err)
	assert.NotContains(t, s.Properties, "base")
	assert.Equal(t, []string{"count"}, s.Required)
}

type customTextValue struct {
	raw string
}

func (c customTextValue) MarshalText() ([]byte, error) {
	return []byte(c.raw), nil
}

func (c *customTextValue) UnmarshalText(b []byte) error {
	c.raw = string(b)
	return nil
}

// textOutOnly has MarshalText but no UnmarshalText, so encoding/json decodes
// it from a JSON number.
type textOutOnly int

func (t textOutOnly) MarshalText() ([]byte, error) {
	return []byte(strconv.Itoa(int(t))), nil
}

// textInOnly has UnmarshalText but no MarshalText, so encoding/json decodes
// it from a JSON string.
type textInOnly int

func (t *textInOnly) UnmarshalText(b []byte) error {
	n, err := strconv.Atoi(string(b))
	*t = textInOnly(n)
	return err
}

// customJSONInt implements json.Marshaler, so encoding/json calls MarshalJSON
// and ignores the ",string" option on fields of this type.
type customJSONInt int

func (c customJSONInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(c))
}

// namedIntPtr is a named pointer type; encoding/json ignores ",string" on it.
type namedIntPtr *int

// encoding/json decodes fields with json:",string" (for string, numeric, and
// boolean types, or pointers to them) and types implementing
// encoding.TextUnmarshaler from JSON strings.
func TestGenerateJSONStringTagAndTextMarshaler(t *testing.T) {
	type payload struct {
		ID      int64                  `json:"id,string"`
		Price   float64                `json:"price,string,omitempty"`
		Enabled bool                   `json:"enabled,string"`
		OptID   *int                   `json:"opt_id,string,omitempty"`
		Slice   []int                  `json:"slice,string"` // ,string ignored by encoding/json on slices
		Token   customTextValue        `json:"token"`
		OptTok  *customTextValue       `json:"opt_token,omitempty"`
		Iface   encoding.TextMarshaler `json:"iface,omitempty"`
		Custom  customJSONInt          `json:"custom,string"`
		Named   namedIntPtr            `json:"named,string"`
		OutOnly textOutOnly            `json:"out_only"`
		InOnly  textInOnly             `json:"in_only"`
	}

	s, err := Generate(payload{})
	assert.NoError(t, err)
	assert.Equal(t, String, s.Properties["id"].Type)
	assert.Equal(t, String, s.Properties["price"].Type)
	assert.Equal(t, String, s.Properties["enabled"].Type)
	assert.Equal(t, String, s.Properties["opt_id"].Type)
	assert.NotNil(t, s.Properties["opt_id"].Nullable)
	assert.True(t, *s.Properties["opt_id"].Nullable)
	assert.Equal(t, Array, s.Properties["slice"].Type)
	assert.Equal(t, String, s.Properties["token"].Type)
	assert.Equal(t, String, s.Properties["opt_token"].Type)
	assert.NotNil(t, s.Properties["opt_token"].Nullable)
	assert.True(t, *s.Properties["opt_token"].Nullable)
	assert.Empty(t, s.Properties["iface"].Type)
	assert.Equal(t, Integer, s.Properties["custom"].Type)
	assert.Equal(t, Integer, s.Properties["named"].Type)
	assert.Equal(t, Integer, s.Properties["out_only"].Type)
	assert.Equal(t, String, s.Properties["in_only"].Type)

	// Each schema type is one that json.Unmarshal accepts.
	var p payload
	assert.NoError(t, json.Unmarshal([]byte(`{"token":"x","out_only":1,"in_only":"2"}`), &p))

	// A struct that embeds a TextUnmarshaler promotes UnmarshalText and
	// decodes from a JSON string in encoding/json.
	type embedsTextMarshaler struct {
		customTextValue
		Extra int `json:"extra"`
	}
	s, err = Generate(embedsTextMarshaler{})
	assert.NoError(t, err)
	assert.Equal(t, String, s.Type)
}

// encoding/json drops conflicting embedded fields at the same depth (unless one
// is explicitly tagged and the other is untagged), and prefers shallower
// embedded fields over deeper ones regardless of declaration order.
func TestGenerateEmbeddedConflictAndDepthRules(t *testing.T) {
	type EmbedA struct {
		ID   string
		Name string `json:"name"`
	}
	type EmbedB struct {
		ID    int
		Extra string `json:"extra"`
	}
	type ConflictingSiblingEmbeds struct {
		EmbedA
		EmbedB
	}

	// Both EmbedA.ID and EmbedB.ID are at depth 1 with the same tag status:
	// encoding/json drops "ID" completely.
	s, err := Generate(ConflictingSiblingEmbeds{}, GenerateOptions{DisallowAdditionalProperties: true})
	assert.NoError(t, err)
	assert.NotContains(t, s.Properties, "ID")
	assert.Contains(t, s.Properties, "name")
	assert.Contains(t, s.Properties, "extra")
	assert.Equal(t, []string{"name", "extra"}, s.Required)

	// At the same depth, an explicitly json-tagged field wins over an untagged field.
	type UntaggedID struct {
		ID int
	}
	type TaggedID struct {
		ID string `json:"ID"`
	}
	type TaggedWins struct {
		UntaggedID
		TaggedID
	}
	s, err = Generate(TaggedWins{})
	assert.NoError(t, err)
	assert.Contains(t, s.Properties, "ID")
	assert.Equal(t, String, s.Properties["ID"].Type)
	assert.Equal(t, []string{"ID"}, s.Required)

	// A shallower embedded field (depth 1) wins over a deeper embedded field
	// (depth 2) even when the deeper struct is embedded first.
	type DeepInner struct {
		ID int `json:"id"`
	}
	type DeepHolder struct {
		DeepInner
	}
	type ShallowHolder struct {
		ID string `json:"id"`
	}
	type DeeperFirst struct {
		DeepHolder
		ShallowHolder
	}
	s, err = Generate(DeeperFirst{})
	assert.NoError(t, err)
	assert.Contains(t, s.Properties, "id")
	assert.Equal(t, String, s.Properties["id"].Type)
	assert.Equal(t, []string{"id"}, s.Required)

	// When the same struct is embedded twice at the same depth (diamond
	// embedding), its promoted fields conflict at that depth and are omitted.
	type SharedLeaf struct {
		Shared string
	}
	type LeftBranch struct {
		SharedLeaf
		Left string `json:"left"`
	}
	type RightBranch struct {
		SharedLeaf
		Right string `json:"right"`
	}
	type Diamond struct {
		LeftBranch
		RightBranch
	}
	s, err = Generate(Diamond{})
	assert.NoError(t, err)
	assert.NotContains(t, s.Properties, "Shared")
	assert.Contains(t, s.Properties, "left")
	assert.Contains(t, s.Properties, "right")
	assert.Equal(t, []string{"left", "right"}, s.Required)
}

type EmbedLoopA struct {
	*EmbedLoopB
	A string `json:"a"`
}

type EmbedLoopB struct {
	*EmbedLoopA
	B    string      `json:"b"`
	Next *EmbedLoopB `json:"next,omitempty"`
}

// Structs that embed each other terminate, and a struct reflected as a field
// type still gets the fields it promotes from an embedded struct that is
// already being reflected further up, as encoding/json encodes them.
func TestGenerateEmbeddingCycle(t *testing.T) {
	s, err := Generate(EmbedLoopA{})
	assert.NoError(t, err)
	assert.Contains(t, s.Properties, "a")
	assert.Contains(t, s.Properties, "b")
	assert.Contains(t, s.Properties, "next")
	assert.Equal(t, []string{"a"}, s.Required)

	next := s.Properties["next"]
	assert.Equal(t, Object, next.Type)
	assert.Contains(t, next.Properties, "a")
	assert.Contains(t, next.Properties, "b")
	assert.Contains(t, next.Properties["next"].Description, "recursive reference")
}

// With ",string" on a field that decodes from a JSON string, encoding/json
// decodes only a JSON string literal inside the string. The schema keeps the
// string type and adds a pattern for that literal.
func TestGenerateJSONStringTagOnStringField(t *testing.T) {
	type payload struct {
		Name     string      `json:"name,string"`
		OptName  *string     `json:"opt_name,string,omitempty"`
		Text     textInOnly  `json:"text,string"`
		Desc     string      `json:"desc,string" description:"Display name"`
		Color    string      `json:"color,string" enum:"red,a<b" default:"red" example:"a<b"`
		Pat      string      `json:"pat,string" pattern:"^\"[a-z]+\"$"`
		Nullable string      `json:"nullable,string" nullable:"true"`
		Plain    string      `json:"plain"`
		Slice    []string    `json:"slice,string"` // ,string ignored by encoding/json on slices
		Custom   customTextS `json:"custom,string"`
	}

	s, err := Generate(payload{})
	assert.NoError(t, err)

	for _, name := range []string{"name", "opt_name", "text", "color", "nullable"} {
		p := s.Properties[name]
		assert.Equal(t, String, p.Type, name)
		assert.NotNil(t, p.Pattern, name)
		assert.Equal(t, encodedStringPattern, *p.Pattern, name)
		assert.Equal(t, encodedStringDescription, p.Description, name)
	}
	assert.True(t, *s.Properties["opt_name"].Nullable)
	assert.True(t, *s.Properties["nullable"].Nullable)

	// Tags replace the generated description and pattern.
	assert.Equal(t, "Display name", s.Properties["desc"].Description)
	assert.Equal(t, encodedStringPattern, *s.Properties["desc"].Pattern)
	assert.Equal(t, `^"[a-z]+"$`, *s.Properties["pat"].Pattern)

	// Enum, default, and example values are encoded like the field's value.
	color := s.Properties["color"]
	assert.Equal(t, []any{`"red"`, `"a<b"`}, color.Enum)
	assert.Equal(t, any(`"red"`), color.Default)
	assert.Equal(t, any(`"a<b"`), color.Example)

	// Fields without an honored ",string" option are unchanged.
	assert.Nil(t, s.Properties["plain"].Pattern)
	assert.Empty(t, s.Properties["plain"].Description)
	assert.Equal(t, Array, s.Properties["slice"].Type)
	assert.Nil(t, s.Properties["slice"].Items.Pattern)
	assert.Nil(t, s.Properties["custom"].Pattern)

	// Number and bool fields with ",string" get no pattern.
	type numbers struct {
		ID int64 `json:"id,string"`
		OK bool  `json:"ok,string"`
	}
	s, err = Generate(numbers{})
	assert.NoError(t, err)
	assert.Nil(t, s.Properties["id"].Pattern)
	assert.Nil(t, s.Properties["ok"].Pattern)
}

// customTextS implements json.Marshaler, so encoding/json ignores ",string"
// on fields of this type.
type customTextS string

func (c customTextS) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

// A value that matches encodedStringPattern decodes into a string field with
// the ",string" option, and the values the pattern rejects fail to decode. The
// one exception is "null", which encoding/json accepts as no value; the
// pattern rejects it, so the schema stays strict but never wrong.
func TestEncodedStringPatternMatchesDecoding(t *testing.T) {
	type payload struct {
		Name string `json:"name,string"`
	}
	re := regexp.MustCompile(encodedStringPattern)

	for _, value := range []string{
		`""`, `"abc"`, `"a\"b"`, `"a\\b"`, `"a\/b"`, `"\b\f\n\r\t"`,
		`"éé"`, `"é😀"`, `"null"`,
	} {
		assert.True(t, re.MatchString(value), value)
		body, err := json.Marshal(map[string]string{"name": value})
		assert.NoError(t, err)
		var p payload
		assert.NoError(t, json.Unmarshal(body, &p), value)
	}

	for _, value := range []string{
		`abc`, `"abc`, `abc"`, ` "abc"`, `"abc" `, `"a"b"`, `"\"`,
		`"a\qb"`, `"\u00g0"`, `"\u00e"`, "\"a\tb\"", "\"a\x00b\"",
	} {
		assert.False(t, re.MatchString(value), value)
		body, err := json.Marshal(map[string]string{"name": value})
		assert.NoError(t, err)
		var p payload
		assert.Error(t, json.Unmarshal(body, &p), value)
	}

	// The schema's encoded enum value decodes to the Go value.
	var p payload
	assert.NoError(t, json.Unmarshal([]byte(`{"name":"\"a<b\""}`), &p))
	assert.Equal(t, "a<b", p.Name)
}
