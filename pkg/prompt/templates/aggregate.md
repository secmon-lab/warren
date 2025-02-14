Find a similar security alert with a new alert to aggregate them.

# Input

## New alert

```json
{{ .new }}
```

## Candidate alerts

{{ range .candidates }}
```json
{{ . }}
```
---
{{ end }}

## Output

The output is a JSON object that matches the following schema:

```json
{{ .schema }}
```

If you find a similar alert, return the alert ID. If not, return an empty string in `alert_id` field.
