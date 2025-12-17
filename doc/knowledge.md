# Warren Knowledge Management

## Overview

Knowledge Management in Warren allows you to capture and preserve organizational security expertise that informs alert triage and incident response. The knowledge system automatically integrates domain-specific information into AI-powered analysis, helping Warren provide more accurate and context-aware recommendations.

**Why Knowledge Management?**
- **Reduce False Positives**: Document known benign patterns that trigger alerts
- **Preserve Expertise**: Capture security team knowledge for consistent decision-making
- **Context-Aware Analysis**: Automatically provide relevant information to AI analysis
- **Continuous Improvement**: Learn from past incidents and encode lessons for future use

**Key Benefits:**
- Automatic integration into alert enrichment and chat analysis
- Topic-based organization for easy maintenance
- Version history to track changes over time
- Collaborative knowledge building across your security team

## Core Concepts

### Topic

A **Topic** is a category or domain that groups related knowledge together. Topics are automatically assigned to alerts based on their schema and inherited by tickets.

**Topic Structure:**
```
aws.cloudtrail          # AWS CloudTrail events
okta.logs               # Okta authentication logs
github.audit            # GitHub audit events
custom.application      # Custom application logs
```

**Topic Assignment:**
1. Alerts receive a topic from their schema (e.g., `aws.cloudtrail`)
2. Tickets inherit the topic from their first alert
3. Knowledge is organized by topic for automatic retrieval

### Knowledge

A **Knowledge** entry contains specific information about a topic. Each knowledge has:
- **Name**: Short, descriptive title (e.g., "Known Safe S3 Bucket Access")
- **Slug**: Unique identifier within the topic (e.g., `safe-s3-buckets`)
- **Content**: Detailed information in Markdown format
- **Topic**: The topic this knowledge belongs to
- **Archived**: Whether the knowledge is still active

**Example Knowledge Entry:**
```markdown
**Name:** Known Safe S3 Bucket Access
**Slug:** safe-s3-buckets
**Topic:** aws.cloudtrail
**Content:**
The following S3 buckets are accessed daily by our data pipeline and should not trigger alerts:
- `company-logs-archive` - Accessed by log aggregator (10.0.1.50)
- `company-backups` - Accessed by backup system (10.0.1.51)

These access patterns occur between 00:00-06:00 UTC and are expected behavior.
```

### Version Management

Warren uses an append-only model for knowledge:
- Creating or updating knowledge adds a new version
- Previous versions remain in history
- Archiving marks knowledge as inactive (logical delete)
- Version history allows tracking changes over time

## Topic Usage

### Automatic Topic Assignment

Topics are automatically assigned based on alert schemas:

```
Alert Schema → Topic
aws.cloudtrail → aws.cloudtrail
okta.system → okta.system
custom.app → custom.app
```

### Topic Naming Best Practices

1. **Use dot notation** for hierarchical organization:
   - `aws.cloudtrail` (good)
   - `aws_cloudtrail` (avoid)

2. **Match alert schemas** when possible for automatic integration

3. **Be specific but not too granular**:
   - `aws.cloudtrail` (good - covers all CloudTrail events)
   - `aws.cloudtrail.s3.getobject` (too specific)

4. **Use consistent naming**:
   - `aws.*` for AWS services
   - `okta.*` for Okta events
   - `github.*` for GitHub events

## Knowledge CRUD Operations

### WebUI Operations

Warren provides a web interface for managing knowledge at `http://your-warren-instance/knowledge`.

**Viewing Knowledge:**
1. Navigate to `/knowledge` to see all topics and their knowledge counts
2. Click on a topic to view all knowledge entries for that topic
3. View version history for any knowledge entry

**Creating Knowledge:**
1. Click "New Knowledge" button
2. Select or enter a topic
3. Provide a name (title) and slug (identifier)
4. Write content in Markdown format
5. Click "Save"

**Editing Knowledge:**
1. Click "Edit" on an existing knowledge entry
2. Modify name and/or content (slug cannot be changed)
3. Click "Save" to create a new version

**Archiving Knowledge:**
1. Click "Archive" on a knowledge entry
2. Confirm the archive action
3. Archived knowledge no longer appears in active listings but remains in history

### Slack Operations

Warren's Slack bot provides commands for knowledge management within any Slack thread.

**List Knowledge for a Topic:**
```
knowledge_list <topic>
```
Example:
```
knowledge_list aws.cloudtrail
```

**Get Specific Knowledge:**
```
knowledge_get <topic> <slug>
```
Example:
```
knowledge_get aws.cloudtrail safe-s3-buckets
```

**Save Knowledge:**
```
knowledge_save <topic> <slug> <name>
Content goes here in markdown format
```
Example:
```
knowledge_save aws.cloudtrail safe-s3-buckets "Known Safe S3 Access"
The following buckets are accessed by automation:
- company-logs-archive
- company-backups
```

**Archive Knowledge:**
```
knowledge_archive <topic> <slug>
```
Example:
```
knowledge_archive aws.cloudtrail old-pattern
```

### GraphQL API Operations

For programmatic access, Warren provides a GraphQL API.

**Query Knowledge by Topic:**
```graphql
query {
  knowledges(topic: "aws.cloudtrail") {
    slug
    name
    content
    topic
    archived
    createdAt
    updatedAt
  }
}
```

**Get Topics with Counts:**
```graphql
query {
  knowledgeTopics {
    topic
    count
  }
}
```

**Create Knowledge:**
```graphql
mutation {
  saveKnowledge(input: {
    topic: "aws.cloudtrail"
    slug: "safe-s3-buckets"
    name: "Known Safe S3 Bucket Access"
    content: "The following S3 buckets..."
  }) {
    slug
    topic
  }
}
```

**Archive Knowledge:**
```graphql
mutation {
  archiveKnowledge(input: {
    topic: "aws.cloudtrail"
    slug: "old-pattern"
  }) {
    slug
    archived
  }
}
```

## Automatic Prompt Integration

Knowledge is automatically integrated into AI analysis prompts in three scenarios:

### 1. Alert Enrichment (Enrich Policy)

When alerts are processed through enrichment policies, Warren automatically:
1. Retrieves knowledge for the alert's topic
2. Includes knowledge in the system prompt
3. Provides context to LLM tasks

**Example:**
```rego
# In your enrich policy
package alert.aws.cloudtrail

prompt "analyze_s3_access" {
  inline = "Analyze this S3 access and determine if it's suspicious"
  format = "text"
}
```

Warren automatically adds knowledge for `aws.cloudtrail` topic to the prompt, so the LLM knows about known safe buckets and access patterns.

### 2. Chat Analysis

When security analysts chat with Warren about tickets:
1. Warren retrieves knowledge for the ticket's topic
2. Knowledge is included in the chat system prompt
3. AI responses incorporate organizational knowledge

**Example Chat:**
```
User: Is this S3 access normal?
Warren: Yes, this access to company-logs-archive from 10.0.1.50 is part of our
        documented data pipeline. This is a known safe pattern that runs daily
        between 00:00-06:00 UTC.
```

### 3. Ticket Metadata Generation

When Warren generates ticket titles and descriptions:
1. Knowledge for the ticket's topic is retrieved
2. Information informs the AI's understanding
3. Titles and descriptions reflect organizational context

**Before Knowledge:**
```
Title: "AWS S3 GetObject Access Detected"
Description: "Multiple S3 GetObject operations detected from IP 10.0.1.50"
```

**After Knowledge:**
```
Title: "Expected Log Aggregator S3 Access"
Description: "Routine S3 access by data pipeline to company-logs-archive (documented pattern)"
```

## Use Cases and Best Practices

### Use Case Examples

#### 1. Known False Positive Patterns
```markdown
**Name:** Okta Admin Tool Access
**Slug:** okta-admin-tool
**Topic:** okta.system
**Content:**
Our IT automation tool (user: automation@company.com) performs these actions hourly:
- User role changes
- Group membership updates
- Permission modifications

These are expected and should not trigger high-severity alerts.
Source: IT-2023-0451
```

#### 2. Environment-Specific Behavior
```markdown
**Name:** Staging Environment Test Activities
**Slug:** staging-test-activities
**Topic:** aws.cloudtrail
**Content:**
Staging environment (account: 123456789012) runs automated security tests that generate:
- Failed authentication attempts (expected rate: ~100/hour)
- Privilege escalation attempts (test scenarios)
- Unusual API patterns (penetration testing)

These are part of our continuous security validation process.
Review schedule: First Monday of each month
Contact: security-testing@company.com
```

#### 3. Documented Incident Response
```markdown
**Name:** Similar Phishing Campaign 2024-Q1
**Slug:** phishing-2024-q1
**Topic:** email.suspicious
**Content:**
In Q1 2024, we experienced a similar phishing campaign:
- Indicators: Emails with "Invoice Update" subject from fake-accounting domains
- Impact: 3 users clicked, no credentials compromised
- Resolution: Email filter updated, user training conducted

**Key Learnings:**
- Check sender domain reputation immediately
- Look for typosquatting (accunting vs accounting)
- Correlate with email gateway logs

Incident Report: INC-2024-0123
```

#### 4. Business Process Documentation
```markdown
**Name:** Monthly Financial Close Activities
**Slug:** monthly-financial-close
**Topic:** aws.cloudtrail
**Content:**
During monthly financial close (last 3 days of month), expect:
- High volume database queries from finance-reports user
- Bulk S3 uploads to company-financial-reports bucket
- Extended CloudTrail session durations (up to 6 hours)

These activities are pre-approved (FIN-PROC-001) and monitored by finance team.
Review with: finance-systems@company.com
```

### Best Practices

#### Knowledge Granularity
✅ **Good**: One knowledge entry per concept
```
✓ "Known Safe CI/CD Pipeline Activities"
✓ "Monthly Maintenance Window Patterns"
✓ "Development Environment Access Patterns"
```

❌ **Avoid**: Multiple unrelated concepts in one entry
```
✗ "All AWS Exceptions" (too broad)
✗ "Various Safe Patterns" (too vague)
```

#### Regular Review and Updates
- **Schedule**: Review knowledge quarterly
- **Archive**: Remove outdated entries
- **Update**: Keep content current with infrastructure changes

**Review Checklist:**
- [ ] Is this pattern still occurring?
- [ ] Have IP addresses or accounts changed?
- [ ] Is the contact person still correct?
- [ ] Should this be archived?

#### Descriptive Naming
✅ **Good Slugs:**
```
safe-s3-buckets          # Clear what it covers
known-admin-ips          # Specific and descriptive
staging-test-patterns    # Indicates scope
```

❌ **Avoid:**
```
misc                     # Too vague
temp                     # Unclear purpose
fix-123                  # Non-descriptive
```

#### Size Management

Each topic has a 10KB limit for all active knowledge combined.

**Strategies for Size Management:**
1. **Be Concise**: Focus on essential information
2. **Use References**: Link to external documentation instead of duplicating
3. **Archive Old Entries**: Remove outdated knowledge
4. **Split Topics**: Use subtopics if needed (aws.cloudtrail.s3, aws.cloudtrail.iam)

**Example - Concise Format:**
```markdown
## Safe IP Addresses
- 10.0.1.50 - Log aggregator (daily 00:00-06:00 UTC)
- 10.0.1.51 - Backup system (nightly)

Details: https://wiki.company.com/security/safe-ips
```

## Limitations and Considerations

### Topic Size Limit
- **Maximum**: 10KB of active knowledge per topic
- **Calculation**: Sum of all active knowledge content lengths
- **Impact**: Exceeding limit prevents adding new knowledge
- **Solution**: Archive old knowledge or split into subtopics

### Append-Only Model
- **Deletions**: No permanent deletion, only logical archiving
- **History**: All versions retained indefinitely
- **Privacy**: Sensitive information remains in history even when archived
- **Consideration**: Avoid storing credentials or PII

### Prompt Size Impact
- Knowledge is included in AI prompts
- Large knowledge bases may affect response times
- Best practice: Keep knowledge focused and relevant

### Synchronization
- Knowledge changes are immediate
- Active AI sessions use knowledge from when they started
- New sessions get latest knowledge automatically

## Troubleshooting

### Knowledge Not Appearing in Analysis

**Problem**: Knowledge doesn't seem to influence AI responses

**Solutions:**
1. **Check Topic Matches**:
   - Verify alert/ticket topic matches knowledge topic exactly
   - Topics are case-sensitive: `aws.cloudtrail` ≠ `AWS.CloudTrail`

2. **Verify Knowledge is Active**:
   - Check knowledge isn't archived
   - View knowledge list to confirm it exists

3. **Review Content**:
   - Knowledge may be present but not relevant to the specific query
   - AI may prioritize other information over knowledge

**Debug Steps:**
```bash
# Check what topic an alert has
warren run analyze <alert-file> --debug

# List knowledge for a topic
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ knowledges(topic: \"aws.cloudtrail\") { name slug } }"}'
```

### Size Limit Error

**Problem**: "Topic knowledge size limit exceeded" error when saving

**Solutions:**
1. **Archive Unused Knowledge**:
   ```
   knowledge_archive aws.cloudtrail old-pattern-1
   knowledge_archive aws.cloudtrail old-pattern-2
   ```

2. **Condense Content**:
   - Remove redundant information
   - Use links instead of full documentation
   - Summarize verbose descriptions

3. **Split Topic**:
   - Create subtopics (e.g., `aws.cloudtrail.s3`, `aws.cloudtrail.iam`)
   - Distribute knowledge across subtopics

### Version History Issues

**Problem**: Need to see what changed in knowledge

**Solution:**
```graphql
query {
  knowledge(topic: "aws.cloudtrail", slug: "safe-s3-buckets") {
    versions {
      version
      name
      content
      createdAt
      archived
    }
  }
}
```

**Reverting Changes:**
1. View version history
2. Copy content from previous version
3. Save as new version with that content

## Related Documentation

- [User Guide](./user_guide.md) - Basic Warren operations and workflows
- [AI Agent Guide](./agent.md) - Using AI-powered analysis and chat features
- [Policy Guide](./policy.md) - Writing alert policies that work with knowledge
- [Model Documentation](./model.md) - Understanding alerts, tickets, and topics
