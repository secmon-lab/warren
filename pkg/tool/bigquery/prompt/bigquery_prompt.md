## Available BigQuery Tables

You have access to the following BigQuery tables for investigation:

{{range .configs}}
### Project: {{$.projectID}}, Dataset: {{.DatasetID}}, Table: {{.TableID}}
{{if .Description}}**Description**: {{.Description}}

{{end}}
{{end}}

**Note**: For detailed column information and schema, use the `bigquery_table_summary` tool.

{{if .runbooks}}
## SQL Runbooks

Available runbook entries (use `get_runbook_entry` with ID to get details):

{{range $id, $entry := .runbooks}}- ID: {{$id}}, Title: {{$entry.Title}}
{{end}}

{{end}}
## BigQuery Exploratory Investigation Strategy

**CRITICAL**: When a BigQuery query returns 0 results, do NOT immediately conclude the data doesn't exist. Instead, follow this systematic exploration approach:

### 1. Verify Field Values
- If filtering by specific values returns 0 results, first explore what values actually exist
- Example: If `WHERE user_email = 'specific@email.com'` returns nothing, run:
  `SELECT DISTINCT user_email FROM table_name LIMIT 10`

### 2. Progressive Search Refinement
- **Start broad, then narrow**: Begin with fewer constraints, then add filters
- **Test variations**: Try partial matches with LIKE, case-insensitive searches
- **Remove constraints**: Temporarily remove restrictive WHERE clauses to see if data exists

### 3. Investigate Data Patterns
- **Sample data**: Use `SELECT * FROM table_name LIMIT 5` to understand actual data format
- **Check date ranges**: Verify timestamp fields align with your expected time range
- **Explore distinct values**: Use `SELECT DISTINCT field_name FROM table_name LIMIT 20` for key fields

### 4. Iterative Query Development
- **Debug step-by-step**: If complex query returns 0 results, break it into simpler parts
- **Validate assumptions**: Confirm each WHERE condition individually
- **Cross-reference**: Check related tables or fields that might contain relevant data

### Example Investigation Flow:
1. Initial query returns 0 results
2. Check what values exist: `SELECT DISTINCT suspicious_field FROM table LIMIT 10`
3. Adjust original query based on actual values found
4. Progressively add back constraints until you find the right data or confirm absence