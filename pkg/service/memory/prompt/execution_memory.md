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
- **Keep**: Tool call successes, successful patterns and approaches, effective data retrieval methods
- **Change**: Tool call errors and their causes, data retrieval failures and why they occurred, inefficient processes that could be optimized
- **Notes**: General observations about the data, unexpected patterns, edge cases, contextual information about the environment

Keep the total response under 2000 characters. If updating existing memory, integrate new learnings without losing valuable past insights. If there are no new learnings, you can return the existing memory as-is.
