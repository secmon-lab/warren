# AbuseIPDB

Check IP address reputation using AbuseIPDB to identify malicious or suspicious IP addresses.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_IPDB_API_KEY` | `--ipdb-api-key` | Yes | AbuseIPDB API key |
| `WARREN_IPDB_BASE_URL` | `--ipdb-base-url` | No | Base URL (default: `https://api.abuseipdb.com/api/v2`) |

## Available Functions

| Function | Description |
|---|---|
| `ipdb_check` | Check IP address reputation. Optional `max_age_in_days` parameter (1-365) to limit report age. |

## Setup

1. Create an account at [abuseipdb.com](https://www.abuseipdb.com/)
2. Navigate to **API** section in your account
3. Create a new API key
4. Set `WARREN_IPDB_API_KEY` environment variable

The tool is automatically enabled when the API key is configured.
