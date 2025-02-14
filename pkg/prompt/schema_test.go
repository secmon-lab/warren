package prompt_test

import (
	"testing"

	"github.com/secmon-lab/warren/pkg/prompt"
)

type TestStruct struct {
	String    string   `json:"string"`
	Int       int      `json:"int"`
	Float     float64  `json:"float"`
	Bool      bool     `json:"bool"`
	StringArr []string `json:"string_arr"`
	Nested    struct {
		Field string `json:"field"`
	} `json:"nested"`
	Ignored string `json:"-"`
}

func TestGenerateSchema(t *testing.T) {
	schema := prompt.GenerateSchema(TestStruct{})

	if got := schema.Schema; got != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("schema[$schema] = %v, want %v", got, "http://json-schema.org/draft-07/schema#")
	}
	if got := schema.Type; got != "object" {
		t.Errorf("schema[type] = %v, want %v", got, "object")
	}
	if got := schema.AdditionalProps; got != false {
		t.Errorf("schema[additionalProperties] = %v, want false", got)
	}

	properties := schema.Properties

	// Test string field
	if got := properties["string"].Type; got != "string" {
		t.Errorf("properties[string][type] = %v, want string", got)
	}

	// Test int field
	if got := properties["int"].Type; got != "integer" {
		t.Errorf("properties[int][type] = %v, want integer", got)
	}

	// Test float field
	if got := properties["float"].Type; got != "number" {
		t.Errorf("properties[float][type] = %v, want number", got)
	}

	// Test bool field
	if got := properties["bool"].Type; got != "boolean" {
		t.Errorf("properties[bool][type] = %v, want boolean", got)
	}

	// Test string array field
	stringArr := properties["string_arr"]
	if got := stringArr.Type; got != "array" {
		t.Errorf("string_arr[type] = %v, want array", got)
	}
	if got := stringArr.Items.Type; got != "string" {
		t.Errorf("string_arr[items][type] = %v, want string", got)
	}

	// Test nested struct field
	nested := properties["nested"]
	if got := nested.Type; got != "object" {
		t.Errorf("nested[type] = %v, want object", got)
	}
	if got := nested.Properties["field"].Type; got != "string" {
		t.Errorf("nested[properties][field][type] = %v, want string", got)
	}

	// Test ignored field
	if _, exists := properties["Ignored"]; exists {
		t.Error("Ignored field should not exist in properties")
	}
}
