# Slack Message Search

Search Slack messages using the Slack search.messages API for investigative context during alert analysis.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_SLACK_TOOL_USER_TOKEN` | `--slack-tool-user-token` | Yes | Slack **User** token (not Bot token). Requires `search:read` scope. |

> **Note:** The Slack search API requires a **User token** (`xoxp-...`), not a Bot token (`xoxb-...`). Bot tokens do not support the `search.messages` API.

## Available Functions

| Function | Description |
|---|---|
| `slack_message_search` | Search Slack messages. Supports parameters: `query`, `sort` (score/timestamp), `sort_dir` (asc/desc), `count`, `page`, `highlight` |

## Setup

1. In your Slack workspace, create or use an existing Slack App
2. Navigate to **OAuth & Permissions**
3. Under **User Token Scopes**, add `search:read`
4. Install the app and copy the **User OAuth Token** (`xoxp-...`)
5. Set `WARREN_SLACK_TOOL_USER_TOKEN` environment variable

The tool is automatically enabled when the user token is configured.
