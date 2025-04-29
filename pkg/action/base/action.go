package base

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollam"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/urfave/cli/v3"
)

type Base struct {
	alertIDs  []types.AlertID
	repo      interfaces.Repository
	sessionID types.SessionID
	policies  map[string]string
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

func New(repo interfaces.Repository, alertIDs []types.AlertID, policies map[string]string, sessionID types.SessionID) *Base {
	return &Base{
		alertIDs:  alertIDs,
		repo:      repo,
		sessionID: sessionID,
		policies:  policies,
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
	return slog.GroupValue()
}

func (x *Base) Specs(ctx context.Context) ([]gollam.ToolSpec, error) {
	return []gollam.ToolSpec{
		{
			Name:        "base.alerts.get",
			Description: "Get a set of alerts with pagination support",
			Parameters: map[string]*gollam.Parameter{
				"limit": {
					Type:        gollam.TypeInteger,
					Description: "Maximum number of alerts to return",
				},
				"offset": {
					Type:        gollam.TypeInteger,
					Description: "Number of alerts to skip",
				},
			},
		},
		{
			Name:        "base.alert.search",
			Description: "Search the alerts by the given query. You can specify the path as Firestore path, and the operation and value to filter the alerts.",
			Parameters: map[string]*gollam.Parameter{
				"path": {
					Type:        gollam.TypeString,
					Description: "The path of the alert",
				},
				"op": {
					Type:        gollam.TypeString,
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
					Type:        gollam.TypeString,
					Description: "The value of the alert",
				},
				"limit": {
					Type:        gollam.TypeInteger,
					Description: "Maximum number of alerts to return. Default is 25.",
				},
				"offset": {
					Type:        gollam.TypeInteger,
					Description: "Number of alerts to skip. Default is 0.",
				},
			},
			Required: []string{"path", "op", "value"},
		},
	}, nil
}

func (x *Base) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "base.alerts.get":
		return x.getAlerts(ctx, args)
	case "base.alert.search":
		return x.searchAlerts(ctx, args)
	case "base.alert.resolve":
		return x.resolveAlert(ctx, args)
	case "base.note.get":
		return x.getNotes(ctx, args)
	case "base.note.put":
		return x.putNote(ctx, args)
	case "base.policy.list":
		return x.listPolicies(ctx, args)
	case "base.policy.get":
		return x.getPolicy(ctx, args)
	default:
		return nil, goerr.New("invalid function name", goerr.V("name", name))
	}
}
