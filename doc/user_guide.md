# Warren User Guide

This guide covers daily operations and workflows for security analysts using Warren to manage security alerts and incidents through Slack and the Web UI.

## Overview

Warren provides two primary interfaces for managing security alerts:
- **Slack Interface**: Real-time notifications and quick actions
- **Web UI**: Detailed analysis and management capabilities

Both interfaces are synchronized, allowing seamless collaboration between team members.

## Slack Interface

### Alert Notifications

When a new security alert is detected, Warren posts it to your configured Slack channel:

```
❗ [Alert Title]
[Alert Description]
————————————————
*ID:* `alert-12345678-abcd-efgh-ijkl-123456789012`
*Schema:* `guardduty`
*Ack:* ⏳
————————————————
*severity:* high
*source_ip:* 192.168.1.100
*region:* us-east-1

[Acknowledge] [Bind to ticket]
```

**Key Elements:**
- **Title**: AI-enhanced title with ❗ emoji prefix
- **Description**: Detailed explanation of the alert
- **ID**: Unique alert identifier
- **Schema**: Alert type/source (e.g., guardduty, sentry)
- **Ack Status**: 
  - ⏳ = Unbound (no ticket)
  - ✅ = Bound (has ticket)
- **Attributes**: Key-value pairs from the alert
- **Action Buttons**: Quick actions for alert processing

### Interactive Buttons

#### Alert Buttons

1. **[Acknowledge]** (Blue/Primary)
   - Creates a new ticket immediately
   - Alert status changes from ⏳ to ✅
   - Ticket message appears in thread

2. **[Bind to ticket]** (Red/Danger)
   - Opens modal to select existing ticket
   - Links alert to chosen ticket
   - Useful for grouping related alerts

#### Ticket Buttons

1. **[Resolve]** (Blue/Primary)
   - Opens resolution modal
   - Select conclusion and add comments
   - Updates ticket status

2. **[Salvage]** (Gray/Default)
   - Finds similar unbound alerts
   - Helps clean up related alerts
   - Uses AI similarity matching

### Modal Dialogs

#### Bind to Ticket Modal

When binding an alert to an existing ticket:

```
Select a ticket to bind this alert to:

Ticket: [Dropdown menu ▼]
  🔍 Suspicious Login Investigation (2h ago)
  🔍 Port Scan Detection (5h ago)
  ✅ Resolved False Positive (1d ago)

Or enter ticket ID directly:
Ticket ID: [________________]

[Cancel] [Bind]
```

**Features:**
- Recent tickets with status icons
- Time since creation
- Manual ID entry option

#### Resolve Ticket Modal

When resolving a ticket:

```
Resolve Ticket

Conclusion: [Dropdown menu ▼]
  👍 Intended - Expected behavior
  🛡️ Unaffected - No impact
  🚫 False Positive - Incorrect alert
  🚨 True Positive - Real incident
  ⬆️ Escalated - Needs escalation

Comment (optional):
[________________________]
[________________________]

[Cancel] [Resolve]
```

**Conclusion Options:**
- **Intended**: Normal, expected behavior
- **Unaffected**: Real attack but no impact
- **False Positive**: Alert fired incorrectly
- **True Positive**: Confirmed security incident
- **Escalated**: Requires higher-level response

#### Salvage Modal

Finding similar alerts:

```
Find Similar Unbound Alerts

Similarity Threshold: [0.9] (0.0-1.0)
Keyword Filter: [____________]

[Refresh]

Found 3 similar alerts:
☐ Suspicious Login from 192.168.1.101
☐ Failed Auth Attempt - Same Region
☐ Multiple Login Failures Detected

[Cancel] [Submit]
```

### Ticket Messages

When a ticket is created or updated:

```
🎫 Suspicious Login Investigation
*ID:* `ticket-87654321-dcba-hgfe-lkji-210987654321` | 🔗 View Details
Investigation of multiple failed login attempts from suspicious IP addresses

*Status:*
🔍 Open
*Assignee:*
👤 @john.doe
*Conclusion:*
🚫 False Positive
*Reason:*
💭 Legitimate user forgot password, confirmed via help desk ticket #12345
————————————————
🔔 Related Alerts
• Suspicious Login from 192.168.1.100
• Failed Authentication - Multiple Attempts
• Account Lockout Triggered
_Showing 3 of 3 alerts_
————————————————
🔍 Finding
*Severity:* 🟡 Medium
*Summary:* Multiple failed login attempts detected
*Reason:*
💭 User confirmed password reset attempt after returning from vacation
*Recommendation:*
💡 Consider implementing password recovery reminders for long absence

[Resolve] [Salvage]
```

**Test Tickets**: Display with 🧪 prefix:
```
🧪 [TEST] Security Drill - Phishing Response
```

### Alert Lists

For aggregated alerts:

```
📑 New list with 15 alerts
————————————————
📋 Alert List
Multiple related alerts detected in short time window
*Status:* 📋 Unbound
*ID:* `list-11111111-aaaa-bbbb-cccc-dddddddddddd`

• Port scan detected on server-01
• Port scan detected on server-02
• Port scan detected on server-03
_Showing 3 of 15 alerts_

[Acknowledge] [Bind to ticket]
```

## Web UI

### Dashboard

The dashboard provides an at-a-glance view of your security posture:

```
┌─────────────────────────┬─────────────────────────┐
│ 📊 Open Tickets         │ 🚨 Unbound Alerts       │
│        12               │        47               │
└─────────────────────────┴─────────────────────────┘

📈 Recent Activity
• john.doe resolved ticket "Suspicious Login" as False Positive (5m ago)
• New alert "Port Scan Detected" created (10m ago)
• jane.smith created ticket "Investigate Unusual Traffic" (1h ago)
```

### Alert Management

Navigate to **Alerts** to see all alerts:

**Filters:**
- Status: All / Unbound / Bound
- Time Range: Last 24h / 7d / 30d / Custom
- Severity: Critical / High / Medium / Low
- Schema: guardduty / sentry / custom

**Bulk Actions:**
- Select multiple alerts with checkboxes
- Create ticket from selected alerts
- Bind selected to existing ticket

### Ticket Management

The **Tickets** page shows all tickets with:

**Views:**
- Open Tickets (default)
- Pending Tickets
- Resolved Tickets
- All Tickets

**For Each Ticket:**
- Status indicator and title
- Alert count and severity
- Assignee information
- Last updated time
- Quick actions menu

### Ticket Detail View

Detailed ticket information includes:

1. **Header Section**
   - Title and status
   - Created/Updated timestamps
   - Assignee selector

2. **Alert Section**
   - List of all bound alerts
   - Alert details and attributes
   - Timeline of when alerts were added

3. **Investigation Section**
   - Comments and findings
   - File attachments
   - External tool results

4. **AI Analysis**
   - Chat with Agent button
   - View AI recommendations
   - Similar ticket suggestions

### Clustering View

Access via **Clusters** to see alert groupings:

```
Clustering Parameters
├─ eps: 0.5 (distance threshold)
├─ minSamples: 3 (minimum cluster size)
└─ Keyword filter: [_________]

Found 5 clusters:

🎯 Cluster: "swift-eagle" (24 alerts)
   Center: "Suspicious Network Activity"
   Keywords: network, scan, port, reconnaissance
   [Create Ticket] [Bind to Existing]

🎯 Cluster: "brave-wolf" (18 alerts)
   Center: "Failed Authentication Attempts"
   Keywords: login, authentication, failed, password
   [Create Ticket] [Bind to Existing]
```

## Common Workflows

### Investigating a New Alert

1. **Initial Assessment**
   - Alert appears in Slack
   - Review title, description, and severity
   - Check attributes for immediate context

2. **Create Ticket**
   - Click [Acknowledge] for simple alerts
   - Or [Bind to ticket] for related investigations

3. **Detailed Analysis**
   - Click "🔗 View Details" in ticket message
   - Opens Web UI for full investigation
   - Access Chat with Agent for AI analysis

4. **Investigation**
   - Use AI Agent to analyze IPs, domains, files
   - Add findings and comments
   - Upload relevant screenshots or logs

5. **Resolution**
   - Click [Resolve] in Slack or Web UI
   - Select appropriate conclusion
   - Document reason for future reference

### Handling Alert Storms

When multiple similar alerts arrive:

1. **Let Clustering Work**
   - Warren automatically groups similar alerts
   - Check Clusters view in Web UI

2. **Review Clusters**
   - Assess cluster size and keywords
   - Identify root cause patterns

3. **Bulk Processing**
   - Create ticket from entire cluster
   - All alerts automatically bound
   - Single investigation for many alerts

4. **Use Salvage**
   - After resolving, click [Salvage]
   - Find any missed similar alerts
   - Bind them to keep things clean

### Managing False Positives

1. **Identify Pattern**
   - Note common attributes
   - Document why it's false positive

2. **Resolve Appropriately**
   - Select "🚫 False Positive" conclusion
   - Add clear explanation

3. **Prevent Recurrence**
   - Work with admin to update policies
   - Consider ignore rules for known-good

4. **Clean Up**
   - Use Salvage to find similar
   - Bulk resolve if appropriate

### Collaborative Investigation

1. **Start in Slack**
   - Alert posted to channel
   - Team sees and can comment

2. **Create Ticket**
   - Acknowledge to create ticket
   - Thread created for discussion

3. **Collaborate**
   - @mention experts in thread
   - Share findings in comments
   - Upload evidence files

4. **Parallel Work**
   - Some in Slack thread
   - Others in Web UI
   - All updates synchronized

5. **Resolution**
   - Consensus on conclusion
   - Document for knowledge base

## Best Practices

### Slack Etiquette

1. **Use Threads**
   - Keep discussions in ticket threads
   - Avoid cluttering main channel

2. **Meaningful Comments**
   - Add context, not just "looking"
   - Share findings as you go

3. **Timely Updates**
   - Acknowledge critical alerts quickly
   - Update status when investigating

### Ticket Management

1. **Descriptive Titles**
   - Edit AI titles if needed
   - Make searchable for future

2. **Thorough Documentation**
   - Record all findings
   - Include negative results too

3. **Appropriate Conclusions**
   - Be accurate for metrics
   - "Escalated" != "too hard"

### Using AI Features

1. **Effective Prompts**
   - Be specific with requests
   - Provide context when needed

2. **Verify Results**
   - AI assists, doesn't replace judgment
   - Cross-check suspicious findings

3. **Learn Patterns**
   - Note what AI catches
   - Improve your recognition

## Tips and Tricks

### Keyboard Shortcuts (Web UI)

- `Ctrl/Cmd + K`: Quick search
- `Ctrl/Cmd + /`: Toggle filters
- `Escape`: Close modals

### Search Operators

In Web UI search:
- `status:open` - Open tickets only
- `severity:high` - High severity alerts
- `created:>2024-01-01` - Recent items
- `assignee:me` - Your tickets

### Time Savers

1. **Bookmark Filtered Views**
   - Save common filter combinations
   - Quick access to your workflow

2. **Use Templates**
   - Create template comments
   - Standardize investigations

3. **Leverage History**
   - Search similar past tickets
   - Reuse resolution reasoning

## Troubleshooting

### Common Issues

**"Can't see alerts in Slack"**
- Verify Warren bot is in channel
- Check your notification settings
- Confirm channel in Warren config

**"Buttons not working"**
- Refresh Slack client
- Check Warren service status
- Try Web UI as alternative

**"Can't log into Web UI"**
- Use "Sign in with Slack"
- Clear cookies and retry
- Check with administrator

**"Alerts not clustering"**
- Verify similarity threshold
- Check if alerts have embeddings
- Allow time for processing

### Getting Help

1. **In-App Help**
   - Hover over ? icons
   - Check tooltips

2. **Team Resources**
   - Ask in #warren-help
   - Check team runbooks

3. **Administrator**
   - Policy questions
   - Access issues
   - Feature requests

## Available Tools

Warren includes several built-in tools that can be used during alert investigation and analysis. These tools are accessible through the chat interface.

### Slack Message Search

Search for messages in your Slack workspace to find related discussions, previous incidents, or context around alerts.

**Tool Name:** `slack_message_search`

**Parameters:**
- `query` (required): The search query using Slack's search operators
- `sort`: Sort order - "score" (relevance) or "timestamp" (newest first)
- `sort_dir`: Sort direction - "asc" or "desc"
- `count`: Number of results to return (default: 20, max: 100)
- `page`: Page number for pagination (default: 1)

**Example Queries:**
```
// Search for messages from a specific user
query: "from:@security-bot"

// Search in a specific channel
query: "in:security-alerts error"

// Search for messages with attachments
query: "has:file malware"

// Complex query with multiple operators
query: "from:@john in:incidents after:2024-01-01"
```

**Configuration:**
To use the Slack Message Search tool, set the following environment variable:
- `WARREN_SLACK_TOOL_USER_TOKEN`: Slack User token with `search:read` permission

**Important:** This tool requires a **User OAuth token**, not a Bot token. The search.messages API only works with User tokens. To obtain a User token:
1. Create a Slack App with OAuth scopes
2. Install the app to your workspace
3. Use the User OAuth Token (starts with `xoxp-`), not the Bot User OAuth Token

**Required Slack App Permissions:**
- `search:read` - Search messages and files in the workspace
- Additional permissions may be required depending on search scope:
  - `channels:history` - Search public channels
  - `groups:history` - Search private channels
  - `im:history` - Search direct messages
  - `mpim:history` - Search group direct messages

### Other Available Tools

- **OTX (AlienVault Open Threat Exchange)**: Threat intelligence lookup
- **VirusTotal**: File and URL reputation checking
- **URLScan**: URL analysis and screenshots
- **Shodan**: Internet-wide device search
- **AbuseIPDB**: IP address reputation
- **BigQuery**: Custom data analysis