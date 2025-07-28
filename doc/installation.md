# Warren Installation Guide

This guide helps you install and configure Warren for different environments, from local development to production deployment. Warren can be configured at different levels depending on your needs.

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

## Configuration Levels

Warren supports three configuration levels, allowing you to start simple and add features as needed:

### Level 1: Local Development (Quick Start)

**Perfect for**: First-time users, demos, local development

**What you get**:
- ✅ Full web UI access
- ✅ AI-powered alert analysis
- ✅ Alert processing pipeline
- ✅ Chat interface with AI agent
- ❌ No data persistence (in-memory only)
- ❌ No team collaboration features

**Requirements**:
- Docker
- Google Cloud account (for Vertex AI)
- No Slack setup required

**Quick Setup**:
Follow our [Getting Started Guide](./getting_started.md) to have Warren running in 5 minutes!

### Level 2: Team Trial

**Perfect for**: Team evaluation, proof of concept

**What you get** (everything from Level 1 plus):
- ✅ Persistent data storage (Firestore)
- ✅ Cloud Storage for attachments (required with Firestore)
- ✅ Slack integration for team notifications
- ✅ User authentication via Slack OAuth
- ✅ Basic security controls

**Additional Requirements**:
- Slack workspace with admin access
- Firestore database setup
- Cloud Storage bucket (automatically used when Firestore is enabled)
- Basic authorization policy (Rego file)

**Quick Setup**:
See [Starting with Level 2](#starting-with-level-2) for detailed upgrade instructions.

### Level 3: Production Ready

**Perfect for**: Production deployments, enterprise environments

**What you get** (everything from Level 2 plus):
- ✅ Secret Manager for secure credential storage
- ✅ Cloud Run deployment with auto-scaling
- ✅ Identity-Aware Proxy for additional security
- ✅ Comprehensive monitoring and logging
- ✅ Policy-based authorization
- ✅ External tool integrations

**Additional Requirements**:
- Cloud Run setup
- IAM configuration
- Production Slack app
- Security policies

**Upgrade from Level 2**:
See [Google Cloud Setup Guide](./installation_gcp.md) for complete production deployment.

## Configuration Checklist

Use this checklist to track your configuration progress:

### Level 1: Local Development
- [ ] Google Cloud account created
- [ ] gcloud CLI installed and authenticated
- [ ] Vertex AI API enabled
- [ ] Docker installed
- [ ] Environment variables configured (WARREN_GEMINI_PROJECT_ID)

### Level 2: Team Trial
- [ ] All Level 1 requirements
- [ ] Slack app created
- [ ] OAuth tokens and secrets configured
- [ ] Firestore database created
- [ ] Cloud Storage bucket created
- [ ] Authorization policy created (basic.rego)
- [ ] Slack channel configured
- [ ] Team members invited to Slack channel

### Level 3: Production Ready
- [ ] All Level 2 requirements
- [ ] Secret Manager configured
- [ ] Cloud Run service deployed
- [ ] Custom domain configured (optional)
- [ ] IAP configured (optional)
- [ ] Monitoring alerts set up
- [ ] Backup strategy implemented
- [ ] Security policies defined
- [ ] External tool API keys configured

## Quick Start for Each Level

### Starting with Level 1

The fastest way to get started:

```bash
# 1. Set up Google Cloud (if not already done)
export PROJECT_ID="your-project-id"
gcloud auth application-default login
gcloud services enable aiplatform.googleapis.com --project=$PROJECT_ID

# 2. Run with Docker
docker run -d \
  --name warren \
  -p 8080:8080 \
  -v ~/.config/gcloud:/home/nonroot/.config/gcloud:ro \
  -e WARREN_GEMINI_PROJECT_ID=$PROJECT_ID \
  -e WARREN_NO_AUTHENTICATION=true \
  -e WARREN_NO_AUTHORIZATION=true \
  -e WARREN_ADDR=0.0.0.0:8080 \
  ghcr.io/secmon-lab/warren:latest serve

# Note: If the image is not available, build locally:
# git clone https://github.com/secmon-lab/warren.git && cd warren
# docker build -t warren:local .
# Then use warren:local instead of ghcr.io/secmon-lab/warren:latest
```

See our [Getting Started Guide](./getting_started.md) for the complete walkthrough.

### Starting with Level 2

To upgrade from Level 1 to Level 2:

```bash
# 1. Create Slack App
# Follow: https://github.com/secmon-lab/warren/blob/main/doc/installation_slack.md
# Get your OAuth token and signing secret

# 2. Enable Firestore and Cloud Storage
gcloud services enable firestore.googleapis.com storage.googleapis.com
gcloud firestore databases create \
  --location=us-central1 \
  --type=firestore-native

# Create Cloud Storage bucket
gsutil mb gs://warren-storage-$PROJECT_ID

# 3. Create authorization policy (required when removing WARREN_NO_AUTHORIZATION)
mkdir -p policies/auth
cat > policies/auth/basic.rego << 'EOF'
package auth

default allow = false

# Allow all requests for now (basic setup)
# In production, implement proper authorization logic based on:
# - input.google (Google OAuth claims)
# - input.iap (Google IAP claims)
# - input.req (HTTP request details)
# - input.env (environment variables)
allow = true
EOF

# For production policies, see: https://github.com/secmon-lab/warren/blob/main/doc/policy.md

# 4. Update your Docker command (now with policy mount)
docker run -d \
  --name warren \
  -p 8080:8080 \
  -v ~/.config/gcloud:/home/nonroot/.config/gcloud:ro \
  -v $(pwd)/policies:/policies:ro \
  -e WARREN_ADDR=0.0.0.0:8080 \
  -e WARREN_POLICY=/policies \
  -e WARREN_GEMINI_PROJECT_ID=$PROJECT_ID \
  -e WARREN_FIRESTORE_PROJECT_ID=$PROJECT_ID \
  -e WARREN_SLACK_OAUTH_TOKEN="xoxb-your-token" \
  -e WARREN_SLACK_SIGNING_SECRET="your-secret" \
  -e WARREN_SLACK_CLIENT_ID="your-client-id" \
  -e WARREN_SLACK_CLIENT_SECRET="your-client-secret" \
  -e WARREN_SLACK_CHANNEL_NAME="security-alerts" \
  -e WARREN_STORAGE_BUCKET="warren-storage-$PROJECT_ID" \
  ghcr.io/secmon-lab/warren:latest serve
```

### Starting with Level 3

For production deployment on Cloud Run:

```bash
# 1. Store secrets securely
echo -n "xoxb-your-token" | gcloud secrets create slack-oauth-token --data-file=-
echo -n "your-signing-secret" | gcloud secrets create slack-signing-secret --data-file=-

# 2. Create Cloud Storage bucket
gsutil mb gs://warren-storage-$PROJECT_ID

# 3. Deploy to Cloud Run
gcloud run deploy warren \
  --image=ghcr.io/secmon-lab/warren:latest \
  --region=us-central1 \
  --set-env-vars="WARREN_FIRESTORE_PROJECT_ID=$PROJECT_ID,WARREN_GEMINI_PROJECT_ID=$PROJECT_ID,WARREN_STORAGE_BUCKET=warren-storage-$PROJECT_ID" \
  --set-secrets="WARREN_SLACK_OAUTH_TOKEN=slack-oauth-token:latest,WARREN_SLACK_SIGNING_SECRET=slack-signing-secret:latest"
```

See [Google Cloud Setup Guide](./installation_gcp.md) for complete production deployment including:
- IAM configuration
- Identity-Aware Proxy setup
- Monitoring and alerting
- Custom domain configuration

## Migration Paths

### Level 1 → Level 2 Migration

1. **Create Slack App**:
   - Go to [api.slack.com/apps](https://api.slack.com/apps)
   - Create new app from manifest
   - Configure OAuth scopes and permissions
   - Install to your workspace

2. **Enable Data Persistence**:
   ```bash
   # Enable Firestore
   gcloud services enable firestore.googleapis.com
   gcloud firestore databases create --location=us-central1
   ```

3. **Update Configuration**:
   - Remove `WARREN_NO_AUTHENTICATION` and `WARREN_NO_AUTHORIZATION`
   - Add Slack credentials
   - Add `WARREN_FIRESTORE_PROJECT_ID`

### Level 2 → Level 3 Migration

1. **Set Up Cloud Infrastructure**:
   - Create Cloud Storage bucket
   - Configure Secret Manager
   - Set up Cloud Run service

2. **Enhance Security**:
   - Configure IAM roles
   - Set up Identity-Aware Proxy
   - Define security policies

3. **Add Production Features**:
   - Configure monitoring
   - Set up backups
   - Add external tool integrations

## Common Configuration Options

### Environment Variables Reference

```bash
# Core Configuration
WARREN_GEMINI_PROJECT_ID      # Required: GCP project for Vertex AI
WARREN_GEMINI_LOCATION        # Default: us-central1

# Authentication (Level 1: disable, Level 2+: configure)
WARREN_NO_AUTHENTICATION      # Set to true for Level 1 only
WARREN_NO_AUTHORIZATION       # Set to true for Level 1 only

# Slack Integration (Level 2+)
WARREN_SLACK_OAUTH_TOKEN      # Bot user OAuth token (xoxb-...)
WARREN_SLACK_SIGNING_SECRET   # For request verification
WARREN_SLACK_CLIENT_ID        # For OAuth flow
WARREN_SLACK_CLIENT_SECRET    # For OAuth flow
WARREN_SLACK_CHANNEL_NAME     # Target channel for alerts

# Storage (Level 2+)
WARREN_FIRESTORE_PROJECT_ID   # For persistent storage
WARREN_STORAGE_BUCKET         # Level 3: For file attachments

# External Tools (Optional)
WARREN_OTX_API_KEY            # AlienVault OTX
WARREN_VIRUSTOTAL_API_KEY     # VirusTotal
WARREN_URLSCAN_API_KEY        # URLScan.io
WARREN_SHODAN_API_KEY         # Shodan
WARREN_ABUSEIPDB_API_KEY      # AbuseIPDB
```

## Prerequisites by Level

### All Levels
- Docker or Docker Desktop
- Google Cloud account
- gcloud CLI installed

### Level 2 Additional
- Slack workspace with admin access
- Ability to create Slack apps

### Level 3 Additional
- Google Cloud billing enabled
- Domain for custom URL (optional)
- SSL certificate (optional)

## Next Steps

- **New to Warren?** Start with our [Getting Started Guide](./getting_started.md) (Level 1)
- **Ready for team use?** Follow the [Slack Configuration Guide](./installation_slack.md) (Level 2)
- **Deploying to production?** See the [Google Cloud Setup Guide](./installation_gcp.md) (Level 3)

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

---

*Warren requires Go 1.24+, Docker 25.0.6+, and Google Cloud SDK 464.0.0+ for development.*