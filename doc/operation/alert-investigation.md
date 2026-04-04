# Alert Investigation Guide

This guide covers how to investigate security alerts using Warren through Slack, the Web UI, and the CLI.

## Investigation Interfaces

Warren provides three interfaces for alert investigation:

| Interface | Best For |
|-----------|----------|
| **Slack** | Real-time notifications, quick actions, team collaboration |
| **Web UI** | Detailed analysis, bulk operations, alert management |
| **CLI** (`warren chat`) | Automated analysis, scripting, local development |

All interfaces are synchronized — changes in one are reflected in the others.

## Slack Interface

### Alert Notifications

When a new alert is detected, Warren posts it to your configured Slack channel with:
- Title and description (AI-enhanced)
- Alert ID and schema
- Key attributes (severity, source IP, etc.)
- Status indicator: unbound or bound to a ticket
- **[Acknowledge]** and **[Bind to ticket]** buttons

### Creating a Ticket

Click **[Acknowledge]** to create a new ticket from an alert. The alert status changes from unbound to bound, and a ticket message appears in the thread.

To add an alert to an existing investigation, click **[Bind to ticket]** and select from recent tickets or enter a ticket ID directly.

### Resolving a Ticket

Click **[Resolve]** on a ticket message to open the resolution modal. Select a conclusion (Intended, Unaffected, False Positive, True Positive, or Escalated) and optionally add a comment.

### Salvaging Similar Alerts

Click **[Salvage]** on a ticket to find similar unbound alerts using AI similarity matching. Select relevant alerts to bind them to the ticket.

### Talking to Warren

Mention `@warren` in a ticket's Slack thread to start an AI-powered investigation. Warren will analyze the ticket's alerts using available tools and sub-agents.

```
@warren Check if the source IPs in this ticket are malicious
@warren Analyze all indicators and update the finding
@warren Search our logs for connections from these IPs
```

### Slack Commands

The following commands are available by mentioning `@warren`:

| Command | Aliases | Description |
|---------|---------|-------------|
| `ticket` | `t` | Create or manage a ticket |
| `refine` | `r` | Run the refine process (review open tickets and consolidate unbound alerts) |
| `repair` | | Repair ticket state |
| `abort` | | Abort current operation |
| `purge` | | Purge and clean up |

## Web UI

### Dashboard

The dashboard provides an overview of your security posture with open ticket count, unbound alert count, and recent activity.

### Alert Management

Navigate to **Alerts** to view, filter, and manage alerts:
- Filter by status (Unbound/Bound), time range, severity, and schema
- Select multiple alerts for bulk operations (create ticket, bind to existing)

### Ticket Management

The **Tickets** page shows all tickets with status, alert count, assignee, and quick actions. Click a ticket for the detail view with:
- All bound alerts and their attributes
- Investigation comments and findings
- **Chat** button for AI-powered analysis via WebSocket

### Chat via Web UI

The ticket detail view includes a chat interface connected via WebSocket (`/ws/chat/ticket/{ticketID}`). This provides the same AI investigation capabilities as Slack mentions, with real-time streaming responses.

## CLI Interface

### Interactive Mode

```bash
warren chat --ticket-id <TICKET_ID>
```

Starts an interactive session for conversational investigation of the specified ticket.

### Single Query Mode

```bash
warren chat --ticket-id <TICKET_ID> \
  --query "Analyze all IPs in this ticket"
```

Executes a single query and exits. Useful for automation.

### Language Setting

```bash
export WARREN_LANG=ja  # Japanese
warren chat --ticket-id <TICKET_ID>
```

## Available Tools

Warren's AI agent has access to the following tools during investigation. Tools without configured API keys are automatically excluded.

### Warren Base Tools

| Tool | Description |
|------|-------------|
| `warren_get_alerts` | Get alerts bound to the current ticket |
| `warren_find_nearest_ticket` | Find similar tickets using AI embeddings |
| `warren_search_tickets_by_words` | Search tickets by keywords |
| `warren_update_finding` | Update the ticket's finding (severity, summary, reason, recommendation) |
| `warren_get_ticket_comments` | Get comments from the ticket's Slack thread |

### Threat Intelligence Tools

| Tool | Description | Details |
|------|-------------|---------|
| VirusTotal | IP, domain, file hash, URL reputation | [README](../../pkg/tool/vt/README.md) |
| OTX | Multi-indicator threat intelligence | [README](../../pkg/tool/otx/README.md) |
| URLScan.io | URL scanning and analysis | [README](../../pkg/tool/urlscan/README.md) |
| Shodan | Internet device search | [README](../../pkg/tool/shodan/README.md) |
| AbuseIPDB | IP reputation scoring | [README](../../pkg/tool/ipdb/README.md) |
| abuse.ch | Malware hash lookup | [README](../../pkg/tool/abusech/README.md) |
| WHOIS | Domain/IP registration lookup | [README](../../pkg/tool/whois/README.md) |

### Code & Device Tools

| Tool | Description | Details |
|------|-------------|---------|
| GitHub | Code search, issue search, file content | [README](../../pkg/tool/github/README.md) |
| Microsoft Intune | Device compliance, sign-in history | [README](../../pkg/tool/intune/README.md) |

### Other Tools

| Tool | Description | Details |
|------|-------------|---------|
| Slack Message Search | Search workspace messages | [README](../../pkg/tool/slack/README.md) |
| Knowledge | Save/retrieve investigation knowledge | [README](../../pkg/tool/knowledge/README.md) |

## Sub-Agents

Sub-agents are specialized AI agents that handle complex, multi-step operations. The main agent delegates to them when needed.

| Sub-Agent | Description | Details |
|-----------|-------------|---------|
| BigQuery Agent | Query security log data via natural language | [README](../../pkg/agents/bigquery/README.md) |
| CrowdStrike Falcon Agent | Query EDR incidents, alerts, and events | [README](../../pkg/agents/falcon/README.md) |
| Slack Search Agent | Search Slack messages for context | [README](../../pkg/agents/slack/README.md) |

## Chat Strategy

Warren uses the `amber` chat execution strategy by default. It parallelizes independent tasks for faster multi-tool investigations.

Configure via `--chat-strategy` flag or `WARREN_CHAT_STRATEGY` environment variable:

```bash
warren serve --chat-strategy amber
```

### Amber Strategy Details

The amber strategy parallelizes independent tasks:

1. **Planning**: LLM creates a structured plan with independent tasks
2. **Phase Execution**: All tasks in a phase run in parallel (separate goroutines)
3. **Replan**: LLM reviews results, adds new tasks if needed, or proceeds to final response
4. **Final Response**: Synthesizes all results into a comprehensive answer

Each task gets its own Slack progress indicator and runs with only the tools it needs.

## MCP Integration

Warren supports extending AI agent capabilities through MCP (Model Context Protocol). You can connect remote MCP servers or bundle custom tool binaries into your Docker image.

See [MCP Integration Guide](./mcp.md) for full configuration, credential helpers, and building custom tools.

## Effective Prompts

### Be Specific
- Bad: "Check this"
- Good: "Check if the source IP 192.168.1.100 is malicious using VirusTotal and AbuseIPDB"

### Provide Context
- Bad: "Find similar"
- Good: "Find similar DDoS attacks from the last 30 days"

### Request Actions
- Bad: "This looks bad"
- Good: "Update the finding with critical severity and recommend immediate IP blocking"

### Batch Operations
- Bad: Multiple separate queries for each indicator
- Good: "Analyze all IPs, domains, and file hashes in this ticket for threats"

## Common Workflows

### Investigating a New Alert

1. Alert appears in Slack
2. Click **[Acknowledge]** to create a ticket
3. Mention `@warren` to start AI analysis
4. Review findings and update ticket
5. Click **[Resolve]** with appropriate conclusion

### Handling Alert Storms

1. Create a ticket from a representative alert
2. Use **[Salvage]** to find and bind similar unbound alerts
3. Investigate once for the entire group of alerts

### Collaborative Investigation

1. Ticket created in Slack — thread becomes the discussion space
2. `@mention` experts and share findings in the thread
3. Use Web UI for detailed analysis in parallel
4. All updates synchronized across interfaces
