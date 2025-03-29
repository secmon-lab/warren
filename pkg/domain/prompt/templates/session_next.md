# 次のアクションの選択

これまでの情報を元に問い合わせに答えるための次のアクションを選択してください。

## Selectable Actions

Choose the next action from the following actions.

## `done`

You have already investigated the alert and no further action is needed to decide the alert severity.

{{ range .actions }}

## `{{ .Name }}`

{{ .Description }}

### Arguments

Arguments are required to execute the action. Do not include any other arguments.
{{ range .Args }}
- `{{ .Name }}` ({{ .Type }}, {{ if .Required }}required{{ else }}optional{{ end }}): {{ .Description }} {{ if .Choices }} Choose one of the following values: {{ end }}
{{ range .Choices }}  - `{{ .Value }}`: {{ .Description }}
{{ end }}
{{ end }}
{{ end }}

## Output Schema

以下のスキーマに従って出力してください。必ず1つのマップ形式で出力するようにしてください。

```json
{{ .schema }}
```