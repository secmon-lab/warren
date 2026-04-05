# Bluebell Strategy

Bluebell extends aster's execution flow with **intent resolution** — automatically selecting the right investigation approach from multiple user-defined prompts.

## Design Philosophy

**Adaptability and problem redefinition.** Bluebell inherits aster's plan→tasks→replan→final flow but adds a critical pre-planning phase: XY problem detection and intent resolution. Instead of assuming every alert needs the same investigation approach, bluebell selects the most appropriate perspective from user-defined prompts and generates a situation-specific investigation directive.

The core insight is that *how* you investigate matters as much as *what* you investigate. A deployment-related alert needs infrastructure analysis, not threat hunting. Bluebell detects this mismatch and reframes the investigation before planning begins.

## Execution Flow

```
Intent Resolution → Plan → Parallel Tasks → Replan → ... → Final Response
```

### Intent Resolution Phase

Before planning, bluebell runs a single LLM call that:

1. **XY Problem Detection**: Assesses whether the user's stated question (X) matches the actual problem indicated by alert data (Y)
2. **Prompt Selection**: Chooses the best-fit prompt from user-defined candidates based on `id` + `description` (Content is NOT passed — token efficient)
3. **Intent Resolution**: Generates a situation-specific investigation directive that addresses the actual problem

The resolved intent replaces generic prompt content with targeted instructions like: *"Alert context indicates this IP belongs to the organization's CI/CD pipeline. Investigate deployment configuration rather than external threat."*

### Candidate Handling

| Candidates | Behavior |
|-----------|----------|
| 0 | Skip intent resolution entirely (aster-equivalent behavior) |
| 1 | Skip selection, resolve intent only |
| 2+ | Select + resolve intent in one LLM call |

## System Prompt Architecture

Bluebell consolidates aster's two templates (`system.md` + `ticketless_system.md`) into a single `system.md` with template conditionals. The template uses a typed `SystemPromptData` struct instead of `map[string]any`.

### Template Variables

```go
type SystemPromptData struct {
    Context        ContextData   // Ticket, Alert, Thread, Channel
    Tools          ToolsData     // Available tool descriptions
    Knowledge      KnowledgeData // Knowledge base tags
    ResolvedIntent string        // Situation-specific directive from intent resolver
    Lang           string        // Response language
    Requester      Requester     // Slack user ID for mentions
}
```

The `ResolvedIntent` field is injected into the `## Investigation Directive` section. Unlike aster's `UserPrompt` which contains raw file content, this is a synthesized, situation-specific instruction.

### Context Handling in Selector

The selector receives:
- `ThreadComments` — Human conversation in the Slack thread
- `SlackHistory` — Recent channel messages

The selector does **NOT** receive `chatCtx.History` (plan/replan LLM session history) to prevent contamination from the planner's reasoning under a different system prompt.

## Prompt File Format

User-defined prompts are markdown files with YAML frontmatter:

```markdown
---
id: infra-incident
description: >
  Infrastructure incident investigation. Use when alerts suggest
  system availability issues, performance degradation, or
  cascading failures rather than security threats.
---

(Content is stored but not passed to the selector.
The description alone drives selection and intent resolution.)
```

- `id` — Required, must be unique across all prompt files
- `description` — Required, used by the selector for prompt selection (1-3 sentences recommended)
- Content (body) — Stored in `PromptEntry.Content` but not used by the selector

### Directory Structure Example

```
prompts/
├── security-investigation.md
├── infra-incident.md
├── compliance-audit.md
└── problem-solving.md
```

## Configuration

| Flag | Env Var | Description |
|------|---------|-------------|
| `--chat-strategy bluebell` | `WARREN_CHAT_STRATEGY` | Select bluebell |
| `--user-system-prompts <dir>` | `WARREN_USER_SYSTEM_PROMPTS` | Directory containing prompt files |
| `--budget-strategy <type>` | `WARREN_BUDGET_STRATEGY` | `none` or `default` |

## Requirements

- **Knowledge service is required.** Bluebell always runs in agent mode with `knowledge_search` available to the planner. Configure knowledge service before using bluebell.

## When to Use Bluebell

- Multiple investigation approaches needed (security, infrastructure, compliance, etc.)
- Alerts that may be misclassified (XY problem scenarios)
- Organizations with domain-specific investigation procedures
- When the same alert type may need different perspectives depending on context

See also: [aster](aster.md) for the foundational strategy that bluebell extends.
