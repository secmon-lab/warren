# Policy Guide

Warren uses [Rego](https://www.openpolicyagent.org/docs/latest/policy-language/) policies to control alert detection and API authorization. This guide explains how to write, test, and manage policies effectively.

## Introduction to Rego

Rego is a declarative policy language designed for expressing complex logic. In Warren:
- **Alert policies** determine which events become security alerts
- **Authorization policies** control API access

Key concepts:
- Policies are collections of rules
- Rules generate data or make decisions
- Input data is available as `input`
- Rules can reference other rules

## Alert Detection Policies

Alert policies transform incoming events into Warren alerts with structured metadata.

### When to Use Alert Policies

While Warren automatically enriches alerts with AI-generated titles, descriptions, and attributes, you should create custom policies when you need:

1. **Deterministic Processing**: Apply predefined rules for consistent alert formatting
2. **Conditional Filtering**: Ignore alerts based on specific criteria (e.g., known false positives)
3. **Alert Multiplication**: Generate multiple alerts from a single event
4. **Custom Enrichment**: Add specific attributes or links before AI processing
5. **Compliance Requirements**: Ensure specific fields are always present for audit purposes

### Policy Evaluation Flow

```mermaid
flowchart TD
    A[Incoming Event] --> B{Policy Exists?}
    B -->|Yes| C[Policy Evaluation]
    B -->|No| D[Direct to AI Enrichment]
    
    C --> E{Alert Generated?}
    E -->|Yes| F[Policy-defined Alert]
    E -->|No/Ignored| G[Event Dropped]
    
    F --> H{Has title/description?}
    H -->|Yes| I[Send to Warren]
    H -->|No| J[AI Enrichment]
    
    D --> J
    J --> K[AI-generated title/description/attrs]
    K --> I
    
    style A fill:#f9f,stroke:#333,stroke-width:2px
    style I fill:#9f9,stroke:#333,stroke-width:2px
    style G fill:#faa,stroke:#333,stroke-width:2px
```

**Key Points:**
- If a policy exists for the webhook path, it's evaluated first
- Policy can filter (ignore) events before they reach AI
- Policy-generated alerts can skip AI enrichment if complete
- Events without policies go directly to AI enrichment

### Structure

Alert policies follow this pattern:

```rego
package alert.schema_name

# Main detection rule
alert contains {
    "title": "Alert Title",
    "description": "Alert Description",
    "attrs": [
        {
            "key": "attribute_name",
            "value": "attribute_value",
            "link": "optional_url"
        }
    ]
} if {
    # Conditions for alert creation
    not ignore
    # Additional conditions...
}

# Ignore rules to filter alerts
ignore if {
    # Conditions to ignore
}
```

### Package Naming

The package name determines the webhook endpoint:
- Package: `alert.guardduty`
- Endpoint: `/hooks/alert/raw/guardduty`

### Webhook Endpoint Examples

**Important**: Warren doesn't have predefined webhook endpoints. You create endpoints by defining policy packages. The package name determines the webhook URL.

Here are some **example patterns** you might use:

| Example Use Case | Package Name | Resulting Webhook Endpoint |
|-----------------|--------------|----------------------------|
| AWS GuardDuty | `alert.guardduty` | `/hooks/alert/raw/guardduty` |
| Custom monitoring | `alert.monitoring` | `/hooks/alert/raw/monitoring` |
| Security scanner | `alert.scanner` | `/hooks/alert/raw/scanner` |
| Your app alerts | `alert.myapp` | `/hooks/alert/raw/myapp` |

**How it works**:
1. You create a policy file with package name (e.g., `package alert.myapp`)
2. Warren automatically creates the webhook endpoint
3. Send your tool's webhooks to that endpoint
4. Your policy processes the incoming data

**Special Integration Patterns**:

| Integration Type | Endpoint Pattern | Use Case |
|-----------------|------------------|----------|
| AWS SNS | `/hooks/alert/sns/{name}` | For AWS services via SNS topics |
| Google Pub/Sub | `/hooks/alert/pubsub/{name}` | For GCP services via Pub/Sub |

**Remember**: These are just examples. You can name your packages anything that makes sense for your security tools and workflows.

### Input Data

The `input` variable contains the entire JSON payload sent to the webhook. The structure depends on your alert source.

Example GuardDuty input:
```json
{
  "Findings": [
    {
      "Title": "Unusual API calls",
      "Description": "API calls from unusual location",
      "Severity": 5.0,
      "Type": "Stealth:IAMUser/AnomalousBehavior",
      "Region": "us-east-1",
      "Resource": {
        "Type": "AccessKey",
        "AccessKeyDetails": {
          "UserName": "admin"
        }
      }
    }
  ]
}
```

### Writing Rules

#### Basic Alert Rule

```rego
package alert.custom

alert contains {
    "title": input.title,
    "description": input.description,
    "attrs": []
} if {
    input.severity >= "medium"
}
```

#### With Attributes

```rego
package alert.webapp

alert contains {
    "title": sprintf("Security Alert: %s", [input.event_type]),
    "description": input.message,
    "attrs": [
        {
            "key": "severity",
            "value": upper(input.severity),
            "link": ""
        },
        {
            "key": "source_ip",
            "value": input.client_ip,
            "link": sprintf("https://ipinfo.io/%s", [input.client_ip])
        },
        {
            "key": "user",
            "value": input.username,
            "link": ""
        }
    ]
} if {
    input.severity in ["high", "critical"]
    not is_internal_ip(input.client_ip)
}

# Helper function
is_internal_ip(ip) if {
    startswith(ip, "10.")
}

is_internal_ip(ip) if {
    startswith(ip, "192.168.")
}
```

#### Processing Arrays

For sources that send multiple events:

```rego
package alert.cloudtrail

# Process each event in the array
alert contains {
    "title": sprintf("AWS %s by %s", [event.eventName, event.userIdentity.userName]),
    "description": event.errorMessage,
    "attrs": build_attrs(event)
} if {
    event := input.Records[_]
    is_suspicious(event)
}

is_suspicious(event) if {
    event.errorCode != null
    event.eventName in ["DeleteBucket", "DeleteDBInstance", "TerminateInstances"]
}

build_attrs(event) = attrs if {
    attrs := [
        {
            "key": "event_name",
            "value": event.eventName,
            "link": ""
        },
        {
            "key": "aws_region",
            "value": event.awsRegion,
            "link": ""
        },
        {
            "key": "source_ip",
            "value": event.sourceIPAddress,
            "link": sprintf("https://ipinfo.io/%s", [event.sourceIPAddress])
        }
    ]
}
```

### Using Ignore Patterns

The ignore pattern helps reduce noise:

```rego
package alert.securityscanner

alert contains {
    "title": sprintf("Port Scan Detected: %s", [input.target_host]),
    "description": sprintf("%d ports scanned on %s", [input.port_count, input.target_host]),
    "attrs": [
        {
            "key": "source_ip",
            "value": input.source_ip,
            "link": ""
        },
        {
            "key": "ports_scanned",
            "value": to_string(input.port_count),
            "link": ""
        }
    ]
} if {
    not ignore
    input.port_count > 100
}

# Ignore authorized scanners
ignore if {
    input.source_ip in ["10.0.1.50", "10.0.1.51"]
}

# Ignore low port counts during business hours
ignore if {
    input.port_count < 1000
    current_hour := time.clock(time.now_ns())[0]
    current_hour >= 9
    current_hour <= 17
}
```

## Authorization Policies

Authorization policies control access to Warren's APIs.

### Understanding Warren's Authorization Flow

When `WARREN_NO_AUTHORIZATION` flag is removed (Level 2 and above), every API request goes through policy evaluation:

1. HTTP request is received
2. Context is built from request data, authentication tokens, and environment
3. Policy engine evaluates `data.auth` with the context
4. Request is allowed only if `allow = true`

### Authorization Flow Sequence

The following sequence diagram shows how Warren processes authorization for API requests:

```mermaid
sequenceDiagram
    participant Client
    participant Warren HTTP Server
    participant Auth Middleware
    participant Policy Engine
    participant Request Handler
    
    Client->>Warren HTTP Server: HTTP Request
    Note over Client: Headers, Body, etc.
    
    Warren HTTP Server->>Auth Middleware: Process Request
    
    Auth Middleware->>Auth Middleware: Extract Auth Context
    Note over Auth Middleware: Google IAP JWT<br/>Google ID Token<br/>SNS Message<br/>HTTP Request
    
    Auth Middleware->>Auth Middleware: Build Context
    Note over Auth Middleware: Context includes:<br/>- google: {}<br/>- iap: {}<br/>- sns: {}<br/>- req: {}<br/>- env: {}
    
    Auth Middleware->>Policy Engine: Query("data.auth", context)
    
    Policy Engine->>Policy Engine: Evaluate auth.rego
    Note over Policy Engine: Check rules against<br/>provided context
    
    alt Authorization Allowed
        Policy Engine-->>Auth Middleware: {allow: true}
        Auth Middleware->>Request Handler: Continue Request
        Request Handler-->>Client: HTTP Response (2xx)
    else Authorization Denied
        Policy Engine-->>Auth Middleware: {allow: false}
        Auth Middleware-->>Client: HTTP 403 Forbidden
        Note over Client: "Authorization failed.<br/>Check your policy."
    else Policy Error
        Policy Engine-->>Auth Middleware: Error
        Auth Middleware-->>Client: HTTP 500 Error
    end
```

**Key Points:**

1. **Context Building**: The middleware automatically extracts authentication information from various sources
2. **Policy Query**: The entire context is passed to the policy engine as `input`
3. **Binary Decision**: The policy must return `allow = true` for the request to proceed
4. **Error Handling**: Both policy errors and denials are handled appropriately

### Context Structure

Authorization policies receive rich context:

```json
{
  "google": {         // Google ID token claims (when present)
    "iss": "https://accounts.google.com",
    "sub": "1234567890",                    // User's unique ID
    "aud": "client-12345.apps.googleusercontent.com",
    "exp": 1311281970,                      // Expiration timestamp
    "iat": 1311280970,                      // Issued at timestamp
    "email": "user@example.com",
    "email_verified": true,
    "name": "Jane Doe",
    "given_name": "Jane",
    "family_name": "Doe",
    "picture": "https://lh3.googleusercontent.com/...",
    "locale": "en",
    "hd": "example.com",                    // Hosted domain (G Suite/Workspace)
    "nonce": "...",                         // If nonce was provided
    "auth_time": 1311280970                 // Authentication time
  },
  "iap": {            // Google IAP JWT claims (when present)
    "iss": "https://cloud.google.com/iap",
    "sub": "accounts.google.com:1234567890", // User's unique ID
    "aud": "/projects/123456/apps/warren",   // Your app's audience
    "exp": 1311281970,                       // Expiration timestamp
    "iat": 1311280970,                       // Issued at timestamp
    "email": "user@example.com",
    "email_verified": true,
    "gcip": {                                // External identity provider data (if used)
      "auth_time": 1553219869,
      "email": "user@example.com",
      "email_verified": false,
      "firebase": {
        "identities": {},
        "sign_in_provider": "password"
      },
      "name": "User Name",
      "picture": "https://...",
      "sub": "..."
    }
  },
  "sns": {            // AWS SNS message data (when present)
    "Type": "Notification",                  // or "SubscriptionConfirmation"
    "MessageId": "22b80b92-fdea-4c2c-8f9d-bdfb0c7bf324",
    "Token": "...",                         // For subscription confirmation
    "TopicArn": "arn:aws:sns:us-east-1:123456789012:MyTopic",
    "Subject": "My Subject",                // Optional
    "Message": "{\"default\":\"Hello\"}",  // The actual message content
    "Timestamp": "2023-01-01T12:00:00.000Z",
    "SignatureVersion": "1",                // or "2"
    "Signature": "...",                     // Message signature
    "SigningCertURL": "https://sns.us-east-1.amazonaws.com/...",
    "SubscribeURL": "https://sns.us-east-1.amazonaws.com/..."  // For confirmations
  },
  "req": {           // HTTP request details
    "method": "POST",
    "path": "/api/tickets",
    "body": "{\"title\":\"Test\"}",         // Request body as string
    "header": {
      "Content-Type": ["application/json"],
      "Authorization": ["Bearer ..."],
      "X-User-Email": ["user@example.com"],  // Custom headers
      "X-Forwarded-For": ["192.168.1.1"]
    }
  },
  "env": {           // ALL environment variables (not just WARREN_*)
    "WARREN_ENV": "production",
    "WARREN_WEBHOOK_TOKEN": "secret-token",
    "WARREN_SLACK_CHANNEL_NAME": "security-alerts",
    "PATH": "/usr/local/bin:/usr/bin:/bin",
    "HOME": "/home/nonroot",
    "USER": "nonroot",
    "HOSTNAME": "warren-abc123"
    // ... ALL environment variables from the system
  }
}
```

### Common Patterns

#### Basic Setup (Getting Started)

For initial setup when transitioning from Level 1 to Level 2:

```rego
package auth

default allow = false

# WARNING: This allows all requests - only for initial setup!
# Replace with proper rules before production use
allow = true

# TODO: Implement proper authorization based on your requirements
# Examples below show common patterns
```

#### Allow Authenticated Users

```rego
package auth

default allow = false

# Allow any authenticated user via Google IAP
allow = true if {
    input.iap.email
}

# Allow specific email domains
allow = true if {
    input.iap.email
    endswith(input.iap.email, "@example.com")
}

# Allow Slack-authenticated users
allow = true if {
    input.req.header["X-User-Email"][0]
    endswith(input.req.header["X-User-Email"][0], "@yourcompany.com")
}
```

#### Combining Multiple Authentication Methods

```rego
package auth

import rego.v1

default allow = false

# Allow Google IAP authenticated users from specific domain
allow if {
    input.iap.email
    endswith(input.iap.email, "@example.com")
}

# Allow specific service accounts via Google ID token
allow if {
    input.google.email
    input.google.email in [
        "monitoring-service@project.iam.gserviceaccount.com",
        "alerting-service@project.iam.gserviceaccount.com"
    ]
}

# Allow webhook requests with valid token
allow if {
    startswith(input.req.path, "/hooks/alert/")
    input.req.header.Authorization[0] == sprintf("Bearer %s", [input.env.WARREN_WEBHOOK_TOKEN])
}

# Allow AWS SNS requests (already verified by middleware)
allow if {
    startswith(input.req.path, "/hooks/alert/sns/")
    input.sns.Type in ["Notification", "SubscriptionConfirmation"]
}

# Allow Slack OAuth authenticated users with admin role
allow if {
    # Check for Slack user header (set by Warren after OAuth)
    input.req.header["X-User-Email"][0]
    input.req.header["X-User-Role"][0] == "admin"
}

# Conditional access based on time and user
allow if {
    # During business hours, allow any authenticated user
    current_hour := time.clock(time.now_ns())[0]
    current_hour >= 9
    current_hour < 17
    input.iap.email  # Any IAP authenticated user
}

allow if {
    # After hours, only allow on-call team
    current_hour := time.clock(time.now_ns())[0]
    outside_business_hours := current_hour < 9 or current_hour >= 17
    outside_business_hours
    input.iap.email in [
        "oncall1@example.com",
        "oncall2@example.com",
        "security-lead@example.com"
    ]
}

# Debug rule - logs authentication attempts (remove in production)
debug_auth[msg] if {
    msg := sprintf("Auth attempt: path=%s, has_iap=%v, has_google=%v, has_sns=%v", [
        input.req.path,
        input.iap != null,
        input.google != null,
        input.sns != null
    ])
}
```

**Best Practices for Combined Authentication:**

1. **Order Rules by Specificity**: Place more specific rules first
2. **Use Clear Comments**: Document what each rule allows and why
3. **Test Thoroughly**: Ensure no unintended access paths
4. **Log for Debugging**: Use debug rules during development (remove in production)
5. **Fail Secure**: Default to `allow = false`

#### Service Account Access

```rego
package auth

# Allow specific service account
allow = true if {
    input.google.email == "warren-service@project.iam.gserviceaccount.com"
}

# Allow based on environment variable
allow = true if {
    input.env.WARREN_SERVICE_ACCOUNT
    input.google.email == input.env.WARREN_SERVICE_ACCOUNT
}
```

#### Path-based Access

```rego
package auth

# Public endpoints
allow = true if {
    input.req.path in ["/health", "/metrics"]
}

# Webhook authentication
allow = true if {
    startswith(input.req.path, "/hooks/alert/")
    valid_webhook_token
}

valid_webhook_token if {
    input.req.header.Authorization[0] == sprintf("Bearer %s", [input.env.WARREN_WEBHOOK_TOKEN])
}
```

## Testing Policies

Warren includes a policy testing framework.

### Test Structure

Create test directories matching your policy structure:

```
policies/
├── alert/
│   └── myservice.rego
├── auth/
│   └── auth.rego           # Your authorization policy
└── test/
    ├── myservice/          # Tests for alert policies
    │   ├── detect/
    │   │   └── test1.json  # Should trigger alerts
    │   └── ignore/
    │       └── test2.json  # Should be ignored
    └── auth/               # Tests for auth policies
        ├── allow/
        │   └── admin.json  # Should be allowed
        └── deny/
            └── anonymous.json  # Should be denied
```

### Authorization Policy Test Examples

**Allow test** (`test/auth/allow/admin.json`):
```json
{
  "iap": {
    "email": "admin@example.com",
    "sub": "1234567890",
    "aud": ["/projects/123456/apps/warren"]
  },
  "req": {
    "method": "POST",
    "path": "/api/tickets",
    "header": {
      "Content-Type": ["application/json"]
    }
  },
  "env": {
    "WARREN_ENV": "production"
  }
}
```

**Allow test** (`test/auth/allow/service_account.json`):
```json
{
  "google": {
    "email": "monitoring-service@project.iam.gserviceaccount.com",
    "sub": "service-account-id",
    "iss": "https://accounts.google.com"
  },
  "req": {
    "method": "POST",
    "path": "/hooks/alert/raw/monitoring",
    "header": {
      "Authorization": ["Bearer ya29.xxx"]
    }
  }
}
```

**Allow test** (`test/auth/allow/webhook.json`):
```json
{
  "req": {
    "method": "POST",
    "path": "/hooks/alert/raw/custom",
    "header": {
      "Authorization": ["Bearer secret-webhook-token-123"],
      "Content-Type": ["application/json"]
    },
    "body": "{\"alert\": \"test\"}"
  },
  "env": {
    "WARREN_WEBHOOK_TOKEN": "secret-webhook-token-123"
  }
}
```

**Deny test** (`test/auth/deny/anonymous.json`):
```json
{
  "req": {
    "method": "GET",
    "path": "/api/tickets",
    "header": {}
  }
}
```

**Deny test** (`test/auth/deny/wrong_domain.json`):
```json
{
  "iap": {
    "email": "user@wrongdomain.com",
    "sub": "9876543210"
  },
  "req": {
    "method": "GET",
    "path": "/api/tickets"
  }
}
```

**Deny test** (`test/auth/deny/invalid_webhook_token.json`):
```json
{
  "req": {
    "method": "POST",
    "path": "/hooks/alert/raw/custom",
    "header": {
      "Authorization": ["Bearer wrong-token"]
    }
  },
  "env": {
    "WARREN_WEBHOOK_TOKEN": "correct-token"
  }
}
```

### Running Tests

The `warren test` command validates your policies against test data:

```bash
# Basic usage
warren test \
  --policy ./policies \
  --test-detect-data ./test/myservice/detect \
  --test-ignore-data ./test/myservice/ignore

# Using aliases
warren test \
  -p ./policies \
  -d ./test/myservice/detect \
  -i ./test/myservice/ignore

# Using environment variables
export WARREN_POLICY=./policies
export WARREN_TEST_DETECT_DATA=./test/myservice/detect
export WARREN_TEST_IGNORE_DATA=./test/myservice/ignore
warren test
```

#### Command Options

- `-p, --policy`: Path to policy directory or file (required)
- `-d, --test-detect-data`: Path to test data that should trigger alerts
- `-i, --test-ignore-data`: Path to test data that should be ignored

#### How It Works

1. **Directory Structure**: Test files are organized by schema name
   - First directory under the test path becomes the schema name
   - Example: `test/myservice/detect/alert1.json` → schema: `myservice`

2. **File Formats**: Supports `.json` and `.jsonl` files
   - `.json`: Single JSON object per file
   - `.jsonl`: Multiple JSON objects (one per line)

3. **Test Execution**:
   - For "detect" data: Test passes if policy generates at least one alert
   - For "ignore" data: Test passes if policy generates no alerts

4. **Output**:
   - ✅ PASS: Expected behavior matches actual behavior
   - ❌ FAIL: Mismatch with details (schema, filename, reason)

### Test Data Format

Test data should match the actual input your webhook receives:

#### Detection test (`test/myservice/detect/test1.json`):
```json
{
  "alert_title": "Suspicious Activity",
  "alert_description": "Unusual behavior detected",
  "severity": "high",
  "source_ip": "192.168.1.100"
}
```

This should trigger your policy to create an alert.

#### Ignore test (`test/myservice/ignore/test2.json`):
```json
{
  "alert_title": "Normal Activity",
  "alert_description": "Regular scan",
  "severity": "low",
  "source_ip": "10.0.1.50"
}
```

This should be filtered out by your policy's ignore rules.

#### Multi-object test (`test/myservice/detect/batch.jsonl`):
```json
{"event": "login_failed", "user": "admin", "ip": "45.227.255.100"}
{"event": "privilege_escalation", "user": "guest", "ip": "192.168.1.50"}
{"event": "data_exfiltration", "size": "10GB", "destination": "unknown.host"}
```

Each line is tested independently against your policy.

## Examples

### Example 1: Severity-based Filtering

```rego
package alert.monitoring

alert contains {
    "title": input.alert_name,
    "description": input.alert_message,
    "attrs": [
        {
            "key": "severity",
            "value": input.severity,
            "link": ""
        }
    ]
} if {
    not ignore
}

# Simple severity threshold
ignore if {
    to_number(input.severity) < 3
}
```

### Example 2: Time-based Rules

```rego
package alert.scheduled

alert contains {
    "title": "Scheduled Job Failed",
    "description": input.error_message,
    "attrs": [
        {
            "key": "job_name",
            "value": input.job_name,
            "link": ""
        }
    ]
} if {
    not ignore_maintenance_window
}

# Ignore during maintenance window (UTC)
ignore_maintenance_window if {
    current_hour := time.clock(time.now_ns())[0]
    current_day := time.weekday(time.now_ns())
    
    # Sunday 2-4 AM UTC
    current_day == "Sunday"
    current_hour >= 2
    current_hour < 4
}
```

### Example 3: Complex Attribute Extraction

```rego
package alert.firewall

alert contains {
    "title": title,
    "description": description,
    "attrs": array.concat(base_attrs, threat_attrs)
} if {
    not ignore
}

title = t if {
    t := sprintf("Firewall Alert: %s from %s", [input.action, input.source_ip])
}

description = d if {
    d := sprintf("Traffic %s: %s:%d -> %s:%d", 
        [input.action, input.source_ip, input.source_port, 
         input.dest_ip, input.dest_port])
}

base_attrs = [
    {
        "key": "action",
        "value": upper(input.action),
        "link": ""
    },
    {
        "key": "protocol",
        "value": input.protocol,
        "link": ""
    }
]

threat_attrs = attrs if {
    input.threat_level > 0
    attrs := [
        {
            "key": "threat_level",
            "value": to_string(input.threat_level),
            "link": ""
        },
        {
            "key": "threat_category",
            "value": input.threat_category,
            "link": ""
        }
    ]
} else = []

ignore if {
    # Ignore internal traffic
    startswith(input.source_ip, "10.")
    startswith(input.dest_ip, "10.")
}
```

### Example 4: Enrichment with Links

```rego
package alert.threats

alert contains {
    "title": sprintf("Malware Detected: %s", [input.malware_name]),
    "description": input.detection_message,
    "attrs": [
        {
            "key": "file_hash",
            "value": input.file_hash,
            "link": sprintf("https://www.virustotal.com/gui/file/%s", [input.file_hash])
        },
        {
            "key": "source_ip",
            "value": input.source_ip,
            "link": sprintf("https://www.abuseipdb.com/check/%s", [input.source_ip])
        },
        {
            "key": "malware_family",
            "value": input.malware_name,
            "link": sprintf("https://malpedia.caad.fkie.fraunhofer.de/search?q=%s", 
                [replace(input.malware_name, " ", "+")])
        }
    ]
} if {
    input.confidence_score > 80
}
```

## Real-World Policy Examples

### AWS GuardDuty Policy

A production-ready policy for AWS GuardDuty findings:

```rego
package alert.guardduty

import rego.v1

# Main alert rule for GuardDuty findings
alert contains {
    "title": title,
    "description": description,
    "attrs": array.concat([
        {
            "key": "severity",
            "value": severity_label,
            "link": ""
        },
        {
            "key": "type",
            "value": input.detail.type,
            "link": aws_doc_link
        },
        {
            "key": "account",
            "value": input.detail.accountId,
            "link": ""
        },
        {
            "key": "region",
            "value": input.detail.region,
            "link": ""
        }
    ], resource_attrs)
} if {
    # Only process GuardDuty findings
    input.source == "aws.guardduty"
    input.detail.type
    
    # Skip informational findings unless critical
    not ignore
}

# Generate human-readable title
title := sprintf("%s in %s", [
    finding_title[input.detail.type],
    input.detail.region
]) if {
    finding_title[input.detail.type]
} else := input.detail.type

# Title mappings for common GuardDuty findings
finding_title := {
    "Recon:EC2/PortProbeUnprotectedPort": "Port Scan Detected",
    "UnauthorizedAccess:EC2/SSHBruteForce": "SSH Brute Force Attack",
    "Trojan:EC2/BlackholeTraffic": "EC2 Instance Communicating with Known Malicious IP",
    "CryptoCurrency:EC2/BitcoinTool.B!DNS": "Cryptocurrency Mining Activity",
    "UnauthorizedAccess:IAMUser/InstanceCredentialExfiltration": "AWS Credentials Compromised",
    "Policy:IAMUser/RootCredentialUsage": "Root Account Activity Detected",
    "Stealth:IAMUser/LoggingConfigurationModified": "CloudTrail Logging Disabled"
}

# Generate detailed description
description := sprintf("%s. Finding ID: %s", [
    input.detail.description,
    input.detail.id
]) if {
    input.detail.description
} else := sprintf("GuardDuty detected suspicious activity: %s", [input.detail.type])

# Convert numeric severity to label
severity_label := "critical" if { input.detail.severity >= 8.0 }
else := "high" if { input.detail.severity >= 6.0 }
else := "medium" if { input.detail.severity >= 4.0 }
else := "low"

# AWS documentation link for finding type
aws_doc_link := sprintf(
    "https://docs.aws.amazon.com/guardduty/latest/ug/guardduty_finding-types-ec2.html#%s",
    [lower(replace(input.detail.type, ":", ""))]
)

# Extract resource-specific attributes
resource_attrs contains attr if {
    input.detail.resource.instanceDetails.instanceId
    attr := {
        "key": "instance_id",
        "value": input.detail.resource.instanceDetails.instanceId,
        "link": sprintf("https://console.aws.amazon.com/ec2/v2/home?region=%s#Instances:instanceId=%s", [
            input.detail.region,
            input.detail.resource.instanceDetails.instanceId
        ])
    }
}

resource_attrs contains attr if {
    some i
    ip := input.detail.resource.instanceDetails.networkInterfaces[i].privateIpAddress
    attr := {
        "key": sprintf("private_ip_%d", [i]),
        "value": ip,
        "link": ""
    }
}

resource_attrs contains attr if {
    input.detail.service.action.networkConnectionAction.remoteIpDetails.ipAddressV4
    attr := {
        "key": "remote_ip",
        "value": input.detail.service.action.networkConnectionAction.remoteIpDetails.ipAddressV4,
        "link": sprintf("https://www.abuseipdb.com/check/%s", [
            input.detail.service.action.networkConnectionAction.remoteIpDetails.ipAddressV4
        ])
    }
}

# Ignore rules
ignore if {
    # Ignore low severity findings in development accounts
    input.detail.accountId in ["123456789012", "098765432109"]  # Dev accounts
    input.detail.severity < 4.0
}

ignore if {
    # Ignore expected scanning from security tools
    input.detail.type == "Recon:EC2/PortProbeUnprotectedPort"
    input.detail.service.action.networkConnectionAction.remoteIpDetails.ipAddressV4 in [
        "10.0.0.10",  # Internal security scanner
        "10.0.0.11"   # Vulnerability assessment tool
    ]
}

# Test data for policy testing
test_guardduty_critical if {
    alert[_] with input as {
        "source": "aws.guardduty",
        "detail": {
            "type": "Trojan:EC2/BlackholeTraffic",
            "severity": 8.5,
            "id": "test-finding-123",
            "description": "EC2 instance i-1234567890abcdef0 is communicating with a known malicious IP",
            "accountId": "111111111111",
            "region": "us-east-1",
            "resource": {
                "instanceDetails": {
                    "instanceId": "i-1234567890abcdef0",
                    "networkInterfaces": [{
                        "privateIpAddress": "10.0.1.50"
                    }]
                }
            },
            "service": {
                "action": {
                    "networkConnectionAction": {
                        "remoteIpDetails": {
                            "ipAddressV4": "192.168.100.200"
                        }
                    }
                }
            }
        }
    }
}
```

### Suricata IDS Policy

A comprehensive policy for Suricata intrusion detection alerts:

```rego
package alert.suricata

import rego.v1

# Main alert rule for Suricata events
alert contains {
    "title": title,
    "description": description,
    "attrs": array.concat(base_attrs, flow_attrs)
} if {
    # Process EVE JSON format
    input.event_type == "alert"
    
    # Must have signature info
    input.alert.signature
    input.alert.signature_id
    
    # Apply filtering
    not ignore
}

# Generate title from signature and category
title := sprintf("[%s] %s", [
    strings.trim_space(input.alert.category),
    input.alert.signature
])

# Create detailed description
description := sprintf(
    "Signature ID %d triggered. Flow: %s:%d -> %s:%d using %s",
    [
        input.alert.signature_id,
        input.src_ip,
        input.src_port,
        input.dest_ip,
        input.dest_port,
        input.proto
    ]
)

# Base attributes for all alerts
base_attrs := [
    {
        "key": "severity",
        "value": severity_mapping[input.alert.severity],
        "link": ""
    },
    {
        "key": "signature_id", 
        "value": sprintf("%d", [input.alert.signature_id]),
        "link": sprintf("https://docs.suricata.io/en/latest/rules/intro.html#sid-signature-id-%d", [
            input.alert.signature_id
        ])
    },
    {
        "key": "category",
        "value": input.alert.category,
        "link": ""
    },
    {
        "key": "protocol",
        "value": input.proto,
        "link": ""
    }
]

# Flow-specific attributes
flow_attrs contains attr if {
    input.src_ip
    attr := {
        "key": "source_ip",
        "value": input.src_ip,
        "link": threat_intel_link(input.src_ip)
    }
}

flow_attrs contains attr if {
    input.dest_ip
    attr := {
        "key": "destination_ip", 
        "value": input.dest_ip,
        "link": threat_intel_link(input.dest_ip)
    }
}

flow_attrs contains attr if {
    input.src_port
    attr := {
        "key": "source_port",
        "value": sprintf("%d", [input.src_port]),
        "link": ""
    }
}

flow_attrs contains attr if {
    input.dest_port
    attr := {
        "key": "destination_port",
        "value": sprintf("%d", [input.dest_port]), 
        "link": service_link(input.dest_port)
    }
}

flow_attrs contains attr if {
    input.http.hostname
    attr := {
        "key": "hostname",
        "value": input.http.hostname,
        "link": sprintf("https://urlscan.io/search/#%s", [input.http.hostname])
    }
}

flow_attrs contains attr if {
    input.http.url
    attr := {
        "key": "url",
        "value": input.http.url,
        "link": ""
    }
}

flow_attrs contains attr if {
    input.dns.query[0].rrname
    attr := {
        "key": "dns_query",
        "value": input.dns.query[0].rrname,
        "link": sprintf("https://www.virustotal.com/gui/domain/%s", [
            strings.trim_suffix(input.dns.query[0].rrname, ".")
        ])
    }
}

# Severity mapping (Suricata uses 1-3, we use low/medium/high/critical)
severity_mapping := {
    1: "high",
    2: "medium", 
    3: "low"
}

# Generate threat intelligence links based on IP type
threat_intel_link(ip) := link if {
    # Skip private IPs
    net.cidr_contains("10.0.0.0/8", ip)
    link := ""
} else := link if {
    net.cidr_contains("172.16.0.0/12", ip)
    link := ""
} else := link if {
    net.cidr_contains("192.168.0.0/16", ip)
    link := ""
} else := sprintf("https://www.abuseipdb.com/check/%s", [ip])

# Service documentation links for common ports
service_link(port) := "https://attack.mitre.org/techniques/T1021/001/" if { port == 22 }  # SSH
else := "https://attack.mitre.org/techniques/T1021/002/" if { port == 445 }  # SMB
else := "https://attack.mitre.org/techniques/T1021/001/" if { port == 3389 } # RDP
else := ""

# Ignore rules for false positive reduction
ignore if {
    # Ignore DNS queries to internal DNS servers
    input.alert.category == "DNS"
    input.dest_ip in ["10.0.0.53", "10.0.0.54"]  # Internal DNS servers
}

ignore if {
    # Ignore vulnerability scans from authorized scanners
    input.alert.signature contains "GPL SCAN"
    input.src_ip in [
        "10.10.10.100",  # Nessus scanner
        "10.10.10.101"   # OpenVAS scanner
    ]
}

ignore if {
    # Ignore low severity alerts from monitoring subnets
    severity_mapping[input.alert.severity] == "low"
    net.cidr_contains("10.99.0.0/16", input.src_ip)  # Monitoring subnet
}

ignore if {
    # Ignore specific noisy signatures
    input.alert.signature_id in [
        2013504,  # ET POLICY GNU/Linux APT User-Agent Outbound likely related to package management
        2013505,  # ET POLICY curl User-Agent Outbound
        2019401   # ET POLICY Spotify P2P Client
    ]
}

# Test cases
test_suricata_malware if {
    result := alert[_] with input as {
        "event_type": "alert",
        "src_ip": "192.168.1.100",
        "src_port": 54321,
        "dest_ip": "185.220.101.45",
        "dest_port": 443,
        "proto": "TCP",
        "alert": {
            "signature": "ET MALWARE Cobalt Strike Beacon Observed",
            "signature_id": 2027067,
            "category": "A Network Trojan was detected",
            "severity": 1
        }
    }
    
    result.title == "[A Network Trojan was detected] ET MALWARE Cobalt Strike Beacon Observed"
    result.attrs[_].key == "severity"
    result.attrs[_].value == "high"
}

test_ignore_internal_dns if {
    count(alert) == 0 with input as {
        "event_type": "alert",
        "src_ip": "10.0.1.50",
        "dest_ip": "10.0.0.53",
        "proto": "UDP",
        "alert": {
            "signature": "ET DNS Query to a Suspicious Domain",
            "signature_id": 2020000,
            "category": "DNS",
            "severity": 2
        }
    }
}
```

### Policy Writing Best Practices

When creating policies for your security tools:

1. **Start with the Tool's Output Format**: Understand the exact JSON structure your tool produces
2. **Focus on High-Value Alerts**: Filter out noise early with `ignore` rules
3. **Enrich with Context**: Add links to threat intelligence and documentation
4. **Test Thoroughly**: Include test cases for both positive and negative scenarios
5. **Document Assumptions**: Comment on why certain decisions were made
6. **Version Control**: Track changes and test in staging before production

## Debugging

### Print Statements

Use `print()` for debugging (visible in debug logs):

```rego
package alert.debug

alert contains result if {
    print("Input data:", input)
    
    severity := input.severity
    print("Severity:", severity)
    
    result := {
        "title": input.title,
        "description": input.description,
        "attrs": []
    }
    
    print("Result:", result)
}
```

View debug output:
```bash
warren serve --log-level=debug --policy ./policies
```

### Common Issues

#### No Alerts Created
- Check package name matches webhook path
- Verify conditions are met
- Use print statements to debug
- Test with minimal policy first

#### Performance Issues
- Avoid complex iterations
- Use helper functions
- Cache computed values
- Minimize external calls

## Deployment

### Policy Management

1. **Version Control**: Store policies in Git
2. **Testing**: Run tests before deployment
3. **Staging**: Test in non-production first
4. **Monitoring**: Watch for policy errors in logs

### Hot Reload

Warren watches policy files for changes:
```bash
warren serve --policy ./policies --watch
```

### Policy Organization

```
policies/
├── alert/
│   ├── aws/
│   │   ├── guardduty.rego
│   │   └── cloudtrail.rego
│   ├── gcp/
│   │   └── scc.rego
│   └── custom/
│       └── webapp.rego
├── auth/
│   ├── api.rego
│   └── webhook.rego
└── lib/
    └── common.rego  # Shared functions
```

## Best Practices

1. **Use Descriptive Names**
   - Clear package names
   - Meaningful variable names
   - Helpful attribute keys

2. **Handle Missing Data**
   ```rego
   title := input.title if {
       input.title
   } else = "Unknown Alert"
   ```

3. **Validate Input**
   ```rego
   alert contains {...} if {
       # Ensure required fields exist
       input.source_ip
       input.severity
       is_valid_severity(input.severity)
   }
   ```

4. **Document Complex Logic**
   ```rego
   # Calculate risk score based on multiple factors
   # High severity + external IP + failed auth = high risk
   risk_score = score if {
       ...
   }
   ```

5. **Test Edge Cases**
   - Empty arrays
   - Missing fields
   - Invalid data types
   - Extreme values

6. **Security Considerations**
   - Never log sensitive data in policies
   - Validate all external inputs
   - Use allowlists over denylists when possible
   - Regularly review and update ignore rules

`TODO: Add a troubleshooting section for common policy errors`

## Advanced Topics

### Dynamic Attributes

Build attributes conditionally:
```rego
attrs[a] if {
    input.user_id
    a := {
        "key": "user",
        "value": input.user_id,
        "link": sprintf("/users/%s", [input.user_id])
    }
}

attrs[a] if {
    input.department
    a := {
        "key": "department",
        "value": input.department,
        "link": ""
    }
}
```

### Multi-source Policies

Handle different input formats:
```rego
package alert.generic

# Format 1: Simple
alert contains {
    "title": input.title,
    "description": input.message,
    "attrs": []
} if {
    input.format == "simple"
}

# Format 2: Detailed
alert contains {
    "title": input.event.name,
    "description": input.event.details,
    "attrs": extract_attrs(input.event.metadata)
} if {
    input.format == "detailed"
}
```

### Policy Composition

Reuse common patterns:
```rego
package lib.common

is_business_hours if {
    hour := time.clock(time.now_ns())[0]
    hour >= 9
    hour < 17
}

is_weekend if {
    day := time.weekday(time.now_ns())
    day in ["Saturday", "Sunday"]
}

severity_number(s) = 5 if { s == "critical" }
severity_number(s) = 4 if { s == "high" }
severity_number(s) = 3 if { s == "medium" }
severity_number(s) = 2 if { s == "low" }
severity_number(s) = 1 if { s == "info" }
```

Use in policies:
```rego
package alert.business

import data.lib.common

alert contains {...} if {
    not common.is_business_hours
    common.severity_number(input.severity) >= 3
}
```

## Troubleshooting Common Issues

### Authorization Policy Not Working

If your authorization policy isn't being evaluated:

1. **Check Package Name**: Must be `package auth` (not `auth.something`)
2. **Verify Policy Mount**: Ensure `-v $(pwd)/policies:/policies:ro` in Docker command
3. **Set WARREN_POLICY**: Environment variable must point to policy directory
4. **Remove NO_AUTHORIZATION**: Ensure flag is not set
5. **Check Logs**: Look for policy evaluation results in debug logs

### Alert Policy Not Triggering

1. **Verify Endpoint**: Package name must match webhook path
2. **Check Input Structure**: Use `print()` to debug input data
3. **Test Conditions**: Ensure your conditions can be satisfied
4. **Review Ignore Rules**: Check if alerts are being filtered

### Additional Troubleshooting Scenarios

#### Policy Not Loading

**Symptoms**: Warren starts but policies don't seem to be evaluated

**Checks**:
1. Verify `WARREN_POLICY` environment variable is set correctly
2. Check file permissions on policy directory (must be readable)
3. Look for policy loading errors in startup logs
4. Ensure `.rego` files have correct syntax

**Solution**:
```bash
# Test policy syntax
opa fmt --list policies/
opa test policies/

# Check Warren logs
docker logs warren | grep -i policy
```

#### Intermittent Authorization Failures

**Symptoms**: Same request sometimes works, sometimes fails

**Common Causes**:
1. **Time-based rules**: Check if policies have time conditions
2. **Token expiration**: IAP/OAuth tokens may expire during testing
3. **Race conditions**: Context building might be incomplete

**Debug Steps**:
```rego
# Add debug logging to your policy
allow if {
    print("Current time:", time.now_ns())
    print("Input context:", input)
    # ... your rules ...
}
```

#### Missing Context Data

**Symptoms**: `input.google` or `input.iap` is empty when expected

**Checks**:
1. Verify authentication headers are being sent
2. Check middleware ordering in server configuration
3. Ensure tokens are valid and not expired

**Debug with curl**:
```bash
# Test with Google ID token
curl -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     https://your-warren/api/tickets

# Test with IAP header
curl -H "x-goog-iap-jwt-assertion: $IAP_JWT" \
     https://your-warren/api/tickets
```

#### Policy Changes Not Taking Effect

**Symptoms**: Modified policy file but behavior doesn't change

**Solutions**:
1. **Docker volume mount**: Ensure you're editing the mounted file
2. **Hot reload**: Warren doesn't auto-reload policies - restart the container
3. **Cache issues**: Clear browser cache if testing via UI

```bash
# Restart Warren to reload policies
docker restart warren

# Or use --watch flag (if implemented)
warren serve --policy ./policies --watch
```

#### Webhook Authorization Failures

**Symptoms**: External tools can't send alerts, getting 403 errors

**Common Issues**:
1. **Wrong token format**: Ensure "Bearer " prefix is included
2. **Environment variable mismatch**: Check `WARREN_WEBHOOK_TOKEN` value
3. **Path matching**: Verify webhook path in policy matches actual endpoint

**Test webhook auth**:
```bash
# Test with correct token
curl -X POST \
  -H "Authorization: Bearer $WARREN_WEBHOOK_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"test": "alert"}' \
  http://localhost:8080/hooks/alert/raw/test
```

#### Debugging Complex Policies

**Use OPA's built-in tools**:
```bash
# Evaluate policy with specific input
opa eval -d policies/auth/auth.rego \
         -i test/auth/allow/admin.json \
         "data.auth.allow"

# Run REPL for interactive testing
opa run policies/
> data.auth.allow with input as {"iap": {"email": "test@example.com"}}
```

## Next Steps

1. Start with simple policies and iterate
2. Test thoroughly with real data
3. Monitor policy performance
4. Share patterns with your team
5. Contribute improvements back to Warren

## Additional Resources

- [Open Policy Agent Documentation](https://www.openpolicyagent.org/docs/latest/)
- [Rego Playground](https://play.openpolicyagent.org/) - Test policies online
- [Warren Examples Repository](https://github.com/secmon-lab/warren/tree/main/examples/policies) - Community contributed policies

## Examples Directory

For ready-to-use policy examples, check the [examples/policies](https://github.com/secmon-lab/warren/tree/main/examples/policies) directory in the Warren repository. It includes:

### Alert Detection Policies
- **AWS Services**: GuardDuty, CloudTrail, Security Hub
- **Network Security**: Suricata, Snort, Zeek
- **Host Security**: OSSEC, Wazuh, Falco
- **Cloud Platforms**: GCP Security Command Center, Azure Sentinel
- **SIEM Integration**: Splunk, Elastic, Datadog

### Authorization Policies
- **Basic Setup**: Simple allow-all for getting started
- **Domain-based**: Allow users from specific email domains
- **Role-based**: Different access levels based on user roles
- **Time-based**: Business hours vs after-hours access
- **Service Accounts**: Allow specific automation accounts

### Testing Examples
- Complete test suites for each policy
- Input data samples from real security tools
- Performance testing scenarios

### Custom Templates
- Webhook receiver template
- Multi-source aggregation template
- Enrichment pipeline template

To use these examples:

```bash
# Clone the repository
git clone https://github.com/secmon-lab/warren.git
cd warren

# Copy example policies
cp -r examples/policies/* /path/to/your/policies/

# Customize for your environment
vi /path/to/your/policies/auth/auth.rego
```

**Contributing**: If you've created policies for security tools not yet covered, please consider submitting a pull request to share with the community!