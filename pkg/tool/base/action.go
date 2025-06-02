package base

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/urfave/cli/v3"
)

type Warren struct {
	repo         interfaces.Repository
	ticketID     types.TicketID
	policyClient interfaces.PolicyClient
	ticket       *ticket.Ticket
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

	typedVal, ok := val.(T)
	if !ok {
		return null, goerr.New("key is not a", goerr.V("key", key))
	}

	return typedVal, nil
}

func New(repo interfaces.Repository, policy interfaces.PolicyClient, ticketID types.TicketID) *Warren {
	return &Warren{
		repo:         repo,
		ticketID:     ticketID,
		policyClient: policy,
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
		slog.Int("alerts.number", len(x.ticket.AlertIDs)),
		slog.String("ticket.id", string(x.ticketID)),
	)
}

const (
	cmdGetAlerts         = "warren.get_alerts"
	cmdFindNearestTicket = "warren.find_nearest_ticket"
	cmdListPolicies      = "warren.list_policies"
	cmdGetPolicy         = "warren.get_policy"
	cmdExecPolicy        = "warren.exec_policy"
)

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
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}
