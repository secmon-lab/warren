Please group the given alerts according to the specified rules.

# Rules

- Alerts with different `schema` should not be grouped together.
- An alert can belong to only one group.
- Similar types of alerts should belong to the same group.

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