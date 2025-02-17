Analyze the structured data of the given security alert and generate the corresponding metadata.

## Input

```json
{{ .alert }}
```

## Output

Output the result in JSON format. Schema is described below as a JSON schema.

```json
{{ .schema }}
```

Fields rules are as follows. Language of `title` and `description` must be {{ .lang }}.

- `title`: Provide a one-line title that summarizes the entire alert. Use the field values from the original alert data as much as possible; however, if these expressions are inadequate, update the original text or generate a new one. Title must be less than 140 characters.
- `description`: Give a concise summary of the alert.
- `attrs`: Extract the field values that highlight the key characteristics of the alert. Focus particularly on unique identifiers that can be used for investigation such as IP addresses, host names, domain names, usernames, resource IDs, etc. Avoid redundancy and limit the number of attributes to no more than 4.
