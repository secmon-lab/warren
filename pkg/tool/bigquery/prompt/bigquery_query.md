Execute a BigQuery query and return the query ID.

## Critical Field Safety Requirements

- **MANDATORY**: Only use field names that are explicitly provided in the schema_fields list
- **FIELD VALIDATION**: Before referencing any field in SQL, verify it exists in the provided schema
- **NO FIELD GUESSING**: Never assume field names exist - only use confirmed field names from schema
- **SAFE SQL PATTERN**: Start with `SELECT * FROM table_name LIMIT 1000` to see structure, then use confirmed fields
- **NESTED FIELDS**: Use dot notation (e.g., `record_field.nested_field`) only if confirmed in schema
- **COLUMN LISTING**: Use `SELECT column_name FROM INFORMATION_SCHEMA.COLUMNS WHERE table_name = 'your_table'` if needed

## Query Execution Restrictions

- Before executing the query, a dry run will be performed and an error will be returned if the query scan size exceeds {{ .limit }} bytes
- If tables have partitioning configured, utilize it to the maximum extent. For example, with Day partitioning, only span across dates when necessary. Conversely, within a single date range, specify as long a time range as possible.
- Only select columns that are necessary for your investigation
- Use LIMIT clauses to reduce scan size when exploring data
- Add WHERE clauses with partition fields to reduce scan size if needed

## Field Validation Protocol

1. **VERIFY FIRST**: Check field exists in schema_fields before using in SQL
2. **START SIMPLE**: Use `SELECT *` first to understand table structure
3. **PROGRESSIVE QUERIES**: Add specific fields only after confirming they exist
4. **ERROR RECOVERY**: If field not found, use only fields from schema_fields list
