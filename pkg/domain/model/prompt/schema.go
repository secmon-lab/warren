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

func ToSchema(v interface{}) *jsonSchema {
	schema := &jsonSchema{
		Schema:          "http://json-schema.org/draft-07/schema#",
		Type:            "object",
		Properties:      make(map[string]jsonSchema),
		AdditionalProps: false,
	}
	// Initialize a map to track processed types to prevent circular references
	processedTypes := make(map[reflect.Type]*jsonSchema)
	parseStruct(reflect.TypeOf(v), schema, processedTypes)

	return schema
}

func (x *jsonSchema) Stringify() (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	if err := enc.Encode(x); err != nil {
		return "", goerr.Wrap(err, "failed to marshal json schema", goerr.V("schema", x))
	}
	return buf.String(), nil
}

func parseStruct(t reflect.Type, schema *jsonSchema, processedTypes map[reflect.Type]*jsonSchema) {
	// If the type has already been processed, copy its schema and return to avoid infinite recursion
	if existingSchema, exists := processedTypes[t]; exists {
		schema.Type = existingSchema.Type
		schema.Properties = existingSchema.Properties
		schema.AdditionalProps = existingSchema.AdditionalProps
		schema.Items = existingSchema.Items
		return
	}
	// Mark this type as processed
	processedTypes[t] = schema

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

		schema.Properties[fieldName] = getFieldSchema(field, processedTypes)
	}
}

func getFieldSchema(field reflect.StructField, processedTypes map[reflect.Type]*jsonSchema) jsonSchema {
	schema := jsonSchema{}

	// If the field is a pointer, get the element type
	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	switch fieldType.Kind() {
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
		itemSchema := getFieldSchema(reflect.StructField{Type: fieldType.Elem()}, processedTypes)
		schema.Items = &itemSchema
	case reflect.Map:
		schema.Type = "object"
		valueSchema := getFieldSchema(reflect.StructField{Type: fieldType.Elem()}, processedTypes)
		schema.AdditionalProps = true
		schema.Properties = make(map[string]jsonSchema)
		schema.Items = &valueSchema
	case reflect.Struct:
		schema.Type = "object"
		schema.Properties = make(map[string]jsonSchema)
		parseStruct(fieldType, &schema, processedTypes)
	}

	// If the field has a `schema:"optional"` tag, set AdditionalProps to true
	if field.Tag.Get("schema") == "optional" {
		schema.AdditionalProps = true
	}

	return schema
}
