# Slack Search Agent

Sub-agent for searching Slack messages to provide investigative context during alert analysis. Useful for finding related discussions, previous incidents, and contextual information from team communications.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_AGENT_SLACK_USER_TOKEN` | `--agent-slack-user-token` | Yes | Slack **User** token with `search:read` scope |

> **Note:** The Slack search API requires a **User token** (`xoxp-...`), not a Bot token (`xoxb-...`).

## Capabilities

- Search Slack messages with full query syntax
- Sort results by relevance score or timestamp
- Paginate through large result sets

## Setup

1. In your Slack workspace, create or use an existing Slack App
2. Navigate to **OAuth & Permissions**
3. Under **User Token Scopes**, add `search:read`
4. Install the app and copy the **User OAuth Token** (`xoxp-...`)
5. Set `WARREN_AGENT_SLACK_USER_TOKEN` environment variable

## Agent Memory

The Slack agent uses agent memory (ID: `slack_search`) to learn effective search patterns and remember useful query strategies from previous investigations.

The agent is automatically registered when the user token is configured.
