# CrowdStrike Falcon Agent Configuration

## Overview

The Falcon agent enables Warren to query CrowdStrike Falcon for security incidents, alerts, behaviors, and CrowdScores. This is a **read-only** agent — it does not modify any data in your Falcon environment.

## Prerequisites

### Create API Client Credentials

1. Log in to the [CrowdStrike Falcon Console](https://falcon.crowdstrike.com/)
2. Navigate to **Support and resources > Resources and tools > API clients and keys**
3. Click **Create API client**
4. Configure the following:
   - **Client Name**: A descriptive name (e.g., `warren-readonly`)
   - **API Scopes**: Select the scopes listed below
5. Click **Create**
6. **Copy and securely store** the Client ID and Client Secret (the secret cannot be retrieved later)

### Required API Scopes

| Scope | Permission | Purpose |
|---|---|---|
| **Incidents** | Read | Search and retrieve incident details, behaviors |
| **Alerts** | Read | Search and retrieve alert details, aggregates |

> **Note:** Only **Read** permission is required. Do not grant Write permission unless needed for other integrations.

## Cloud Regions

CrowdStrike operates in multiple cloud regions. Set the base URL matching your Falcon tenant region:

| Cloud Region | Base URL |
|---|---|
| US-1 (default) | `https://api.crowdstrike.com` |
| US-2 | `https://api.us-2.crowdstrike.com` |
| EU-1 | `https://api.eu-1.crowdstrike.com` |
| US-GOV-1 | `https://api.laggar.gcw.crowdstrike.com` |
| US-GOV-2 | `https://api.us-gov-2.crowdstrike.mil` |

You can find your region by checking the domain in your Falcon console URL (e.g., `falcon.us-2.crowdstrike.com` means US-2).

## Deployment

### Environment Variables

```bash
export WARREN_AGENT_FALCON_CLIENT_ID="your-client-id"
export WARREN_AGENT_FALCON_CLIENT_SECRET="your-client-secret"
export WARREN_AGENT_FALCON_BASE_URL="https://api.crowdstrike.com"  # Optional, defaults to US-1
```

### CLI Flags

```bash
warren serve \
  --agent-falcon-client-id=your-client-id \
  --agent-falcon-client-secret=your-client-secret \
  --agent-falcon-base-url=https://api.crowdstrike.com
```

### Cloud Run

```bash
gcloud run services update warren \
  --set-env-vars="WARREN_AGENT_FALCON_CLIENT_ID=your-client-id" \
  --set-env-vars="WARREN_AGENT_FALCON_CLIENT_SECRET=your-client-secret" \
  --set-env-vars="WARREN_AGENT_FALCON_BASE_URL=https://api.crowdstrike.com"
```

> **Security Note:** For production, use Secret Manager for the client secret rather than plain environment variables:
> ```bash
> gcloud run services update warren \
>   --set-secrets="WARREN_AGENT_FALCON_CLIENT_SECRET=falcon-secret:latest"
> ```

## Authentication

The agent uses [OAuth2 Client Credentials Flow](https://www.falconpy.io/Service-Collections/OAuth2.html) to authenticate with the CrowdStrike API:

1. Sends `client_id` and `client_secret` to `POST /oauth2/token`
2. Receives a bearer token (valid for 30 minutes)
3. Automatically refreshes the token before expiry

No manual token management is required.

## Available Tools

The agent exposes the following read-only tools to the LLM:

### Incidents

| Tool | Description |
|---|---|
| `falcon_search_incidents` | Search for incidents using FQL filters |
| `falcon_get_incidents` | Get detailed incident information by ID |

### Alerts

| Tool | Description |
|---|---|
| `falcon_search_alerts` | Search and retrieve alerts (combined endpoint with cursor pagination) |
| `falcon_get_alerts` | Get detailed alert information by composite ID |

### Behaviors

| Tool | Description |
|---|---|
| `falcon_search_behaviors` | Search for behaviors using FQL filters |
| `falcon_get_behaviors` | Get detailed behavior information by ID |

### CrowdScores

| Tool | Description |
|---|---|
| `falcon_get_crowdscores` | Get CrowdScore values for your environment |

## Troubleshooting

### Agent not appearing in available tools

Verify that both `WARREN_AGENT_FALCON_CLIENT_ID` and `WARREN_AGENT_FALCON_CLIENT_SECRET` are set. The agent is skipped if either is empty.

### 401 Unauthorized errors

- Verify your Client ID and Client Secret are correct
- Check that the API client has not been revoked in the Falcon console
- Ensure the base URL matches your Falcon tenant region

### 403 Forbidden errors

- Verify the API client has the required scopes (`Incidents: Read`, `Alerts: Read`)
- Check that the API client is active and not expired

### Empty results

- Verify FQL filter syntax (values must be quoted, e.g., `status:'new'` not `status:new`)
- Check the time range — CrowdStrike may have data retention limits
- Use simpler filters first, then refine
