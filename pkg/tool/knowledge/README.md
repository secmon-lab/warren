# Knowledge Management

Save, retrieve, and manage agent knowledge in an append-only model. Knowledge is organized by topic and identified by slug.

## Configuration

No configuration is required. The knowledge tool is always available.

## Available Functions

| Function | Description |
|---|---|
| `knowledge_list` | List all knowledge slugs in the current topic |
| `knowledge_get` | Get specific knowledge by slug, or all knowledges in the topic |
| `knowledge_save` | Save or update knowledge (by slug, name, content). Maximum 10KB per topic. |
| `knowledge_archive` | Archive a knowledge entry (logical delete, frees quota) |

## Usage

Knowledge entries are scoped to a topic (typically a ticket) and identified by a unique slug. The agent uses knowledge to accumulate findings during investigation.

- **Slug**: A short identifier for the knowledge entry (e.g., `ip-reputation`, `user-activity`)
- **Name**: A human-readable name for the entry
- **Content**: The knowledge content (Markdown supported)
- **Quota**: Maximum 10KB of knowledge per topic to prevent unbounded growth
