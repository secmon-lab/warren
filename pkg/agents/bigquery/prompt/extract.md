# Task
Extract the raw query result records from the conversation history and return them as a JSON array.

# Key Principle: Understand User Intent
- Do NOT just extract data literally based on the user's wording
- UNDERSTAND what the user actually wants to know or achieve
- INTERPRET the user's intent from their request and the context
- Select records that FULFILL the user's actual needs, not just match keywords

# Guidelines
1. First, understand the user's true intent from their original request
2. Look for query results, data tables, or structured data in the conversation history
3. Parse the data carefully - it may be in table format, JSON, or plain text
4. Convert each row/record into a JSON object with proper field names and values
5. Preserve ALL field names and values exactly as they appear in the results
6. If multiple queries were executed, intelligently select records that fulfill the user's intent
7. Return ONLY the records array - no wrapper object, no additional fields
8. **IMPORTANT**: Each record MUST be a non-empty object with the actual field names and values from the query results

# Examples of Intent Understanding
- User asks "recent suspicious activities" → Extract records showing anomalies, not just recent records
- User asks "who accessed this resource" → Extract records with user/identity information
- User asks "how many times" → Extract all relevant records (LLM will count, but return raw data)

# Output Format Example

**Input (table format in conversation)**:
```
+----------+---------------------+------------------+
| user_id  | login_time          | ip_address       |
+----------+---------------------+------------------+
| user123  | 2024-11-25 10:30:00 | 192.168.1.100    |
| user456  | 2024-11-26 14:20:00 | 10.0.0.50        |
+----------+---------------------+------------------+
```

**Output (JSON object with records array)**:
```json
{
  "records": [
    {
      "user_id": "user123",
      "login_time": "2024-11-25 10:30:00",
      "ip_address": "192.168.1.100"
    },
    {
      "user_id": "user456",
      "login_time": "2024-11-26 14:20:00",
      "ip_address": "10.0.0.50"
    }
  ]
}
```

# Important
- The output must be a JSON OBJECT with a "records" field
- The "records" field contains an ARRAY of record objects
- Each record object must contain the actual field names and values
- Preserve original field names and data types
- **DO NOT return empty objects** - each object must contain the actual field values
- If no records match the user's intent, return: `{"records": []}`
- Focus on user's INTENT, not literal wording
