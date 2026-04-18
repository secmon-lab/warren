Analyze the structured data of the given security alert and generate the corresponding metadata.

## Input

```json
{{ .alert }}
```

## Available Tags

The following tags are registered in the system. Use them to classify this alert by selecting the most relevant ones by name. Do NOT invent new tag names — only pick from the list below. If no tag is a good fit, leave `tags` empty.

{{ .available_tags }}

## Output

Output the result in JSON format. Schema is described below as a JSON schema.

```json
{{ .schema }}
```

Fields rules are as follows. Language of `title` and `description` must be {{ .lang }}.

- `title`: Provide a one-line title that summarizes the entire alert in natural language, clearly indicating what actor/subject performed what action against which resource/target and what potential impact occurred. The title should be written as a natural sentence or phrase that describes the security event, incorporating the key elements (actor, action, target, impact) in a readable format. Use the field values from the original alert data as much as possible; however, transform technical identifiers into more readable forms when appropriate. Title must be less than 140 characters.
- `description`: Give a concise summary of the alert.
- `attrs`: Extract the field values that highlight the key characteristics of the alert. Focus particularly on unique identifiers that can be used for investigation such as IP addresses, host names, domain names, usernames, resource IDs, etc. Avoid redundancy and aim for 3-9 attributes that provide comprehensive context for investigation.
- `tags`: Select zero or more tag names from the "Available Tags" list above that best classify this alert. Each entry MUST exactly match a `name` from that list (case-sensitive). Do not output tag names that are not in the list. If no tag applies, output an empty array.
