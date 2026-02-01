# Example Security Alerts

This directory contains sample JSON alert payloads from various security monitoring products. These can be used for testing and demonstration purposes with Warren's alert ingestion pipeline.

## Files

| File | Product | Category | Severity | Scenario |
|---|---|---|---|---|
| `scc.json` | Google Cloud Security Command Center | Cloud Monitoring | HIGH | Malware distribution domain (`download-update.online`) communication from Compute Engine |
| `guardduty.json` | AWS GuardDuty | Cloud Monitoring | 8 (High) | EC2 instance querying Cobalt Strike C2 domain (`xr7v2k9q.com`) |
| `crowdstrike_falcon.json` | CrowdStrike Falcon | EDR | Critical | Cobalt Strike beacon communicating with C2 (`1.12.66.17` / `p3nh8xzv.net`) |
| `defender_endpoint.json` | Microsoft Defender for Endpoint | EDR / AntiVirus | High | Mimikatz credential dumping, Meterpreter C2 (`43.208.238.110`) |
| `falco.json` | Falco | Container Runtime Security | Critical | Reverse shell to AsyncRAT C2 (`155.94.163.103:8080`) in Kubernetes container |

## Usage

Send any of these alerts to Warren using the raw alert webhook endpoint:

```bash
curl -X POST http://localhost:8080/hooks/alert/raw/{schema} \
  -H "Content-Type: application/json" \
  -d @examples/alert/guardduty.json
```

Replace `{schema}` with a schema name that matches your ingestion policy (e.g., `guardduty`, `crowdstrike`, `defender`, `falco`, `scc`).

## Notes on Indicators

- **Victim-side hosts/IPs**: Non-existent names and RFC 5737 / private ranges (`10.x.x.x`, `172.31.x.x`, `198.51.100.x`, `example-inc.local`)
- **Malicious indicators**: Attack-side IPs are real IOCs sourced from [ThreatFox](https://threatfox.abuse.ch/) (`1.12.66.17`, `43.208.238.110`, `155.94.163.103`, `161.97.182.121`). Domains are fictitious but realistic (DGA-style `xr7v2k9q.com`, `p3nh8xzv.net` for C2; `download-update.online` for malware distribution)
- **Schemas**: Each JSON follows the actual output format of the respective product's API or event stream
