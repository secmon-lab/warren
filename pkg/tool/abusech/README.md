# abuse.ch (MalwareBazaar)

Query malware information from MalwareBazaar by file hash for malware identification and analysis.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_ABUSECH_AUTH_KEY` | `--abusech-api-key` | Yes | MalwareBazaar API key |
| `WARREN_ABUSECH_BASE_URL` | `--abusech-base-url` | No | Base URL (default: `https://mb-api.abuse.ch/api/v1`) |

## Available Functions

| Function | Description |
|---|---|
| `abusech.bazaar.query` | Query malware information by file hash (MD5, SHA1, or SHA256) |

## Setup

1. Create an account at [bazaar.abuse.ch](https://bazaar.abuse.ch/)
2. Navigate to your account settings to find your API key
3. Set `WARREN_ABUSECH_AUTH_KEY` environment variable

The tool is automatically enabled when the API key is configured.
