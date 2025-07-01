Please summarize the given set of security alerts. You will need to output the following information:

- `title`: Title of the alert set (around 100 characters max). This should be human readable and concise, written in natural language that clearly describes what actor/subject performed what action against which resource/target and what potential impact occurred. The title should be written as a natural sentence or phrase that describes the security incident, transforming technical identifiers into more readable forms when appropriate. In particular, generate a title that makes the type of alert and threat immediately apparent.
- `description`: Description of the alert set (around 300 characters max). This should be human readable and concise. In particular, generate an explanation that provides a deeper overview than the title, allowing for quick understanding of the alert situation.
- `summary`: This differs from the above two in that it will be used later by AI to analyze Alerts. There is no character limit. Instead, please summarize the following information:
  - Indicators: Summarize identifiers that appear in alerts. This will be used for IoC searches on external sites and log searches.
  - Affected Assets: Summarize assets affected that appear in alerts. This is important information for measuring scope and degree of impact.
  - Context: Summarize the context that appears in alerts. This is important information for understanding the background of alerts and threats.
  - Correlation: If there is noteworthy information about connections between alerts, please summarize it.
  - Impact: If there is information that already indicates the impact status of alerts, please summarize it. Information indicating success/failure of attacks is particularly important.

{{ if .summary }}

## Input (Summary)

{{ .summary }}
{{ end }}

{{ if .schema }}
## Output

Output the result in JSON format. Schema is described below as a JSON schema. It must one map object, not an array.

```json
{{ .schema }}
```

The language of fields must be {{ .lang }}.

### Example

```json
{
  "title": "Test Alert List",
  "description": "This is a test alert list",
  "summary": "This is a test alert list"
}
```
{{ end }}
