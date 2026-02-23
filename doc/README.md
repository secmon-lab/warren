# Warren Documentation

## Overview

Warren is an AI-powered security alert management platform that streamlines incident response through intelligent alert processing, collaborative workflows, and seamless integrations. This documentation provides comprehensive guidance for security analysts and system administrators to effectively deploy and use Warren.

## Documentation Structure

| Document | Description | Target Audience |
|----------|-------------|----------------|
| [Getting Started](./getting_started.md) | Quick introduction and first steps with Warren | New users, Security analysts |
| [Installation Overview](./installation.md) | Deployment options and setup overview | System administrators |
| [Installation - Google Cloud](./installation_gcp.md) | Detailed Google Cloud setup | System administrators |
| [Installation - Slack](./installation_slack.md) | Slack integration configuration | System administrators |
| [Configuration Reference](./configuration.md) | Complete reference for all environment variables and CLI flags | System administrators |
| [Data Models](./model.md) | Understanding Alerts, Tickets, and core concepts | All users |
| [User Guide](./user_guide.md) | Daily operations and workflow management | Security analysts |
| [AI Agent Guide](./agent.md) | Using Chat features and AI-powered analysis | Security analysts |
| [Knowledge Management](./knowledge.md) | Managing organizational security knowledge | All users |
| [Integration Guide](./integration.md) | API reference and external system integration | System administrators |
| [Policy Guide](./policy.md) | Writing and managing Rego policies | System administrators |
| [Experimental Features](./experimental.md) | Experimental features (Refine) | All users |

## Quick Start Paths

### For Security Analysts
1. **[Getting Started](./getting_started.md)** - Understand Warren's value and basic concepts
2. **[Data Models](./model.md)** - Learn about Alerts and Tickets
3. **[User Guide](./user_guide.md)** - Master daily operations
4. **[AI Agent Guide](./agent.md)** - Leverage AI for investigations
5. **[Knowledge Management](./knowledge.md)** - Capture and use organizational expertise

### For System Administrators
1. **[Getting Started](./getting_started.md)** - Overview of Warren's architecture
2. **[Installation Overview](./installation.md)** - Choose deployment method
3. **[Google Cloud Setup](./installation_gcp.md)** + **[Slack Setup](./installation_slack.md)** - Configure infrastructure
4. **[Configuration Reference](./configuration.md)** - Configure Warren's settings
5. **[Policy Guide](./policy.md)** - Customize alert detection
6. **[Integration Guide](./integration.md)** - Connect external systems

## Key Concepts

- **Alert**: A security event that requires attention
- **Ticket**: A container for related alerts that tracks investigation progress
- **Policy**: Rego rules that determine which events become alerts
- **Clustering**: AI-powered grouping of similar alerts
- **Agent**: AI assistant for security analysis and investigation
- **Knowledge**: Organizational security expertise that informs AI analysis

## Finding Help

- **GitHub Issues**: [Report bugs or request features](https://github.com/secmon-lab/warren/issues)
- **Discussions**: [Ask questions and share experiences](https://github.com/secmon-lab/warren/discussions)
- **Security**: For security concerns, please see our [Security Policy](https://github.com/secmon-lab/warren/security/policy)

## Documentation Versions

This documentation is for Warren v1.0+. For the latest updates and version-specific information, please check the [GitHub releases](https://github.com/secmon-lab/warren/releases).