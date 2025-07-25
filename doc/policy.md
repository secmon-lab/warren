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