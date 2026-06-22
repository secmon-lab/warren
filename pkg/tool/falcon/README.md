# CrowdStrike Falcon Tool Configuration

## Overview

The Falcon tool enables Warren to query CrowdStrike Falcon for incidents, alerts, behaviors, devices, CrowdScores, and raw EDR telemetry events (Next-Gen SIEM). Security analysts can pivot from alert context into endpoint detection data during an investigation.

This is a **read-only** tool — it does not modify any data in your CrowdStrike environment. It is backed by [`github.com/gollem-dev/tools/falcon`](https://github.com/gollem-dev/tools).

> **Note:** This replaces the former CrowdStrike Falcon *sub-agent* (`WARREN_AGENT_FALCON_*`). See [doc/migration/v0.18.0.md](../../../doc/migration/v0.18.0.md).

## Configuration

| Environment Variable | CLI Flag | Default | Required |
|---|---|---|---|
| `WARREN_FALCON_CLIENT_ID` | `--falcon-client-id` | - | Yes |
| `WARREN_FALCON_CLIENT_SECRET` | `--falcon-client-secret` | - | Yes |
| `WARREN_FALCON_BASE_URL` | `--falcon-base-url` | `https://api.crowdstrike.com` | No |

The tool is enabled only when both `WARREN_FALCON_CLIENT_ID` and `WARREN_FALCON_CLIENT_SECRET` are set; otherwise it is silently skipped.

## Setup

1. Sign in to the [CrowdStrike Falcon console](https://falcon.crowdstrike.com/).
2. Go to **Support and resources > API clients and keys**.
3. Click **Create API client** and grant the following **read** scopes:
   - Incidents (read)
   - Alerts (read)
   - Hosts / Devices (read)
   - Event search / Next-Gen SIEM (read), if EDR event search is needed
4. Copy the **Client ID** and **Client Secret**.
5. Set the **Base URL** matching your CrowdStrike cloud region (US-1 is the default; US-2, EU-1, US-GOV-1 differ).

## Available Functions

| Function | Description |
|---|---|
| `falcon_search_incidents` | Search incidents via FQL |
| `falcon_get_incidents` | Fetch incident details by IDs |
| `falcon_search_alerts` | Search alerts via FQL |
| `falcon_get_alerts` | Fetch alert details by composite IDs |
| `falcon_search_behaviors` | Search behaviors via FQL |
| `falcon_get_behaviors` | Fetch behavior details by IDs |
| `falcon_search_devices` | Search devices (hosts) via FQL |
| `falcon_get_devices` | Fetch device details by IDs |
| `falcon_get_crowdscores` | Retrieve CrowdScore environment risk values |
| `falcon_search_events` | Search raw EDR telemetry events (Next-Gen SIEM) |

To avoid overwhelming the LLM, search functions return a bounded number of records per call and keep the overflow in an in-memory page store; the LLM fetches subsequent pages by passing the returned `page_token` back to the same function.

## Authentication

The tool uses the CrowdStrike OAuth2 client credentials flow (`POST {base_url}/oauth2/token`), caches the bearer token, and retries once on a 401 by clearing the cached token. No manual token management is required.
