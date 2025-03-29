Choose the next action to investigate the alert.

You must respond with the next action according to the following JSON schema format. The response must be valid JSON, DO NOT include any other text or markdown quotes.

```json
{{ .schema }}
```

# Actions

Choose the next action from the following actions.

## `done`

You have already investigated the alert and no further action is needed to decide the alert severity.

{{ range .actions }}
## `{{ .Name }}`

{{ .Description }}

### Arguments

Arguments are required to execute the action. Do not include any other arguments.
{{ range .Args }}
- `{{ .Name }}` ({{ .Type }}, {{ if .Required }}required{{ else }}optional{{ end }}): {{ .Description }} {{ if .Choices }} Choose one of the following values: {{ end }}
{{ range .Choices }}  - `{{ .Value }}`: {{ .Description }}
{{ end }}
{{ end }}
{{ end }}
