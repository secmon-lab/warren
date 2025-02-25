Please group the given alerts by similarity according to the specified rules. This aggregation aims to facilitate the bulk processing of a large number of alerts.

# Rules

- Alerts with different `schema` should not be grouped together.
- An alert must belong to only one group.
- Similar types of alerts should belong to the same group.
- Aim to have a maximum of {{ .max_groups }} groups. However, if aggregation proves difficult, this limit may be exceeded.

# Input

Below are the alerts to be grouped.

{{ range .alerts }}
----------------
```json
{{ . }}
```
{{ end }}

# Output

Return the grouped alert IDs.

## Schema
The response must adhere to the following JSON schema format.

```json
{{ .schema }}
```

## Example of Response

Below is an example of the response.
```json
{{ .example }}
```

Please ensure that the alerts are categorized correctly based on their schemas, and provide the output in the specified JSON format.