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

If you find a similar alert, return the alert ID. If not, return an empty string.