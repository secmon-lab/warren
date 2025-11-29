# Task
Extract the raw Slack messages from the conversation history and return them as a JSON array.

# Key Principle: Understand User Intent
- Do NOT just extract messages literally based on the user's wording
- UNDERSTAND what the user actually wants to find or learn
- INTERPRET the user's intent from their request and the context
- Select messages that FULFILL the user's actual information needs, not just match keywords

# Guidelines
1. First, understand the user's true intent from their original request
2. Look for tool execution results in the conversation history (e.g., `slack_search_messages` tool responses)
3. Extract the message data that answers the user's actual question or need
4. Do NOT summarize or paraphrase message content - include full original text
5. For each message, include: text, user, channel, timestamp
6. If multiple searches were performed, intelligently select messages that fulfill the user's intent
7. Return ONLY the messages array - no wrapper object, no additional fields

# Examples of Intent Understanding
- User asks "people having authentication problems" → Extract messages about auth issues, not just containing "authentication"
- User asks "what did X say about Y" → Extract X's messages related to topic Y
- User asks "recent discussions on Z" → Extract conversational messages about Z, not just mentions

# Output Format
Return a JSON array directly:
[
  {
    "text": "full message text here",
    "user": "user_id or name",
    "channel": "channel_id or name",
    "timestamp": "timestamp"
  },
  ...
]

# Important
- The output must be a JSON ARRAY at the top level
- Each element must have: text, user, channel, timestamp fields
- Preserve original message text exactly as it appears
- If no messages match the user's intent, return an empty array: []
- Focus on user's INTENT, not literal wording
