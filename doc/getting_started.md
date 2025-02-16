# Getting Started

## Slack App Configuration

1. Create a Slack App
2. (Optional) Configure the app "Display Information"
3. Add permissions
    - Move "OAuth & Permissions" -> "Scopes"
    - Click "Add an OAuth Scope" and add the following:
        - `app_mentions:read`
        - `channels:history`
        - `chat:write`
        - `files:write`
        - `reactions:read`
4. Install the app to your workspace
    - Move "OAuth & Permissions" -> "Install App" or "Request to install app"
5. Get the Bot secrets
    - Move "OAuth & Permissions" -> "OAuth Tokens" -> "Bot User OAuth Token"
      - It will be used as `WARREN_SLACK_OAUTH_TOKEN` environment variable
    - Move "Basic Information" -> "App Credentials" -> "Signing Secret"
      - It will be used as `WARREN_SLACK_SIGNING_SECRET` environment variable
6. Create a Slack channel (if you don't have one)
    - It will be used as `WARREN_SLACK_CHANNEL_NAME`. It's not required `#` prefix.

