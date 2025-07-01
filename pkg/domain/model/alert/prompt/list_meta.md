Analyze the structured data of the given security alert set and generate the corresponding metadata. You should have pre-analyzed summary of the alert set.

## Input (Summary)

{{ .summary }}

## Output

Output the result in JSON format. Schema is described below as a JSON schema. It must one map object, not an array.

```json
{{ .schema }}
```

Fields rules are as follows. Language of `title` and `description` must be {{ .lang }}.

- `title`: Provide a one-line title that summarizes the entire alert group, clearly indicating what actor/subject performed what action against which resource/target and what potential impact occurred. The title should follow the pattern **"Actor/Subject → Action → Resource/Target → Impact"** where possible and accurately represent the characteristics of the alert group. It should be immediately understandable to security analysts about what kind of events are included. Use the field values from the original alert data as much as possible; however, if these expressions are inadequate, update the original text or generate a new one. Title must be less than 140 characters. (around 100 characters max)
- `description`: Give a concise summary of the alert. (around 250 characters max)
- `attributes`: Attributes of the alert set (around 10 items max) - extract characteristics that represent the set of alerts from the alert contents

### Example

```json
{
  "title": "Test Alert List",
  "description": "This is a test alert list",
  "attributes": [
    {
      "key": "attribute1",
      "value": "value1"
    }
  ]
}
```
