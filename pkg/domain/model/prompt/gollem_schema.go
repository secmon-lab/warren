package prompt

import (
	"reflect"

	"github.com/m-mizutani/gollem"
)

// ToGollemSchema converts a Go struct to gollem.ResponseSchema
func ToGollemSchema(name, description string, v interface{}) *gollem.ResponseSchema {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := &gollem.ResponseSchema{
		Name:        name,
		Description: description,
		Schema:      toGollemParameter(t),
	}

	return schema
}

func toGollemParameter(t reflect.Type) *gollem.Parameter {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return &gollem.Parameter{
			Type: gollem.TypeString,
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &gollem.Parameter{
			Type: gollem.TypeInteger,
		}
	case reflect.Float32, reflect.Float64:
		return &gollem.Parameter{
			Type: gollem.TypeNumber,
		}
	case reflect.Bool:
		return &gollem.Parameter{
			Type: gollem.TypeBoolean,
		}
	case reflect.Slice:
		itemParam := toGollemParameter(t.Elem())
		return &gollem.Parameter{
			Type:  gollem.TypeArray,
			Items: itemParam,
		}
	case reflect.Struct:
		return structToGollemParameter(t)
	default:
		// Fallback to object for unknown types
		return &gollem.Parameter{
			Type: gollem.TypeObject,
		}
	}
}

func structToGollemParameter(t reflect.Type) *gollem.Parameter {
	param := &gollem.Parameter{
		Type:       gollem.TypeObject,
		Properties: make(map[string]*gollem.Parameter),
	}

	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get JSON tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := jsonTag
		if fieldName == "" {
			fieldName = field.Name
		}

		// Remove omitempty and other options from tag
		for idx := 0; idx < len(fieldName); idx++ {
			if fieldName[idx] == ',' {
				fieldName = fieldName[:idx]
				break
			}
		}

		// Get description from comment or tag
		description := field.Tag.Get("description")

		// Create parameter for this field
		fieldParam := toGollemParameter(field.Type)
		if description != "" {
			fieldParam.Description = description
		}

		param.Properties[fieldName] = fieldParam

		// Check if field is required (no omitempty tag and not a pointer)
		fullTag := field.Tag.Get("json")
		isOmitempty := false
		for j := 0; j < len(fullTag); j++ {
			if fullTag[j] == ',' && j+9 < len(fullTag) && fullTag[j+1:j+10] == "omitempty" {
				isOmitempty = true
				break
			}
		}

		if !isOmitempty && field.Type.Kind() != reflect.Ptr {
			required = append(required, fieldName)
		}
	}

	if len(required) > 0 {
		param.Required = required
	}

	return param
}
