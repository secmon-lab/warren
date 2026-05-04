Create an execution plan for the following user request.

## User Message
{{ .message }}

# Planning Rules

1. **Phased execution**: You don't need to plan everything upfront. Plan only the immediate next phase of tasks. After all tasks in a phase complete, you will be asked to replan.
2. **Parallel execution**: ALL tasks in a phase are executed simultaneously in parallel. Tasks CANNOT depend on each other within the same phase.
3. **Sequential dependencies**: If a task depends on the result of another task, put the dependent task in a later phase (via replan).
4. **Replan opportunity**: After each phase completes, you will see all task results and can add new tasks, adjust the approach, or finish.
5. **Direct response**: If the question can be answered from your existing context or from knowledge search results alone, set tasks to an empty array and put your answer in the message field. Do not create tasks when no additional tool execution is needed.
6. **Message required**: Always include a message — it will be shown to the user as an initial response before tasks begin.
7. **Tool assignment**: Each task must specify which ToolSet names it needs. Only specified ToolSets will be available to that task.
8. **Clear purpose**: Each task must have a clear, specific purpose. The description should be detailed enough for an independent agent to execute it.
9. **Acceptance criteria per task**: Each task MUST have clear acceptance criteria that specify the concrete conditions under which that task is considered complete. This helps evaluate progress during replanning.
10. **LLM selection per task**: Each task MUST set `llm_id` to one of the entries in the "Available LLMs" section below. Choose the cheapest LLM whose description fits the task — do not always pick the most powerful one. Routing tasks to lighter LLMs is the primary cost control.

# Available LLMs

You MUST select one LLM for each task by setting `llm_id`. Choose based on the task's complexity, expected token volume, and tool usage. Read each entry's description carefully — descriptions document when to use that entry and when to avoid it.

{{ .available_llms }}

Selection guidelines:
- Lightweight summaries / single-tool lookups / simple yes-or-no checks → pick the cheapest entry whose description matches
- Multi-step tool use, deep reasoning, security incident triage → pick a high-capability entry
- When two entries are functionally equivalent (e.g. quota distribution variants), distribute load across them so no single quota pool is exhausted

# Response Format

Respond with a JSON object containing:
- `message`: Initial response message to show the user (required, use Slack markdown format)
- `tasks`: Array of tasks to execute in parallel (can be empty for direct responses)

Each task must have:
- `id`: Unique identifier (e.g., "task-1", "check-ip", "analyze-logs")
- `title`: Short descriptive title
- `description`: Detailed instructions for the task agent
- `acceptance_criteria`: A single clear, measurable condition that defines when this task is complete (required). Must be a concrete, verifiable statement (e.g., "Determine whether the source IP is malicious or benign", "Confirm or rule out data exfiltration").
- `tools`: Array of ToolSet names this task needs
- `llm_id`: ID of the LLM to use (must match one of the entries in the "Available LLMs" section)
