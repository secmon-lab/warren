# Shodan

Search internet-connected device information using Shodan, including host details, open ports, and services.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_SHODAN_API_KEY` | `--shodan-api-key` | Yes | Shodan API key |
| `WARREN_SHODAN_BASE_URL` | `--shodan-base-url` | No | Base URL (default: `https://api.shodan.io`) |

## Available Functions

| Function | Description |
|---|---|
| `shodan_host` | Get host information by IP address (open ports, services, banners) |
| `shodan_domain` | Get domain information (subdomains, DNS records) |
| `shodan_search` | Search the Shodan database using query syntax (with configurable result limit) |

## Setup

1. Create an account at [shodan.io](https://www.shodan.io/)
2. Navigate to **My Account** to find your API key
3. Set `WARREN_SHODAN_API_KEY` environment variable

The tool is automatically enabled when the API key is configured.
