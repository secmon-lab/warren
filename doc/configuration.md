# Warren Configuration Guide

This document provides a comprehensive reference for all configuration options in Warren, including environment variables, command-line flags, and configuration files.

## Configuration Hierarchy

Warren uses the following precedence for configuration (highest to lowest):

1. **Command-line flags** - Override all other settings
2. **Environment variables** - Override configuration files
3. **Configuration files** - Base configuration
4. **Default values** - Built-in defaults

## Core Configuration

### Server Settings

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_ADDR` | `--addr` | `127.0.0.1:8080` | HTTP server address |
| `WARREN_LOG_LEVEL` | `--log-level` | `info` | Log level (debug, info, warn, error) |
| `WARREN_LOG_FORMAT` | `--log-format` | `console` | Log format (json, console) |
| `WARREN_LOG_QUIET` | `--log-quiet` | `false` | Quiet mode (suppress non-error logs) |
| `WARREN_LOG_STACKTRACE` | `--log-stacktrace` | `true` | Show stacktrace in error logs |
| `WARREN_LOG_OUTPUT` | `--log-output` | `stderr` | Log output destination |

### Google Cloud Settings

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_GEMINI_PROJECT_ID` | `--gemini-project-id` | - | **Required** - Google Cloud project ID for Vertex AI |
| `WARREN_GEMINI_LOCATION` | `--gemini-location` | `us-central1` | Vertex AI location |
| `WARREN_GEMINI_MODEL` | `--gemini-model` | `gemini-1.5-flash` | Gemini model name |
| `WARREN_FIRESTORE_PROJECT_ID` | `--firestore-project-id` | - | **Required** - Firestore project ID |
| `WARREN_FIRESTORE_DATABASE_ID` | `--firestore-database-id` | `(default)` | Firestore database ID |
| `WARREN_STORAGE_PROJECT_ID` | `--storage-project-id` | - | Cloud Storage project ID |
| `WARREN_STORAGE_BUCKET` | `--storage-bucket` | - | Cloud Storage bucket name |

### Slack Configuration

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_SLACK_OAUTH_TOKEN` | `--slack-oauth-token` | - | Slack bot OAuth token (xoxb-...) **Required for Slack features** |
| `WARREN_SLACK_SIGNING_SECRET` | `--slack-signing-secret` | - | Slack app signing secret |
| `WARREN_SLACK_CLIENT_ID` | `--slack-client-id` | - | Slack OAuth client ID |
| `WARREN_SLACK_CLIENT_SECRET` | `--slack-client-secret` | - | Slack OAuth client secret |
| `WARREN_SLACK_CHANNEL_NAME` | `--slack-channel-name` | - | Default Slack channel (without #) |
| `WARREN_FRONTEND_URL` | `--frontend-url` | - | Web UI URL for Slack links |

### Authentication

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_SERVICE_ACCOUNT` | - | - | Expected service account email (used in policy evaluation via `input.env.WARREN_SERVICE_ACCOUNT`) |

## External Tool Integration

### Threat Intelligence Tools

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_VT_API_KEY` | `--vt-api-key` | - | VirusTotal API key |
| `WARREN_OTX_API_KEY` | `--otx-api-key` | - | AlienVault OTX API key |
| `WARREN_URLSCAN_API_KEY` | `--urlscan-api-key` | - | URLScan.io API key |
| `WARREN_SHODAN_API_KEY` | `--shodan-api-key` | - | Shodan API key |
| `WARREN_ABUSEIPDB_API_KEY` | `--abuseipdb-api-key` | - | AbuseIPDB API key |

### BigQuery Configuration

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_BIGQUERY_PROJECT_ID` | `--bigquery-project-id` | - | BigQuery project ID |
| `WARREN_BIGQUERY_DATASET_ID` | `--bigquery-dataset-id` | - | BigQuery dataset ID |
| `WARREN_BIGQUERY_CONFIG` | `--bigquery-config` | - | Path to BigQuery config YAML (can specify multiple) |
| `WARREN_BIGQUERY_RUNBOOK_PATH` | `--bigquery-runbook-path` | - | Path to SQL runbook directory (can specify multiple) |
| `WARREN_BIGQUERY_CREDENTIALS` | `--bigquery-credentials` | - | Path to Google Cloud credentials JSON |
| `WARREN_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT` | `--bigquery-impersonate-service-account` | - | Service account email for impersonation |
| `WARREN_BIGQUERY_STORAGE_BUCKET` | `--bigquery-storage-bucket` | - | GCS bucket for query results |
| `WARREN_BIGQUERY_STORAGE_PREFIX` | `--bigquery-storage-prefix` | - | GCS object path prefix |
| `WARREN_BIGQUERY_TIMEOUT` | `--bigquery-timeout` | `5m` | Query execution timeout |
| `WARREN_BIGQUERY_SCAN_LIMIT` | `--bigquery-scan-limit` | `10GB` | Query scan limit |

## Policy Configuration

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_POLICY` | `--policy` | - | Path to policy files/directories |
| `WARREN_WATCH` | `--watch` | `false` | Watch policy files for changes |

## Chat/Agent Configuration

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_LANG` | `--lang` | `en` | Response language (en, ja, etc.) |
| `WARREN_MCP_CONFIG` | `--mcp-config` | - | Path to MCP config YAML |

## Command-Specific Options

### `warren serve`

```bash
warren serve \
  --addr=127.0.0.1:8080 \
  --log-level=debug \
  --policy=/path/to/policies \
  --slack-channel-name=security-alerts \
  --gemini-project-id=my-project \
  --firestore-project-id=my-project \
  --enable-graphql \
  --enable-graphiql
```

Additional serve options:
- `--enable-graphql` - Enable GraphQL API endpoint
- `--enable-graphiql` - Enable GraphiQL playground UI

### `warren chat`

```bash
warren chat \
  --ticket-id=ticket-12345 \
  --lang=en \
  --mcp-config=mcp.yaml
```

Chat-specific options:
- `--ticket-id` - Required ticket ID to analyze
- `--query` - Single query for non-interactive mode
- `--list` - List previous chat sessions

### `warren test`

```bash
warren test \
  --policy=/path/to/policies \
  --filter=guardduty
```

## Configuration Files

### MCP Configuration (YAML)

```yaml
# mcp-config.yaml
servers:
  - name: "custom-tools"
    type: "sse"
    url: "https://tools.example.com/mcp"
    headers:
      Authorization: "Bearer ${MCP_API_KEY}"
    
  - name: "local-scanner"
    type: "stdio"
    command: "/usr/local/bin/scanner"
    args: ["--mcp-mode"]
    env:
      SCANNER_CONFIG: "/etc/scanner/config.yaml"
```

### BigQuery Configuration (YAML)

```yaml
# bigquery-config.yaml
dataset_id: security_logs
table_id: events
description: "Security event logs"
columns:
  - name: timestamp
    type: TIMESTAMP
    description: "Event timestamp"
  - name: source_ip
    type: STRING
    description: "Source IP address"
```

## Environment Variable Expansion

Warren supports environment variable expansion in configuration values:

- `${VAR_NAME}` - Expands to the value of VAR_NAME
- `${VAR_NAME:-default}` - Uses default if VAR_NAME is not set

Example:
```bash
export WARREN_FRONTEND_URL="https://${SERVICE_NAME}-${PROJECT_ID}.a.run.app"
```

## Development Mode

For local development, you must provide Google Cloud services and Slack credentials:

```bash
# Minimal required settings
export WARREN_LOG_LEVEL=debug
export WARREN_LOG_FORMAT=console
export WARREN_FIRESTORE_PROJECT_ID=your-project
export WARREN_GEMINI_PROJECT_ID=your-project
export WARREN_SLACK_OAUTH_TOKEN=xoxb-your-token
export WARREN_SLACK_SIGNING_SECRET=your-secret
export WARREN_POLICY=./policies

warren serve
```

Note: Warren does not currently support an `--env-file` flag. Use environment variables or command-line flags.

## Production Deployment

For production, store sensitive values in Secret Manager:

```bash
# Non-sensitive environment variables
export WARREN_GEMINI_PROJECT_ID=prod-project
export WARREN_GEMINI_LOCATION=us-central1
export WARREN_SLACK_CHANNEL_NAME=security-alerts

# Sensitive values from Secret Manager
gcloud run services update warren \
  --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-token:latest" \
  --set-secrets="WARREN_SLACK_SIGNING_SECRET=slack-secret:latest"
```

## Troubleshooting Configuration

### Debug Configuration Loading

```bash
# Verbose logging to see configuration details
WARREN_LOG_LEVEL=debug warren serve
```

Note: The `--show-config` flag is not implemented.

### Common Issues

1. **"Missing required configuration"**
   - Required fields: `WARREN_FIRESTORE_PROJECT_ID`, `WARREN_GEMINI_PROJECT_ID`
   - Slack features require: `WARREN_SLACK_OAUTH_TOKEN`
   - Ensure environment variables are exported

2. **"Invalid configuration value"**
   - Check data types (numbers vs strings)
   - Verify enum values (e.g., log levels: debug, info, warn, error)
   - Log format must be "json" or "console"

3. **"Configuration file not found"**
   - Use absolute paths for file configurations
   - Check file permissions
   - Policy paths can be directories or individual files

## Configuration Best Practices

1. **Use Secret Manager** for sensitive values (API keys, tokens)
2. **Use environment variables** for deployment-specific settings
3. **Use configuration files** for complex structures (MCP, BigQuery)
4. **Document custom configurations** in your deployment scripts
5. **Validate configurations** before deployment using `--dry-run`