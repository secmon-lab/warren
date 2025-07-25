# Warren Installation Overview

Warren is an AI-powered security alert management tool that integrates with Slack and Google Cloud services. This guide provides an overview of the installation process with links to detailed guides for each component.

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

## Installation Options

Warren can be deployed in several ways:

### 1. **Cloud Run Deployment** (Recommended)
- Fully managed serverless platform
- Automatic scaling and high availability
- Integrated with Google Cloud services
- See [Google Cloud Setup Guide](./installation_gcp.md) for details

### 2. **Kubernetes Deployment**
- Deploy to any Kubernetes cluster (GKE, EKS, AKS, or self-managed)
- Full control over deployment configuration
- Suitable for organizations with existing K8s infrastructure
- Helm charts available in the repository

## Minimal Viable Setup

For users who want to quickly try Warren, this section provides the minimum configuration needed to get started.

> **Important Note**: Warren is designed as a cloud-native application and requires both Google Cloud services (Firestore and Vertex AI) and Slack integration to run the `serve` command. The full server cannot run without these dependencies.

### Option 1: Policy Testing Only (No External Services Required)

If you only want to test alert detection policies without running the full server:

```bash
# Clone the repository
git clone https://github.com/secmon-lab/warren.git
cd warren

# Create test data structure
mkdir -p test-data/guardduty
echo '{
  "source": "aws.guardduty",
  "detail": {
    "type": "Recon:EC2/PortProbeUnprotectedPort",
    "severity": 5.0,
    "accountId": "123456789012",
    "region": "us-east-1"
  }
}' > test-data/guardduty/sample.json

# Test the policy
go run main.go test \
  --policy ./policies \
  --test-detect-data ./test-data
```

This allows you to:
- ✅ Test and develop Rego policies locally
- ✅ Validate alert detection logic
- ✅ No external services required
- ✅ CI/CD friendly for policy validation
- ❌ No web UI or API access
- ❌ No alert processing pipeline
- ❌ No Slack notifications

### Option 2: Minimal Setup with Google Cloud and Slack

Warren requires both Google Cloud services and Slack integration to run. Here's the minimal setup:

```bash
# 1. Create a new project
export PROJECT_ID=warren-test-$(date +%s)
gcloud projects create $PROJECT_ID
gcloud config set project $PROJECT_ID

# 2. Enable billing (required for Firestore and Vertex AI)
# Visit: https://console.cloud.google.com/billing

# 3. Enable required APIs
gcloud services enable \
  firestore.googleapis.com \
  aiplatform.googleapis.com

# 4. Create Firestore database
gcloud firestore databases create \
  --location=us-central1 \
  --type=firestore-native

# 5. Create a minimal Slack app
# Follow the Quick Setup section in installation_slack.md to get:
# - WARREN_SLACK_OAUTH_TOKEN (xoxb-...)
# - WARREN_SLACK_SIGNING_SECRET

# 6. Run Warren locally
export WARREN_FIRESTORE_PROJECT_ID=$PROJECT_ID
export WARREN_GEMINI_PROJECT_ID=$PROJECT_ID
export WARREN_GEMINI_LOCATION=us-central1
export WARREN_SLACK_OAUTH_TOKEN="xoxb-your-token"
export WARREN_SLACK_SIGNING_SECRET="your-signing-secret"
export WARREN_SLACK_CHANNEL_NAME="test-alerts"
export WARREN_LOG_LEVEL=debug

go run main.go serve --addr=:8080
```

This gives you:
- ✅ Web UI access at http://localhost:8080
- ✅ Alert processing pipeline  
- ✅ AI-powered alert enrichment
- ✅ Persistent storage
- ✅ Basic Slack integration (required)
- ✅ Slack OAuth authentication
- ❌ No external tool integrations (optional)

### Option 3: Minimal Cloud Run Deployment

Deploy Warren on Cloud Run with minimal configuration:

```bash
# Use the project from Option 2
# First, store Slack credentials in Secret Manager
echo -n "xoxb-your-token" | gcloud secrets create slack-oauth-token --data-file=-
echo -n "your-signing-secret" | gcloud secrets create slack-signing-secret --data-file=-

# Deploy Warren
gcloud run deploy warren \
  --image=ghcr.io/secmon-lab/warren:latest \
  --region=us-central1 \
  --set-env-vars="WARREN_FIRESTORE_PROJECT_ID=$PROJECT_ID,WARREN_GEMINI_PROJECT_ID=$PROJECT_ID,WARREN_SLACK_CHANNEL_NAME=test-alerts" \
  --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-oauth-token:latest,WARREN_SLACK_SIGNING_SECRET=slack-signing-secret:latest"
```

After deployment:
- ✅ Public URL for web access
- ✅ Full alert processing pipeline
- ✅ AI-powered features
- ✅ Slack integration
- ✅ Scalable and managed
- ⚠️ Requires authentication - Configure IAP or use Slack OAuth

### Testing Your Minimal Setup

1. **Send a test alert**:
```bash
curl -X POST http://localhost:8080/hooks/alert/raw/test \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Test Security Alert",
    "description": "This is a test alert",
    "severity": "medium"
  }'
```

2. **Access the Web UI**:
- Navigate to http://localhost:8080
- You'll be automatically logged in as the dev user
- View alerts and create tickets

3. **Test GraphQL API**:
```bash
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ unboundAlerts(limit: 10) { alerts { id title } } }"}'
```

### Upgrading to Full Setup

When ready for production features:

1. **Add Slack Integration**: Follow [Slack Configuration Guide](./installation_slack.md)
2. **Enable AI Features**: Set up Vertex AI per [Google Cloud Setup](./installation_gcp.md#7-vertex-ai-setup)
3. **Configure Authentication**: Set up proper OAuth with Slack or Google IAP
4. **Configure External Tools**: Add API keys for threat intelligence

## Prerequisites

Before starting the installation, ensure you have:

- **Development Tools**:
  - Go 1.23.4 or later (for local development)
  - Docker 25.0.6 or later (for building images)
  - Google Cloud SDK 464.0.0 or later
  - Google Cloud CLI 464.0.0 or later

- **Access Requirements**:
  - Google Cloud Project with billing enabled
  - Slack workspace with admin permissions
  - GitHub account (for accessing Warren repository)

## Quick Start

> **Looking for the simplest setup?** Jump to [Minimal Viable Setup](#minimal-viable-setup) for local development options.

For a production Cloud Run deployment:

1. **Set up Slack Integration**
   ```bash
   # Follow the Slack setup guide to create your Slack app
   # and collect required credentials
   ```
   → See [Slack Configuration Guide](./installation_slack.md)

2. **Configure Google Cloud**
   ```bash
   # Set project and enable APIs
   export PROJECT_ID="your-project-id"
   gcloud config set project $PROJECT_ID
   
   # Enable essential services
   gcloud services enable \
     run.googleapis.com \
     firestore.googleapis.com \
     aiplatform.googleapis.com
   ```
   → See [Google Cloud Setup Guide](./installation_gcp.md) for complete setup

3. **Deploy Warren**
   ```bash
   # Deploy using pre-built image
   gcloud run deploy warren \
     --image=ghcr.io/secmon-lab/warren:latest \
     --region=us-central1 \
     --set-env-vars="WARREN_SLACK_CHANNEL_NAME=security-alerts" \
     --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-token:latest"
   ```

## Detailed Installation Guides

### 📱 [Slack Configuration](./installation_slack.md)
Complete guide for setting up Slack integration:
- Creating and configuring a Slack app
- Setting up OAuth scopes and permissions
- Configuring event subscriptions and interactivity
- Testing the Slack integration

### ☁️ [Google Cloud Setup](./installation_gcp.md)
Comprehensive Google Cloud configuration:
- Project setup and API enablement
- Firestore, Storage, and Vertex AI configuration
- Service account and IAM permissions
- Secret Manager setup
- Cloud Run deployment

### 🛠️ Advanced Configuration
Additional configuration options are covered in these guides:
- **All configuration options** - See [Configuration Reference](./configuration.md) for complete list
- **Custom Docker image creation** - See [Google Cloud Setup](./installation_gcp.md#42-build-and-push-image)
- **Policy configuration and testing** - See [Policy Guide](./policy.md)
- **External tool integration** - See [Integration Guide](./integration.md#external-tool-integration)
- **BigQuery setup** - See [Google Cloud Setup](./installation_gcp.md#10-optional-bigquery-setup)
- **Monitoring and logging** - See [Google Cloud Setup](./installation_gcp.md#11-monitoring-and-logging)

## Verification

After installation, verify your deployment:

1. **Health Check**
   ```bash
   curl https://your-warren-url/graphql \
     -H "Content-Type: application/json" \
     -d '{"query":"query { __typename }"}'
   ```

2. **Slack Integration**
   - Invite Warren bot to your channel: `/invite @warren`
   - Test basic functionality: `@warren help`

3. **Web UI Access**
   - Navigate to your Warren URL
   - Log in with Slack OAuth
   - Verify dashboard loads correctly

## Next Steps

1. **Configure Policies**: Set up alert detection rules in [Policy Guide](./policy.md)
2. **Test Alert Processing**: Send test alerts to verify the pipeline
3. **Set Up Integrations**: Connect external security tools
4. **Train Your Team**: Review the [User Guide](./user_guide.md)

## Troubleshooting

Common issues and solutions:

- **Slack bot not responding**: Check OAuth token and channel permissions
- **Authentication errors**: Verify Slack OAuth redirect URLs
- **No alerts appearing**: Check policy configuration and logs
- **Performance issues**: Review Cloud Run scaling settings

For detailed troubleshooting, check the logs:
```bash
gcloud logs read "resource.type=cloud_run_revision AND resource.labels.service_name=warren" --limit=50
```

## Support

- **Documentation**: You're reading it! Check other guides for specific topics
- **Issues**: [GitHub Issues](https://github.com/secmon-lab/warren/issues)
- **Discussions**: [GitHub Discussions](https://github.com/secmon-lab/warren/discussions)