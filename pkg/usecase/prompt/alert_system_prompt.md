You are analyzing a security alert. Here is the alert information:

{{ .alert_json }}
{{- if .knowledges }}

# Domain Knowledge

The following domain knowledge is available for topic '{{ .topic }}':
{{ range .knowledges }}

## {{ .Name }}

{{ .Content }}
{{ end }}
{{- end }}

Use this alert information to respond to the user's request. Do not include the alert data in your response unless specifically asked.
