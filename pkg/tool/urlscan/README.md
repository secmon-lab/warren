# URLScan.io

Scan URLs for malicious content, phishing, and other threats using urlscan.io.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_URLSCAN_API_KEY` | `--urlscan-api-key` | Yes | URLScan.io API key |
| `WARREN_URLSCAN_BASE_URL` | `--urlscan-base-url` | No | Base URL (default: `https://urlscan.io/api/v1`) |
| `WARREN_URLSCAN_BACKOFF` | `--urlscan-backoff` | No | Polling backoff interval (default: `3s`) |
| `WARREN_URLSCAN_TIMEOUT` | `--urlscan-timeout` | No | Scan timeout (default: `30s`) |

## Available Functions

| Function | Description |
|---|---|
| `urlscan_scan` | Submit a URL for scanning. The scan runs asynchronously with polling until results are available or the timeout is reached. |

## Setup

1. Create an account at [urlscan.io](https://urlscan.io/)
2. Navigate to **Settings & API > API Keys**
3. Create a new API key
4. Set `WARREN_URLSCAN_API_KEY` environment variable

The tool is automatically enabled when the API key is configured.
