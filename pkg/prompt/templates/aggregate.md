Find a similar security alert with a new alert to aggregate them. Please pay close attention to the following points when looking for similar alerts:

- Pay careful attention to the type of alert. As a general rule, try not to aggregate different types of alerts.
- Even if the alert types are the same, do not aggregate them if they occur on different hosts, IP addresses, or resources where risk assessment and response should be conducted according to different criteria.

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

The output must be a JSON object that matches the following schema:

```json
{{ .schema }}
```

DO NOT return array schema, just return a single JSON object including `alert_id` field.

If you find a similar alert, return the alert ID. If not, return an empty string in `alert_id` field.
