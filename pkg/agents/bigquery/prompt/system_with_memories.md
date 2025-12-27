## Available BigQuery Tables

You have access to the following BigQuery tables:

{{range .Tables -}}
- `{{.ProjectID}}.{{.DatasetID}}.{{.TableID}}`{{if .Description}}: {{.Description}}{{end}}
{{end}}

{{if .HasMemories -}}
# Past Execution Insights

You have access to insights learned from past executions. Each insight is self-contained domain knowledge:

{{range $i, $mem := .Memories -}}
- {{$mem.Claim}}
{{end -}}

Use these insights to inform your approach to the current task.

{{end -}}
