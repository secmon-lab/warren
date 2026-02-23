# Experimental Features

This document describes experimental features in Warren. These features are functional but may change in future releases.

## Refine

The `refine` feature periodically reviews and organizes Warren's operational state. It performs two main tasks:

1. **Open Ticket Review** - Reviews all open tickets and posts follow-up messages when tickets appear stagnant
2. **Unbound Alert Consolidation** - Groups unbound alerts (alerts not yet linked to any ticket) that share a common cause and proposes ticket creation

### How It Works

#### Task 1: Open Ticket Review

For each open ticket, Warren collects the ticket metadata, linked alerts, and comment history, then asks the LLM whether a follow-up is needed. If the LLM generates a follow-up message, it is posted to the ticket's Slack thread. If the message is empty, no action is taken.

The LLM considers factors such as:
- How long the ticket has been stagnant
- Whether an assignee has responded
- Whether a previous refine follow-up was posted recently (to avoid nagging)

#### Task 2: Unbound Alert Consolidation

This task runs in three phases:

1. **Summarize** - Each unbound alert is individually analyzed by the LLM to extract key features (identities, parameters, context, suspected root cause)
2. **Consolidate** - All summaries are presented to the LLM to identify groups of alerts likely caused by the same underlying issue. Each group has a primary (most representative) alert selected by the LLM.
3. **Propose** - For each group, a Slack message is posted to the primary alert's thread (with `reply_broadcast` so it also appears in the channel). The message includes:
   - The consolidation reason
   - Links to all candidate alerts
   - A **Create Ticket** button

When the **Create Ticket** button is pressed, Warren creates a new ticket from all alerts in the group.

### Usage

#### CLI

Run the refine process as a standalone command. This is suitable for cron-based scheduling.

```bash
warren refine \
  --gemini-project-id <GCP_PROJECT_ID> \
  --firestore-project-id <GCP_PROJECT_ID> \
  --slack-oauth-token <SLACK_TOKEN> \
  --slack-channel-name <CHANNEL_NAME>
```

All LLM flags (`--gemini-*`, `--claude-*`) are supported. Firestore and Slack are optional — without Firestore, an in-memory repository is used; without Slack, results are printed to the console.

| Flag | Env Var | Required | Description |
|------|---------|----------|-------------|
| `--gemini-project-id` | `WARREN_GEMINI_PROJECT_ID` | Yes | GCP Project ID for Vertex AI |
| `--gemini-model` | `WARREN_GEMINI_MODEL` | No | Gemini model (default: `gemini-2.5-flash`) |
| `--gemini-location` | `WARREN_GEMINI_LOCATION` | No | GCP location (default: `us-central1`) |
| `--claude-project-id` | `WARREN_CLAUDE_PROJECT_ID` | No | GCP Project ID for Claude Vertex AI |
| `--claude-model` | `WARREN_CLAUDE_MODEL` | No | Claude model (default: `claude-sonnet-4@20250514`) |
| `--claude-location` | `WARREN_CLAUDE_LOCATION` | No | GCP location for Claude (default: `us-east5`) |
| `--firestore-project-id` | `WARREN_FIRESTORE_PROJECT_ID` | No | Firestore project ID |
| `--firestore-database-id` | `WARREN_FIRESTORE_DATABASE_ID` | No | Firestore database ID (default: `(default)`) |
| `--slack-oauth-token` | `WARREN_SLACK_OAUTH_TOKEN` | No | Slack OAuth token |
| `--slack-channel-name` | `WARREN_SLACK_CHANNEL_NAME` | No | Slack channel name |

#### Slack Command

In Slack, mention Warren with the `refine` command:

```
@warren refine
```

The command runs asynchronously. Warren will reply with a confirmation message and post results as they become available.

### Scheduling Example

To run refine every 6 hours using cron:

```cron
0 */6 * * * warren refine --gemini-project-id my-project --firestore-project-id my-project --slack-oauth-token xoxb-... --slack-channel-name alerts
```

For Google Cloud environments, you can use Cloud Scheduler to trigger the refine command via Cloud Run jobs.

### Limitations

- Unbound alert processing is capped at 100 alerts per run to manage LLM costs
- Each consolidation group contains 2-10 alerts
- The LLM may not always produce follow-up messages even when tickets are stagnant — this is by design to avoid noise
