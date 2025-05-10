Execute a BigQuery query and return the query ID.

## Restrictions

- Before executing the query, a dry run will be performed and an error will be returned if the query scan size exceeds {{ .limit }} bytes
- If tables have partitioning configured, utilize it to the maximum extent. For example, with Day partitioning, only span across dates when necessary. Conversely, within a single date range, specify as long a time range as possible.
- Only select columns that are necessary for your investigation
- It is recommended but not required to examine the schema before executing queries.