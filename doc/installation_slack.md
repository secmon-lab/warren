# Slack Integration Setup for Warren

This guide walks you through creating and configuring a Slack app for Warren, including all required permissions, OAuth configuration, and webhook setup.

## Prerequisites

- Admin access to a Slack workspace
- Warren service URL (from Google Cloud deployment)
- Basic understanding of Slack app configuration

## 1. Create a Slack App

### 1.1. Initial App Creation

1. Go to [Slack API Apps page](https://api.slack.com/apps)
2. Click **"Create New App"**
3. Choose **"From scratch"**
4. Configure:
   - **App Name**: `Warren Security Bot` (or your preferred name)
   - **Workspace**: Select your target workspace
5. Click **"Create App"**

### 1.2. App Configuration Basics

After creation, you'll be on the app configuration page. Save the following from **Basic Information**:

- **App ID**: (You'll see this in the URL and app info)
- **Client ID**: Under "App Credentials"
- **Client Secret**: Under "App Credentials" (keep this secure!)
- **Signing Secret**: Under "App Credentials" (required for webhook verification)

## 2. Configure OAuth & Permissions

### 2.1. Bot Token Scopes

Navigate to **"OAuth & Permissions"** in the sidebar, then scroll to **"Scopes"**.

Add these **Bot Token Scopes**:

| Scope | Purpose |
|-------|---------|
| `app_mentions:read` | Detect when users mention @warren |
| `channels:history` | Read message history in public channels |
| `channels:read` | View basic channel information |
| `chat:write` | Post messages and create threads |
| `files:write` | Upload files and images |
| `reactions:read` | Read emoji reactions on messages |
| `users:read` | Look up user information |
| `usergroups:read` | Access user group information |
| `team:read` | Read workspace information |

### 2.2. User Token Scopes (for Web UI OAuth)

In the same **"Scopes"** section, add these **User Token Scopes**:

| Scope | Purpose |
|-------|---------|
| `openid` | OpenID Connect authentication |
| `email` | Access user email for identification |
| `profile` | Access user profile information |

These scopes enable users to log into Warren's web interface using their Slack credentials.

### 2.3. OAuth Redirect URLs

Still in **"OAuth & Permissions"**, add your redirect URL:

1. Scroll to **"Redirect URLs"**
2. Click **"Add New Redirect URL"**
3. Enter: `https://YOUR-WARREN-URL/api/auth/callback`
   - Replace `YOUR-WARREN-URL` with your actual Warren service URL
   - Example: `https://warren-abc123-uc.a.run.app/api/auth/callback`
4. Click **"Add"**
5. Click **"Save URLs"**

## 3. Install App to Workspace

### 3.1. Initial Installation

1. Go to **"OAuth & Permissions"**
2. Click **"Install to Workspace"**
3. Review the permissions summary
4. Click **"Allow"**

### 3.2. Save OAuth Tokens

After installation, you'll see:

- **Bot User OAuth Token**: Starts with `xoxb-`
  - Save this as `WARREN_SLACK_OAUTH_TOKEN`
- **User OAuth Token**: Starts with `xoxp-` (not needed for Warren)

## 4. Configure Event Subscriptions

### 4.1. Enable Events

1. Navigate to **"Event Subscriptions"** in the sidebar
2. Toggle **"Enable Events"** to On
3. Enter Request URL: `https://YOUR-WARREN-URL/hooks/slack/event`

Warren will respond to Slack's verification challenge automatically. You should see "Verified" ✓.

### 4.2. Subscribe to Bot Events

In the **"Subscribe to bot events"** section, add:

| Event | Purpose |
|-------|---------|
| `app_mention` | Respond when @warren is mentioned |
| `message.channels` | Monitor messages in channels where Warren is present |

Click **"Save Changes"** at the bottom.

## 5. Configure Interactivity

### 5.1. Enable Interactive Components

1. Navigate to **"Interactivity & Shortcuts"** in the sidebar
2. Toggle **"Interactivity"** to On
3. Enter Request URL: `https://YOUR-WARREN-URL/hooks/slack/interaction`
4. Click **"Save Changes"**

This enables Warren's interactive buttons and modal dialogs.

## 6. Create Warren Channel

### 6.1. Channel Setup

1. In your Slack workspace, create a channel for Warren alerts
   - Suggested name: `#security-alerts` or `#warren-alerts`
   - Can be public or private based on your security requirements

2. Add Warren bot to the channel:
   - Click on the channel name at the top
   - Select "Integrations" or "Settings" → "Add apps"
   - Search for your Warren app and add it
   - Or use the app's home page to join channels

3. Note the channel name (without #) for Warren configuration
   - This will be `WARREN_SLACK_CHANNEL_NAME`

## 7. Configure Warren Environment

### 7.1. Required Configuration

Warren needs these Slack-related environment variables:

| Variable | Value | Source |
|----------|-------|--------|
| `WARREN_SLACK_OAUTH_TOKEN` | `xoxb-...` | Bot User OAuth Token |
| `WARREN_SLACK_SIGNING_SECRET` | `abc123...` | App Signing Secret |
| `WARREN_SLACK_CLIENT_ID` | `123456789.123456789` | App Client ID |
| `WARREN_SLACK_CLIENT_SECRET` | `abc123...` | App Client Secret |
| `WARREN_SLACK_CHANNEL_NAME` | `security-alerts` | Your channel name |

### 7.2. Update Cloud Run Service

If using Google Cloud Run:

```bash
# Update with Slack configuration
gcloud run services update warren \
    --region=YOUR-REGION \
    --set-env-vars="WARREN_SLACK_CHANNEL_NAME=security-alerts" \
    --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-oauth-token:latest" \
    --set-secrets="WARREN_SLACK_SIGNING_SECRET=slack-signing-secret:latest" \
    --set-secrets="WARREN_SLACK_CLIENT_ID=slack-client-id:latest" \
    --set-secrets="WARREN_SLACK_CLIENT_SECRET=slack-client-secret:latest"
```

## 8. Test Slack Integration

### 8.1. Basic Integration Test

1. Send a test alert to your Warren webhook endpoint (see Integration Guide)

2. Verify that the alert appears in your configured Slack channel with:
   - Alert title and description
   - Interactive buttons (Acknowledge, Bind to ticket)
   - Proper formatting

### 8.2. Verify Event Handling

1. Check Cloud Run logs:
   ```bash
   gcloud logs read "resource.type=cloud_run_revision" --limit=50
   ```

2. Look for:
   - Successful Slack signature verification
   - Event receipt confirmations
   - No authentication errors

### 8.3. Test Interactive Components

1. Send a test alert to Warren
2. Verify interactive buttons appear on alert messages
3. Click a button and confirm modal appears

## 9. Optional: App Customization

### 9.1. App Appearance

In **"Basic Information"** → **"Display Information"**:

- **App icon**: Upload a custom icon (recommended: 512x512px)
- **App name**: How it appears in Slack
- **Short description**: Brief description for app directory
- **Background color**: Match your brand colors

### 9.2. Bot User Customization

In **"App Home"**:

- **Bot Name**: Change how @warren appears
- **Always Show My Bot as Online**: Enable for better visibility

## 10. Troubleshooting

### Common Issues

1. **"Request URL verification failed"**
   - Verify Warren is running and accessible
   - Check the URL is correct (https, no trailing slash)
   - Review Warren logs for verification errors

2. **"Invalid signing secret"**
   - Ensure signing secret is correctly set in Warren
   - No extra spaces or newlines in the secret

3. **Bot not responding**
   - Verify bot is in the channel (check channel integrations)
   - Check OAuth token is valid
   - Review permission scopes
   - Ensure bot has been added to the channel through Slack UI

4. **OAuth redirect fails**
   - Confirm redirect URL exactly matches configuration
   - Include https:// and full path

### Troubleshooting Commands

For debugging purposes, Warren supports these Slack commands when mentioned:
- `@warren list` (or `l`, `ls`) - List current alerts
- `@warren aggregate` (or `a`, `aggr`) - Group related alerts  
- `@warren ticket` (or `t`) - Manage tickets
- `@warren repair` - Repair functionality

These commands are primarily for troubleshooting. Normal operation is through the Web UI and interactive buttons in Slack messages.

### Debug Checklist

- [ ] Warren service is running and accessible
- [ ] All 4 Slack secrets are correctly configured
- [ ] Bot is invited to the target channel
- [ ] Event subscriptions show "Verified"
- [ ] Interactive components URL is verified
- [ ] OAuth redirect URL matches exactly

### Viewing Slack API Calls

Enable debug logging in Warren to see detailed Slack API interactions:

```bash
# If using Cloud Run, update with debug flag
gcloud run services update warren \
    --set-env-vars="WARREN_LOG_LEVEL=debug"
```

## 11. Security Best Practices

1. **Rotate Credentials Regularly**
   - Regenerate signing secret periodically
   - Update OAuth tokens if compromised

2. **Limit Scope**
   - Only grant necessary permissions
   - Use private channels for sensitive alerts

3. **Monitor Access**
   - Review app installation list
   - Audit bot activity in channels

4. **Webhook Security**
   - Warren automatically verifies Slack signatures
   - Ensure HTTPS is always used
   - Don't expose webhook URLs publicly

## Next Steps

1. Test the complete alert flow
2. Configure alert policies
3. Train team on Slack interactions
4. Set up monitoring and alerts

## Additional Resources

- [Slack API Documentation](https://api.slack.com/docs)
- [Slack App Security Best Practices](https://api.slack.com/security-best-practices)
- [Warren GitHub Repository](https://github.com/secmon-lab/warren)