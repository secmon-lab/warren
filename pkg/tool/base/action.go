package base

import (
	"context"
	"log/slog"

	"github.com/gollem-dev/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/toolset"
	"github.com/urfave/cli/v3"
)

// SlackUpdateFunc is a callback function to update Slack messages when ticket is updated
type SlackUpdateFunc func(ctx context.Context, ticket *ticket.Ticket) error

type Warren struct {
	repo        interfaces.Repository
	ticketID    types.TicketID
	slackUpdate SlackUpdateFunc
	llmClient   gollem.LLMClient

	tools gollem.ToolSet
}

var _ interfaces.Tool = &Warren{}

func (x *Warren) Helper() *cli.Command {
	return nil
}

func New(repo interfaces.Repository, ticketID types.TicketID, opts ...func(*Warren)) *Warren {
	w := &Warren{
		repo:     repo,
		ticketID: ticketID,
	}

	for _, opt := range opts {
		opt(w)
	}

	// Build the type-safe tool set once the dependencies from opts are wired in.
	// Each tool's schema is inferred from its typed input struct, replacing the
	// hand-written ToolSpec literals and the getArg map[string]any extraction.
	w.tools = toolset.New(
		gollem.MustNewTool(cmdGetAlerts, descGetAlerts, w.getAlerts),
		gollem.MustNewTool(cmdFindNearestTicket, descFindNearestTicket, w.findNearestTicket),
		gollem.MustNewTool(cmdSearchTicketsByWords, descSearchTicketsByWords, w.searchTicketsByWords),
		gollem.MustNewTool(cmdUpdateFinding, descUpdateFinding, w.updateFinding),
		gollem.MustNewTool(cmdGetTicketSessionMessages, descGetTicketSessionMessages, w.getTicketSessionMessages),
		gollem.MustNewTool(cmdSearchSessionMessages, descSearchSessionMessages, w.searchSessionMessages),
	)

	return w
}

func WithSlackUpdate(updateFunc SlackUpdateFunc) func(*Warren) {
	return func(w *Warren) {
		w.slackUpdate = updateFunc
	}
}

func WithLLMClient(client gollem.LLMClient) func(*Warren) {
	return func(w *Warren) {
		w.llmClient = client
	}
}

func (x *Warren) ID() string {
	return "warren"
}

func (x *Warren) Description() string {
	return "Warren ticket operations including alerts, findings, and comments"
}

func (x *Warren) Flags() []cli.Flag {
	return nil
}

func (x *Warren) Configure(ctx context.Context) error {
	return nil
}

func (x *Warren) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("ticket.id", string(x.ticketID)),
	)
}

// Prompt returns additional instructions for the system prompt
func (x *Warren) Prompt(ctx context.Context) (string, error) {
	return "", nil
}

const (
	cmdGetAlerts                = "warren_get_alerts"
	cmdFindNearestTicket        = "warren_find_nearest_ticket"
	cmdSearchTicketsByWords     = "warren_search_tickets_by_words"
	cmdUpdateFinding            = "warren_update_finding"
	cmdGetTicketSessionMessages = "warren_get_ticket_session_messages"
	cmdSearchSessionMessages    = "warren_search_session_messages"

	// Default values for search operations
	DefaultSearchTicketsLimit    = 10
	DefaultSearchTicketsDuration = 30
	DefaultCommentsLimit         = 50
)

func IgnorableTool(name string) bool {
	switch name {
	case cmdUpdateFinding:
		return true
	default:
		return false
	}
}

// Tool descriptions. Kept as constants so the typed-tool registration in New
// stays readable and the wire-level descriptions remain unchanged.
const (
	descGetAlerts                = "Get a set of alerts that is bound to the ticket with pagination support"
	descFindNearestTicket        = "Search the previous tickets that are similar to the current ticket"
	descSearchTicketsByWords     = "Search tickets using natural language query or keywords. Uses semantic similarity to find relevant tickets."
	descUpdateFinding            = "Update the finding information of the current ticket with analysis results"
	descGetTicketSessionMessages = "Get chat messages (user inputs, AI responses, traces, plans, warnings) from every Session attached to the current ticket. Supersedes warren_get_ticket_comments by returning both human-authored messages and AI-produced outputs across Slack/Web/CLI channels in a unified shape."
	descSearchSessionMessages    = "Full-text search across every Session.Message attached to the current ticket. Returns the top matching messages (case-insensitive substring match). Use this when you need to look up prior discussion or investigation traces by keyword rather than by source/type."
)

// Typed inputs for each tool. The schema (field names, types, required,
// descriptions) is inferred from these struct tags by gollem.NewTool.
type getAlertsInput struct {
	Limit  int64 `json:"limit" description:"Maximum number of alerts to return"`
	Offset int64 `json:"offset" description:"Number of alerts to skip"`
}

type findNearestTicketInput struct {
	Limit    int64 `json:"limit" description:"Maximum number of tickets to return"`
	Duration int64 `json:"duration" description:"Duration of the ticket in days"`
}

type searchTicketsByWordsInput struct {
	Query    string `json:"query" required:"true" description:"Search query using natural language or keywords to find similar tickets"`
	Limit    int64  `json:"limit" description:"Maximum number of tickets to return (default: 10)"`
	Duration int64  `json:"duration" description:"Duration to search back in days (default: 30)"`
}

type updateFindingInput struct {
	Summary        string `json:"summary" required:"true" description:"Summary of the investigation results analyzed by the agent"`
	Severity       string `json:"severity" required:"true" description:"Severity level of the finding. Must be one of: 'low', 'medium', 'high', 'critical'"`
	Reason         string `json:"reason" required:"true" description:"Detailed reasoning and justification for the severity assessment"`
	Recommendation string `json:"recommendation" required:"true" description:"Recommended actions based on the analysis results"`
}

type getTicketSessionMessagesInput struct {
	Source string `json:"source" description:"Optional filter by Session source: 'slack', 'web', or 'cli'. Omit to include all sources."`
	Type   string `json:"type" description:"Optional filter by Message type: 'user', 'trace', 'plan', 'response', or 'warning'. Omit to include all types."`
	Limit  int64  `json:"limit" description:"Maximum number of messages to return (default: 50)"`
	Offset int64  `json:"offset" description:"Number of messages to skip for pagination (default: 0)"`
}

type searchSessionMessagesInput struct {
	Query string `json:"query" required:"true" description:"Case-insensitive substring to search for across message content."`
	Limit int64  `json:"limit" description:"Maximum number of messages to return (default: 50)"`
}

func (x *Warren) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return x.tools.Specs(ctx)
}

func (x *Warren) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return x.tools.Run(ctx, name, args)
}
