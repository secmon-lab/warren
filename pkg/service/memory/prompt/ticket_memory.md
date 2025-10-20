You are tasked with extracting organizational security knowledge from a resolved ticket.

## Ticket Information
{{ .ticket }}

## Comments History
{{ .comments }}

{{ if .existing_memory }}
## Existing Memory
{{ .existing_memory.Insights }}
{{ end }}

Generate a JSON response with insights:

{{ .json_schema }}

Focus on:
- Organizational context (environment, systems, policies)
- Common false positive patterns and how to identify them
- Typical true positive indicators
- Key points for threat assessment
- Important investigation checkpoints
- Organization-specific security risk factors
- Similar past incidents and their resolutions

Keep the response under 2000 characters. If updating existing memory, integrate new insights without losing valuable past knowledge.
