# Warren Policy Guide

Warren uses [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) policies powered by Open Policy Agent (OPA) to provide flexible and programmable control over alert processing, enrichment, and access control.

## Table of Contents

- [Overview](#overview)
- [Alert Lifecycle](#alert-lifecycle)
- [Policy Types](#policy-types)
  - [Alert Policy](#alert-policy)
  - [Enrich Policy](#enrich-policy)
  - [Commit Policy](#commit-policy)
  - [Authorization Policy](#authorization-policy)
- [Pipeline Architecture](#pipeline-architecture)
- [Getting Started](#getting-started)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Warren's policy system enables you to:

- **Transform** incoming security events into structured alerts
- **Enrich** alerts with additional context using AI and external tools
- **Control** alert routing, severity, and metadata
- **Authorize** API access with flexible authentication rules

All policies are written in Rego, a declarative language designed for expressing complex policy logic. Warren evaluates these policies at different stages of the alert processing pipeline.

## Alert Lifecycle

An alert in Warren goes through several stages from ingestion to notification:

```mermaid
flowchart TD
    A[Event Ingested] --> B[Stage 1: Alert Policy]
    B -->|Fill metadata by GenAI| D[Stage 2: Enrich Policy]
    D --> E[Stage 3: Commit Policy]
    E --> F[Alert Published]

    B -.->|Filter/Ignore| G[Event Dropped]

    subgraph B_box [" "]
        B
        B_desc["• Transforms raw event into Alert objects<br/>• Can filter/ignore events<br/>• Can generate multiple alerts from one event<br/>• Sets initial metadata (title, description, attrs)"]
    end

    subgraph D_box [" "]
        D
        D_desc["• Defines enrichment tasks (query/agent)<br/>• Tasks can query LLM or use AI agents with tools<br/>• Results stored in task ID map"]
    end

    subgraph E_box [" "]
        E
        E_desc["• Has access to alert + enrichment results<br/>• Can override metadata based on enrichment<br/>• Determines publish type (alert/notice/discard)<br/>• Sets final routing (channel)"]
    end

    style A fill:#e1f5ff
    style F fill:#d4edda
    style G fill:#f8d7da
    style B fill:#fff3cd
    style D fill:#d1ecf1
    style E fill:#d4f1d4
    style B_box fill:#fffef0
    style D_box fill:#f0f8ff
    style E_box fill:#f0fff0
```

### Key Lifecycle Concepts

1. **Immutability**: Once an alert is created, its core data doesn't change. Policies can set metadata, but the original event data remains intact.

2. **One Event, Multiple Alerts**: Alert policies can generate multiple alerts from a single event (e.g., processing an array of findings).

3. **Three Policy Stages**:
   - **Alert Policy**: Transforms raw events into alerts, filters unwanted events
   - **Enrich Policy**: Defines additional AI analysis tasks (query/agent)
   - **Commit Policy**: Makes final routing and publishing decisions based on enrichment results

4. **Progressive Enhancement**: Each policy stage adds more context:
   - Alert policy: Initial structure and filtering
   - Enrich policy: Additional investigation tasks
   - Commit policy: Final decision and routing

> **Note**: Warren automatically fills missing titles/descriptions with AI between Alert and Enrich policies. This is not controlled by policies.

## Policy Types

Warren uses four types of policies, each evaluated at different stages of the pipeline:

### Alert Policy

**Package**: `alert.{schema_name}`
**When**: First stage - transforms raw events into alerts
**Input**: Raw event data from webhook
**Output**: Alert metadata (title, description, attributes)

Alert policies define how external events become Warren alerts. The package name determines the webhook endpoint:

- Package `alert.guardduty` → Endpoint `/hooks/alert/raw/guardduty`
- Package `alert.custom` → Endpoint `/hooks/alert/raw/custom`

**Structure**:

```rego
package alert.{schema_name}

# Main rule - can generate multiple alerts
alert contains {
    "title": "Alert title",
    "description": "Alert description",
    "attrs": [
        {
            "key": "severity",
            "value": "high",
            "link": ""
        }
    ]
} if {
    # Conditions for alert creation
    input.severity >= 5
    not ignore
}

# Optional ignore rule for filtering
ignore if {
    input.source == "test"
}
```

**Key Points**:

- Use `alert contains` to generate alerts (can produce multiple from one event)
- Use `ignore` to filter out unwanted events
- If no policy exists, Warren creates a default alert and uses AI for metadata
- Attributes can include links to external tools (VirusTotal, IPinfo, etc.)

### Enrich Policy

**Package**: `enrich`
**When**: After alert creation and AI metadata generation
**Input**: Complete alert object with metadata
**Output**: Task definitions (query/agent types)

Enrich policies define additional analysis to perform on alerts using AI:

**Structure**:

```rego
package enrich

# Query tasks - simple LLM questions
query contains {
    "id": "check_ioc",
    "prompt": "threat_analysis.md",  # Prompt file path
    "format": "json"                 # or "text"
} if {
    input.schema == "guardduty"
}

# Agent tasks - AI agents with tool access
agent contains {
    "id": "investigate_ip",
    "inline": "Investigate the source IP address",  # Inline prompt
    "format": "text"
} if {
    has_external_ip
}

has_external_ip if {
    some attr in input.metadata.attributes
    attr.key == "source_ip"
    not startswith(attr.value, "10.")
}
```

**Task Types**:

1. **Query Tasks**: Simple LLM queries for analysis
   - Uses `gollem.GenerateContent()`
   - Fast, no tool access
   - Good for classification, summarization

2. **Agent Tasks**: AI agents with tool access
   - Uses `gollem.Agent` with tool execution
   - Can call security tools (VirusTotal, BigQuery, etc.)
   - Slower but more powerful

**Key Points**:

- Task IDs are used to reference results in commit policy
- Use `prompt` for file-based prompts, `inline` for simple text
- Format can be `"text"` or `"json"` (JSON enables structured parsing)
- Agent tasks have access to tools configured in Warren

### Commit Policy

**Package**: `commit`
**When**: After enrichment tasks complete
**Input**: Alert object + enrichment results map
**Output**: Final metadata overrides and publish decision

Commit policies make final decisions about alert handling based on original alert data and enrichment results:

**Structure**:

```rego
package commit

# Override title based on enrichment
title := sprintf("CONFIRMED THREAT: %s", [input.alert.metadata.title]) if {
    input.enrich.check_ioc.is_malicious == true
}

# Override description
description := input.enrich.check_ioc.analysis if {
    input.enrich.check_ioc.analysis
}

# Set notification channel
channel := "security-urgent" if {
    input.enrich.check_ioc.severity == "critical"
}

# Add attributes from enrichment
attr contains {
    "key": "threat_score",
    "value": input.enrich.check_ioc.score,
    "link": ""
} if {
    input.enrich.check_ioc.score
}

# Determine publish type
publish := "discard" if {
    input.enrich.check_ioc.is_false_positive == true
}

publish := "notice" if {
    input.alert.metadata.severity == "low"
}

# Default: publish as full alert
publish := "alert"
```

**Publish Types**:

- `"alert"` (default): Full alert with ticket creation and Slack thread
- `"notice"`: Simple notification only, no ticket
- `"discard"`: Drop the alert, no notification

**Input Structure**:

```json
{
  "alert": {
    "id": "alert-123",
    "schema": "guardduty",
    "metadata": {
      "title": "Alert title",
      "description": "Alert description",
      "attributes": [...]
    },
    "data": { /* original event */ }
  },
  "enrich": {
    "check_ioc": {
      "is_malicious": true,
      "score": "8.5",
      "analysis": "..."
    },
    "investigate_ip": "..."
  }
}
```

**Key Points**:

- Access enrichment results via `input.enrich.{task_id}`
- Can override any metadata field
- Use `publish` to control alert disposition
- Default behavior is to publish as full alert

### Authorization Policy

**Package**: `auth`
**When**: On every API request
**Input**: Request context (IAP, Google ID, SNS, headers, env)
**Output**: `allow = true` or `false`

Authorization policies control access to Warren's API endpoints:

**Structure**:

```rego
package auth

default allow = false

# Allow authenticated users from company domain
allow if {
    input.iap.email
    endswith(input.iap.email, "@example.com")
}

# Allow service accounts
allow if {
    input.google.email == "monitoring@project.iam.gserviceaccount.com"
}

# Allow webhook with token
allow if {
    startswith(input.req.path, "/hooks/alert/")
    input.req.header.Authorization[0] == sprintf("Bearer %s", [input.env.WARREN_WEBHOOK_TOKEN])
}
```

**Context Available**:

- `input.iap.*`: Google IAP JWT claims
- `input.google.*`: Google ID token claims
- `input.sns.*`: AWS SNS message data
- `input.req.*`: HTTP request details (method, path, headers, body)
- `input.env.*`: All environment variables

See the original policy.md sections on Authorization Policies for detailed examples and context structure.

## Pipeline Architecture

Warren's alert processing pipeline is implemented in `pkg/usecase/alert_pipeline.go`:

```mermaid
flowchart TB
    Start[ProcessAlertPipeline] --> A[EvaluateAlertPolicy]
    A --> B{For each alert}

    B --> C[ConvertNamesToTags]
    C --> D[FillMetadata]
    D --> E[EvaluateEnrichPolicy]
    E --> F[ExecuteTasks]
    F --> G[EvaluateCommitPolicy]

    G --> H{More alerts?}
    H -->|Yes| B
    H -->|No| I[Return AlertPipelineResult]

    A -.-> A1[Alerts]
    C -.-> C1[Tag IDs]
    D -.-> D1[AI title/desc/embedding]
    E -.-> E1[EnrichPolicyResult]
    F -.-> F1[EnrichResults]
    G -.-> G1[CommitPolicyResult]

    style Start fill:#e1f5ff
    style I fill:#d4edda
    style B fill:#fff3cd
    style H fill:#fff3cd
```

**Event Notifications**:

The pipeline emits events through the `Notifier` interface for real-time monitoring:

- `NotifyAlertPolicyResult`: Alert policy evaluation complete
- `NotifyEnrichPolicyResult`: Enrich policy evaluation complete
- `NotifyEnrichTaskPrompt`: Enrichment task prompt being sent
- `NotifyEnrichTaskResponse`: Enrichment task response received
- `NotifyCommitPolicyResult`: Commit policy evaluation complete
- `NotifyError`: Error occurred during processing

These events power:
- Console output (colored, formatted logs)
- Slack thread updates (real-time pipeline progress)
- Debugging and observability

**Pure Pipeline vs. Full Handling**:

- `ProcessAlertPipeline()`: Pure function, no side effects (no DB, no Slack)
- `HandleAlert()`: Complete handling including DB save and Slack posting

This separation enables testing and reusability.

## Getting Started

### 1. Basic Alert Policy

Start with a simple alert policy that passes events through:

```rego
package alert.myservice

alert contains {
    "title": input.title,
    "description": input.message,
    "attrs": []
} if {
    input.severity != "info"
}
```

Save as `policies/alert/myservice.rego` and send events to `/hooks/alert/raw/myservice`.

### 2. Add AI Enrichment

Define enrichment tasks for additional analysis:

```rego
package enrich

query contains {
    "id": "analyze_severity",
    "inline": "Based on the alert data, is this a true security threat or false positive? Respond with JSON: {\"is_threat\": boolean, \"confidence\": number, \"reasoning\": string}",
    "format": "json"
} if {
    input.schema == "myservice"
}
```

Save as `policies/enrich/enrich.rego`.

### 3. Make Routing Decisions

Use commit policy to route based on enrichment:

```rego
package commit

# Route high-confidence threats to urgent channel
channel := "security-urgent" if {
    input.enrich.analyze_severity.is_threat == true
    input.enrich.analyze_severity.confidence > 0.8
}

# Discard confirmed false positives
publish := "discard" if {
    input.enrich.analyze_severity.is_threat == false
    input.enrich.analyze_severity.confidence > 0.9
}

# Default to normal channel
channel := "security-alerts"
publish := "alert"
```

Save as `policies/commit/commit.rego`.

### 4. Test Your Policies

Create test data and verify behavior:

```bash
warren test \
  --policy ./policies \
  --test-detect-data ./test/myservice/detect \
  --test-ignore-data ./test/myservice/ignore
```

## Best Practices

### Alert Policy Best Practices

1. **Filter Early**: Use `ignore` rules to drop noise before AI processing
2. **Structured Attributes**: Extract key fields as attributes for filtering/clustering
3. **Useful Links**: Add links to external tools (VirusTotal, IPinfo, AWS Console)
4. **Handle Arrays**: Use `event := input.Records[_]` pattern for batch events
5. **Validate Input**: Check required fields exist before accessing them

### Enrich Policy Best Practices

1. **Use Query for Simple Tasks**: Query tasks are faster than agents
2. **Request JSON Format**: Structured responses are easier to parse in commit policy
3. **Limit Agent Tasks**: Agents are powerful but slower - use sparingly
4. **Descriptive Task IDs**: Use meaningful IDs like `check_threat_intel`, not `task1`
5. **Conditional Enrichment**: Only run tasks when relevant (check alert schema/attributes)

### Commit Policy Best Practices

1. **Default to Alert**: Always provide default `publish = "alert"` and `channel`
2. **Conservative Discarding**: Only discard with high confidence
3. **Preserve Context**: When overriding title/description, include original context
4. **Use Enrichment Wisely**: Don't blindly trust AI - validate confidence scores
5. **Test Edge Cases**: Verify behavior when enrichment tasks fail or return unexpected data

### General Policy Best Practices

1. **Version Control**: Store policies in Git
2. **Test Thoroughly**: Create test cases for both positive and negative scenarios
3. **Document Decisions**: Comment why certain rules exist
4. **Monitor Performance**: Watch for slow policies in logs
5. **Iterate Gradually**: Start simple, add complexity as needed

## Examples

### Complete GuardDuty Pipeline

**Alert Policy** (`policies/alert/guardduty.rego`):

```rego
package alert.guardduty

alert contains {
    "title": sprintf("%s in %s", [input.detail.type, input.detail.region]),
    "description": input.detail.description,
    "attrs": [
        {
            "key": "severity",
            "value": severity_label,
            "link": ""
        },
        {
            "key": "account",
            "value": input.detail.accountId,
            "link": ""
        },
        {
            "key": "finding_id",
            "value": input.detail.id,
            "link": sprintf("https://console.aws.amazon.com/guardduty/home?region=%s#/findings?search=id%%3D%s", [
                input.detail.region,
                input.detail.id
            ])
        }
    ]
} if {
    input.source == "aws.guardduty"
    input.detail.severity >= 4.0  # Medium and above
}

severity_label := "critical" if { input.detail.severity >= 8.0 }
else := "high" if { input.detail.severity >= 6.0 }
else := "medium"
```

**Enrich Policy** (`policies/enrich/enrich.rego`):

```rego
package enrich

# Analyze GuardDuty findings with AI
query contains {
    "id": "analyze_finding",
    "inline": "Analyze this GuardDuty finding. Provide: 1) Is this a real threat or false positive? 2) Recommended actions. 3) Urgency level. Respond in JSON: {\"is_threat\": boolean, \"confidence\": number, \"actions\": string[], \"urgency\": string}",
    "format": "json"
} if {
    input.schema == "guardduty"
}

# Use agent to investigate external IPs
agent contains {
    "id": "investigate_ip",
    "inline": "Investigate the remote IP address in this GuardDuty finding using available threat intelligence tools. Summarize your findings.",
    "format": "text"
} if {
    input.schema == "guardduty"
    has_remote_ip
}

has_remote_ip if {
    some attr in input.metadata.attributes
    attr.key == "remote_ip"
}
```

**Commit Policy** (`policies/commit/commit.rego`):

```rego
package commit

# Override title for confirmed threats
title := sprintf("🚨 CONFIRMED THREAT: %s", [input.alert.metadata.title]) if {
    input.enrich.analyze_finding.is_threat == true
    input.enrich.analyze_finding.confidence > 0.85
}

# Add threat intelligence findings
attr contains {
    "key": "threat_intel",
    "value": input.enrich.investigate_ip,
    "link": ""
} if {
    input.enrich.investigate_ip
}

# Route based on urgency
channel := "security-critical" if {
    input.enrich.analyze_finding.urgency == "critical"
}

channel := "security-urgent" if {
    input.enrich.analyze_finding.urgency == "high"
}

# Discard false positives
publish := "discard" if {
    input.enrich.analyze_finding.is_threat == false
    input.enrich.analyze_finding.confidence > 0.9
}

# Default routing
channel := "security-alerts"
publish := "alert"
```

### Severity-Based Routing

Simple commit policy for routing by severity:

```rego
package commit

channel := "security-critical" if {
    input.alert.metadata.severity == "critical"
}

channel := "security-high" if {
    input.alert.metadata.severity == "high"
}

# Send low severity as notices only
publish := "notice" if {
    input.alert.metadata.severity == "low"
}

channel := "security-info" if {
    input.alert.metadata.severity == "low"
}

# Default
channel := "security-alerts"
publish := "alert"
```

## Additional Resources

- [Open Policy Agent Documentation](https://www.openpolicyagent.org/docs/latest/)
- [Rego Language Reference](https://www.openpolicyagent.org/docs/latest/policy-reference/)
- [Rego Playground](https://play.openpolicyagent.org/) - Test policies online
- [Warren GitHub Repository](https://github.com/secmon-lab/warren) - Source code and examples

## Troubleshooting

### Policy Not Loading

**Symptoms**: Warren starts but policies don't seem to be evaluated

**Solutions**:
1. Check `WARREN_POLICY` environment variable points to policy directory
2. Verify file permissions (policies must be readable)
3. Check syntax: `opa check policies/`
4. Look for errors in Warren startup logs

### Enrichment Tasks Not Running

**Symptoms**: Commit policy receives empty `input.enrich`

**Solutions**:
1. Verify enrich policy package is exactly `enrich` (not `enrich.something`)
2. Check task conditions - they may not match your alert
3. Ensure `WARREN_LLM_*` environment variables are set
4. Check Warren logs for task execution errors

### Commit Policy Not Applying

**Symptoms**: Alert metadata doesn't reflect commit policy changes

**Solutions**:
1. Verify commit policy package is exactly `commit`
2. Check that field names match exactly (`title`, `description`, `channel`, `attr`, `publish`)
3. Use `print()` statements to debug: `print("Setting channel:", channel)`
4. Verify enrichment task IDs match: `input.enrich.{task_id}`

### Authorization Always Denied

**Symptoms**: API requests fail with 403 Forbidden

**Solutions**:
1. Check auth policy package is exactly `auth` (not `auth.api`)
2. Verify `default allow = false` has at least one `allow if` rule
3. Debug context: Add `print("Auth input:", input)` to see what's available
4. Start permissive: `allow = true` temporarily to test (remove in production!)
5. Check that `WARREN_NO_AUTHORIZATION` flag is NOT set (removes auth entirely)

For more detailed troubleshooting and examples, see the sections in the original policy documentation.
