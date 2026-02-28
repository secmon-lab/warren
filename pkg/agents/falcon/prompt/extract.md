# Task
Extract the raw CrowdStrike Falcon records from the conversation history and return them as a JSON object with a "records" field.

# Key Principle: Understand User Intent
- Do NOT just extract data literally based on the user's wording
- UNDERSTAND what the user actually wants to know or achieve
- INTERPRET the user's intent from their request and the context
- Select records that FULFILL the user's actual needs, not just match keywords

# Guidelines
1. First, understand the user's true intent from their original request
2. Look for API response data in the conversation history (e.g., `falcon_search_alerts`, `falcon_get_incidents` tool responses)
3. Parse the data carefully — responses contain nested structures with resources/entities
4. Convert each record into a JSON object preserving the key fields
5. Preserve ALL relevant field names and values exactly as they appear
6. If multiple API calls were made, intelligently combine and deduplicate records
7. **IMPORTANT**: Each record MUST be a non-empty object with the actual field names and values

# Record Types

## Incidents
Key fields: incident ID, status, fine_score, start/end time, hosts, users, tactics, techniques

## Alerts
Key fields: composite_id, status, severity, timestamp, hostname, tactic, technique, filename, cmdline, description

## Behaviors
Key fields: behavior_id, tactic, technique, severity, pattern_disposition, device info, timestamp

## CrowdScores
Key fields: id, timestamp, score, adjusted_score

# Output Format
Return a JSON object with a "records" field:
```json
{
  "records": [
    {
      "field1": "value1",
      "field2": "value2"
    }
  ]
}
```

# Important
- The output must be a JSON OBJECT with a "records" field
- The "records" field contains an ARRAY of record objects
- Each record object must contain the actual field names and values from the API response
- **DO NOT return empty objects** — each object must contain the actual field values
- If no records match the user's intent, return: `{"records": []}`
- Flatten deeply nested structures where appropriate for readability
- Focus on user's INTENT, not literal wording
