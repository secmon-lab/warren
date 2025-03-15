Analyze the structured data of the given security alert set and generate the corresponding metadata. If you find similarity between alerts, you should mention it in the description. If no similarities are found, there is no need to force mentioning it.

## Input

{{ range .alerts }}
```json
{{ . }}
```
{{ end }}

## Output

Output the result in JSON format. Schema is described below as a JSON schema.

```json
{{ .schema }}
```

Fields rules are as follows. Language of `title` and `description` must be {{ .lang }}.

- `title`: Provide a one-line title that summarizes the entire alert. The title should accurately represent the characteristics of the alert group and be immediately understandable to security analysts about what kind of events are included. Use the field values from the original alert data as much as possible; however, if these expressions are inadequate, update the original text or generate a new one. Title must be less than 140 characters.
- `description`: Give a concise summary of the alert.

### Example

```json
{
  "title": "Test Alert List",
  "description": "This is a test alert list"
}
```
