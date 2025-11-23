# Warren AI Agent Guide

Warren's AI Agent provides an intelligent command-line interface for investigating security incidents using natural language. The agent analyzes alerts within the context of a specific ticket, executing various security tools and providing actionable insights.

## Overview

The AI Agent serves as your security analysis assistant, capable of:
- Analyzing alerts bound to a specific ticket using threat intelligence sources
- Executing complex investigations through natural language commands
- Finding patterns and similar incidents in historical data
- Updating the ticket's findings based on analysis results

### Important Concept: Ticket Context

The AI Agent always operates within the context of a specific ticket. When you use the chat command, you must provide a ticket ID. The agent will:
- Access all alerts associated with that ticket
- Read the ticket's metadata and current findings
- Update the ticket's findings when requested
- Search for similar tickets based on the current ticket's content

This ticket-centric approach ensures that all AI analysis is properly tracked and associated with your investigation workflow.

### Architecture

```mermaid
graph TD
    A[User Query] --> B[AI Agent]
    B --> C{Plan Mode?}
    C -->|Simple| D[Direct Execution]
    C -->|Complex| E[Create Plan]
    E --> F[Execute Steps]
    
    B --> G[Tool Selection]
    G --> H[Warren Tools]
    G --> I[Security Tools]
    G --> J[BigQuery]
    G --> K[MCP Tools]
    
    H --> L[Results]
    I --> L
    J --> L
    K --> L
    
    L --> M[Format Response]
    M --> N[User Response]
```

## Using the Chat Interface

The AI Agent is accessed through the command-line interface. There is no Web UI integration at this time.

### Command Line Interface

#### Interactive mode:
```bash
warren chat --ticket-id ticket-12345678-abcd-efgh-ijkl-123456789012
```

This starts an interactive session where you can have a conversation with the agent about the specified ticket.

#### Single query mode:
```bash
warren chat --ticket-id ticket-12345678-abcd-efgh-ijkl-123456789012 \
  --query "Analyze all IPs in this ticket"
```

This executes a single query and exits, useful for automation or quick checks.

### Required Option

- `--ticket-id` or `-t`: The ticket ID to analyze (required)

### Optional Options

- `--query` or `-q`: Single query to execute (if not provided, starts interactive mode)

Additionally, all standard Warren configuration options are available (Firestore, LLM, Policy, Storage, MCP settings). See the [Configuration Reference](./configuration.md) for details.

## Tool Configuration

### Setting Up External Tool API Keys

Before using the security intelligence tools, you need to configure their API keys. Warren will gracefully skip tools without configured API keys and inform you when a tool is unavailable.

#### Configuration Methods

1. **Environment Variables** (Recommended for production):
```bash
export WARREN_VT_API_KEY="your-virustotal-api-key"
export WARREN_OTX_API_KEY="your-alienault-otx-key"
export WARREN_URLSCAN_API_KEY="your-urlscan-api-key"
export WARREN_SHODAN_API_KEY="your-shodan-api-key"
export WARREN_IPDB_API_KEY="your-abuseipdb-key"
export WARREN_ABUSECH_API_KEY="your-abusech-key"
```

2. **Command-line Flags** (For testing):
```bash
warren serve \
  --vt-api-key="your-virustotal-api-key" \
  --otx-api-key="your-otx-api-key" \
  --urlscan-api-key="your-urlscan-api-key" \
  --shodan-api-key="your-shodan-api-key" \
  --ipdb-api-key="your-abuseipdb-key" \
  --abusech-api-key="your-abusech-key"
```

3. **Google Secret Manager** (For Google Cloud deployments):
```bash
# Create secrets
echo -n "your-api-key" | gcloud secrets create vt-api-key --data-file=-

# Use in Cloud Run
gcloud run services update warren \
  --set-secrets="WARREN_VT_API_KEY=vt-api-key:latest"
```

#### Obtaining API Keys

- **VirusTotal**: Sign up at [virustotal.com](https://www.virustotal.com), go to API Key section
- **OTX**: Register at [otx.alienvault.com](https://otx.alienvault.com), find API key in settings
- **URLScan**: Create account at [urlscan.io](https://urlscan.io), API key in profile
- **Shodan**: Sign up at [shodan.io](https://www.shodan.io), API key in account settings
- **AbuseIPDB**: Register at [abuseipdb.com](https://www.abuseipdb.com), generate key in account

#### Default Behavior

- **Missing API Keys**: Tools without configured keys are automatically excluded from the available tool list
- **Failed API Calls**: Agent reports the error and continues with other tools
- **Rate Limits**: Agent respects rate limits and reports when limits are hit

When tools are not configured, they simply won't appear in the agent's available tools. The agent will use whatever tools are available to complete the requested analysis.

## Available Tools

### Warren Base Tools

These tools integrate directly with Warren's data and operate within the context of your current ticket:

#### `warren_get_alerts`
Retrieves all alerts that are bound to the current ticket.
```
"Show me all alerts in this ticket"
"What alerts are we investigating?"
```

#### `warren_find_nearest_ticket`
Finds similar tickets based on AI embeddings of the current ticket.
```
"Find similar incidents"
"Show me tickets like this one"
"Are there any related investigations?"
```

#### `warren_search_tickets_by_words`
Searches all tickets using keywords or natural language.
```
"Search for tickets mentioning ransomware"
"Find all tickets with IP 192.168.1.100"
"Show me other tickets from the same user"
```

#### `warren_update_finding`
Updates the current ticket's finding field with your analysis results.
```
"Update finding with critical severity"
"Set this as a true positive incident"
"Add investigation summary to findings"
```

#### `warren_get_ticket_comments`
Retrieves comments from the current ticket's Slack thread.
```
"Show me all comments on this ticket"
"What has the team discussed about this?"
"Get the latest Slack discussion"
```

### Security Intelligence Tools

#### VirusTotal Integration
- `vt_ip`: Check IP reputation
- `vt_domain`: Analyze domains
- `vt_file_hash`: Check file hashes
- `vt_url`: Scan URLs

Example:
```
"Check if IP 192.168.1.100 is malicious"
"Analyze the domain suspicious-site.com"
```

#### AbuseIPDB
- `ipdb_check`: IP reputation scoring and abuse reports

Example:
```
"Is this IP reported for abuse?"
"Check the reputation of 192.168.1.100"
```

#### Abuse.ch
- `abusech.bazaar.query`: Query malware samples and IoCs

Example:
```
"Search for this file hash in malware databases"
"Check if this hash is known malware"
```

#### Shodan
- `shodan_host`: Internet device information for specific IP
- `shodan_domain`: Search devices associated with a domain
- `shodan_search`: Search exposed services with queries

Example:
```
"What services are running on this IP?"
"Search for exposed MongoDB instances"
"Find all devices for example.com domain"
```

#### URLScan
- `urlscan_scan`: Submit and scan a URL

Example:
```
"Scan this suspicious URL"
"Analyze https://suspicious-site.com"
```

#### OTX (Open Threat Exchange)
- `otx_ipv4`: IPv4 address threat intelligence
- `otx_ipv6`: IPv6 address threat intelligence
- `otx_domain`: Domain threat intelligence
- `otx_hostname`: Hostname analysis
- `otx_file_hash`: File hash intelligence

Example:
```
"Get threat intelligence for this IP"
"Check if this file hash is known malware"
```

### BigQuery Agent

The BigQuery Agent is a specialized sub-agent that handles BigQuery data retrieval operations. It automatically selects appropriate tables, constructs queries, and retrieves data based on natural language requests.

**Important**: The agent delegates to a specialized BigQuery sub-agent that only retrieves data. The main agent must analyze the returned data and provide answers.

#### Configuration

The BigQuery Agent requires configuration before use. There are two ways to configure it:

**1. Using a Configuration File (Recommended)**:

Create a YAML configuration file specifying available tables:

```yaml
# bigquery-config.yaml
tables:
  - project_id: "my-project"
    dataset_id: "security_logs"
    table_id: "authentication_events"
    description: "User authentication and login events"

  - project_id: "my-project"
    dataset_id: "security_logs"
    table_id: "network_traffic"
    description: "Network connection logs and traffic data"

scan_size_limit: "10GB"  # Optional: Maximum scan size per query
query_timeout: "5m"      # Optional: Query timeout duration
```

Use with Warren:
```bash
warren chat --ticket-id TICKET_ID \
  --agent-bigquery-config bigquery-config.yaml \
  --agent-bigquery-project-id my-project
```

**2. Using Environment Variables**:

```bash
export WARREN_AGENT_BIGQUERY_CONFIG=/path/to/bigquery-config.yaml
export WARREN_AGENT_BIGQUERY_PROJECT_ID=my-project
export WARREN_AGENT_BIGQUERY_SCAN_SIZE_LIMIT=10GB

warren chat --ticket-id TICKET_ID
```

**Additional Options**:

- `--agent-bigquery-runbook-dir`: Path to SQL runbook files or directories containing predefined queries
- `--agent-bigquery-impersonate-service-account`: Service account email to impersonate for BigQuery operations

**SQL Runbooks**:

You can provide predefined SQL queries as "runbooks" that the agent can use:

```sql
-- saved-queries/recent-logins.sql
-- description: Get recent login attempts for a specific user
-- parameters: username, days
SELECT
  timestamp,
  user,
  source_ip,
  status
FROM `{project}.{dataset}.authentication_events`
WHERE user = @username
  AND timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL @days DAY)
ORDER BY timestamp DESC
```

Use runbooks:
```bash
warren chat --ticket-id TICKET_ID \
  --agent-bigquery-config config.yaml \
  --agent-bigquery-runbook-dir ./saved-queries
```

#### Available Tool

When configured, the BigQuery Agent provides one tool to the main agent:

##### `query_bigquery`
Retrieve data from BigQuery tables using natural language.
```
"Query login attempts from this IP in the last 30 days"
"Show me all events from this user yesterday"
"Find network connections to suspicious domains"
```

The agent will:
1. Understand your data retrieval requirements
2. Select the appropriate table(s)
3. Construct and execute the SQL query
4. Return the raw data for analysis

**Note**: If the BigQuery Agent is not configured (no config file provided), this tool will not be available.

### Slack Agent

The Slack Agent is a specialized sub-agent that searches for messages in your Slack workspace. It understands natural language search requests, automatically determines search keywords and variations, and retrieves relevant messages.

**Important**: The agent delegates to a specialized Slack search sub-agent that retrieves messages. The main agent must analyze the returned messages and provide answers.

#### Configuration

The Slack Agent requires a Slack User OAuth Token with `search:read` scope. This is different from the bot token used for posting alerts.

**Setting up Slack User Token**:

1. Go to your Slack App configuration at [api.slack.com/apps](https://api.slack.com/apps)
2. Navigate to "OAuth & Permissions"
3. Under "User Token Scopes", add the `search:read` scope
4. Install/Reinstall the app to your workspace
5. Copy the "User OAuth Token" (starts with `xoxp-`)

**Configure the token**:

Using environment variable (recommended):
```bash
export WARREN_AGENT_SLACK_USER_TOKEN=xoxp-your-user-token-here
warren chat --ticket-id TICKET_ID
```

Using command-line flag:
```bash
warren chat --ticket-id TICKET_ID \
  --agent-slack-user-token xoxp-your-user-token-here
```

**Why User Token?**

The Slack search API requires a user token (not a bot token) because it searches across all channels visible to the user, including private channels and DMs. Make sure to use a user token with appropriate channel access.

#### Available Tool

When configured, the Slack Agent provides one tool to the main agent:

##### `search_slack`
Search for messages in Slack workspace using natural language.
```
"Find discussions about authentication issues"
"People reporting performance problems in the last week"
"Error messages in #security-alerts channel"
```

**Parameters**:
- `request`: Natural language description of what to find (NOT search keywords)
- `limit`: Maximum number of messages to return (default: 50, max: 200)

**Important Usage Notes**:

❌ **DON'T** specify search keywords:
- "search for authentication keyword"
- "messages containing 'error'"
- "find keyword login"

✅ **DO** describe the concept/situation:
- "people having authentication problems"
- "discussions about performance issues"
- "error reports in #security-alerts channel"

The Slack Agent automatically:
1. Determines appropriate search keywords and variations
2. Handles multilingual terms
3. Searches across channels and time periods
4. Returns relevant messages with full context

**Example Usage**:

```
User: What were people saying about the outage yesterday?

Agent: [Calls search_slack with request="discussions about outage yesterday"]

Slack Agent returns: 15 messages from #incidents and #engineering discussing
the database outage on 2025-11-22...

Agent: Based on the Slack messages, the team identified a database connection
pool exhaustion issue that started at 14:30. [Provides analysis]
```

**Note**: If the Slack Agent is not configured (no user token provided), this tool will not be available.

## MCP (Model Context Protocol) Integration

Warren supports extending capabilities through MCP:

### What is MCP?

MCP allows Warren to connect to external tool servers that provide additional capabilities. These can be:
- Remote services (via HTTP/SSE)
- Local executables
- Custom integrations

### Configuring MCP

Create a YAML configuration file:

```yaml
# mcp-config.yaml
servers:
  - name: "custom-intel"
    type: "sse"
    url: "https://intel-api.example.com/mcp"
    headers:
      Authorization: "Bearer YOUR_API_KEY"
    
  - name: "local-scanner"
    type: "stdio"
    command: "/usr/local/bin/scanner-mcp"
    args: ["--mode", "mcp"]
```

Use with Warren:
```bash
warren chat --ticket-id TICKET_ID --mcp-config mcp-config.yaml
```

### Available MCP Types

1. **SSE (Server-Sent Events)**
   - Real-time streaming tools
   - Remote API integrations

2. **HTTP**
   - RESTful tool endpoints
   - Request-response patterns

3. **STDIO**
   - Local executables
   - Command-line tools wrapped as MCP

## Using the Chat Interface

### Understanding Ticket Context

Before using commands, remember that the agent always works within the context of your specified ticket:
- All alerts referenced are those bound to the ticket
- Findings updates modify the current ticket
- Similarity searches compare against the current ticket
- The agent cannot access alerts or data from other tickets unless explicitly searching

### Basic Commands

Simple questions about the current ticket:
```
"Summarize this incident"
"What's the severity of these alerts?"
"When did this attack start?"
"How many alerts are in this ticket?"
```

### Investigation Commands

Comprehensive analysis of the ticket's alerts:
```
"Analyze all IPs and domains in this ticket"
"Check if any indicators are malicious"
"Find the attack pattern in these alerts"
"What services are affected?"
```

### Complex Investigations

Multi-step analysis with plan mode:
```
"Investigate this incident thoroughly:
1. Check all IPs against threat intel
2. Search for similar past incidents
3. Query logs for related activity
4. Summarize findings and recommend actions"
```

### Updating the Current Ticket

Modify the ticket's finding field:
```
"Update the finding with high severity"
"Mark this as a false positive with explanation"
"Add recommendation to block these IPs"
"Summarize the investigation results in the finding"
```

## Plan Mode

For complex requests, the agent automatically enters plan mode:

```
User: Perform a complete investigation of this incident

Agent: I'll help you perform a complete investigation. Let me break this down:

📋 Creating plan...

### Investigation Plan
- [ ] Analyze alert indicators
- [ ] Check threat intelligence  
- [ ] Search historical data
- [ ] Summarize findings

⏳ Starting execution...

✅ [1/4] Analyzing alert indicators...
[Tool execution details]

✅ [2/4] Checking threat intelligence...
[Tool execution details]
```

Plan mode provides:
- Clear task breakdown
- Progress tracking
- Transparent execution
- Error handling

## Effective Prompts

### Be Specific
❌ "Check this"
✅ "Check if the source IP 192.168.1.100 is malicious"

### Provide Context
❌ "Find similar"
✅ "Find similar DDoS attacks from the last 30 days"

### Request Actions
❌ "This looks bad"
✅ "Update the finding with critical severity and recommend immediate IP blocking"

### Batch Operations
❌ Multiple separate queries
✅ "Analyze all IPs, domains, and file hashes in this ticket for threats"

## Advanced Usage

### Chaining Investigations

```
"First check if these IPs are malicious, then search our BigQuery logs 
for any successful connections from confirmed bad IPs"
```

### Custom Analysis

```
"Compare this incident with the one from last week (ticket-xxx) and 
identify common patterns"
```

### Automated Workflows

```
"Every time you find a malicious IP:
1. Check our logs for any connections
2. See if other tickets mention it
3. Add it to the finding summary"
```

## Slack Integration

The chat command itself does not post to Slack. However, when you update the ticket's findings using the agent, those updates will be reflected in:
- The ticket's finding field in the database
- The Web UI when viewing the ticket
- Any subsequent Slack messages about the ticket will show the updated findings

To share your agent analysis with the team, use the `warren_update_finding` tool to save your conclusions to the ticket.

## Best Practices

### 1. Start Broad, Then Focus
Begin with general analysis, then dive into specific concerns:
```
"Analyze this incident" → "Focus on the command injection attempts"
```

### 2. Verify Critical Findings
Double-check important results:
```
"The IP showed up as malicious in OTX. Check VirusTotal and AbuseIPDB too"
```

### 3. Document Thoroughly
Update findings with clear explanations:
```
"Update finding: Confirmed ransomware attack. Initial access via phishing email.
Recommendation: Isolate affected systems and begin incident response procedure."
```

### 4. Use Historical Context
Leverage past incidents:
```
"Compare with similar tickets to see if this is part of a campaign"
```

### 5. Automate Repetitive Tasks
Let the agent handle routine checks:
```
"For each alert, check all IPs and domains against threat intel"
```

## Troubleshooting

### "Tool not found"
- Verify tool name is correct
- Check if external API keys are configured
- Ensure MCP servers are running

### "No response from agent"
- Check ticket ID is valid
- Verify you have access to the ticket
- Check Warren logs for errors

### "Timeout errors"
- Complex queries may take time
- Consider breaking into smaller requests
- Check external service availability

### "Cannot update finding"
- Ensure you have write permissions
- Verify ticket is not archived
- Check the finding format

## Examples

### Example 1: Basic Threat Analysis
```
User: Check if any IPs in this ticket are malicious

Agent: I'll analyze all IPs found in the alerts bound to ticket ticket-12345...

[Executes vt_ip, otx_ipv4, ipdb_check for each IP]

Found 3 IPs in this ticket's alerts:
- 192.168.1.100: Clean across all sources
- 10.0.0.50: Internal IP, skipped
- 185.220.100.240: MALICIOUS - Listed in 45/87 VT engines, 
  OTX shows it's part of Tor exit nodes
```

### Example 2: Historical Pattern Search
```
User: Find similar SQL injection attacks in the past month

Agent: I'll search for SQL injection patterns in recent tickets.

[Executes warren_search_tickets_by_words with SQL injection keywords]
[Executes bigquery_query for web logs with SQL patterns]

Found 7 similar incidents:
- 5 tickets with SQL injection attempts
- All targeting the same /api/users endpoint  
- Coming from 3 different IP ranges
- Pattern suggests automated scanning tool
```

### Example 3: Incident Summary
```
User: Summarize this incident and update the finding

Agent: I'll analyze all alerts in this ticket and create a comprehensive summary.

[Analyzes ticket's alerts, checks indicators, reviews patterns]

Based on the 23 alerts in this ticket:
- Attack type: Coordinated brute force against SSH services
- Affected systems: 15 servers in production environment
- Attack source: Known botnet IPs (confirmed via threat intel)
- Impact: No successful logins detected in logs

[Executes warren_update_finding]

✅ Updated ticket's finding field with High severity classification and investigation summary.
```

## Language Support

The agent responds in the language configured at the global level:

```bash
# Set language globally (affects all content generation)
# You can use any word to specify language
export WARREN_LANG=ja  # Japanese
export WARREN_LANG=日本語 # Japanese
export WARREN_LANG=es  # Spanish
export WARREN_LANG=en  # English (default)

warren chat --ticket-id TICKET_ID
```

Supported languages depend on the underlying LLM model (Gemini or Claude).

## Next Steps

1. **Explore Tools**: Try each tool category to understand capabilities
2. **Create Workflows**: Develop standard investigation patterns
3. **Configure MCP**: Add custom tools for your environment
4. **Share Knowledge**: Document useful prompts for your team