package swarm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
)

//go:embed prompt/system.md
var systemPromptTemplate string

//go:embed prompt/ticketless_system.md
var ticketlessSystemPromptTemplate string

//go:embed prompt/plan.md
var planPromptTemplate string

//go:embed prompt/replan.md
var replanPromptTemplate string

//go:embed prompt/task.md
var taskPromptTemplate string

//go:embed prompt/final.md
var finalPromptTemplate string

// planningContext holds the shared context for planning operations.
type planningContext struct {
	message        string
	ticket         *ticket.Ticket
	alerts         []*alert.Alert
	tools          []gollem.ToolSet
	subAgents      []*agent.SubAgent
	memoryContext  string
	userPrompt     string
	lang           lang.Lang
	requesterID    string
	threadComments []ticket.Comment
	knowledges     []*knowledge.Knowledge
	slackHistory   []model.HistoryMessage
}

// generateSystemPrompt generates the shared system prompt containing static context.
func generateSystemPrompt(ctx context.Context, pc *planningContext) (string, error) {
	ticketJSON, alertJSON, alertCount := marshalContext(pc)

	return prompt.GenerateWithStruct(ctx, systemPromptTemplate, map[string]any{
		"ticket_json":           ticketJSON,
		"alert_json":            alertJSON,
		"alert_count":           alertCount,
		"tools_description":     describeTools(ctx, pc.tools),
		"subagents_description": describeSubAgents(pc.subAgents),
		"memory_context":        pc.memoryContext,
		"user_prompt":           pc.userPrompt,
		"lang":                  pc.lang,
		"requester_id":          pc.requesterID,
		"thread_comments":       pc.threadComments,
		"knowledges":            pc.knowledges,
		"topic":                 pc.ticket.Topic,
		"history_messages":      pc.slackHistory,
	})
}

// generatePlanPrompt generates the planning user message prompt.
func generatePlanPrompt(ctx context.Context, pc *planningContext) (string, error) {
	return prompt.Generate(ctx, planPromptTemplate, map[string]any{
		"message": pc.message,
	})
}

// generateReplanPrompt generates the replan user message prompt with completed results.
func generateReplanPrompt(ctx context.Context, pc *planningContext, allResults []*phaseResult, currentPhase int) (string, error) {
	return prompt.Generate(ctx, replanPromptTemplate, map[string]any{
		"message":           pc.message,
		"completed_results": formatCompletedResults(allResults),
		"current_phase":     currentPhase,
	})
}

// generateTaskPrompt generates the system prompt for a task agent.
func generateTaskPrompt(ctx context.Context, task TaskPlan) (string, error) {
	return prompt.Generate(ctx, taskPromptTemplate, map[string]any{
		"title":               task.Title,
		"description":         task.Description,
		"acceptance_criteria": task.AcceptanceCriteria,
	})
}

// generateFinalPrompt generates the final response user message prompt.
func generateFinalPrompt(ctx context.Context, pc *planningContext, allResults []*phaseResult) (string, error) {
	return prompt.GenerateWithStruct(ctx, finalPromptTemplate, map[string]any{
		"message":           pc.message,
		"completed_results": formatCompletedResults(allResults),
		"lang":              pc.lang,
		"requester_id":      pc.requesterID,
	})
}

// ticketlessPlanningContext holds the shared context for ticketless planning operations.
type ticketlessPlanningContext struct {
	message       string
	tools         []gollem.ToolSet
	subAgents     []*agent.SubAgent
	memoryContext string
	userPrompt    string
	lang          lang.Lang
	requesterID   string
	knowledges    []*knowledge.Knowledge
	history       []model.HistoryMessage
}

// generateTicketlessSystemPrompt generates the system prompt for ticketless chat.
func generateTicketlessSystemPrompt(ctx context.Context, pc *ticketlessPlanningContext) (string, error) {
	return prompt.GenerateWithStruct(ctx, ticketlessSystemPromptTemplate, map[string]any{
		"history_messages":      pc.history,
		"tools_description":     describeTools(ctx, pc.tools),
		"subagents_description": describeSubAgents(pc.subAgents),
		"memory_context":        pc.memoryContext,
		"user_prompt":           pc.userPrompt,
		"lang":                  pc.lang,
		"requester_id":          pc.requesterID,
		"knowledges":            pc.knowledges,
	})
}

func marshalContext(pc *planningContext) (string, string, int) {
	ticketJSON, err := json.MarshalIndent(pc.ticket, "", "  ")
	if err != nil {
		ticketJSON = []byte(fmt.Sprintf(`{"error": "failed to marshal ticket: %s"}`, err))
	}

	alertCount := len(pc.alerts)
	var alertJSON []byte
	if alertCount > 0 {
		alertJSON, err = json.MarshalIndent(pc.alerts[0], "", "  ")
		if err != nil {
			alertJSON = []byte(fmt.Sprintf(`{"error": "failed to marshal alert: %s"}`, err))
		}
	} else {
		alertJSON = []byte(`{}`)
	}

	return string(ticketJSON), string(alertJSON), alertCount
}

// describeTools generates a text description of available tools.
func describeTools(ctx context.Context, toolSets []gollem.ToolSet) string {
	var b strings.Builder
	for _, ts := range toolSets {
		specs, err := ts.Specs(ctx)
		if err != nil {
			continue
		}
		for _, spec := range specs {
			fmt.Fprintf(&b, "- `%s`: %s\n", spec.Name, spec.Description)
		}

		if tool, ok := ts.(interfaces.Tool); ok {
			additionalPrompt, err := tool.Prompt(ctx)
			if err == nil && additionalPrompt != "" {
				fmt.Fprintf(&b, "  Additional: %s\n", additionalPrompt)
			}
		}
	}
	if b.Len() == 0 {
		return "(no tools available)"
	}
	return b.String()
}

// describeSubAgents generates a text description of available sub-agents.
func describeSubAgents(subAgents []*agent.SubAgent) string {
	if len(subAgents) == 0 {
		return "(no sub-agents available)"
	}
	var b strings.Builder
	for _, sa := range subAgents {
		inner := sa.Inner()
		spec := inner.Spec()
		fmt.Fprintf(&b, "- `%s`: %s\n", spec.Name, spec.Description)
		if hint := sa.PromptHint(); hint != "" {
			fmt.Fprintf(&b, "  Hint: %s\n", hint)
		}
	}
	return b.String()
}

// formatCompletedResults formats all completed phase results for inclusion in prompts.
func formatCompletedResults(allResults []*phaseResult) string {
	var b strings.Builder
	for _, pr := range allResults {
		fmt.Fprintf(&b, "## Phase %d\n\n", pr.phase)
		for i, r := range pr.results {
			fmt.Fprintf(&b, "### Task: %s (ID: %s)\n", pr.tasks[i].Title, pr.tasks[i].ID)
			if r.Error != nil {
				fmt.Fprintf(&b, "**Status**: Failed\n**Error**: %s\n\n", r.Error.Error())
			} else if r.BudgetExceeded {
				fmt.Fprintf(&b, "**Status**: Budget Exceeded (terminated early)\n**Result**:\n<task-output>\n%s\n</task-output>\n\n", r.Result)
			} else {
				fmt.Fprintf(&b, "**Status**: Completed\n**Result**:\n<task-output>\n%s\n</task-output>\n\n", r.Result)
			}
		}
	}
	if b.Len() == 0 {
		return "(no results yet)"
	}
	return b.String()
}

// FormatMemories formats agent memories for inclusion in planning prompt.
func FormatMemories(memories []*memory.AgentMemory) string {
	if len(memories) == 0 {
		return ""
	}
	var b strings.Builder
	for _, m := range memories {
		fmt.Fprintf(&b, "- **%s**: %s\n", m.Query, m.Claim)
	}
	return b.String()
}

// Schema definitions for structured LLM responses.

var taskSchema = &gollem.Parameter{
	Type: gollem.TypeObject,
	Properties: map[string]*gollem.Parameter{
		"id":          {Type: gollem.TypeString, Description: "Unique task ID", Required: true},
		"title":       {Type: gollem.TypeString, Description: "Short task title", Required: true},
		"description": {Type: gollem.TypeString, Description: "Detailed task instructions", Required: true},
		"acceptance_criteria": {
			Type:        gollem.TypeString,
			Description: "A clear, measurable condition that defines when this task is considered complete",
			Required:    true,
		},
		"tools": {
			Type:        gollem.TypeArray,
			Items:       &gollem.Parameter{Type: gollem.TypeString},
			Description: "Tool names to make available for this task",
			Required:    true,
		},
		"sub_agents": {
			Type:        gollem.TypeArray,
			Items:       &gollem.Parameter{Type: gollem.TypeString},
			Description: "Sub-agent names to make available for this task (e.g. falcon, bigquery, slack)",
			Required:    true,
		},
	},
}

var planSchema = &gollem.Parameter{
	Type: gollem.TypeObject,
	Properties: map[string]*gollem.Parameter{
		"message": {Type: gollem.TypeString, Description: "Initial message to display to the user", Required: true},
		"tasks": {
			Type:        gollem.TypeArray,
			Items:       taskSchema,
			Description: "Tasks to execute in parallel (empty for direct response)",
			Required:    true,
		},
	},
}

var replanSchema = &gollem.Parameter{
	Type: gollem.TypeObject,
	Properties: map[string]*gollem.Parameter{
		"message": {
			Type:        gollem.TypeString,
			Description: "Status update message about progress and next steps, shown to the user before the next phase",
		},
		"tasks": {
			Type:        gollem.TypeArray,
			Items:       taskSchema,
			Description: "New tasks for the next phase (empty = proceed to final response)",
			Required:    true,
		},
	},
}

// parsePlanResult parses the LLM response into a PlanResult.
func parsePlanResult(texts []string) (*PlanResult, error) {
	if len(texts) == 0 {
		return nil, goerr.New("no response from planning LLM")
	}
	var result PlanResult
	if err := json.Unmarshal([]byte(texts[0]), &result); err != nil {
		return nil, goerr.Wrap(err, "failed to parse plan result", goerr.V("raw", texts[0]))
	}
	return &result, nil
}

// parseReplanResult parses the LLM response into a ReplanResult.
func parseReplanResult(texts []string) (*ReplanResult, error) {
	if len(texts) == 0 {
		return nil, goerr.New("no response from replan LLM")
	}
	var result ReplanResult
	if err := json.Unmarshal([]byte(texts[0]), &result); err != nil {
		return nil, goerr.Wrap(err, "failed to parse replan result", goerr.V("raw", texts[0]))
	}
	return &result, nil
}
