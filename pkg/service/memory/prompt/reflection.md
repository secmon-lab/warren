# Agent Task Reflection

You are analyzing the execution of an agent task to extract new insights and evaluate the usefulness of provided memories.

## Task Query

**Query:** {{ .TaskQuery }}

## Provided Memories

The following memories were retrieved and made available during execution:

{{ if gt (len .UsedMemories) 0 }}
{{ range .UsedMemories }}
- **ID {{ .ID }}**: {{ .Claim }}
{{ end }}
{{ else }}
No memories were provided for this task.
{{ end }}

## Execution History

The agent's execution proceeded as follows:

{{ .ExecutionHistory }}

## Your Task

Please analyze the execution and provide:

### 1. New Claims (Insights)

Extract specific, actionable domain knowledge learned during this execution. Each claim MUST be:

- **Self-Contained**: Readable and understandable WITHOUT knowing the original query or task context
- **Concrete**: Specific facts, patterns, or rules discovered (not vague generalizations)
- **Actionable**: Directly useful for similar future tasks
- **Contextualized**: Explains WHY it matters or what problem it solves

**CRITICAL**: The claim should make sense to someone who has NEVER seen the original task. Include all necessary context within the claim itself.

**Format**: "[Specific context/entity] [What was discovered] - [Why it matters/What problem it solves]"

Examples of **EXCELLENT** claims (self-contained, specific, actionable):
- "BigQuery table 'project.dataset.events' has field 'user_id' as INT64 type, not STRING - attempting to filter with email addresses like 'user@example.com' will cause type mismatch errors, use numeric IDs only"
- "Slack search requires both 'from:@username' AND 'in:#channel' syntax for filtering by user in specific channel - using only 'from:' searches all channels and may return irrelevant results"
- "Table 'security_logs' in project 'prod-security' is partitioned by 'event_date' column - queries without WHERE clause on event_date will scan entire table (10TB+) and fail with cost limit errors"
- "GitHub API /repos/{owner}/{repo}/pulls endpoint has rate limit of 100 requests/min per token - exceeding returns 429 status with Retry-After header, implement exponential backoff starting at 60s"

Examples of **BAD** claims (missing context, not self-contained):
- "The task was successful" (not actionable, no domain knowledge)
- "user_id is INT64" (which table? which project? why does it matter?)
- "Need to use the correct filter" (what filter? where? for what purpose?)
- "Login events require action='login'" (which table? which system? what happens if you don't?)
- "API has rate limit" (which API? what limit? how to handle?)
- "Used BigQuery" (not domain knowledge, just a tool reference)

**Key Question**: If you showed this claim to another agent who has NEVER worked on this task, would they understand:
1. WHAT the claim is about (which system/table/API/field)?
2. WHAT was discovered (the specific fact or rule)?
3. WHY it matters (what problem does it solve or prevent)?

If the answer to ANY of these is "NO", the claim needs more context.

### 2. Helpful Memories

Identify which provided memories (by ID) were **helpful** during execution:
- Provided accurate, relevant information
- Guided the approach or solution
- Helped avoid mistakes or errors
- Contributed to successful completion

### 3. Harmful Memories

Identify which provided memories (by ID) were **harmful** during execution:
- Contained incorrect or outdated information
- Misled the execution or caused errors
- Were irrelevant or distracting
- Needed to be corrected or ignored

## Output Format

Respond with a JSON object matching this schema:

```json
{{ .JSONSchema }}
```

**Important Guidelines:**
- Only list memory IDs that were actually referenced in the execution
- NewClaims should contain only novel insights, not repetition of existing memories
- If no new insights were gained, NewClaims can be an empty array
- If all memories were neutral (neither helpful nor harmful), leave those arrays empty
- Be specific and honest in your evaluation
