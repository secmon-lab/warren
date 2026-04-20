package bluebell

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	knowledgeModel "github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/system.md
var systemPromptRaw string

//go:embed prompt/plan.md
var planPromptTemplate string

//go:embed prompt/replan.md
var replanPromptTemplate string

//go:embed prompt/task.md
var taskPromptTemplate string

//go:embed prompt/final.md
var finalPromptTemplate string

// systemPromptTemplate has frontmatter stripped (system.md is the only prompt file with frontmatter).
var systemPromptTemplate string

func init() {
	systemPromptTemplate = stripFrontmatter(systemPromptRaw)
}

// stripFrontmatter removes YAML frontmatter (---...---) from the beginning of a string.
func stripFrontmatter(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return s
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.TrimLeft(strings.Join(lines[i+1:], "\n"), "\n")
		}
	}
	return s
}

// SystemPromptData is the data passed to system.md template.
// Field names map directly to template variables (e.g., {{ .Context.Ticket }}).
type SystemPromptData struct {
	Context          ContextData
	Tools            ToolsData
	Knowledge        KnowledgeData
	UserSystemPrompt string // user-provided static system prompt (--user-system-prompt, environment info etc.)
	ResolvedIntent   string // situation-specific investigation directive from intent resolver
	Lang             lang.Lang
	Requester        Requester
}

// ContextData holds the investigation target context.
// Ticket/Alert fields are populated for ticket-based chat;
// Channel.History is populated for ticketless chat.
type ContextData struct {
	Ticket  string
	Alert   AlertData
	Thread  ThreadData
	Channel ChannelData
}

// AlertData holds representative alert information.
type AlertData struct {
	Data  string
	Count int
}

// ThreadData holds the session.Message timeline for the current Session.
type ThreadData struct {
	SessionMessages []*sessModel.Message
}

// ChannelData holds Slack channel history messages.
type ChannelData struct {
	History []model.HistoryMessage
}

// ToolsData holds tool descriptions.
type ToolsData struct {
	Description string
}

// KnowledgeData holds knowledge base tag information.
type KnowledgeData struct {
	Tags []*knowledgeModel.KnowledgeTag
}

// Requester holds requester information for mentions.
type Requester struct {
	ID string
}

// planningContext holds the shared context for planning operations.
type planningContext struct {
	message          string
	ticket           *ticket.Ticket
	alerts           []*alert.Alert
	tools            []interfaces.ToolSet
	userSystemPrompt string
	resolvedIntent   string
	lang             lang.Lang
	requesterID      string
	sessionMessages  []*sessModel.Message
	slackHistory     []model.HistoryMessage
	knowledgeService *svcknowledge.Service
}

// generateSystemPrompt generates the shared system prompt.
// Handles both ticket-based and ticketless chat via template conditionals.
func generateSystemPrompt(ctx context.Context, pc *planningContext) (string, error) {
	data := SystemPromptData{
		Tools: ToolsData{
			Description: describeTools(ctx, pc.tools),
		},
		Knowledge: KnowledgeData{
			Tags: fetchKnowledgeTags(ctx, pc.knowledgeService),
		},
		UserSystemPrompt: pc.userSystemPrompt,
		ResolvedIntent:   pc.resolvedIntent,
		Lang:             pc.lang,
		Requester: Requester{
			ID: pc.requesterID,
		},
	}

	// Populate ticket/alert context if available (ticket-based chat)
	if pc.ticket != nil && pc.ticket.ID != "" {
		ticketJSON, alertJSON, alertCount := marshalContext(pc)
		data.Context = ContextData{
			Ticket: ticketJSON,
			Alert: AlertData{
				Data:  alertJSON,
				Count: alertCount,
			},
			Thread: ThreadData{
				SessionMessages: pc.sessionMessages,
			},
		}
	}

	// Populate channel history (for ticketless or as additional context)
	if len(pc.slackHistory) > 0 {
		data.Context.Channel = ChannelData{
			History: pc.slackHistory,
		}
	}

	return prompt.GenerateWithStruct(ctx, systemPromptTemplate, data)
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
func generateTaskPrompt(ctx context.Context, task TaskPlan, knowledgeSvc *svcknowledge.Service) (string, error) {
	data := map[string]any{
		"title":               task.Title,
		"description":         task.Description,
		"acceptance_criteria": task.AcceptanceCriteria,
	}

	if knowledgeSvc != nil {
		tags, err := knowledgeSvc.ListTags(ctx)
		if err != nil {
			logging.From(ctx).Warn("failed to list knowledge tags for task prompt", "error", err)
		} else if len(tags) > 0 {
			data["knowledge_tags"] = tags
		}
	}

	return prompt.GenerateWithStruct(ctx, taskPromptTemplate, data)
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

// describeTools generates a text description of available tools grouped by ToolSet.
func describeTools(ctx context.Context, toolSets []interfaces.ToolSet) string {
	var b strings.Builder
	for _, ts := range toolSets {
		specs, err := ts.Specs(ctx)
		if err != nil {
			continue
		}
		var toolNames []string
		for _, spec := range specs {
			toolNames = append(toolNames, spec.Name)
		}
		fmt.Fprintf(&b, "- `%s` — %s\n", ts.ID(), ts.Description())
		if len(toolNames) > 0 {
			fmt.Fprintf(&b, "  Tools: %s\n", strings.Join(toolNames, ", "))
		}

		additionalPrompt, err := ts.Prompt(ctx)
		if err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "failed to get prompt from tool set", goerr.V("id", ts.ID())))
		} else if additionalPrompt != "" {
			fmt.Fprintf(&b, "  %s\n", additionalPrompt)
		}
	}
	if b.Len() == 0 {
		return "(no tools available)"
	}
	return b.String()
}

// formatCompletedResults formats all completed phase results for inclusion in prompts.
func formatCompletedResults(allResults []*phaseResult) string {
	var b strings.Builder
	for _, pr := range allResults {
		fmt.Fprintf(&b, "## Phase %d\n\n", pr.phase)

		if pr.questionResult != nil {
			qr := pr.questionResult
			fmt.Fprintf(&b, "### Question Asked to Operator\n")
			fmt.Fprintf(&b, "**Question**: %s\n", qr.Question)
			fmt.Fprintf(&b, "**Options**: %s\n", strings.Join(qr.Options, ", "))
			fmt.Fprintf(&b, "**Selected**: %s\n", qr.Answer)
			if qr.Comment != "" {
				fmt.Fprintf(&b, "**Comment**: %s\n", qr.Comment)
			}
			fmt.Fprintln(&b)
			continue
		}

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

// fetchKnowledgeTags retrieves all knowledge tags for embedding in the planner prompt.
func fetchKnowledgeTags(ctx context.Context, svc *svcknowledge.Service) []*knowledgeModel.KnowledgeTag {
	if svc == nil {
		return nil
	}
	tags, err := svc.ListTags(ctx)
	if err != nil {
		logging.From(ctx).Warn("failed to list knowledge tags for planning", "error", err)
		return nil
	}
	return tags
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
			Description: "ToolSet names to make available for this task",
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

var questionSchema = &gollem.Parameter{
	Type: gollem.TypeObject,
	Properties: map[string]*gollem.Parameter{
		"question": {
			Type:        gollem.TypeString,
			Description: "The question to ask the security operator",
			Required:    true,
		},
		"options": {
			Type: gollem.TypeArray,
			Items: &gollem.Parameter{
				Type: gollem.TypeString,
			},
			Description: "Answer choices for the operator to select from. Must be specific and comprehensive. The last option MUST always be 'None of the above' or equivalent.",
			Required:    true,
		},
		"reason": {
			Type:        gollem.TypeString,
			Description: "Why this question is needed — what information gap it fills",
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
		"question": {
			Type:        gollem.TypeObject,
			Description: "Ask the security operator a question. Use ONLY as a last resort after exhausting all available tools. If set, tasks are ignored — the question is asked first, then replanning occurs with the answer. The question MUST include concrete answer choices (options). Do NOT set this if the information can be obtained through tools (BigQuery, VirusTotal, WHOIS, etc.).",
			Properties:  questionSchema.Properties,
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
