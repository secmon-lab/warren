# MCP (Model Context Protocol) Integration

Warren supports extending AI agent capabilities by connecting to external MCP servers. This allows you to add custom tools — such as internal threat intelligence APIs, proprietary scanners, or organization-specific data sources — without modifying Warren's source code.

## Overview

MCP tools are available in both **Slack chat** (`@warren` in threads) and **CLI chat** (`warren chat`). When configured, MCP tools appear alongside built-in tools (VirusTotal, Shodan, etc.) and the AI agent can invoke them during investigation.

```
Warren Agent
  ├── Built-in tools (VirusTotal, OTX, Shodan, ...)
  ├── Sub-agents (BigQuery, Falcon, Slack)
  └── MCP tools (your custom servers)
        ├── Remote: SSE / HTTP
        └── Local: STDIO (bundled in Docker image)
```

## Configuration

Create a YAML config file and pass it via `--mcp-config` flag or `WARREN_MCP_CONFIG` environment variable:

```bash
# CLI
warren chat --ticket-id <TICKET_ID> --mcp-config mcp-config.yaml

# Environment variable (for serve mode / Cloud Run)
export WARREN_MCP_CONFIG=/etc/warren/mcp-config.yaml
```

## Remote MCP Servers

Connect to MCP servers hosted externally over the network.

### Server Types

| Type | Transport | Use Case |
|------|-----------|----------|
| `sse` | Server-Sent Events | Streaming responses, long-running operations |
| `http` | Streamable HTTP | Simple request-response |

### Basic Configuration

```yaml
servers:
  - name: "threat-intel"
    type: "sse"
    url: "https://intel-api.example.com/mcp/sse"
    headers:
      Authorization: "Bearer YOUR_API_TOKEN"

  - name: "asset-db"
    type: "http"
    url: "https://assets.example.com/mcp"
    headers:
      X-API-Key: "YOUR_API_KEY"
```

### Credential Helper

For MCP servers requiring dynamic authentication (e.g., Google IAP, short-lived tokens), use a credential helper instead of static headers:

```yaml
servers:
  - name: "iap-protected-server"
    type: "http"
    url: "https://iap-protected.example.com/mcp"
    helper:
      command: "./scripts/iap-token.sh"
      args: ["https://iap-protected.example.com"]
```

The helper command must output JSON to stdout:

```json
{
  "headers": {"Authorization": "Bearer eyJhbGciOi..."},
  "expires_at": "2026-03-01T12:00:00Z"
}
```

- `headers`: HTTP headers to inject into requests (overrides static `headers` on conflict)
- `expires_at` (optional): RFC 3339 timestamp; Warren caches the result until this time. Without it, the helper runs on every request

Credential helpers only apply to remote servers (`sse` and `http`), not `stdio`.

### Disabling a Server

Temporarily disable a server without removing its configuration:

```yaml
servers:
  - name: "maintenance-server"
    type: "http"
    url: "https://example.com/mcp"
    disabled: true
```

## Local MCP Servers (STDIO)

Run MCP servers as local processes. Warren communicates with them via stdin/stdout.

### Basic Configuration

```yaml
local:
  - name: "filesystem"
    command: "npx"
    args: ["@modelcontextprotocol/server-filesystem", "/tmp"]
    env:
      NODE_ENV: "production"
```

### Full Example

Combining remote and local servers:

```yaml
servers:
  - name: "threat-intel"
    type: "sse"
    url: "https://intel-api.example.com/mcp/sse"
    headers:
      Authorization: "Bearer ${INTEL_API_KEY}"

local:
  - name: "custom-scanner"
    command: "/usr/local/bin/my-scanner"
    args: ["--mcp"]
```

## Building Custom MCP Tools into Docker

For production deployments on Cloud Run or similar environments, you can build custom MCP tool binaries directly into the Warren Docker image. This eliminates external dependencies and ensures your tools are always available.

### Step 1: Implement an MCP Server

Create a standalone MCP server. Any language works as long as the binary speaks the MCP STDIO protocol. Here is a Go example using `github.com/modelcontextprotocol/go-sdk`:

```go
// cmd/my-tool/main.go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "github.com/modelcontextprotocol/go-sdk/server"
)

func main() {
    s := server.NewMCPServer(mcp.Implementation{
        Name:    "my-tool",
        Version: "1.0.0",
    })

    // Define and register a tool
    s.AddTool(mcp.Tool{
        Name:        "lookup_asset",
        Description: "Look up an asset by hostname or IP",
        InputSchema: mcp.ToolInputSchema{
            Type: "object",
            Properties: map[string]mcp.Property{
                "query": {Type: "string", Description: "Hostname or IP address"},
            },
            Required: []string{"query"},
        },
    }, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        query := req.Params.Arguments["query"].(string)

        // Your lookup logic here
        result := map[string]string{
            "hostname": query,
            "owner":    "platform-team",
            "env":      "production",
        }

        data, _ := json.Marshal(result)
        return &mcp.CallToolResult{
            Content: []mcp.Content{mcp.TextContent{Text: fmt.Sprintf("Asset info: %s", string(data))}},
        }, nil
    })

    _ = server.RunStdioServer(s)
}
```

### Step 2: Create a Dockerfile

Use the official Warren Docker image from GHCR as the base, and add your custom tool binary:

```dockerfile
# Build custom MCP tool
FROM golang:1.24-alpine AS build-tool
WORKDIR /tool
COPY my-tool/go.mod my-tool/go.sum ./
RUN go mod download
COPY my-tool/ ./
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /my-tool ./cmd/my-tool

# Use Warren's official image as base
FROM ghcr.io/secmon-lab/warren:latest

# Copy custom tool binary
COPY --from=build-tool /my-tool /usr/local/bin/my-tool
```

You can pin a specific version instead of `latest`:

```dockerfile
FROM ghcr.io/secmon-lab/warren:v0.7.0
```

### Step 3: Configure MCP to Use the Bundled Tool

Create `mcp-config.yaml`:

```yaml
local:
  - name: "my-tool"
    command: "/usr/local/bin/my-tool"
```

### Step 4: Deploy

Embed the config file and environment variable directly in the Dockerfile:

```dockerfile
COPY mcp-config.yaml /etc/warren/mcp-config.yaml
ENV WARREN_MCP_CONFIG=/etc/warren/mcp-config.yaml
```

The complete Dockerfile looks like this:

```dockerfile
# Build custom MCP tool
FROM golang:1.24-alpine AS build-tool
WORKDIR /tool
COPY my-tool/go.mod my-tool/go.sum ./
RUN go mod download
COPY my-tool/ ./
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /my-tool ./cmd/my-tool

# Use Warren's official image as base
FROM ghcr.io/secmon-lab/warren:latest
COPY --from=build-tool /my-tool /usr/local/bin/my-tool
COPY mcp-config.yaml /etc/warren/mcp-config.yaml
ENV WARREN_MCP_CONFIG=/etc/warren/mcp-config.yaml
```

## Troubleshooting

- **"failed to create MCP server client"**: Check URL accessibility and authentication headers
- **"credential helper command failed"**: Verify the helper script is executable and outputs valid JSON
- **Tools not appearing**: Confirm the config file path is correct and the server is not `disabled: true`
- **STDIO server not starting**: Ensure the binary exists at the specified path and has execute permissions
