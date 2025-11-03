You are analyzing the execution history that you just completed to extract learnings.

{{ if .existing_memory }}
## Current Accumulated Knowledge
The following is the knowledge accumulated from past executions:

**Keep (Successful Patterns):**
{{ .existing_memory.Keep }}

**Change (Areas for Improvement):**
{{ .existing_memory.Change }}

**Notes (Other Insights):**
{{ .existing_memory.Notes }}
{{ end }}

{{ if .error }}
## Execution Result
This execution ended with an error:
{{ .error }}
{{ end }}

Based on the execution history in this session and the existing knowledge above, generate a JSON response:

{{ .json_schema }}

Focus on:
- **Summary**: A concise 1-2 sentence overview of what this execution accomplished and the key learnings. This will be used for semantic search to find relevant past experiences.

- **Keep** (Successful Execution Strategies): Document specific, actionable patterns that worked well
  - Name the exact tools used and how they were used (e.g., "Used BigQuery agent with query: SELECT ... WHERE timestamp > ...", "VirusTotal lookup with hash: ...")
  - Successful query patterns, search conditions, or command examples
  - Effective data retrieval methods and their results
  - Which tool combinations worked well together

- **Change** (Areas for Improvement): Document specific failures and their root causes
  - Which tools failed and exactly why (e.g., "BigQuery query failed because field name was 'event_time' not 'timestamp'")
  - Root causes of data retrieval failures (wrong field names, incorrect types, missing filters)
  - Inefficient processes and concrete optimization suggestions
  - Specific mistakes to avoid (e.g., "Don't use user.id for searches, use user.email instead")

- **Notes**: Additional insights
  - General observations about the data schema or environment
  - Unexpected patterns or edge cases discovered
  - Contextual information that might be useful later

Keep the total response under 2000 characters. If updating existing memory, integrate new learnings without losing valuable past insights. If there are no new learnings, you can return the existing memory as-is.
