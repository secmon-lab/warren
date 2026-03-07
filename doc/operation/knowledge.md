# Knowledge Management

Knowledge Management in Warren allows you to capture and preserve organizational security expertise that informs alert triage and incident response. Knowledge is automatically integrated into AI-powered analysis.

## Why Knowledge Management?

- **Reduce False Positives**: Document known benign patterns
- **Preserve Expertise**: Capture team knowledge for consistent decisions
- **Context-Aware Analysis**: Automatically provide context to AI
- **Continuous Improvement**: Learn from past incidents

## Core Concepts

### Topic

A **Topic** groups related knowledge together. Topics are automatically assigned based on alert schema (e.g., `aws.cloudtrail`, `okta.system`).

- Alerts receive a topic from their schema
- Tickets inherit the topic from their first alert
- Knowledge is organized by topic for automatic retrieval

### Knowledge Entry

Each knowledge entry has:
- **Name**: Short, descriptive title
- **Slug**: Unique identifier within the topic
- **Content**: Detailed information (Markdown)
- **Topic**: The topic this knowledge belongs to

Example:
```markdown
Name: Known Safe S3 Bucket Access
Slug: safe-s3-buckets
Topic: aws.cloudtrail

The following S3 buckets are accessed daily by our data pipeline:
- company-logs-archive - Accessed by log aggregator (10.0.1.50)
- company-backups - Accessed by backup system (10.0.1.51)

These access patterns occur between 00:00-06:00 UTC.
```

### Version Management

Warren uses an append-only model:
- Creating or updating adds a new version
- Previous versions remain in history
- Archiving marks knowledge as inactive (logical delete)

## Managing Knowledge

### Web UI

Navigate to `/knowledge` to view, create, edit, and archive knowledge entries.

### Agent Tools

During investigation, the AI agent can manage knowledge using built-in tools:

| Tool | Description |
|------|-------------|
| `knowledge_list` | List all knowledge slugs in the topic |
| `knowledge_get` | Get specific knowledge by slug |
| `knowledge_save` | Save or update knowledge |
| `knowledge_archive` | Archive a knowledge entry |

### GraphQL API

```graphql
# List knowledge by topic
query { knowledges(topic: "aws.cloudtrail") { slug, name, content } }

# Get topics with counts
query { knowledgeTopics { topic, count } }

# Save knowledge
mutation {
  saveKnowledge(input: {
    topic: "aws.cloudtrail"
    slug: "safe-s3-buckets"
    name: "Known Safe S3 Access"
    content: "..."
  }) { slug, topic }
}

# Archive
mutation {
  archiveKnowledge(input: { topic: "aws.cloudtrail", slug: "old-pattern" }) {
    slug, archived
  }
}
```

## Automatic Integration

Knowledge is automatically included in AI prompts in three scenarios:

### 1. Alert Enrichment

When processing alerts through enrichment policies, Warren includes topic-relevant knowledge in the system prompt.

### 2. Chat Analysis

When analysts chat with Warren about tickets, knowledge for the ticket's topic is included in the context.

### 3. Metadata Generation

When generating ticket titles and descriptions, knowledge informs the AI's understanding of organizational context.

## Best Practices

### Knowledge Granularity

One entry per concept:
- "Known Safe CI/CD Pipeline Activities"
- "Monthly Maintenance Window Patterns"

Avoid overly broad entries like "All AWS Exceptions".

### Topic Naming

- Use dot notation: `aws.cloudtrail` (not `aws_cloudtrail`)
- Match alert schemas for automatic integration
- Be specific but not too granular

### Size Management

Each topic has a **10KB limit** for all active knowledge.

Strategies:
- Be concise — focus on essential information
- Use links to external documentation
- Archive old entries
- Split into subtopics if needed

### Regular Review

- Review knowledge quarterly
- Archive outdated entries
- Update content with infrastructure changes

## Limitations

- **10KB per topic**: Archive old entries or split topics if exceeded
- **Append-only**: No permanent deletion, only logical archiving
- **No PII**: Avoid storing credentials or sensitive personal information
- **Prompt size**: Large knowledge bases may affect response times
