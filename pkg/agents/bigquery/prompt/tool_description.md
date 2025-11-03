## BigQuery Agent

You can query BigQuery tables using the `query_bigquery` tool. The agent will automatically check table schemas and construct appropriate queries.

**Important Guidelines**:
- The agent MUST check table schemas before constructing queries
- Results will be returned as raw data records without summarization
- All query fields will be preserved in the response

**How to Use**:
- Do NOT specify table names or SQL details in your query
- Focus on describing WHAT information you need, not HOW to get it
- Be clear about the data you want to retrieve (e.g., "login failures in the last 24 hours")
- The agent will automatically select appropriate tables and construct queries

{{if .HasTables -}}
### Available Tables

{{range .Tables -}}
- **`{{.ProjectID}}.{{.DatasetID}}.{{.TableID}}`**{{if .Description}}: {{.Description}}{{end}}
{{end}}
{{end -}}
{{if .ScanSizeLimit -}}
**Scan Size Limit**: {{.ScanSizeLimit}}
{{end -}}
{{if .QueryTimeout -}}
**Query Timeout**: {{.QueryTimeout}}
{{end -}}
