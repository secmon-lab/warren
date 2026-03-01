# MCP (Model Context Protocol) Configuration

Warren supports connecting to external MCP servers to extend the AI agent's available tools. This document covers MCP configuration, including the credential helper feature for dynamic authentication.

## Overview

Warren can connect to three types of MCP servers:

| Type | Transport | Description |
|------|-----------|-------------|
| `sse` | Server-Sent Events | Remote server using SSE transport |
| `http` | Streamable HTTP | Remote server using HTTP transport |
| `stdio` | Standard I/O | Local process communicating via stdin/stdout |

## Configuration File

Specify the MCP configuration file via CLI flag or environment variable:

```bash
warren serve --mcp-config=mcp-config.yaml
# or
export WARREN_MCP_CONFIG=mcp-config.yaml
```

### Basic Structure

```yaml
# Remote MCP servers
servers:
  - name: "web-search"
    type: "sse"
    url: "https://api.example.com/mcp/sse"
    headers:
      Authorization: "Bearer YOUR_API_TOKEN"

  - name: "database"
    type: "http"
    url: "https://db.example.com/mcp"
    headers:
      Authorization: "Bearer YOUR_DB_TOKEN"

# Local MCP servers (stdio-based)
local:
  - name: "filesystem"
    command: "npx"
    args: ["@modelcontextprotocol/server-filesystem", "/tmp"]
    env:
      NODE_ENV: "production"
```

### Disabling a Server

Add `disabled: true` to temporarily disable a server without removing its configuration:

```yaml
servers:
  - name: "maintenance-server"
    type: "http"
    url: "https://example.com/mcp"
    disabled: true
```

## Credential Helper

For MCP servers that require dynamic authentication (e.g., Google IAP, short-lived tokens), you can configure a **credential helper** — an external command that generates HTTP headers at runtime.

### How It Works

1. Before each MCP request (or when the cached credential expires), Warren executes the helper command
2. The helper command outputs a JSON object to stdout containing headers and an optional expiry time
3. Warren injects the headers into the MCP HTTP request

### Configuration

Add a `helper` field to a server configuration:

```yaml
servers:
  - name: "iap-protected-server"
    type: "http"
    url: "https://iap-protected.example.com/mcp"
    headers:
      Content-Type: "application/json"
    helper:
      command: "./scripts/iap-token.sh"
      args: ["https://iap-protected.example.com"]
      env:
        GOOGLE_APPLICATION_CREDENTIALS: "/path/to/sa-key.json"
```

### Helper Output Format

The helper command must output valid JSON to stdout:

```json
{
  "headers": {
    "Authorization": "Bearer eyJhbGciOi...",
    "X-Custom-Header": "value"
  },
  "expires_at": "2026-03-01T12:00:00Z"
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `headers` | Yes | Key-value map of HTTP headers to inject |
| `expires_at` | No | RFC 3339 timestamp. If provided, the result is cached until this time |

### Header Merge Priority

When both static `headers` and `helper` produce the same header key, the **helper output takes precedence**:

1. Static `headers` are applied first (base)
2. Helper-generated headers override any matching keys

### Caching

- If `expires_at` is provided, the helper result is cached in-memory until that time
- When the cache expires, the helper command is re-executed on the next request
- Without `expires_at`, the helper is executed on every request
- Cache is cleared on process restart

### Error Handling

| Scenario | Behavior |
|----------|----------|
| Command not found | MCP client creation fails with error |
| Non-zero exit code | MCP client creation fails with error (stderr is logged) |
| Invalid JSON output | MCP client creation fails with error |
| Timeout (default 30s) | MCP client creation fails with error |
| Cache refresh failure | Request fails with error |

The helper's stderr output is always logged for debugging purposes.

### Applicability

Credential helpers are only applicable to remote MCP servers (`sse` and `http` types). Local `stdio` servers do not use HTTP and therefore do not need credential helpers.

## Use Case: Google IAP Authentication

Here is an example helper script for Google Identity-Aware Proxy:

```bash
#!/bin/bash
# scripts/iap-token.sh
# Usage: ./iap-token.sh <target-audience>

if [ -z "$1" ]; then
  echo "Usage: $0 <target-audience>" >&2
  exit 1
fi

TARGET_AUDIENCE="$1"
TOKEN=$(gcloud auth print-identity-token --audiences="$TARGET_AUDIENCE" 2>/dev/null)

if [ $? -ne 0 ]; then
  echo "Failed to get identity token" >&2
  exit 1
fi

# Token is valid for ~1 hour, cache for 50 minutes
EXPIRES=$(date -u -v+50M '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -d '+50 minutes' '+%Y-%m-%dT%H:%M:%SZ')

cat <<EOF
{
  "headers": {
    "Authorization": "Bearer ${TOKEN}"
  },
  "expires_at": "${EXPIRES}"
}
EOF
```

## Troubleshooting

### Helper command fails

1. Verify the command is executable: `chmod +x ./scripts/helper.sh`
2. Test the command manually: `./scripts/helper.sh`
3. Check stderr output in Warren logs (DEBUG level)
4. Ensure the output is valid JSON with a `headers` field

### Headers not being applied

1. Confirm the server `type` is `sse` or `http` (not `stdio`)
2. Check that the helper output contains the expected header keys
3. If using caching, verify `expires_at` has not passed

### Timeout issues

The default helper timeout is 30 seconds. If your credential helper needs more time (e.g., for network calls), consider optimizing the helper script or implementing local caching within the script itself.
