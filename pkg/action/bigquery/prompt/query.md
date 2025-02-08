Generate a SQL query to retrieve data from the table for security alert investigation.

# Requirements

- The query must be optimized for performance and cost.
- The query must be optimized for read only access to the data.
- The query must be optimized for the data volume and complexity.
- The query must have a limit of {{ .limit }} rows.

# Schema

The schema of the table `{{ .table_id }}` is as follows:

```json
{{ .schema }}
```

# Output

Output must be json format and have `query` field with SQL query. For example:

```json
{
  "query": "SELECT * FROM `{{ .table_id }}`"
}
```
