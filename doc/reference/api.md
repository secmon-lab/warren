# API Reference

This document covers Warren's webhook endpoints, GraphQL API, and MCP integration.

## Webhook Endpoints

Warren provides three webhook endpoints for receiving security alerts. Each endpoint supports a `{schema}` path parameter that determines which Rego policy processes the alert.

### Raw Alert Endpoint

**Endpoint**: `POST /hooks/alert/raw/{schema}`

Direct HTTP integration for custom tools and scripts.

```bash
curl -X POST https://your-warren/hooks/alert/raw/custom \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "title": "Suspicious Network Activity",
    "severity": "high",
    "source_ip": "192.168.1.100"
  }'
```

### Google Pub/Sub Endpoint

**Endpoint**: `POST /hooks/alert/pubsub/{schema}`

For Google Cloud services with guaranteed delivery and automatic retries.

```bash
# Create topic and push subscription
gcloud pubsub topics create warren-alerts
gcloud pubsub subscriptions create warren-alerts-sub \
  --topic=warren-alerts \
  --push-endpoint=https://your-warren/hooks/alert/pubsub/guardduty \
  --push-auth-service-account=warren-sa@project.iam.gserviceaccount.com
```

### AWS SNS Endpoint

**Endpoint**: `POST /hooks/alert/sns/{schema}`

Native AWS integration with automatic signature verification and subscription confirmation.

### Processing Flow

1. **Authentication** — Verify request authenticity (signatures, tokens)
2. **Schema Resolution** — `{schema}` maps to `package ingest.{schema}` policy
3. **Policy Evaluation** — Raw data passed to Rego policy
4. **Alert Creation** — AI generates embeddings, stores in Firestore
5. **Notification** — Posts to Slack, runs clustering

### Custom Schemas

Define custom schemas with Rego policies:

```rego
package ingest.myservice

alert contains {
    "title": input.alert_title,
    "description": input.alert_description,
    "attrs": [{"key": "severity", "value": input.severity, "link": ""}]
} if {
    input.severity in ["low", "medium", "high", "critical"]
}
```

Then send alerts to `/hooks/alert/raw/myservice`.

## GraphQL API

**Endpoint**: `POST /graphql`
**GraphiQL**: `GET /graphiql` (if `--enable-graphiql` is set)

### Authentication

Include authentication cookies from Slack OAuth:
```bash
curl -X POST https://your-warren/graphql \
  -H "Content-Type: application/json" \
  -H "Cookie: token_id=...; token_secret=..." \
  -d '{"query": "{ __typename }"}'
```

### Core Types

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
  finding: Finding
  createdAt: String!
  updatedAt: String!
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
  centerAlert: Alert!
  alerts: [Alert!]!
}
```

### Queries

```graphql
# Get ticket details
query GetTicket($id: ID!) {
  ticket(id: $id) {
    id, title, status, conclusion
    alerts { id, title, schema }
    assignee { name, email }
    comments { content, author { name }, createdAt }
  }
}

# List unbound alerts
query UnboundAlerts($limit: Int, $offset: Int) {
  unboundAlerts(limit: $limit, offset: $offset) {
    alerts { id, title, schema, createdAt }
    hasMore, totalCount
  }
}

# Get alert clusters
query GetClusters($eps: Float, $minSamples: Int) {
  alertClusters(eps: $eps, minSamples: $minSamples, limit: 50) {
    clusters { id, size, keywords, centerAlert { title } }
    totalAlerts, noise
  }
}
```

### Mutations

```graphql
# Create ticket from alerts
mutation CreateTicket($alertIds: [ID!]!) {
  createTicketFromAlerts(alertIds: $alertIds) {
    id, title
  }
}

# Resolve ticket
mutation ResolveTicket($id: ID!, $conclusion: String!, $reason: String!) {
  updateTicketConclusion(id: $id, conclusion: $conclusion, reason: $reason) {
    id, status, conclusion
  }
}

# Bind alerts to ticket
mutation BindAlerts($ticketId: ID!, $alertIds: [ID!]!) {
  bindAlertsToTicket(ticketId: $ticketId, alertIds: $alertIds) {
    id, alerts { id }
  }
}
```

### Code Examples

#### Python

```python
import requests

url = "https://your-warren/graphql"
headers = {"Content-Type": "application/json", "Cookie": "token_id=...; token_secret=..."}

response = requests.post(url, json={
    "query": 'query { ticket(id: "ticket-123") { id title status } }'
}, headers=headers)

print(response.json()["data"]["ticket"])
```

#### TypeScript

```typescript
const response = await fetch('https://your-warren/graphql', {
  method: 'POST',
  credentials: 'include',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    query: `query { tickets(statuses: ["open"], limit: 10) { tickets { id title } } }`
  })
});
const data = await response.json();
```

## MCP (Model Context Protocol)

Warren supports extending AI agent capabilities through MCP. For full configuration details, credential helpers, and building custom tools, see [MCP Integration Guide](../operation/mcp.md).

## Authentication Methods

Warren supports multiple authentication methods for API access:

1. **Google IAP** — JWT token validation from Identity-Aware Proxy headers
2. **OAuth with Slack** — Web UI authentication via Slack OAuth flow
3. **Custom Policy** — Implement token validation in Rego:
   ```rego
   package auth
   allow = true if {
       input.req.header.Authorization[0] == "Bearer valid-token"
   }
   ```
4. **Service Account** — Google Cloud service-to-service with ID tokens
