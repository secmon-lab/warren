Previous task phases have completed. Review the results and decide what to do next.

# Original User Message
{{ .message }}

# Completed Task Results

{{ .completed_results }}

Current phase: {{ .current_phase }}

# Decision Instructions

1. Review all completed task results above
2. Evaluate the gathered information against the acceptance criteria established in the initial plan. Determine which criteria are already met and which still need investigation.
3. Consider whether the current approach is still valid given the results so far. If findings suggest the initial hypothesis was wrong or the approach is ineffective, adjust the direction accordingly.
4. If more information is needed, create new tasks for the next phase (all will run in parallel)
5. If sufficient information is gathered (all acceptance criteria met), return an empty tasks array to proceed to final response generation
6. If any tasks failed, decide whether to retry with a different approach or proceed without that information
7. Each new task must specify which tools and sub-agents it needs
8. If execution is difficult (e.g., all approaches have failed, results are inconclusive), try a different approach or proceed with the best available information.

# Response Format

Respond with a JSON object containing:
- `message`: (optional) A status update message to show the user about progress so far and what will be done next. Shown in Slack before the next phase begins.
- `tasks`: Array of new tasks for the next phase (empty array = proceed to final response)

Each task must have:
- `id`: Unique identifier
- `title`: Short descriptive title
- `description`: Detailed instructions
- `tools`: Array of tool names
- `sub_agents`: Array of sub-agent names

# Budget-Exceeded Tasks

If any tasks were terminated due to budget exhaustion, their results contain handover information about what was accomplished and what remains.
When replanning budget-exceeded tasks:
- Break the remaining work into smaller, more focused tasks
- Prioritize the most critical remaining investigation items
- Each new task should have a clear, achievable scope within budget constraints
- Avoid recreating the same broad task — specify exactly what data to collect
