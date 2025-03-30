# Instructions

Choose the next action to take. You can select and execute actions repeatedly. You don't need to take actions if they're not necessary.


{{ if .results }}
## Input

Here is the result of the previous action.

{{ range .results }}
### {{ .Message }}
{{ range .Rows }}
```
{{ . }}
```
---
{{ end }}
{{ end }}
{{ end }}

## Output

You can respond with the following:

### Function

You can call the functions from function declarations.

### Exit

You can end the session by calling `exit` function. When you call this function, you don't need to respond with any message.

### Text

You can respond with a text message. The message is stored into memory. You can retrieve the stored message by calling `base.notes` function. It's useful to analyze big data. You can put the summary of the analysis into the message.

