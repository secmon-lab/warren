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
	repo         interfaces.Repository
	ticketID     types.TicketID
	policyClient interfaces.PolicyClient
	slackUpdate  SlackUpdateFunc
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

func New(repo interfaces.Repository, policy interfaces.PolicyClient, ticketID types.TicketID, opts ...func(*Warren)) *Warren {
	w := &Warren{
		repo:         repo,
		ticketID:     ticketID,
		policyClient: policy,
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
	cmdGetAlerts         = "warren.get_alerts"
	cmdFindNearestTicket = "warren.find_nearest_ticket"
	cmdListPolicies      = "warren.list_policies"
	cmdGetPolicy         = "warren.get_policy"
	cmdExecPolicy        = "warren.exec_policy"
	cmdUpdateFinding     = "warren.update_finding"
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
			Name:        cmdListPolicies,
			Description: "List all policies in Rego to detect an alert",
		},
		{
			Name:        cmdGetPolicy,
			Description: "Get a policy in Rego to detect an alert by name",
			Parameters: map[string]*gollem.Parameter{
				"name": {
					Type:        gollem.TypeString,
					Description: "The name of the policy. It must be in the list of policies returned by `warren.list_policies`",
				},
			},
			Required: []string{"name"},
		},
		{
			Name:        cmdExecPolicy,
			Description: "Execute a policy in Rego to detect an alert by schema and input data. It returns detected alert data.",
			Parameters: map[string]*gollem.Parameter{
				"schema": {
					Type:        gollem.TypeString,
					Description: "The schema of the alert. It must be in 'package line' in the policy.",
				},
				"input": {
					Type:        gollem.TypeString,
					Description: "The input data to the policy. It must be a JSON string.",
				},
			},
			Required: []string{"schema", "input"},
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
	}, nil
}

func (x *Warren) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case cmdGetAlerts:
		return x.getAlerts(ctx, args)
	case cmdFindNearestTicket:
		return x.findNearestTicket(ctx, args)
	case cmdListPolicies:
		return x.listPolicies(ctx, args)
	case cmdGetPolicy:
		return x.getPolicy(ctx, args)
	case cmdExecPolicy:
		return x.execPolicy(ctx, args)
	case cmdUpdateFinding:
		return x.updateFinding(ctx, args)
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}
