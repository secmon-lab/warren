# Memory Feedback Evaluation

You are evaluating how useful a past memory was for completing a task.

## Memory Being Evaluated

**Task Query:** {{ .MemoryTaskQuery }}

**KPT Summary:**
{{- if gt (len .MemorySuccesses) 0 }}
- **Keep (Successes):**
  {{- range .MemorySuccesses }}
  - {{ . }}
  {{- end }}
{{- end }}
{{- if gt (len .MemoryProblems) 0 }}
- **Problems:**
  {{- range .MemoryProblems }}
  - {{ . }}
  {{- end }}
{{- end }}
{{- if gt (len .MemoryImprovements) 0 }}
- **Try (Improvements):**
  {{- range .MemoryImprovements }}
  - {{ . }}
  {{- end }}
{{- end }}
{{- if and (eq (len .MemorySuccesses) 0) (eq (len .MemoryProblems) 0) (eq (len .MemoryImprovements) 0) }}
- No KPT data available
{{- end }}

## Current Task

**Query:** {{ .TaskQuery }}

## Execution Result

{{ if .ExecError }}
**Status:** Failed
**Error:** {{ .ExecError }}
{{ else }}
**Status:** Successful
{{ end }}

{{ if .ExecResultContent }}
**LLM Response:**
```
{{ .ExecResultContent }}
```
{{ end }}

## Evaluation Criteria

Please evaluate the memory on three dimensions:

### 1. Relevance (0-3 points)
Was the memory relevant to the current task?
- 0 = Not relevant at all (completely different domain/problem)
- 1 = Somewhat relevant (related but not directly applicable)
- 2 = Relevant (applicable to this task)
- 3 = Highly relevant (directly addresses the same problem)

### 2. Support (0-4 points)
Did the memory help find the solution or avoid mistakes?
- 0 = No help (ignored or not applicable)
- 1 = Minor help (provided some context)
- 2 = Moderate help (guided approach or avoided some mistakes)
- 3 = Significant help (key insights that shaped the solution)
- 4 = Critical help (solution would have failed without it)

### 3. Impact (0-3 points)
Did the memory contribute to the final result?
- 0 = No impact (result would be the same without it)
- 1 = Minor impact (slightly improved approach)
- 2 = Moderate impact (noticeably improved result)
- 3 = Major impact (significantly improved success or prevented failure)

## Output Format

Respond with a JSON object matching this schema:

```json
{{ .JSONSchema }}
```

Provide thoughtful scores based on the actual impact of the memory on this execution.
