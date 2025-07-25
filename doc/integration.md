# Integration Guide

Warren provides multiple integration points for receiving security alerts, querying data, and connecting with external security tools. This guide covers webhook endpoints, GraphQL API, and tool integrations.

## Overview

Warren acts as a central hub for security alert management with:
- **Webhook endpoints** for receiving alerts from various sources
- **GraphQL API** for flexible data queries and mutations
- **External tool integrations** for threat intelligence
- **Policy-based authorization** for secure access control

## Webhook Endpoints

### Purpose and Architecture

Webhook endpoints are Warren's primary mechanism for receiving security alerts from external systems. They serve as the entry point for the entire alert processing pipeline:

1. **Alert Collection**: Gather security events from diverse sources (cloud providers, SIEMs, monitoring tools)
2. **Normalization**: Transform various alert formats into Warren's standardized model
3. **Policy Evaluation**: Apply Rego policies to determine if events qualify as alerts
4. **Enrichment**: Enhance alerts with AI-generated metadata and threat intelligence
5. **Distribution**: Route alerts to Slack and create tickets for investigation

All webhook endpoints use policy-based authorization through the `auth` package, providing flexible access control. This ensures that only authorized systems can submit alerts while maintaining security.

### Alert Ingestion Endpoints

Warren provides three types of webhook endpoints, each designed for specific integration patterns:

#### 1. Raw Alert Endpoint

**Purpose**: Direct integration with security tools that can send HTTP requests. This is the simplest integration method for custom tools, scripts, or systems that can format their alerts as JSON.

**Use Cases**:
- Custom security scripts and tools
- Direct integration from security appliances
- Development and testing
- Simple one-off integrations

**Endpoint**: `/hooks/alert/raw/{schema}`  
**Method**: `POST`  
**Content-Type**: `application/json`

Example request:
```bash
curl -X POST https://your-warren-instance/hooks/alert/raw/custom \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "title": "Suspicious Network Activity",
    "description": "Unusual outbound connections detected",
    "severity": "high",
    "source_ip": "192.168.1.100",
    "destination_port": 4444,
    "timestamp": "2025-01-07T10:30:00Z"
  }'
```

**Response**: HTTP 200 OK (empty body on success)

The `{schema}` parameter determines which policy will process the alert. Common schemas:
- `guardduty` - AWS GuardDuty alerts
- `sentry` - Application error monitoring
- `custom` - Custom security tools

#### 2. Google Pub/Sub Endpoint

**Purpose**: Asynchronous, reliable alert delivery from Google Cloud services. Pub/Sub provides guaranteed delivery, automatic retries, and decoupling between alert producers and Warren.

**Use Cases**:
- Google Cloud Security Command Center alerts
- Cloud Logging export for security events
- Integration with Google Cloud security tools
- High-volume alert processing with buffering
- Multi-region alert collection

**Benefits**:
- Automatic retry and dead-letter handling
- Scalable message buffering
- Decoupled architecture
- Built-in authentication via Google ID tokens

**Endpoint**: `/hooks/alert/pubsub/{schema}`  
**Method**: `POST`  
**Authentication**: Google ID token in Authorization header

Pub/Sub message format:
```json
{
  "message": {
    "data": "base64_encoded_alert_json",
    "messageId": "1234567890",
    "publishTime": "2025-01-07T10:30:00Z"
  }
}
```

Setup instructions:
1. Create a Pub/Sub topic for alerts
2. Create a push subscription with URL: `https://your-warren/hooks/alert/pubsub/your-schema`
3. Grant Warren's service account the `roles/pubsub.subscriber` role

Example using gcloud:
```bash
# Create topic
gcloud pubsub topics create warren-alerts

# Create push subscription
gcloud pubsub subscriptions create warren-alerts-sub \
  --topic=warren-alerts \
  --push-endpoint=https://your-warren/hooks/alert/pubsub/guardduty \
  --push-auth-service-account=warren-sa@project.iam.gserviceaccount.com
```

#### 3. AWS SNS Endpoint

**Purpose**: Native integration with AWS security services and event-driven architectures. SNS provides fan-out capabilities, allowing multiple systems to receive the same alerts while Warren processes them for investigation.

**Use Cases**:
- AWS GuardDuty findings
- AWS Security Hub alerts
- CloudWatch alarms for security events
- AWS Config compliance violations
- Custom AWS Lambda security functions
- Multi-account security event aggregation

**Benefits**:
- Native AWS service integration
- Cross-region event delivery
- Multiple subscriber support (Warren + other tools)
- Built-in message verification
- Automatic subscription confirmation

**Security Features**:
- Cryptographic signature verification
- Certificate validation
- Message integrity checking
- Replay attack prevention

**Endpoint**: `/hooks/alert/sns/{schema}`  
**Method**: `POST`  
**Authentication**: SNS signature verification

Warren automatically:
- Verifies SNS message signatures for security
- Handles subscription confirmations transparently
- Extracts and processes alert data from SNS envelope
- Validates message timestamps to prevent replay attacks

SNS topic configuration:
1. Create SNS topic in AWS
2. Add HTTPS subscription with Warren endpoint
3. Confirm subscription (automatic)

Example SNS message:
```json
{
  "Type": "Notification",
  "MessageId": "12345",
  "TopicArn": "arn:aws:sns:us-east-1:123456789012:alerts",
  "Message": "{\"alert\":\"data\"}",
  "Timestamp": "2025-01-07T10:30:00.000Z",
  "SignatureVersion": "1",
  "Signature": "...",
  "SigningCertURL": "..."
}
```

### How Webhook Processing Works

When an alert arrives at any webhook endpoint:

1. **Authentication & Authorization**
   - Verify request authenticity (signatures, tokens)
   - Check authorization policies
   
2. **Schema Resolution**
   - The `{schema}` parameter determines which Rego policy to use
   - Example: `/hooks/alert/raw/guardduty` uses `alert.guardduty` policy

3. **Policy Evaluation**
   - Raw data is passed to the Rego policy as `input`
   - Policy decides if the event should create an alert
   - Policy extracts and formats metadata

4. **Alert Creation**
   - If policy returns alert data, Warren creates an Alert entity
   - AI generates embeddings for similarity analysis
   - Alert is stored in Firestore

5. **Notification & Clustering**
   - Alert posted to Slack if unbound
   - Clustering algorithms group similar alerts
   - Analysts can create tickets from alerts

### Custom Alert Schemas

Define custom schemas using Rego policies to handle any alert format:

1. Create policy file: `policies/alert/myservice.rego`
```rego
package alert.myservice

alert contains {
    "title": input.alert_title,
    "description": input.alert_description,
    "attrs": [
        {
            "key": "severity",
            "value": input.severity,
            "link": ""
        },
        {
            "key": "service",
            "value": input.service_name,
            "link": ""
        }
    ]
} if {
    input.severity in ["low", "medium", "high", "critical"]
}
```

2. Send alerts to `/hooks/alert/raw/myservice`

## GraphQL API

Warren provides a comprehensive GraphQL API for querying and managing security data.

### Endpoint

**URL**: `https://your-warren-instance/graphql`  
**Method**: `POST`  
**Content-Type**: `application/json`

**GraphiQL Interface**: `https://your-warren-instance/graphiql` (if enabled)

### Authentication

Include authentication cookies or headers:
```bash
curl -X POST https://your-warren/graphql \
  -H "Content-Type: application/json" \
  -H "Cookie: token_id=...; token_secret=..." \
  -d '{"query": "{ __typename }"}'
```

### Schema Overview

#### Core Types

```graphql
type Ticket {
  id: ID!
  title: String!
  description: String!
  status: TicketStatus!
  conclusion: AlertConclusion
  reason: String
  alerts: [Alert!]!
  assignee: User
  createdAt: String!
  updatedAt: String!
  slackThread: SlackThread
  comments: [Comment!]!
  finding: Finding
  isTest: Boolean!
}

type Alert {
  id: ID!
  title: String!
  description: String!
  schema: String!
  data: JSON!
  attributes: [Attribute!]!
  createdAt: String!
  ticketId: ID
}

type AlertCluster {
  id: String!
  size: Int!
  keywords: [String!]!
  centerAlertId: ID!
  centerAlert: Alert!
  alerts: [Alert!]!
}
```

### Common Queries

#### Get Ticket Details
```graphql
query GetTicket($id: ID!) {
  ticket(id: $id) {
    id
    title
    description
    status
    conclusion
    alerts {
      id
      title
      schema
      createdAt
    }
    assignee {
      id
      name
      email
    }
    comments {
      id
      content
      author {
        name
      }
      createdAt
    }
  }
}
```

#### List Unbound Alerts
```graphql
query UnboundAlerts($limit: Int, $offset: Int) {
  unboundAlerts(limit: $limit, offset: $offset) {
    alerts {
      id
      title
      description
      schema
      createdAt
    }
    hasMore
    totalCount
  }
}
```

#### Find Similar Tickets
```graphql
query SimilarTickets($ticketId: ID!, $threshold: Float!) {
  similarTickets(
    ticketId: $ticketId
    threshold: $threshold
    limit: 10
  ) {
    tickets {
      id
      title
      status
      similarity
    }
  }
}
```

#### Get Alert Clusters
```graphql
query GetClusters($eps: Float, $minSamples: Int) {
  alertClusters(
    eps: $eps
    minSamples: $minSamples
    limit: 50
  ) {
    clusters {
      id
      size
      keywords
      centerAlert {
        title
      }
    }
    totalAlerts
    noise
  }
}
```

### Common Mutations

#### Create Ticket from Alerts
```graphql
mutation CreateTicket($alertIds: [ID!]!, $title: String, $description: String) {
  createTicketFromAlerts(
    alertIds: $alertIds
    title: $title
    description: $description
  ) {
    id
    title
    alerts {
      id
    }
  }
}
```

#### Update Ticket Status
```graphql
mutation UpdateStatus($id: ID!, $status: String!) {
  updateTicketStatus(id: $id, status: $status) {
    id
    status
    updatedAt
  }
}
```

#### Resolve Ticket
```graphql
mutation ResolveTicket($id: ID!, $conclusion: String!, $reason: String!) {
  updateTicketConclusion(
    id: $id
    conclusion: $conclusion
    reason: $reason
  ) {
    id
    status
    conclusion
    reason
  }
}
```

#### Bind Alerts to Ticket
```graphql
mutation BindAlerts($ticketId: ID!, $alertIds: [ID!]!) {
  bindAlertsToTicket(
    ticketId: $ticketId
    alertIds: $alertIds
  ) {
    id
    alerts {
      id
    }
  }
}
```

### Code Examples

#### JavaScript/TypeScript
```typescript
import { GraphQLClient } from 'graphql-request';

const client = new GraphQLClient('https://your-warren/graphql', {
  credentials: 'include',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Query example
const query = `
  query GetOpenTickets {
    tickets(statuses: ["open"], limit: 10) {
      tickets {
        id
        title
        alertCount
      }
    }
  }
`;

const data = await client.request(query);
console.log(data.tickets);

// Mutation example
const mutation = `
  mutation CreateTicket($alertIds: [ID!]!) {
    createTicketFromAlerts(alertIds: $alertIds) {
      id
      title
    }
  }
`;

const result = await client.request(mutation, {
  alertIds: ['alert-123', 'alert-456']
});
```

#### Python
```python
import requests
import json

url = "https://your-warren/graphql"
headers = {
    "Content-Type": "application/json",
    "Cookie": "token_id=...; token_secret=..."
}

# Query example
query = """
query GetTicket($id: ID!) {
  ticket(id: $id) {
    id
    title
    status
  }
}
"""

response = requests.post(url, json={
    "query": query,
    "variables": {"id": "ticket-12345"}
}, headers=headers)

data = response.json()
print(data["data"]["ticket"])

# Mutation example
mutation = """
mutation UpdateStatus($id: ID!, $status: String!) {
  updateTicketStatus(id: $id, status: $status) {
    id
    status
  }
}
"""

response = requests.post(url, json={
    "query": mutation,
    "variables": {
        "id": "ticket-12345",
        "status": "resolved"
    }
}, headers=headers)
```

#### cURL
```bash
# Query
curl -X POST https://your-warren/graphql \
  -H "Content-Type: application/json" \
  -H "Cookie: token_id=...; token_secret=..." \
  -d '{
    "query": "query { tickets(limit: 5) { tickets { id title } } }"
  }'

# Mutation
curl -X POST https://your-warren/graphql \
  -H "Content-Type: application/json" \
  -H "Cookie: token_id=...; token_secret=..." \
  -d '{
    "query": "mutation($id: ID!) { updateTicketStatus(id: $id, status: \"pending\") { id status } }",
    "variables": {"id": "ticket-12345"}
  }'
```

## External Tool Integration

Warren integrates with various security intelligence tools. Configure API keys through environment variables or command-line flags.

### Configuration

Set API keys using environment variables:
```bash
export WARREN_VT_API_KEY="your-virustotal-key"
export WARREN_OTX_API_KEY="your-otx-key"
export WARREN_URLSCAN_API_KEY="your-urlscan-key"
export WARREN_SHODAN_API_KEY="your-shodan-key"
export WARREN_ABUSEIPDB_API_KEY="your-abuseipdb-key"
```

Or use command-line flags:
```bash
warren serve \
  --vt-api-key="your-virustotal-key" \
  --otx-api-key="your-otx-key" \
  --urlscan-api-key="your-urlscan-key"
```

### Available Integrations

#### VirusTotal
- IP reputation checks
- Domain analysis
- File hash lookups
- URL scanning

#### OTX (Open Threat Exchange)
- Threat intelligence for IPs
- Domain threat data
- File hash intelligence
- URL reputation

#### URLScan
- Website screenshot and analysis
- DOM inspection
- Network request analysis
- Threat detection

#### Shodan
- Internet-wide port scanning data
- Service identification
- Vulnerability information
- Historical data

#### AbuseIPDB
- IP abuse reports
- Confidence scores
- Attack categories
- Geographic data

### Using Tools via Chat

Tools are automatically available in the AI Agent chat:
```
warren chat --ticket-id TICKET_ID

> Check IP 192.168.1.100 with VirusTotal
> Analyze domain example.com using OTX
> Search Shodan for vulnerable services on this IP
```

## Authentication Methods

Warren supports multiple authentication methods:

### 1. Google IAP (Identity-Aware Proxy)
For Google Cloud deployments, validates JWT tokens from IAP headers.

### 2. OAuth with Slack
Web UI authentication flow:
1. User clicks "Sign in with Slack"
2. Redirected to Slack OAuth
3. Warren validates and creates session
4. Cookies set for authentication

### 3. API Token (Custom)
Implement custom token validation in policies:
```rego
package auth

allow = true if {
    input.req.header.Authorization[0] == "Bearer valid-token"
}
```

### 4. Service Account (Google Cloud)
For service-to-service authentication:
```bash
# Get ID token
TOKEN=$(gcloud auth print-identity-token)

# Use in requests
curl -H "Authorization: Bearer $TOKEN" https://your-warren/api/...
```

## Best Practices

### Error Handling

Always check response status and handle errors:
```javascript
try {
  const response = await fetch('/graphql', {
    method: 'POST',
    body: JSON.stringify({ query }),
  });
  
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  
  const data = await response.json();
  if (data.errors) {
    console.error('GraphQL errors:', data.errors);
  }
} catch (error) {
  console.error('Request failed:', error);
}
```

### Rate Limiting

Be mindful of rate limits:
- Space out bulk operations
- Use pagination for large queries
- Cache responses when appropriate

### Security Considerations

1. **Validate Webhooks**
   - Verify signatures for SNS
   - Check source IPs if possible
   - Use HTTPS always

2. **Secure API Keys**
   - Never commit keys to code
   - Use Secret Manager or env vars
   - Rotate keys regularly

3. **Limit Scope**
   - Request minimum permissions
   - Use read-only tokens when possible
   - Implement policy restrictions

### Monitoring

Monitor integration health:
```bash
# Check webhook endpoint
curl -f https://your-warren/health

# Monitor in logs
gcloud logs read "resource.type=cloud_run_revision" \
  --filter="textPayload:webhook" \
  --limit=50
```

## Examples

### Example 1: GuardDuty Integration

1. Create SNS topic for GuardDuty
2. Configure GuardDuty to publish to SNS
3. Add Warren webhook as subscriber
4. Create policy for GuardDuty alerts

### Example 2: Custom SIEM Integration

1. Create custom alert schema
2. Write policy for parsing SIEM format
3. Configure SIEM to POST to Warren
4. Test with sample alerts

### Example 3: Automated Response

```python
# Query for high-severity alerts
query = """
query GetHighSeverityAlerts {
  unboundAlerts(
    keyword: "severity:critical"
    limit: 100
  ) {
    alerts {
      id
      title
      attributes {
        key
        value
      }
    }
  }
}
"""

# Create tickets for critical alerts
for alert in alerts:
    create_ticket(alert['id'], priority='high')
```

## Troubleshooting

### Webhook Issues

**"401 Unauthorized"**
- Check authentication headers
- Verify policy allows access
- Review auth logs

**"Alert not created"**
- Check alert schema matches policy
- Verify required fields present
- Review policy evaluation logs

### GraphQL Issues

**"Query complexity exceeded"**
- Reduce query depth
- Use pagination
- Split into multiple queries

**"Field not found"**
- Check schema documentation
- Verify field names and types
- Use GraphiQL for exploration

### Tool Integration Issues

**"Tool not available"**
- Verify API key is set
- Check key permissions
- Test tool directly

## Support

For additional help:
- Check Warren logs for detailed errors
- Review policy evaluation results
- Consult the [GitHub repository](https://github.com/secmon-lab/warren)