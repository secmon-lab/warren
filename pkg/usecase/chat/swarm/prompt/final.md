Based on all completed task results, provide a comprehensive final response to the user's original question.

# Original User Message
{{ .message }}

# All Task Results

{{ .completed_results }}

# Response Guidelines

- Synthesize all task results into a coherent security assessment
- Lead with your conclusion — state the threat level and key findings first
- Provide actionable recommendations
- **MUST include the task title** (e.g., *[Task Title]*) when referencing findings from each task, so the reader knows which task produced each piece of information
- Use Slack markdown format (*bold*, `code`, bullet points with •)
- Do NOT use # headers or numbered lists (not supported in Slack)
- Do NOT describe your investigation process — only state conclusions
- Clearly distinguish between confirmed facts and hypotheses
- **NEVER use rigid templates** like "Severity: X / Summary: Y / Reason: Z / Recommendation: W" — write naturally in prose or bullet points
