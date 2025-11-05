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

## SQL Runbooks

SQL runbooks are pre-written query templates for common investigation patterns.

### Runbook File Format

Create `.sql` files with metadata in comments:

```sql
-- Title: Failed Login Investigation
-- Description: Query to find failed login attempts by IP address
SELECT
  timestamp,
  user_email,
  source_ip,
  error_code
FROM `project.dataset.auth_logs`
WHERE
  event_type = 'login_failed'
  AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
ORDER BY timestamp DESC
LIMIT 100
```

**Metadata Format** (case insensitive):
- `-- Title: <title>` or `-- title: <title>`
- `-- Description: <description>` or `-- description: <description>`

If no title is specified, the filename (without `.sql`) is used.

### How Runbooks Work

1. **Loading**: Runbooks are loaded at agent initialization
2. **Listing**: Agent's system prompt includes all runbook IDs, titles, and descriptions
3. **Retrieval**: Agent can use `get_runbook` tool to fetch full SQL content
4. **Adaptation**: Agent adapts runbook SQL for specific investigations

## Deployment

### Environment Variables

```bash
export WARREN_AGENT_BIGQUERY_CONFIG="/path/to/config.yaml"
export WARREN_AGENT_BIGQUERY_PROJECT_ID="my-project"
export WARREN_AGENT_BIGQUERY_SCAN_SIZE_LIMIT="10GB"  # Optional override
export WARREN_AGENT_BIGQUERY_RUNBOOK_DIR="/path/to/runbooks,/path/to/more"  # Optional
```

### CLI Flags

```bash
warren serve \
  --agent-bigquery-config=/path/to/config.yaml \
  --agent-bigquery-project-id=my-project \
  --agent-bigquery-runbook-dir=/path/to/runbooks \
  --agent-bigquery-runbook-dir=/path/to/more/runbooks  # Can specify multiple times
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

### Service Account Impersonation

You can configure the BigQuery agent to impersonate a service account for all BigQuery operations. This is useful when:
- Running with different permissions than the default credentials
- Implementing least-privilege access patterns
- Testing with specific service account permissions

#### Configuration

Set via CLI flag or environment variable:

```bash
# CLI flag
warren serve \
  --agent-bigquery-config=/path/to/config.yaml \
  --agent-bigquery-impersonate-service-account=bigquery-reader@my-project.iam.gserviceaccount.com

# Environment variable
export WARREN_AGENT_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT="bigquery-reader@my-project.iam.gserviceaccount.com"
warren serve --agent-bigquery-config=/path/to/config.yaml
```

#### Required Permissions

The identity running Warren (ADC) must have the `roles/iam.serviceAccountTokenCreator` role on the target service account:

```bash
gcloud iam service-accounts add-iam-policy-binding \
  bigquery-reader@my-project.iam.gserviceaccount.com \
  --member="serviceAccount:warren-service@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/iam.serviceAccountTokenCreator"
```

The impersonated service account needs BigQuery permissions:

```bash
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:bigquery-reader@my-project.iam.gserviceaccount.com" \
  --role="roles/bigquery.jobUser"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:bigquery-reader@my-project.iam.gserviceaccount.com" \
  --role="roles/bigquery.dataViewer"
```

### Service Account Permissions

Required IAM roles for the service account (either default ADC or impersonated):
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
