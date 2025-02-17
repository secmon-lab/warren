Generate a SQL query to retrieve data from the table for security alert investigation.

# Requirements

- The query must be optimized for performance and cost.
- You must use the `LIMIT` clause to limit the number of rows to {{ .limit }}.
- You must consider filter conditions to focus on the relevant data that you need to investigate.
- You must use timestamp partition in the query and minimize the data scanned if the table is partitioned by timestamp. However, you should maximize the data scanned within partition type.
  - Since the billing amount is the same within the same partition type, maximize the use of the time window specified by the partition type.
  - If the table is partitioned by day, you should scan data within the same day.
  - Especially when checking for resource abuse, maximize the use of the time window specified by the partition. If deemed necessary, you may retrieve data beyond the partition.
- You must refer only the columns that are necessary for the investigation to minimize the data scanned.

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
