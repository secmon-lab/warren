# Warren Installation Guide

Warren is an AI-powered security alert management tool that integrates with Slack and Google Cloud services. This guide will walk you through the complete deployment process on Google Cloud.

## Architecture

Warren consists of:
- **Go Backend**: REST API and GraphQL server with AI-powered alert processing
- **React Frontend**: Modern web UI for ticket management and dashboard
- **Slack Integration**: Bot for collaborative incident response
- **Google Cloud Services**: Firestore, Cloud Storage, Vertex AI (Gemini), Secret Manager

```mermaid
graph TB
    %% External Users and Services
    User[👤 Security Analyst]
    SlackUser[👥 Slack Users]
    Browser[🌐 Web Browser]
    
    %% Slack Infrastructure
    Slack[📱 Slack Workspace]
    SlackBot[🤖 Warren Bot]
    
    %% External Security Services
    subgraph "External Security Services"
        OTX[🔍 OTX<br/>Threat Intelligence]
        URLScan[🔗 URLScan<br/>URL Analysis]
        VirusTotal[🦠 VirusTotal<br/>File/URL Scanning]
        Shodan[🌐 Shodan<br/>Internet Scanning]
        AbuseChip[⚠️ AbuseChip<br/>IP Reputation]
        IPDB[📍 IPDB<br/>IP Geolocation]
    end
    
    %% Google Cloud Platform
    subgraph "Google Cloud Platform"
        direction TB
        
        %% Container and Deployment
        subgraph "Container Infrastructure"
            ArtifactRegistry[📦 Artifact Registry<br/>Docker Images]
            CloudRun[☁️ Cloud Run<br/>Warren Application]
        end
        
        %% AI and Analytics
        subgraph "AI & Analytics"
            VertexAI[🧠 Vertex AI<br/>Gemini Model]
            BigQuery[📊 BigQuery<br/>Security Data Analytics]
        end
        
        %% Storage and Security
        subgraph "Data & Security"
            Firestore[🗃️ Firestore<br/>Alerts & Tickets]
            CloudStorage[💾 Cloud Storage<br/>Files & Artifacts]
            SecretManager[🔐 Secret Manager<br/>API Keys & Tokens]
        end
        
        %% Cloud Run Application Components
        subgraph "Warren Application (Cloud Run)"
            direction LR
            Backend[⚙️ Go Backend<br/>REST API & GraphQL]
            Frontend[🖥️ React Frontend<br/>Dashboard & UI]
            PolicyEngine[📋 Policy Engine<br/>Rego Evaluation]
        end
    end
    
    %% Alert Sources
    subgraph "Alert Sources"
        GuardDuty[🛡️ AWS GuardDuty]
        SIEM[📊 SIEM Systems]
        SecurityTools[🔧 Security Tools]
        SNS[📧 AWS SNS]
        PubSub[📮 Google Pub/Sub]
    end
    
    %% User Interactions
    User --> Browser
    Browser --> Frontend
    SlackUser --> Slack
    Slack --> SlackBot
    
    %% Slack Integration
    SlackBot <--> Backend
    Backend --> Slack
    
    %% Alert Processing Flow
    GuardDuty --> SNS
    SIEM --> PubSub
    SecurityTools --> Backend
    SNS --> Backend
    PubSub --> Backend
    
    %% Core Application Flow
    Backend --> PolicyEngine
    PolicyEngine --> VertexAI
    Backend --> Firestore
    Backend --> CloudStorage
    Backend <--> Frontend
    
    %% External Service Integration
    Backend --> OTX
    Backend --> URLScan
    Backend --> VirusTotal
    Backend --> Shodan
    Backend --> AbuseChip
    Backend --> IPDB
    Backend --> BigQuery
    
    %% Infrastructure Dependencies
    CloudRun --> SecretManager
    CloudRun --> ArtifactRegistry
    
    %% Styling
    classDef gcpService fill:#4285f4,stroke:#333,stroke-width:2px,color:#fff
    classDef slackService fill:#4a154b,stroke:#333,stroke-width:2px,color:#fff
    classDef externalService fill:#ff6b6b,stroke:#333,stroke-width:2px,color:#fff
    classDef warrenApp fill:#51cf66,stroke:#333,stroke-width:2px,color:#fff
    classDef userService fill:#ffd43b,stroke:#333,stroke-width:2px,color:#000
    
    class ArtifactRegistry,CloudRun,VertexAI,BigQuery,Firestore,CloudStorage,SecretManager gcpService
    class Slack,SlackBot slackService
    class OTX,URLScan,VirusTotal,Shodan,AbuseChip,IPDB,GuardDuty,SIEM,SecurityTools,SNS,PubSub externalService
    class Backend,Frontend,PolicyEngine warrenApp
    class User,SlackUser,Browser userService
```

## Prerequisites

- Go 1.23.4 or later
- Docker 25.0.6 or later
- Google Cloud SDK 464.0.0 or later
- Google Cloud CLI 464.0.0 or later
- A Slack workspace with admin permissions

## 1. Slack App Configuration

### 1.1. Create a Slack App

1. Go to [Slack API](https://api.slack.com/apps) and click "Create New App"
2. Choose "From scratch" and provide:
   - App Name: `Warren Security Bot`
   - Workspace: Select your workspace
3. Click "Create App"

### 1.2. Configure OAuth Scopes

Navigate to "OAuth & Permissions" → "Scopes" and add the following Bot Token Scopes:

#### Required Bot Token Scopes
- `app_mentions:read` - View messages that directly mention @warren
- `channels:history` - View messages and other content in public channels
- `channels:read` - View basic information about public channels
- `chat:write` - Post, update, and delete messages
- `files:write` - Upload, edit, and delete files
- `reactions:read` - View emoji reactions and their associated content
- `users:read` - View people in a workspace (for user profile retrieval)
- `usergroups:read` - View user groups in a workspace
- `team:read` - View the workspace name, email domain, and icon

### 1.3. Install App to Workspace

1. Navigate to "OAuth & Permissions" → "Install App"
2. Click "Install to Workspace" or "Request to Install App"
3. Grant the requested permissions

### 1.4. Collect Slack Credentials

Save these values for later configuration:

1. **Bot User OAuth Token**: 
   - Go to "OAuth & Permissions" → "OAuth Tokens for Your Workspace"
   - Copy the "Bot User OAuth Token" (starts with `xoxb-`)
   - This will be used as `WARREN_SLACK_OAUTH_TOKEN`

2. **Signing Secret**:
   - Go to "Basic Information" → "App Credentials"
   - Copy the "Signing Secret"
   - This will be used as `WARREN_SLACK_SIGNING_SECRET`

3. **Client ID and Secret** (for Web UI OAuth):
   - Copy "Client ID" from "Basic Information" → "App Credentials"
   - Copy "Client Secret" from "Basic Information" → "App Credentials"
   - These will be used as `WARREN_SLACK_CLIENT_ID` and `WARREN_SLACK_CLIENT_SECRET`

### 1.6. Configure OAuth Settings (Important)

**Set OAuth Scopes for User Authentication:**

1. Go to "OAuth & Permissions" → "Scopes" → "User Token Scopes"
2. Add the following User Token Scopes for Web UI authentication:
   - `openid` - Required for OpenID Connect authentication
   - `email` - Access user's email address
   - `profile` - Access user's profile information

**Note:** These scopes are separate from Bot Token Scopes and are specifically needed for the Web UI OAuth flow that allows users to log into Warren's web interface.

### 1.5. Create Slack Channel

Create a dedicated channel for Warren notifications (e.g., `#security-alerts`). The channel name (without `#`) will be used as `WARREN_SLACK_CHANNEL_NAME`.

## 2. Google Cloud Configuration

### 2.1. Create and Setup Google Cloud Project

```bash
# Set your project ID
export PROJECT_ID="your-warren-project"
export REGION="us-central1"

# Create a new project (optional)
gcloud projects create $PROJECT_ID

# Set the project as default
gcloud config set project $PROJECT_ID

# Enable required APIs
gcloud services enable \
    run.googleapis.com \
    secretmanager.googleapis.com \
    firestore.googleapis.com \
    storage-component.googleapis.com \
    aiplatform.googleapis.com \
    artifactregistry.googleapis.com \
    cloudbuild.googleapis.com
```

### 2.2. Create Required Resources

#### Create Firestore Database

```bash
# Create Firestore database in Native mode
gcloud firestore databases create --region=$REGION
```

#### Create Cloud Storage Bucket

```bash
# Create bucket for Warren storage
export STORAGE_BUCKET="${PROJECT_ID}-warren-storage"
gsutil mb gs://$STORAGE_BUCKET
```

#### Create Artifact Registry Repository

```bash
# Create Docker repository for Warren images
gcloud artifacts repositories create warren \
    --repository-format=docker \
    --location=$REGION \
    --description="Warren Docker images"
```

## 3. Policy Configuration

Warren uses Rego policies to detect and process security alerts. Create your policy files:

### 3.1. Create Policy Directory Structure

```bash
mkdir -p policies/alert
```

### 3.2. Sample Alert Detection Policy

Create `policies/alert/guardduty.rego`:

```rego
package alert.guardduty

# Main rule to detect alerts
alert contains {
    "title": sprintf("GuardDuty Alert: %s", [input.Findings[0].Title]),
    "description": input.Findings[0].Description,
    "attrs": [
        {
            "key": "severity",
            "value": sprintf("%.1f", [input.Findings[0].Severity]),
            "link": ""
        },
        {
            "key": "type",
            "value": input.Findings[0].Type,
            "link": ""
        },
        {
            "key": "region",
            "value": input.Findings[0].Region,
            "link": ""
        }
    ]
} if {
    # Only process findings that are not ignored
    not ignore
    # Ensure we have at least one finding
    count(input.Findings) > 0
    # Filter out informational findings
    input.Findings[0].Severity > 2.0
}

# Ignore rule for specific finding types
ignore if {
    # Skip S3 server access logging alerts
    input.Findings[0].Type == "Stealth:S3/ServerAccessLoggingDisabled"
}

ignore if {
    # Skip low severity findings during business hours
    input.Findings[0].Severity <= 4.0
    # Add time-based logic here if needed
}
```

### 3.3. Sample Auth Policy (Optional)

Create `policies/auth.rego` for API authorization:

```rego
package auth

default allow = false

# Allow all authenticated users
allow = true if {
    input.iap.email
}

# Allow specific service accounts
allow = true if {
    input.env.WARREN_SERVICE_ACCOUNT
    input.google.service_account == input.env.WARREN_SERVICE_ACCOUNT
}
```

### 3.4. BigQuery Configuration (Optional)

If you plan to use BigQuery integration, create `bigquery-config.yml`:

```yaml
dataset_id: security_logs
table_id: events
description: "Security event logs for threat analysis"
columns:
  - name: timestamp
    description: "Event timestamp"
    value_example: "2024-03-20T12:00:00Z"
    type: TIMESTAMP
  - name: source_ip
    description: "Source IP address"
    value_example: "192.168.1.1"
    type: STRING
  - name: user_id
    description: "User identifier"
    value_example: "user123"
    type: STRING
  - name: event_type
    description: "Type of security event"
    value_example: "login_failure"
    type: STRING
  - name: metadata
    description: "Additional event metadata"
    type: RECORD
    fields:
      - name: user_agent
        description: "User agent string"
        value_example: "Mozilla/5.0..."
        type: STRING
      - name: risk_score
        description: "Calculated risk score"
        value_example: "75"
        type: INTEGER
partitioning:
  field: "timestamp"
  type: "time"
  time_unit: "daily"
```

### 3.5. SQL Runbooks (Optional)

Create SQL runbook files for automated analysis. Create `runbooks/suspicious_logins.sql`:

```sql
-- Title: Detect Suspicious Login Patterns
-- Description: Identifies login attempts from unusual locations or patterns
-- that might indicate potential security threats or compromised accounts.

SELECT 
    user_id,
    source_ip,
    COUNT(*) as attempt_count,
    MIN(timestamp) as first_attempt,
    MAX(timestamp) as last_attempt,
    STRING_AGG(DISTINCT user_agent LIMIT 3) as user_agents
FROM security_logs.events
WHERE event_type = 'login_attempt'
  AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
  AND (
    -- Multiple failed attempts from same IP
    (event_result = 'failure' AND source_ip IN (
      SELECT source_ip 
      FROM security_logs.events 
      WHERE event_type = 'login_attempt' 
        AND event_result = 'failure'
        AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
      GROUP BY source_ip 
      HAVING COUNT(*) >= 5
    ))
    OR 
    -- Login from unusual geolocation
    (country_code NOT IN ('US', 'CA', 'GB'))
  )
GROUP BY user_id, source_ip
ORDER BY attempt_count DESC, last_attempt DESC
LIMIT 100;
```

## 4. Docker Image Creation

### 4.1. Create Custom Dockerfile

Create a `Dockerfile` that extends the Warren base image:

```dockerfile
# Use Warren base image
FROM ghcr.io/secmon-lab/warren:latest

# Switch to root to copy files
USER root

# Copy your policies and configurations
COPY policies/ /app/policies/
COPY bigquery-config.yml /app/bigquery-config.yml
COPY runbooks/ /app/runbooks/

# Copy any additional configuration files
COPY config/ /app/config/

# Set proper ownership
RUN chown -R nonroot:nonroot /app/policies /app/config /app/runbooks /app/bigquery-config.yml

# Switch back to nonroot user
USER nonroot

# Set default command
ENTRYPOINT ["/warren"]
CMD ["serve", \
     "--policy", "/app/policies", \
     "--bigquery-config", "/app/bigquery-config.yml", \
     "--bigquery-runbook-path", "/app/runbooks"]
```

### 4.2. Build and Push Image

```bash
# Configure Docker for Artifact Registry
gcloud auth configure-docker ${REGION}-docker.pkg.dev

# Build the image
export IMAGE_TAG="${REGION}-docker.pkg.dev/${PROJECT_ID}/warren/warren:latest"
docker build -t $IMAGE_TAG .

# Push to Artifact Registry
docker push $IMAGE_TAG
```

## 5. Secret Management

### 5.1. Create Secrets in Secret Manager

```bash
# Slack OAuth Token
echo -n "xoxb-your-slack-bot-token" | gcloud secrets create slack-oauth-token --data-file=-

# Slack Signing Secret
echo -n "your-slack-signing-secret" | gcloud secrets create slack-signing-secret --data-file=-

# Slack Client ID
echo -n "your-slack-client-id" | gcloud secrets create slack-client-id --data-file=-

# Slack Client Secret
echo -n "your-slack-client-secret" | gcloud secrets create slack-client-secret --data-file=-

# Optional: External service API keys
echo -n "your-otx-api-key" | gcloud secrets create otx-api-key --data-file=-
echo -n "your-urlscan-api-key" | gcloud secrets create urlscan-api-key --data-file=-
echo -n "your-vt-api-key" | gcloud secrets create vt-api-key --data-file=-
```

### 5.2. Create Service Account

```bash
# Create service account for Warren
gcloud iam service-accounts create warren-service \
    --description="Warren Security Bot Service Account" \
    --display-name="Warren Service"

# Grant necessary permissions
export SERVICE_ACCOUNT="warren-service@${PROJECT_ID}.iam.gserviceaccount.com"

# Firestore permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/datastore.user"

# Storage permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/storage.objectAdmin"

# Secret Manager permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/secretmanager.secretAccessor"

# Vertex AI permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/aiplatform.user"

# BigQuery permissions (if using BigQuery integration)
gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/bigquery.jobUser"

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member="serviceAccount:$SERVICE_ACCOUNT" \
    --role="roles/bigquery.dataViewer"
```

## 6. Deploy to Cloud Run

### 6.1. Create deployment script

Create `deploy.sh`:

```bash
#!/bin/bash

export PROJECT_ID="your-warren-project"
export REGION="us-central1"
export SERVICE_NAME="warren"
export IMAGE_TAG="${REGION}-docker.pkg.dev/${PROJECT_ID}/warren/warren:latest"
export SERVICE_ACCOUNT="warren-service@${PROJECT_ID}.iam.gserviceaccount.com"
export STORAGE_BUCKET="${PROJECT_ID}-warren-storage"

gcloud run deploy $SERVICE_NAME \
    --image=$IMAGE_TAG \
    --region=$REGION \
    --service-account=$SERVICE_ACCOUNT \
    --platform=managed \
    --allow-unauthenticated \
    --memory=2Gi \
    --cpu=2 \
    --concurrency=80 \
    --max-instances=10 \
    --timeout=300 \
    --port=8080 \
    --set-env-vars="WARREN_GEMINI_PROJECT_ID=${PROJECT_ID}" \
    --set-env-vars="WARREN_GEMINI_LOCATION=${REGION}" \
    --set-env-vars="WARREN_FIRESTORE_PROJECT_ID=${PROJECT_ID}" \
    --set-env-vars="WARREN_STORAGE_BUCKET=${STORAGE_BUCKET}" \
    --set-env-vars="WARREN_STORAGE_PROJECT_ID=${PROJECT_ID}" \
    --set-env-vars="WARREN_SLACK_CHANNEL_NAME=security-alerts" \
    --set-env-vars="WARREN_FRONTEND_URL=https://${SERVICE_NAME}-${PROJECT_ID}.a.run.app" \
    --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-oauth-token:latest" \
    --set-secrets="WARREN_SLACK_SIGNING_SECRET=slack-signing-secret:latest" \
    --set-secrets="WARREN_SLACK_CLIENT_ID=slack-client-id:latest" \
    --set-secrets="WARREN_SLACK_CLIENT_SECRET=slack-client-secret:latest" \
    --set-secrets="WARREN_OTX_API_KEY=otx-api-key:latest" \
    --set-secrets="WARREN_URLSCAN_API_KEY=urlscan-api-key:latest" \
    --set-secrets="WARREN_VT_API_KEY=vt-api-key:latest"
```

### 6.2. Execute Deployment

```bash
chmod +x deploy.sh
./deploy.sh
```

### 6.3. Get Service URL

```bash
export SERVICE_URL=$(gcloud run services describe warren --region=$REGION --format='value(status.url)')
echo "Warren is deployed at: $SERVICE_URL"
```

## 7. Configure Slack Webhooks

### 7.1. Set Event Subscriptions

1. Go to your Slack App settings → "Event Subscriptions"
2. Enable Events and set Request URL to:
   ```
   https://your-warren-service-url/slack/event
   ```
3. Subscribe to Bot Events:
   - `app_mention`
   - `message.channels`

### 7.2. Set Interactive Components

1. Go to "Interactivity & Shortcuts"
2. Enable Interactivity and set Request URL to:
   ```
   https://your-warren-service-url/slack/interaction
   ```

### 7.3. Set OAuth Redirect URLs

1. Go to "OAuth & Permissions" → "Redirect URLs"
2. Add the following Redirect URLs:
   ```
   https://your-warren-service-url/api/auth/callback
   ```
   
   **Important Notes:**
   - Replace `your-warren-service-url` with your actual Cloud Run service URL
   - This URL handles the OAuth callback flow for Web UI authentication
   - The URL must match exactly (including https://) 
   - You can add multiple URLs for different environments (staging, production)
   
   **Example for Cloud Run:**
   ```
   https://warren-your-project-id-a.run.app/api/auth/callback
   ```

## 8. Verify Installation

### 8.1. Health Check

```bash
curl -f $SERVICE_URL/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"query { __typename }"}'
```

### 8.2. Test Slack Integration

1. Invite Warren bot to your Slack channel:
   ```
   /invite @warren
   ```

2. Test basic functionality:
   ```
   @warren help
   ```

### 8.3. Test Alert Processing

Send a test alert to verify policy evaluation:

```bash
curl -X POST "$SERVICE_URL/alert/raw/guardduty" \
  -H "Content-Type: application/json" \
  -d '{
    "Findings": [{
      "Title": "Test Alert",
      "Description": "This is a test alert",
      "Severity": 5.0,
      "Type": "Test:Alert",
      "Region": "us-east-1"
    }]
  }'
```

## 9. Optional: Setup IAP Authentication

For production deployments, consider enabling Identity-Aware Proxy:

```bash
# Enable IAP
gcloud iap web enable --resource-type=backend-services \
    --oauth2-client-id=YOUR_OAUTH_CLIENT_ID \
    --oauth2-client-secret=YOUR_OAUTH_CLIENT_SECRET \
    --service=warren

# Update Cloud Run to require authentication
gcloud run services update warren \
    --region=$REGION \
    --clear-env-vars=WARREN_DEV_USER \
    --ingress=all
```

## 10. Monitoring and Maintenance

### 10.1. View Logs

```bash
# Application logs
gcloud logs read "resource.type=cloud_run_revision AND resource.labels.service_name=warren" \
    --limit=50 \
    --format="table(timestamp,textPayload)"

# Error logs
gcloud logs read "resource.type=cloud_run_revision AND resource.labels.service_name=warren AND severity>=ERROR" \
    --limit=50
```

### 10.2. Update Deployment

```bash
# Build and push new image
docker build -t $IMAGE_TAG .
docker push $IMAGE_TAG

# Deploy new version
gcloud run deploy warren \
    --image=$IMAGE_TAG \
    --region=$REGION
```

## Environment Variables Reference

| Variable | Description | Required | Example |
|----------|-------------|----------|---------|
| `WARREN_GEMINI_PROJECT_ID` | GCP Project ID for Vertex AI | Yes | `my-project` |
| `WARREN_GEMINI_LOCATION` | GCP Location for Vertex AI | Yes | `us-central1` |
| `WARREN_FIRESTORE_PROJECT_ID` | Firestore project ID | Yes | `my-project` |
| `WARREN_SLACK_OAUTH_TOKEN` | Slack Bot OAuth token | Yes | `xoxb-...` |
| `WARREN_SLACK_SIGNING_SECRET` | Slack signing secret | Yes | `abc123...` |
| `WARREN_SLACK_CHANNEL_NAME` | Default Slack channel | Yes | `security-alerts` |
| `WARREN_SLACK_CLIENT_ID` | Slack OAuth Client ID | No | `123456.789` |
| `WARREN_SLACK_CLIENT_SECRET` | Slack OAuth Client Secret | No | `secret123` |
| `WARREN_FRONTEND_URL` | Frontend URL for OAuth | No | `https://warren.example.com` |
| `WARREN_STORAGE_BUCKET` | Cloud Storage bucket | No | `my-warren-storage` |
| `WARREN_STORAGE_PROJECT_ID` | Cloud Storage project ID | No | `my-project` |
| `WARREN_BIGQUERY_PROJECT_ID` | BigQuery project ID | No | `my-project` |
| `WARREN_BIGQUERY_CONFIG` | BigQuery config file path | No | `/app/bigquery-config.yml` |
| `WARREN_OTX_API_KEY` | OTX API key | No | `key123...` |
| `WARREN_URLSCAN_API_KEY` | URLScan API key | No | `key123...` |
| `WARREN_VT_API_KEY` | VirusTotal API key | No | `key123...` |

## Troubleshooting

### Common Issues

1. **Policy Evaluation Errors**
   - Check policy syntax with `warren test --policy /path/to/policies`
   - Verify policy package names match API endpoints

2. **Slack Integration Issues**
   - Verify webhook URLs are accessible from Slack
   - Check OAuth scopes match requirements
   - Ensure signing secret is correct

3. **Permission Issues**
   - Verify service account has required IAM roles
   - Check Secret Manager access permissions

4. **Performance Issues**
   - Increase Cloud Run memory/CPU allocation
   - Monitor Vertex AI quota usage
   - Check Firestore read/write limits

For additional support, check the [Warren GitHub repository](https://github.com/secmon-lab/warren) or create an issue.

