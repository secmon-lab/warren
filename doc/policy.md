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

### Context Structure

Authorization policies receive rich context:

```json
{
  "google": {},      // Google ID token claims
  "iap": {},         // Google IAP JWT claims
  "sns": {},         // AWS SNS message data
  "req": {           // HTTP request
    "method": "POST",
    "path": "/api/tickets",
    "body": "...",
    "header": {}
  },
  "env": {}          // Environment variables (WARREN_*)
}
```

### Common Patterns

#### Allow Authenticated Users

```rego
package auth

default allow = false

# Allow any authenticated user
allow = true if {
    input.iap.email
}

# Allow specific email domains
allow = true if {
    input.iap.email
    endswith(input.iap.email, "@example.com")
}
```

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

Create test directories:
```
policies/
├── alert/
│   └── myservice.rego
└── test/
    └── myservice/
        ├── detect/
        │   └── test1.json
        └── ignore/
            └── test2.json
```

### Running Tests

```bash
# Test all policies
warren test --policy ./policies

# Test specific policy
warren test --policy ./policies --filter myservice
```

### Test Data Format

Detection test (`test/myservice/detect/test1.json`):
```json
{
  "alert_title": "Suspicious Activity",
  "alert_description": "Unusual behavior detected",
  "severity": "high",
  "source_ip": "192.168.1.100"
}
```

This should be detected by the policy and create an alert.

Ignore test (`test/myservice/ignore/test2.json`):
```json
{
  "alert_title": "Normal Activity",
  "alert_description": "Regular scan",
  "severity": "low",
  "source_ip": "10.0.1.50"
}
```

This should be ignored by the policy.

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

## Next Steps

1. Start with simple policies and iterate
2. Test thoroughly with real data
3. Monitor policy performance
4. Share patterns with your team
5. Contribute improvements back to Warren