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
- `{{ .Name }}` ({{ .Type }}, {{ if .Required }}required{{ else }}optional{{ end }}): {{ .Description }} {{ if .Values }} Choose one of the following values: {{ end }}
{{ if .Values }}{{ range .Values }}  - `{{ .Value }}`: {{ .Description }}
{{ end }}
{{ end }}
{{ end }}
{{ end }}
