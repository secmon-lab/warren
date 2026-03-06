# Chat Strategy Guide

Warren provides two chat execution strategies for AI-powered security alert analysis. The strategy determines how the AI agent plans and executes tasks when responding to user messages in Slack threads.

## Overview

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `legacy` (default) | Sequential plan-and-execute using gollem's `planexec` strategy | Stable, proven approach for most workloads |
| `swarm` | Parallel task execution with phased planning | Faster investigations with multiple independent tasks |

## Selecting a Strategy

Set the `--chat-strategy` flag (or `WARREN_CHAT_STRATEGY` environment variable) when running `warren serve`:

```bash
# Use swarm strategy
warren serve --chat-strategy swarm

# Or via environment variable
export WARREN_CHAT_STRATEGY=swarm
warren serve
```

The default is `legacy` for backward compatibility.

## Legacy Strategy

The legacy strategy uses gollem's built-in `planexec` approach:

1. The LLM creates a plan with sequential steps
2. Each step is executed one at a time
3. After each step, the LLM reflects and adjusts
4. Results are accumulated and a final response is generated

This is a straightforward, sequential execution model. It works well when tasks have natural dependencies or when the investigation flow is linear.

## Swarm Strategy

The swarm strategy introduces parallel task execution with a phased planning model. It is designed for investigations where multiple independent actions can be performed simultaneously.

### Execution Flow

```
User message
    |
    v
[Planning Phase] -- LLM generates a structured plan (tasks + message)
    |
    v
[Post initial message to Slack]
    |
    v
[Phase 1: Execute all tasks in parallel (goroutines)]
    |
    v
[Replan] -- LLM reviews results, decides next steps
    |
    +-- New tasks? --> [Phase 2: Execute in parallel] --> [Replan] --> ...
    |
    +-- No more tasks --> [Final Response] --> Done
```

### Key Concepts

#### Phased Execution

Tasks are organized into phases. All tasks within a single phase run in parallel. If a task depends on the result of another, the planner places it in a subsequent phase via the replan step.

#### Planning

The planner LLM receives:
- The user's message
- Ticket information and a representative alert (JSON)
- A list of all available tools and sub-agents (names + descriptions)
- Past agent memories (if memory service is configured)
- Chat history from previous interactions on the same ticket

It outputs a structured JSON plan containing an initial message and a list of tasks.

#### Tasks

Each task specifies:
- **Title and description**: What the task should accomplish
- **Tools**: Which tools the task agent can use (filtered from the full set)
- **Sub-agents**: Which sub-agents the task agent can invoke (e.g., `falcon`, `bigquery`, `slack`)

Each task runs as an independent gollem agent in its own goroutine with:
- Its own Slack context block showing progress (prefixed with `[Task Title]`)
- Only the tools and sub-agents explicitly assigned to it
- Error isolation from other tasks

#### Replan

After all tasks in a phase complete, the planner reviews:
- All results from all completed phases
- The original user question
- Available tools and sub-agents

It then decides whether to:
- Add new tasks for the next phase
- Return an empty task list to proceed to final response generation

#### Direct Response

If the user's question can be answered immediately without tool usage, the planner returns an empty task list with just a message. No task execution or replan occurs.

### Error Handling

- **Task failure**: If one task fails, other parallel tasks continue. The error is reported to the replanner, which decides whether to retry or proceed.
- **Planning failure**: If the planning LLM call fails, the session is aborted and the user is notified.
- **Phase limit**: A maximum phase count (default: 10) prevents infinite replan loops. When reached, the system proceeds to final response with whatever results are available.

### Slack Progress Display

Each task gets its own updatable Slack message (context block) with the task title as prefix:

```
*[Check source IP]* Querying VirusTotal...
*[Analyze logs]* Running BigQuery query...
*[Search similar tickets]* Found 3 matches
```

The overall plan progress is shown in a separate context block displaying completed and in-progress phases.

### Trace Recording

The swarm strategy records structured traces compatible with Warren's trace system:

```
Root Agent Execute
  +-- planning (LLM call)
  +-- task-1 (agent execute with tools/sub-agents)
  +-- task-2 (agent execute with tools/sub-agents)
  +-- replan-phase-1 (LLM call)
  +-- task-3 (from replan)
  +-- final-response (LLM call)
```

Task agents and their sub-agent calls (Falcon, BigQuery, etc.) are nested under the task's trace span.

## Comparison

| Aspect | Legacy | Swarm |
|--------|--------|-------|
| Task execution | Sequential | Parallel within each phase |
| Planning model | gollem planexec strategy | Custom plan → execute → replan loop |
| Tool assignment | All tools available to agent | Per-task tool filtering |
| Sub-agent assignment | All sub-agents available | Per-task sub-agent filtering |
| Slack progress | Single trace message | Per-task context blocks |
| Error isolation | Agent stops on error | Failed tasks don't affect others |
| History | Maintained across sessions | Maintained via planning session |
| Memory integration | N/A | Searches past insights for planning |

## Configuration Reference

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `WARREN_CHAT_STRATEGY` | `--chat-strategy` | `legacy` | Chat execution strategy (`legacy` or `swarm`) |
