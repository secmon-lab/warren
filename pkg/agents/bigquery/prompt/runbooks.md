{{if .Runbooks}}
## SQL Runbooks

Available SQL runbook templates (use `get_runbook` tool with ID to get full SQL content):

{{range $id, $entry := .Runbooks}}- **ID**: `{{$id}}`
  **Title**: {{$entry.Title}}{{if $entry.Description}}
  **Description**: {{$entry.Description}}{{end}}
{{end}}

These runbooks contain pre-written SQL queries for common investigation patterns. Use the `get_runbook` tool to retrieve the full SQL content, which you can then adapt for your specific investigation needs.

{{end}}
