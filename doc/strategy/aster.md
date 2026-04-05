# Aster Strategy

Aster is Warren's default chat strategy, designed for security alert investigation.

## Design Philosophy

**Simplicity and robustness.** Aster focuses on security investigation with a single-path design. One system prompt drives a consistent investigation across all phases, while multi-agent parallel task execution handles data collection separately from analysis. The strategy works with minimal configuration — knowledge service is optional, and a single `--user-system-prompt` file is sufficient for customization.

The core principle is **separation of concerns**: task agents collect data without analysis (preventing bias), while the planner synthesizes findings into actionable conclusions.

## Execution Flow

```
Plan → Parallel Tasks → Replan → ... → Final Response
```

1. **Plan**: The planner creates a set of parallel investigation tasks based on the user's message and alert context. If the question can be answered from context alone, it responds directly (tasks=[]).
2. **Task Execution**: All tasks in a phase run simultaneously. Each task agent uses filtered tools to collect raw data — no analysis, no interpretation.
3. **Replan**: After tasks complete, the planner reviews results and decides: create more tasks, ask the user a question, or proceed to final response.
4. **Final Response**: Synthesizes all task results into a coherent security assessment.

## System Prompt Architecture

Aster uses two separate system prompt templates:
- `system.md` — For ticket-based investigations (includes ticket/alert JSON)
- `ticketless_system.md` — For general questions without a ticket

The system prompt is generated once and shared across Plan, Replan, and Final phases. Task agents use a separate `task.md` prompt.

### User System Prompt

A single prompt file can be provided via `--user-system-prompt <path>` to add custom instructions. The file content is injected as-is into the `## User System Prompt` section of the system prompt template.

## Configuration

| Flag | Env Var | Description |
|------|---------|-------------|
| `--chat-strategy aster` | `WARREN_CHAT_STRATEGY` | Select aster (default) |
| `--user-system-prompt <path>` | `WARREN_USER_SYSTEM_PROMPT` | Path to custom prompt file |
| `--budget-strategy <type>` | `WARREN_BUDGET_STRATEGY` | `none` or `default` |

## Features

- **Budget Tracking**: Limits tool execution cost per task with soft/hard limits and graceful handover
- **HITL (Human-in-the-Loop)**: Requires human approval for sensitive tools (e.g., `web_fetch`)
- **Knowledge Service** (optional): When configured, the planner can search the knowledge base before creating tasks
- **Session Monitoring**: Background polling detects user-initiated session aborts

## When to Use Aster

- Standard security alert investigation
- Simple setups without multiple investigation approaches
- When knowledge service is not available
- When a single, consistent investigation perspective is sufficient

See also: [bluebell](bluebell.md) for adaptive investigation with multiple prompts.
