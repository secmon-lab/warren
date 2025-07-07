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

## Documentation

- [Installation](./doc/installation.md)
- [Getting Started](./doc/getting_started.md)

## License

Apache 2.0 License

