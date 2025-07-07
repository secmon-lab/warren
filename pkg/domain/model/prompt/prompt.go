package prompt

import (
	"bytes"
	"context"
	"encoding/json"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
)

func stringify(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "", goerr.Wrap(err, "failed to marshal", goerr.V("data", v))
	}
	return buf.String(), nil
}

func Generate(ctx context.Context, tmpl string, data map[string]any) (string, error) {
	builtTemplate, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	values := make(map[string]any)
	for k, v := range data {
		switch v := v.(type) {
		case string:
			values[k] = v
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			values[k] = v
		case float32, float64:
			values[k] = v
		default:
			raw, err := stringify(v)
			if err != nil {
				return "", err
			}
			values[k] = string(raw)
		}
	}

	var buf bytes.Buffer
	if err := builtTemplate.Execute(&buf, values); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}
	return buf.String(), nil
}

// GenerateWithStruct generates a prompt with data passed directly to template without JSON marshaling.
// This allows direct field access in templates (e.g., {{ .ticket.Title }})
func GenerateWithStruct(ctx context.Context, tmpl string, data map[string]any) (string, error) {
	builtTemplate, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse template")
	}

	var buf bytes.Buffer
	if err := builtTemplate.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute template")
	}
	return buf.String(), nil
}
