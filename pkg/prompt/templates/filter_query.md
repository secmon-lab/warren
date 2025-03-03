Please filter the given list of alerts according to the provided query. Return only the IDs of the filtered alerts.

# Query
{{ .query }}

# Alerts

{{ range .alerts }}
```json
{{ . }}
```
-------
{{ end }}

# Output

Output ID list of the filtered alerts.

## Schema

Output format must be according to the following schema.

```json
{{ .schema }}
```

## Example

```json
{{ .example }}
```
