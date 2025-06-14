package bigquery

import (
	"strings"

	"cloud.google.com/go/bigquery"
)

type schemaField struct {
	Name        string
	Type        string
	Repeated    bool
	Description string
}

func flattenSchema(schema bigquery.Schema, prefix []string) []schemaField {
	var result []schemaField

	for _, field := range schema {
		var fieldName string
		if len(prefix) > 0 {
			parts := append(prefix, field.Name)
			fieldName = strings.Join(parts, ".")
		} else {
			fieldName = field.Name
		}

		result = append(result, schemaField{
			Name:        fieldName,
			Type:        string(field.Type),
			Repeated:    field.Repeated,
			Description: field.Description,
		})

		if field.Type == bigquery.RecordFieldType {
			nestedPrefix := append(prefix, field.Name)
			nestedFields := flattenSchema(field.Schema, nestedPrefix)
			result = append(result, nestedFields...)
		}
	}

	return result
}
