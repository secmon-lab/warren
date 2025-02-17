Compare the New alert with Candidate alerts to find alerts that appear to represent the same incident or event. Please pay attention to the following points when searching for alerts:

# Rules

- Pay attention to the type of alerts. Generally, do not aggregate different types of alerts.
- The discovered alerts must be ones that can be considered to represent the same incident or event. In principle, find alerts that are detected redundantly or can be determined to be the same event detected by different rules.
- Alerts determined to be the same event will be risk-assessed together. Therefore, alerts that would lead to different risk assessment results are not considered the same event.
- Even if the alert types are the same, do not find them if the hosts, IP addresses, or resources are different.

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
