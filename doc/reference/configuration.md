# Configuration Reference

This is the single source of truth for all Warren configuration options.

## Configuration Hierarchy

Warren uses the following precedence (highest to lowest):

1. **Command-line flags** — Override all other settings
2. **Environment variables** — Override defaults
3. **Default values** — Built-in defaults

## Core Settings

### Server

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_ADDR` | `--addr` | `127.0.0.1:8080` | HTTP server address |
| `WARREN_LOG_LEVEL` | `--log-level` | `info` | Log level (debug, info, warn, error) |
| `WARREN_LOG_FORMAT` | `--log-format` | `console` | Log format (json, console) |
| `WARREN_LOG_QUIET` | `--log-quiet` | `false` | Suppress non-error logs |
| `WARREN_LOG_STACKTRACE` | `--log-stacktrace` | `true` | Show stacktrace in error logs |
| `WARREN_LOG_OUTPUT` | `--log-output` | `stderr` | Log output destination |
| `WARREN_ASYNC_ALERT_HOOK` | `--async-alert-hook` | - | Async webhook processing (raw, pubsub, sns, all) |

### LLM

LLM provider/model configuration moved to a dedicated TOML file. The legacy
`--gemini-*` / `--claude-*` flags and `--disable-llm` were removed; LLM is
required.

| Environment Variable | CLI Flag       | Default | Description                                         |
| -------------------- | -------------- | ------- | --------------------------------------------------- |
| `WARREN_LLM_CONFIG`  | `--llm-config` | -       | **Required** — path to the LLM TOML config file.    |

See [`llm.md`](./llm.md) for the full TOML schema and
[`llm.toml.example`](./llm.toml.example) for a working sample. The file
declares one `[agent]` block (planner + task allow-list), one or more `[[llm]]`
entries (Claude on Vertex / API key, Gemini on Vertex), and a single
`[embedding]` block (Gemini Vertex only). API keys can be injected via
`{{ .Env.VAR }}` template substitution so the file itself stays free of
secrets.

### Firestore

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_FIRESTORE_PROJECT_ID` | `--firestore-project-id` | - | **Required** — Firestore project ID |
| `WARREN_FIRESTORE_DATABASE_ID` | `--firestore-database-id` | `(default)` | Firestore database ID |

### Cloud Storage

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_STORAGE_PROJECT_ID` | `--storage-project-id` | - | Cloud Storage project ID |
| `WARREN_STORAGE_BUCKET` | `--storage-bucket` | - | Cloud Storage bucket name |

### Slack

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_SLACK_OAUTH_TOKEN` | `--slack-oauth-token` | - | Bot OAuth token (`xoxb-...`) |
| `WARREN_SLACK_SIGNING_SECRET` | `--slack-signing-secret` | - | App signing secret |
| `WARREN_SLACK_CLIENT_ID` | `--slack-client-id` | - | OAuth client ID |
| `WARREN_SLACK_CLIENT_SECRET` | `--slack-client-secret` | - | OAuth client secret |
| `WARREN_SLACK_CHANNEL_NAME` | `--slack-channel-name` | - | Default channel (without #) |
| `WARREN_FRONTEND_URL` | `--frontend-url` | - | Web UI URL for Slack links |

### Authentication

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_NO_AUTHENTICATION` | `--no-authentication` | `false` | Disable authentication |
| `WARREN_NO_AUTHORIZATION` | `--no-authorization` | `false` | Disable authorization |
| `WARREN_SERVICE_ACCOUNT` | - | - | Expected service account email (for policy `input.env`) |

### Policy

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_POLICY` | `--policy` | - | Path to policy files/directories |
| `WARREN_WATCH` | `--watch` | `false` | Watch policy files for changes |

### Chat / Agent

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_LANG` | `--lang` | `en` | Response language (en, ja, etc.) |
| `WARREN_CHAT_STRATEGY` | `--chat-strategy` | `aster` | Chat execution strategy (default: `aster`) |
| `WARREN_MCP_CONFIG` | `--mcp-config` | - | Path to MCP config YAML |

## Threat Intelligence Tools

Tools are automatically enabled when their API key is configured. Missing keys are silently skipped.

### VirusTotal

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_VT_API_KEY` | `--vt-api-key` | - | VirusTotal API key |
| `WARREN_VT_BASE_URL` | `--vt-base-url` | `https://www.virustotal.com/api/v3` | API base URL |

### OTX (AlienVault)

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_OTX_API_KEY` | `--otx-api-key` | - | OTX API key |
| `WARREN_OTX_BASE_URL` | `--otx-base-url` | `https://otx.alienvault.com/api/v1` | API base URL |

### URLScan.io

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_URLSCAN_API_KEY` | `--urlscan-api-key` | - | URLScan.io API key |
| `WARREN_URLSCAN_BASE_URL` | `--urlscan-base-url` | `https://urlscan.io/api/v1` | API base URL |
| `WARREN_URLSCAN_BACKOFF` | `--urlscan-backoff` | `3s` | Polling backoff interval |
| `WARREN_URLSCAN_TIMEOUT` | `--urlscan-timeout` | `30s` | Scan timeout |

### Shodan

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_SHODAN_API_KEY` | `--shodan-api-key` | - | Shodan API key |
| `WARREN_SHODAN_BASE_URL` | `--shodan-base-url` | `https://api.shodan.io` | API base URL |

### AbuseIPDB

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_IPDB_API_KEY` | `--ipdb-api-key` | - | AbuseIPDB API key |
| `WARREN_IPDB_BASE_URL` | `--ipdb-base-url` | `https://api.abuseipdb.com/api/v2` | API base URL |

### abuse.ch (MalwareBazaar)

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_ABUSECH_AUTH_KEY` | `--abusech-api-key` | - | MalwareBazaar API key |
| `WARREN_ABUSECH_BASE_URL` | `--abusech-base-url` | `https://mb-api.abuse.ch/api/v1` | API base URL |

## Code & Device Tools

### GitHub App

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_GITHUB_APP_ID` | `--github-app-id` | - | GitHub App ID |
| `WARREN_GITHUB_APP_INSTALLATION_ID` | `--github-app-installation-id` | - | App installation ID |
| `WARREN_GITHUB_APP_PRIVATE_KEY` | `--github-app-private-key` | - | App private key (PEM) |
| `WARREN_GITHUB_APP_CONFIG` | `--github-app-config` | - | YAML config file path(s) |

### Microsoft Intune

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_INTUNE_TENANT_ID` | `--intune-tenant-id` | - | Azure AD tenant ID |
| `WARREN_INTUNE_CLIENT_ID` | `--intune-client-id` | - | Azure AD client ID |
| `WARREN_INTUNE_CLIENT_SECRET` | `--intune-client-secret` | - | Azure AD client secret |
| `WARREN_INTUNE_BASE_URL` | `--intune-base-url` | `https://graph.microsoft.com/v1.0` | Graph API base URL |

### Slack Message Search (Tool)

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_SLACK_TOOL_USER_TOKEN` | `--slack-tool-user-token` | - | Slack User token (`xoxp-...`) with `search:read` scope |

## Sub-Agent Configuration

### BigQuery Agent

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_AGENT_BIGQUERY_CONFIG` | `--agent-bigquery-config` | - | YAML config file path (required) |
| `WARREN_AGENT_BIGQUERY_PROJECT_ID` | `--agent-bigquery-project-id` | - | GCP project ID |
| `WARREN_AGENT_BIGQUERY_SCAN_SIZE_LIMIT` | `--agent-bigquery-scan-size-limit` | - | Override scan limit |
| `WARREN_AGENT_BIGQUERY_RUNBOOK_DIR` | `--agent-bigquery-runbook-dir` | - | SQL runbook paths (multiple allowed) |
| `WARREN_AGENT_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT` | `--agent-bigquery-impersonate-service-account` | - | Service account to impersonate |

### CrowdStrike Falcon Agent

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_AGENT_FALCON_CLIENT_ID` | `--agent-falcon-client-id` | - | CrowdStrike API client ID |
| `WARREN_AGENT_FALCON_CLIENT_SECRET` | `--agent-falcon-client-secret` | - | CrowdStrike API client secret |
| `WARREN_AGENT_FALCON_BASE_URL` | `--agent-falcon-base-url` | `https://api.crowdstrike.com` | API base URL |

### Slack Search Agent

| Environment Variable | CLI Flag | Default | Description |
|---|---|---|---|
| `WARREN_AGENT_SLACK_USER_TOKEN` | `--agent-slack-user-token` | - | Slack User token (`xoxp-...`) with `search:read` scope |

## Command-Specific Options

### `warren serve`

| CLI Flag | Default | Description |
|----------|---------|-------------|
| `--enable-graphql` | `false` | Enable GraphQL API endpoint |
| `--enable-graphiql` | `false` | Enable GraphiQL playground |

### `warren chat`

| CLI Flag | Default | Description |
|----------|---------|-------------|
| `--ticket-id`, `-t` | - | **Required** — Ticket ID to analyze |
| `--query`, `-q` | - | Single query (non-interactive mode) |
| `--list` | `false` | List previous chat sessions |

### `warren refine`

| CLI Flag | Default | Description |
|----------|---------|-------------|
| Uses core LLM, Firestore, and Slack settings. | | See [Refine documentation](../concepts.md#refine). |

## Configuration Files

### MCP Configuration

```yaml
servers:
  - name: "custom-intel"
    type: "sse"
    url: "https://intel-api.example.com/mcp"
    headers:
      Authorization: "Bearer ${MCP_API_KEY}"

local:
  - name: "local-scanner"
    command: "/usr/local/bin/scanner"
    args: ["--mcp-mode"]
```

### BigQuery Agent Configuration

```yaml
tables:
  - project_id: my-project
    dataset_id: security_logs
    table_id: events
    description: Security event logs

scan_size_limit: "10GB"
query_timeout: 5m
```

See [BigQuery Agent README](../../pkg/agents/bigquery/README.md) for detailed configuration.

## Asynchronous Alert Processing

Enable async processing for webhook types that require fast HTTP responses:

```bash
warren serve --async-alert-hook all
# Or selectively: --async-alert-hook raw --async-alert-hook pubsub
```

When enabled:
- Returns HTTP 200 immediately after request validation
- Processes alerts in the background
- Processing errors are logged (no client feedback)

## Production Best Practices

1. Store sensitive values in Secret Manager:
   ```bash
   gcloud run services update warren \
     --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-token:latest"
   ```
2. Use environment variables for deployment-specific settings
3. Use configuration files for complex structures (MCP, BigQuery)
4. Enable async webhooks for high-volume sources
