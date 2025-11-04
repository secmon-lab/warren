# BigQuery Agent Configuration

## Configuration File

Create a YAML file defining BigQuery tables:

```yaml
projects:
  - id: my-security-project
    description: Security data warehouse
    datasets:
      - id: cloudtrail_logs
        description: AWS CloudTrail audit logs
        tables:
          - id: events
            description: CloudTrail events with user activity, API calls, and resource changes

scan_size_limit: "10GB"  # Required: Maximum bytes scanned per query
query_timeout: 5m        # Optional: Default 5m
```

## Configuration Fields

### Projects Structure

```yaml
projects:
  - id: <project-id>              # Required: GCP project ID
    description: <description>     # Optional: Inherited by child elements
    datasets:
      - id: <dataset-id>           # Required: Dataset ID
        description: <description> # Optional: Inherited by tables
        tables:
          - id: <table-id>         # Required: Table ID
            description: <desc>    # Recommended: Detailed table description
```

### Global Settings

- `scan_size_limit` (required): Maximum bytes scanned per query
  - Format: `"1GB"`, `"10GB"`, `"1TB"`, etc.
- `query_timeout` (optional): Query timeout duration
  - Format: `"5m"`, `"30s"`, etc.
  - Default: `5m`

## Table Descriptions

**Good descriptions improve agent performance significantly.**

Good example:
```yaml
description: |
  AWS CloudTrail events containing:
  - Authentication events (ConsoleLogin, AssumeRole)
  - API calls (eventName, requestParameters, responseElements)
  - Failed attempts (errorCode, errorMessage)
  - Fields: sourceIPAddress, userAgent, mfaUsed
  - Partitioned by event_date (last 90 days)
```

Bad example:
```yaml
description: CloudTrail data  # Too vague
```

## Deployment

### Environment Variables

```bash
export WARREN_AGENT_BIGQUERY_CONFIG="/path/to/config.yaml"
export WARREN_AGENT_BIGQUERY_PROJECT_ID="my-project"
export WARREN_AGENT_BIGQUERY_SCAN_SIZE_LIMIT="10GB"  # Optional override
```

### CLI Flags

```bash
warren serve \
  --agent-bigquery-config=/path/to/config.yaml \
  --agent-bigquery-project-id=my-project
```

### Cloud Run

```bash
gcloud run services update warren \
  --set-env-vars="WARREN_AGENT_BIGQUERY_CONFIG=/app/config.yaml" \
  --set-env-vars="WARREN_AGENT_BIGQUERY_PROJECT_ID=my-project"
```

Note: Config file must be in the container image or mounted.

## Authentication

Uses [Application Default Credentials (ADC)](https://cloud.google.com/docs/authentication/application-default-credentials).

### Local Development

```bash
gcloud auth application-default login
```

### Service Account Permissions

Required IAM roles:
- `roles/bigquery.jobUser`
- `roles/bigquery.dataViewer`

```bash
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:warren-service@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/bigquery.jobUser"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:warren-service@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/bigquery.dataViewer"
```

## Example Configurations

### Minimal

```yaml
projects:
  - id: my-project
    datasets:
      - id: security_logs
        tables:
          - id: events
            description: Security event logs

scan_size_limit: "1GB"
```

### Multi-Project

```yaml
projects:
  - id: prod-logs
    description: Production logs
    datasets:
      - id: aws_cloudtrail
        tables:
          - id: events_2024
            description: |
              CloudTrail events from 2024:
              - Authentication (ConsoleLogin, AssumeRole)
              - S3 access (GetObject, PutObject)
              - IAM changes
              - Partitioned by event_date

      - id: gcp_audit
        tables:
          - id: admin_activity
            description: GCP admin activity logs

  - id: staging-logs
    datasets:
      - id: app_logs
        tables:
          - id: authentication
            description: App authentication events

scan_size_limit: "5GB"
query_timeout: 3m
```
