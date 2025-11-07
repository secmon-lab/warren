You are analyzing a BigQuery agent task execution to help future LLM agents perform similar tasks better.
Your analysis will be used as reference for future executions, so be SPECIFIC and ACTIONABLE.

**Task Information:**
- Original Query: {{.Query}}
- Duration: {{.Duration}}
- Status: {{.Status}}

**Conversation History:**
{{.History}}

**Execution Details:**
{{.ExecutionSummary}}
{{- if .Error}}
- Error: {{.Error}}
{{- end}}

**Instructions:**

Focus on DOMAIN KNOWLEDGE that helps future agents construct queries efficiently.
Capture learnings about data structure, field semantics, and query patterns.

1. **Successes (Keep)**:
   - List 2-4 DOMAIN KNOWLEDGE discoveries that worked
   - Focus on: field semantics, data formats, search patterns, which fields to use for what purpose
   - Each item: ~250 chars, max 500 chars
   - Empty array if execution failed
   - Example: "To find authentication failures, use error_code='AUTH_FAILED' in security_events.event_details (not event_type). User identifier is in user.email field (STRING), not user.id which is internal numeric ID"

2. **Problems**:
   - List 1-3 DOMAIN KNOWLEDGE mistakes or unexpected data behaviors
   - Focus on: wrong field assumptions, unexpected data formats, semantic misunderstandings
   - Each item: ~250 chars, max 500 chars
   - Empty array if none
   - Example: "Assumed 'timestamp' field for time filtering but actual field is 'event_time' (TIMESTAMP type). Assumed user_id was STRING email but it's INT64 internal ID"

3. **Improvements (Try)**:
   - List 2-4 SPECIFIC DOMAIN KNOWLEDGE and EFFICIENCY INSIGHTS to apply next time
   - Focus on: which fields to check, expected data formats, query patterns for specific searches
   - Include EFFICIENCY LEARNINGS: how to reduce trial-and-error, what to check first to avoid wasted queries
   - Each item: ~250 chars, max 500 chars
   - Empty array if none needed
   - Example: "For user activity searches: always use user.email (STRING) not user_id (INT64). For error searches: check error_code field values first (e.g., 'AUTH_FAILED', 'PERMISSION_DENIED')"
   - Example (efficiency): "Before querying by IP address, first check table schema to confirm field name (could be 'client_ip', 'source_ip', or 'remote_addr'). This avoids 2-3 failed query attempts with wrong field names"

**Critical Rules:**
- Focus on DATA SEMANTICS: which fields mean what, expected values, data formats
- Focus on SEARCH PATTERNS: how to find specific types of events/data
- Focus on EFFICIENCY INSIGHTS: what steps could have been skipped, what to verify upfront, how to reduce query iterations
- Include CONCRETE field names, expected values, data types
- Describe MISCONCEPTIONS: what you thought vs what it actually was
- Capture PROCESS IMPROVEMENTS: if you tried 5 queries to get the right field name, note "check schema first" to reduce future attempts
- Do NOT describe generic query optimization (partition filtering, LIMIT, etc.) - focus on THIS dataset's specifics
- Each string should be standalone domain knowledge or actionable efficiency tip
- Keep length: target ~250 chars, max 500 chars per item

**Output Format (JSON only, no markdown code blocks):**
{
  "successes": ["specific domain knowledge 1", "specific domain knowledge 2"],
  "problems": ["specific problem 1"],
  "improvements": ["specific improvement 1", "specific improvement 2"]
}
