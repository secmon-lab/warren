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

## Google Cloud Configuration

### Initialize Google Cloud Project
1. Create a Google Cloud Project
2. Enable the following APIs
    - Cloud Run API
    - Secret Manager API
    - Firestore API
    - Pub/Sub API (if you want to use Pub/Sub as a trigger)

### Create a service account
1. Move IAM & Admin -> Service Accounts
2. Click "Create Service Account"
3. Set the following permissions
    - `roles/secretmanager.secretAccessor`
    - `roles/firestore.user`

