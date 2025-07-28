# warren
AI agent and Slack based security alert management tool

<p align="center">
  <img src="./doc/images/logo2.png" height="128" />
</p>

## Concept

Warren is a security alert management platform that combines AI-powered analysis with collaborative incident response. It processes security alerts from various sources, evaluates them using policy-driven detection rules, and facilitates team collaboration through integrated Slack workflows and a modern web interface.

The platform addresses key challenges in security operations:
- **Alert Fatigue**: Reduces noise by grouping similar alerts and using AI to prioritize based on context
- **Manual Triage**: Automates initial analysis using external threat intelligence sources
- **Knowledge Silos**: Centralizes alert investigation and maintains institutional knowledge
- **Response Coordination**: Streamlines team communication through native Slack integration

Warren operates on a ticket-based workflow where security alerts are grouped into manageable tickets, analyzed using AI agents, and tracked through resolution.

## How it works

### Alert Processing Pipeline

1. **Alert Ingestion**: Security alerts arrive via HTTP endpoints from sources like AWS GuardDuty, SIEM systems, or Pub/Sub messaging
2. **Policy Evaluation**: Incoming data is processed through Rego policies that determine whether events qualify as security alerts
3. **Alert Grouping**: Similar alerts are automatically clustered using semantic similarity to reduce duplicate work
4. **Ticket Creation**: Alert groups are organized into tickets for structured investigation and tracking

### AI-Powered Analysis

Warren integrates with Google Vertex AI (Gemini) to provide intelligent analysis capabilities:
- **Metadata Extraction**: Automatically generates titles, descriptions, and key attributes from raw alert data
- **Threat Intelligence**: Queries external services (OTX, VirusTotal, URLScan, Shodan, AbuseIPDB) for indicators of compromise
- **Security Analysis**: Provides context-aware analysis and recommendations based on alert patterns and historical data
- **Interactive Chat**: Enables conversational analysis where security analysts can ask questions about specific tickets

### Collaborative Workflow

- **Slack Integration**: Native bot integration for real-time notifications, interactive buttons, and team discussions
- **Web Dashboard**: React-based interface for ticket management, alert browsing, and investigation tracking
- **GraphQL API**: Flexible API for custom integrations and automation workflows
- **Policy Management**: Version-controlled Rego policies for customizable alert detection rules

### External Integrations

Warren connects to multiple security intelligence sources:
- **Threat Intelligence**: AlienVault OTX, VirusTotal
- **URL Analysis**: URLScan.io
- **IP Reputation**: AbuseIPDB, Shodan
- **Malware Analysis**: abuse.ch MalwareBazaar
- **Security Analytics**: Google BigQuery for data analysis and pattern detection

### Architecture

The system follows clean architecture principles with clear separation between:
- **Domain Layer**: Core business logic for alerts, tickets, and policies
- **Service Layer**: Application services coordinating business operations
- **Interface Layer**: HTTP/GraphQL APIs and Slack integration
- **Infrastructure Layer**: Database (Firestore), storage (Cloud Storage), and external service adapters

This design ensures maintainability, testability, and flexibility for different deployment environments.

## Quick Start

Get Warren running in 5 minutes with Docker:

```bash
# Set up Google Cloud authentication
export PROJECT_ID="your-gcp-project"
gcloud auth application-default login
gcloud services enable aiplatform.googleapis.com --project=$PROJECT_ID

# Run Warren with Docker
docker run -d \
  --name warren \
  -p 8080:8080 \
  -v ~/.config/gcloud:/home/nonroot/.config/gcloud:ro \
  -e WARREN_GEMINI_PROJECT_ID=$PROJECT_ID \
  -e WARREN_NO_AUTHENTICATION=true \
  -e WARREN_NO_AUTHORIZATION=true \
  -e WARREN_ADDR=0.0.0.0:8080 \
  ghcr.io/secmon-lab/warren:latest serve

# Or build locally if the image is not available:
# git clone https://github.com/secmon-lab/warren.git && cd warren
# docker build -t warren:local .
# Then use warren:local instead

# Send a test alert
curl -X POST http://localhost:8080/hooks/alert/raw/test \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Alert", "description": "Testing Warren", "severity": "high"}'
```

Visit http://localhost:8080 to see your alerts!

## Documentation

- [Getting Started Guide](./doc/getting_started.md) - 5-minute quick start with Docker
- [Installation Guide](./doc/installation.md) - From local development to production deployment
- [User Guide](./doc/user_guide.md) - How to use Warren effectively
- [Configuration Reference](./doc/configuration.md) - All configuration options
- [Policy Guide](./doc/policy.md) - Writing alert detection policies

## License

Apache 2.0 License

