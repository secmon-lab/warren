# Sub-Agent Development Rules

## Package Structure

Each sub-agent lives under `pkg/agents/<name>/` with the following files:

```
pkg/agents/<name>/
├── factory.go      # AgentFactory implementation, CLI flags, Configure()
├── agent.go        # Agent core, SubAgent creation, middleware
├── tool.go         # Internal tools (API calls)
├── auth.go         # Authentication logic (if needed)
├── prompt.go       # System prompt builder
├── extract.go      # Record extraction from session history
├── export_test.go  # Test-only exports
├── prompt/
│   ├── system.md   # System prompt template
│   └── extract.md  # Extraction prompt template
└── README.md       # Configuration guide
```

## Factory Pattern

- Implement `agents.AgentFactory` interface
- CLI flags: `--agent-<name>-<param>`, env vars: `WARREN_AGENT_<NAME>_<PARAM>`
- `Configure()` returns `(nil, nil)` when required credentials are not set (skip registration)
- Log configuration at INFO level with credentials summary (never log secrets, only their length)

## Middleware Pattern (agent.go)

All sub-agents follow this middleware flow:
1. **Pre-execution**: Memory search via `memoryService.SearchAndSelectMemories`
2. **Execution**: Internal agent processes the query
3. **Post-execution**: Memory save, record extraction, internal field cleanup

## msg.Trace Usage

Use `msg.Trace` extensively to show real-time progress to the user in Slack threads. This is critical for observability.

### Required Trace Points in tool.go

- **Before each API call**: What operation is starting (with key parameters like filter/query)
  - `msg.Trace(ctx, "🔍 Searching alerts (filter: \`%s\`)", filter)`
- **After successful API call**: Confirmation with result summary
  - `msg.Trace(ctx, "✅ Alert search completed")`
- **On API failure**: Error details
  - `msg.Trace(ctx, "❌ Alert search failed: %v", err)`
- **On retry**: What's being retried and why
  - `msg.Trace(ctx, "🔄 Received 401, refreshing token and retrying...")`
- **For async operations**: Progress updates (e.g., polling status)
  - `msg.Trace(ctx, "⏳ Event search job created (job_id: \`%s\`), polling...")`
  - `msg.Trace(ctx, "⚠️ Event search reached poll limit, returning %d partial results")`

### Required Trace Points in agent.go

- **On request received**: Show the incoming query
  - `msg.Trace(ctx, "🦅 *[Falcon Agent]* Request: \`%s\`", request)`

### Emoji Convention

| Emoji | Usage |
|-------|-------|
| 🔍 | Search/query operations |
| 📋 | Retrieving details by ID |
| 📊 | Metrics/scores retrieval |
| ✅ | Successful completion |
| ❌ | Failure/error |
| 🔄 | Retry/refresh |
| ⏳ | Async waiting/polling |
| ⚠️ | Warning/partial results |

## Error Handling

### Authentication Errors
- Log token refresh failures at WARN level with status code and response body
- Accept both HTTP 200 and 201 for OAuth token responses (CrowdStrike returns 201)
- Log API call failures at WARN level with status, path, and response body

### Sub-Agent Error Detection
- When record extraction yields 0 records AND the response text contains error indicators (auth failures, connectivity issues), return an error instead of `"status": "success"` with empty records
- This prevents silent failures from being treated as "no data found"

## HTTP Client Best Practices

- URL-encode query parameter values (`url.QueryEscape`)
- Use `safe.Close(ctx, resp.Body)` for response body cleanup
- Implement 401 retry: clear cached token, retry once
- Log all API request paths at DEBUG level for traceability

## Testing

- Use `export_test.go` for exposing test-only wrappers
- Use `httptest.NewServer` for mocking external APIs
- Include a spec count test to catch accidental tool additions/removals
- Test error cases (auth failure, API errors, 401 retry)

## Registration

Add the factory to `pkg/agents/agents.go`:
```go
var All = []AgentFactory{
    &slack.Factory{},
    &bigquery.Factory{},
    &falcon.Factory{}, // Add new agent here
}
```
