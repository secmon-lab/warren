# VirusTotal

Search threat indicators (IP addresses, domains, file hashes, URLs) from VirusTotal for malware and threat intelligence analysis.

## Configuration

| Environment Variable | CLI Flag | Required | Description |
|---|---|---|---|
| `WARREN_VT_API_KEY` | `--vt-api-key` | Yes | VirusTotal API key |
| `WARREN_VT_BASE_URL` | `--vt-base-url` | No | Base URL (default: `https://www.virustotal.com/api/v3`) |

## Available Functions

| Function | Description |
|---|---|
| `vt_ip` | Search IPv4/IPv6 address reputation and associated information |
| `vt_domain` | Search domain reputation, DNS records, and associated information |
| `vt_file_hash` | Search file hash (MD5/SHA1/SHA256) for malware analysis results |
| `vt_url` | Search URL for scan results and reputation |

## Setup

1. Create a VirusTotal account at [virustotal.com](https://www.virustotal.com/)
2. Navigate to your [API key page](https://www.virustotal.com/gui/my-apikey)
3. Copy your API key
4. Set `WARREN_VT_API_KEY` environment variable

The tool is automatically enabled when the API key is configured. It is skipped if the key is not set.
