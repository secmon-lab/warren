package bigquery

import (
	"bytes"
	_ "embed"
	"text/template"
)

//go:embed prompt/bigquery_query.md
var bigqueryQueryPromptTemplate string

func bigqueryQueryPrompt(scanLimit string) string {
	prompt := template.Must(template.New("bigquery_query").Parse(bigqueryQueryPromptTemplate))
	var buf bytes.Buffer
	if err := prompt.Execute(&buf, map[string]string{
		"limit": scanLimit,
	}); err != nil {
		return ""
	}
	return buf.String()
}
