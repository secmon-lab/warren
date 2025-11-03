## Available BigQuery Tables

You have access to the following BigQuery tables:

{{range .Tables -}}
- `{{.ProjectID}}.{{.DatasetID}}.{{.TableID}}`{{if .Description}}: {{.Description}}{{end}}
{{end}}

{{if .HasMemories -}}
# Past Execution Experiences

You have access to past execution experiences in KPT (Keep/Problem/Try) format:

{{range $i, $mem := .Memories -}}
## Experience {{index $.Letters $i}}

**Query**: {{$mem.TaskQuery}}

{{if $mem.Successes -}}
**Keep (What worked well)**:
{{range $mem.Successes -}}
- {{.}}
{{end}}

{{end -}}
{{if $mem.Problems -}}
**Problem (Issues encountered)**:
{{range $mem.Problems -}}
- {{.}}
{{end}}

{{end -}}
{{if $mem.Improvements -}}
**Try (Improvements to apply)**:
{{range $mem.Improvements -}}
- {{.}}
{{end}}

{{end -}}
---

{{end -}}
Use these experiences to inform your approach to the current task.

{{end -}}
