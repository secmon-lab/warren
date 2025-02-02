package prompt

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/m-mizutani/goerr/v2"
)

type jsonSchema struct {
	Schema          string                `json:"$schema"`
	Type            string                `json:"type"`
	Properties      map[string]jsonSchema `json:"properties"`
	AdditionalProps bool                  `json:"additionalProperties"`
	Items           *jsonSchema           `json:"items,omitempty"`
}

func generateSchema(v interface{}) *jsonSchema {
	schema := &jsonSchema{
		Schema:          "http://json-schema.org/draft-07/schema#",
		Type:            "object",
		Properties:      make(map[string]jsonSchema),
		AdditionalProps: false,
	}
	parseStruct(reflect.TypeOf(v), schema)

	return schema
}

func (x *jsonSchema) Stringify() (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	if err := enc.Encode(x); err != nil {
		return "", goerr.Wrap(err, "failed to marshal json schema")
	}
	return buf.String(), nil
}

func parseStruct(t reflect.Type, schema *jsonSchema) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := jsonTag
		if fieldName == "" {
			fieldName = field.Name
		}

		schema.Properties[fieldName] = getFieldSchema(field)
	}
}

func getFieldSchema(field reflect.StructField) jsonSchema {
	schema := jsonSchema{}

	switch field.Type.Kind() {
	case reflect.String:
		schema.Type = "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = "integer"
	case reflect.Float32, reflect.Float64:
		schema.Type = "number"
	case reflect.Bool:
		schema.Type = "boolean"
	case reflect.Slice:
		schema.Type = "array"
		itemSchema := getFieldSchema(reflect.StructField{Type: field.Type.Elem()})
		schema.Items = &itemSchema
	case reflect.Map:
		schema.Type = "object"
		valueSchema := getFieldSchema(reflect.StructField{Type: field.Type.Elem()})
		schema.AdditionalProps = true
		schema.Properties = make(map[string]jsonSchema)
		schema.Items = &valueSchema
	case reflect.Struct:
		schema.Type = "object"
		schema.Properties = make(map[string]jsonSchema)
		parseStruct(field.Type, &schema)
	}

	if field.Tag.Get("schema") == "optional" {
		schema.AdditionalProps = true
	}

	return schema
}
