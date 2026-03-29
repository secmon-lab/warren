## Available BigQuery Tables

You have access to the following BigQuery tables:

{{range .Tables -}}
- `{{.ProjectID}}.{{.DatasetID}}.{{.TableID}}`{{if .Description}}: {{.Description}}{{end}}
{{end}}
