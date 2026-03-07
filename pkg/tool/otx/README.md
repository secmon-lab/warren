# OTX (AlienVault Open Threat Exchange)

Search threat indicators from AlienVault OTX, including IP addresses, domains, hostnames, and file hashes.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_OTX_API_KEY` | `--otx-api-key` | Yes | OTX API key |
| `WARREN_OTX_BASE_URL` | `--otx-base-url` | No | Base URL (default: `https://otx.alienvault.com/api/v1`) |

## Available Functions

| Function | Description |
|---|---|
| `otx_ipv4` | Search IPv4 address for threat intelligence |
| `otx_ipv6` | Search IPv6 address for threat intelligence |
| `otx_domain` | Search domain for threat intelligence |
| `otx_hostname` | Search hostname for threat intelligence |
| `otx_file_hash` | Search file hash for threat intelligence |

## Setup

1. Create an account at [otx.alienvault.com](https://otx.alienvault.com/)
2. Navigate to **Settings > API Integration**
3. Copy your OTX API key
4. Set `WARREN_OTX_API_KEY` environment variable

The tool is automatically enabled when the API key is configured.
