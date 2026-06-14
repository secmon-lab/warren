package bigquery

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed prompt/bigquery_prompt.md
var bigquerySystemPromptTemplate string

func bigquerySystemPrompt(data map[string]any) (string, error) {
	tmpl := template.Must(template.New("bigquery_prompt").Parse(bigquerySystemPromptTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
