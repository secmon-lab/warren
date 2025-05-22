package base

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/urfave/cli/v3"
)

type Base struct {
	alertIDs []types.AlertID
	repo     interfaces.Repository
	ticketID types.TicketID
	policies map[string]string
}

var _ interfaces.Tool = &Base{}

func (x *Base) Helper() *cli.Command {
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

func New(repo interfaces.Repository, alertIDs []types.AlertID, policies map[string]string, ticketID types.TicketID) *Base {
	return &Base{
		alertIDs: alertIDs,
		repo:     repo,
		ticketID: ticketID,
		policies: policies,
	}
}

func (x *Base) Name() string {
	return "base"
}

func (x *Base) Flags() []cli.Flag {
	return nil
}

func (x *Base) Configure(ctx context.Context) error {
	return nil
}

func (x *Base) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("alerts.length", len(x.alertIDs)),
		slog.String("ticket.id", string(x.ticketID)),
	)
}

func (x *Base) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "base.alerts.get",
			Description: "Get a set of alerts with pagination support",
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
			Name:        "base.alert.search",
			Description: "Search the alerts by the given query. You can specify the path as Firestore path, and the operation and value to filter the alerts.",
			Parameters: map[string]*gollem.Parameter{
				"path": {
					Type:        gollem.TypeString,
					Description: "The path of the alert parameter to filter",
				},
				"op": {
					Type:        gollem.TypeString,
					Description: "The operation of the alert",
					Enum: []string{
						"==",
						"!=",
						">",
						">=",
						"<",
						"<=",
					},
				},
				"value": {
					Type:        gollem.TypeString,
					Description: "The value of the alert",
				},
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of alerts to return. Default is 25.",
				},
				"offset": {
					Type:        gollem.TypeInteger,
					Description: "Number of alerts to skip. Default is 0.",
				},
			},
			Required: []string{"path", "op", "value"},
		},
		{
			Name:        "base.policy.list",
			Description: "List all policies",
		},
		{
			Name:        "base.policy.get",
			Description: "Get a policy by name",
			Parameters: map[string]*gollem.Parameter{
				"name": {
					Type:        gollem.TypeString,
					Description: "The name of the policy",
				},
			},
			Required: []string{"name"},
		},
	}, nil
}

func (x *Base) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "base.alerts.get":
		return x.getAlerts(ctx, args)
	case "base.alert.search":
		return x.searchAlerts(ctx, args)
		/* TODO:
		case "base.alert.similar":
			return x.getSimilarAlerts(ctx, args)
		*/
	case "base.policy.list":
		return x.listPolicies(ctx, args)
	case "base.policy.get":
		return x.getPolicy(ctx, args)
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}
