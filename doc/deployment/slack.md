# Slack Integration Setup

This guide covers creating and configuring a Slack app for Warren.

## Prerequisites

- Admin access to a Slack workspace
- Warren service URL (from [GCP deployment](./gcp.md) or local setup)

## 1. Create a Slack App

1. Go to [Slack API Apps page](https://api.slack.com/apps)
2. Click **"Create New App"** > **"From scratch"**
3. Name it (e.g., `Warren Security Bot`) and select your workspace
4. Save from **Basic Information**:
   - **Client ID** and **Client Secret** (under "App Credentials")
   - **Signing Secret** (under "App Credentials")

## 2. Configure OAuth & Permissions

Navigate to **"OAuth & Permissions"**.

### Bot Token Scopes

| Scope | Purpose |
|-------|---------|
| `app_mentions:read` | Detect @warren mentions |
| `channels:history` | Read channel message history |
| `channels:read` | View channel information |
| `chat:write` | Post messages and create threads |
| `files:write` | Upload files and images |
| `reactions:read` | Read emoji reactions |
| `users:read` | Look up user information |
| `usergroups:read` | Access user group information |
| `team:read` | Read workspace information |

### User Token Scopes (for Web UI OAuth)

| Scope | Purpose |
|-------|---------|
| `openid` | OpenID Connect authentication |
| `email` | User email for identification |
| `profile` | User profile information |

### Redirect URL

Add: `https://YOUR-WARREN-URL/api/auth/callback`

## 3. Install App

1. Go to **"OAuth & Permissions"** > **"Install to Workspace"**
2. Click **"Allow"**
3. Save the **Bot User OAuth Token** (`xoxb-...`)

## 4. Event Subscriptions

1. Navigate to **"Event Subscriptions"**
2. Enable Events, enter Request URL: `https://YOUR-WARREN-URL/hooks/slack/event`
3. Subscribe to bot events:
   - `app_mention`
   - `message.channels`
4. Save Changes

## 5. Interactivity

1. Navigate to **"Interactivity & Shortcuts"**
2. Enable Interactivity
3. Request URL: `https://YOUR-WARREN-URL/hooks/slack/interaction`
4. Save Changes

## 6. Create Alert Channel

1. Create a channel (e.g., `#security-alerts`)
2. Add Warren bot to the channel
3. Note the channel name (without `#`)

## 7. Configure Warren

Set the following environment variables:

| Variable | Value |
|----------|-------|
| `WARREN_SLACK_OAUTH_TOKEN` | Bot User OAuth Token (`xoxb-...`) |
| `WARREN_SLACK_SIGNING_SECRET` | App Signing Secret |
| `WARREN_SLACK_CLIENT_ID` | App Client ID |
| `WARREN_SLACK_CLIENT_SECRET` | App Client Secret |
| `WARREN_SLACK_CHANNEL_NAME` | Channel name (without `#`) |

For Cloud Run:

```bash
gcloud run services update warren \
    --region=YOUR-REGION \
    --set-env-vars="WARREN_SLACK_CHANNEL_NAME=security-alerts" \
    --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-oauth-token:latest" \
    --set-secrets="WARREN_SLACK_SIGNING_SECRET=slack-signing-secret:latest" \
    --set-secrets="WARREN_SLACK_CLIENT_ID=slack-client-id:latest" \
    --set-secrets="WARREN_SLACK_CLIENT_SECRET=slack-client-secret:latest"
```

## 8. Test Integration

```bash
# Send a test alert
curl -X POST https://YOUR-WARREN-URL/hooks/alert/raw/test \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Alert", "description": "Testing Slack integration", "severity": "low"}'
```

Verify: alert appears in your Slack channel with interactive buttons.

## Available Slack Commands

Mention `@warren` in a ticket thread to use these commands:

| Command | Aliases | Description |
|---------|---------|-------------|
| `ticket` | `t` | Create or manage a ticket |
| `refine` | `r` | Review open tickets and consolidate unbound alerts |
| `repair` | | Repair ticket state |
| `abort` | | Abort current operation |
| `purge` | | Purge and clean up |

## Troubleshooting

- **Request URL verification failed**: Verify Warren is running and accessible
- **Invalid signing secret**: Check for extra spaces/newlines in the secret
- **Bot not responding**: Verify bot is added to the channel and OAuth token is valid
- **OAuth redirect fails**: Confirm redirect URL matches exactly

For all configuration options, see [Configuration Reference](../reference/configuration.md).
