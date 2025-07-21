package base

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/urfave/cli/v3"
)

// SlackUpdateFunc is a callback function to update Slack messages when ticket is updated
type SlackUpdateFunc func(ctx context.Context, ticket *ticket.Ticket) error

type Warren struct {
	repo        interfaces.Repository
	ticketID    types.TicketID
	slackUpdate SlackUpdateFunc
	llmClient   gollem.LLMClient
}

var _ interfaces.Tool = &Warren{}

func (x *Warren) Helper() *cli.Command {
	return nil
}

func getArg[T any](args map[string]any, key string) (T, error) {
	var null T
	val, ok := args[key]
	if !ok {
		return null, nil
	}

	// Handle special case for numeric types from JSON (which come as float64)
	if reflect.TypeOf(null).Kind() == reflect.Int64 {
		if floatVal, ok := val.(float64); ok {
			result := int64(floatVal)
			return any(result).(T), nil
		}
	}

	typedVal, ok := val.(T)
	if !ok {
		return null, goerr.New("invalid parameter type",
			goerr.V("key", key),
			goerr.V("expected_type", reflect.TypeOf(null).String()),
			goerr.V("actual_type", reflect.TypeOf(val).String()),
			goerr.V("value", val))
	}

	return typedVal, nil
}

func New(repo interfaces.Repository, ticketID types.TicketID, opts ...func(*Warren)) *Warren {
	w := &Warren{
		repo:     repo,
		ticketID: ticketID,
	}

	for _, opt := range opts {
		opt(w)
	}

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

func (x *Warren) Name() string {
	return "warren"
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
	cmdGetAlerts            = "warren_get_alerts"
	cmdFindNearestTicket    = "warren_find_nearest_ticket"
	cmdSearchTicketsByWords = "warren_search_tickets_by_words"
	cmdUpdateFinding        = "warren_update_finding"
	cmdGetTicketComments    = "warren_get_ticket_comments"

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

func (x *Warren) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        cmdGetAlerts,
			Description: "Get a set of alerts that is bound to the ticket with pagination support",
			Parameters: map[string]*gollem.Parameter{
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of alerts to return",
				},
				"offset": {
					Type:        gollem.TypeInteger,
					Description: "Number of alerts to skip",
				},
			},
		},
		{
			Name:        cmdFindNearestTicket,
			Description: "Search the previous tickets that are similar to the current ticket",
			Parameters: map[string]*gollem.Parameter{
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of tickets to return",
				},
				"duration": {
					Type:        gollem.TypeInteger,
					Description: "Duration of the ticket in days",
				},
			},
		},
		{
			Name:        cmdSearchTicketsByWords,
			Description: "Search tickets using natural language query or keywords. Uses semantic similarity to find relevant tickets.",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "Search query using natural language or keywords to find similar tickets",
				},
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of tickets to return (default: 10)",
				},
				"duration": {
					Type:        gollem.TypeInteger,
					Description: "Duration to search back in days (default: 30)",
				},
			},
			Required: []string{"query"},
		},
		{
			Name:        cmdUpdateFinding,
			Description: "Update the finding information of the current ticket with analysis results",
			Parameters: map[string]*gollem.Parameter{
				"summary": {
					Type:        gollem.TypeString,
					Description: "Summary of the investigation results analyzed by the agent",
				},
				"severity": {
					Type:        gollem.TypeString,
					Description: "Severity level of the finding. Must be one of: 'low', 'medium', 'high', 'critical'",
				},
				"reason": {
					Type:        gollem.TypeString,
					Description: "Detailed reasoning and justification for the severity assessment",
				},
				"recommendation": {
					Type:        gollem.TypeString,
					Description: "Recommended actions based on the analysis results",
				},
			},
			Required: []string{"summary", "severity", "reason", "recommendation"},
		},
		{
			Name:        cmdGetTicketComments,
			Description: "Get comments associated with the current ticket with pagination support",
			Parameters: map[string]*gollem.Parameter{
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of comments to return (default: 50)",
				},
				"offset": {
					Type:        gollem.TypeInteger,
					Description: "Number of comments to skip for pagination (default: 0)",
				},
			},
		},
	}, nil
}

func (x *Warren) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case cmdGetAlerts:
		return x.getAlerts(ctx, args)
	case cmdFindNearestTicket:
		return x.findNearestTicket(ctx, args)
	case cmdSearchTicketsByWords:
		return x.searchTicketsByWords(ctx, args)
	case cmdUpdateFinding:
		return x.updateFinding(ctx, args)
	case cmdGetTicketComments:
		return x.getTicketComments(ctx, args)
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}
