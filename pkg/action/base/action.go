package base

import (
	"context"
	"encoding/json"
	"log/slog"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/urfave/cli/v3"
)

type Base struct {
	alerts alert.Alerts
	repo   interfaces.Repository
}

func New(repo interfaces.Repository, alerts alert.Alerts) *Base {
	return &Base{
		alerts: alerts,
		repo:   repo,
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

func (x *Base) Specs() []*genai.FunctionDeclaration {
	return []*genai.FunctionDeclaration{
		{
			Name:        "base.get_alerts",
			Description: "Get a set of alerts with pagination support",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"limit": {
						Type:        genai.TypeInteger,
						Description: "Maximum number of alerts to return",
					},
					"offset": {
						Type:        genai.TypeInteger,
						Description: "Number of alerts to skip",
					},
				},
			},
		},
		{
			Name:        "base.exit",
			Description: "End the agent session and submit the final conclusion",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"conclusion": {
						Type:        genai.TypeString,
						Description: "The final conclusion in Slack markdown format",
					},
				},
			},
		},
	}
}

func (x *Base) Execute(ctx context.Context, name string, args map[string]any) (*action.Result, error) {
	switch name {
	case "base.get_alerts":
		return x.getAlerts(ctx, args)
	case "base.exit":
		return x.exit(ctx, args)
	default:
		return nil, goerr.New("unknown function", goerr.V("name", name))
	}
}

func (x *Base) getAlerts(ctx context.Context, args map[string]any) (*action.Result, error) {
	var limit, offset int64

	if limitVal, ok := args["limit"].(float64); ok {
		limit = int64(limitVal)
	}
	if offsetVal, ok := args["offset"].(float64); ok {
		offset = int64(offsetVal)
	}

	alerts := x.alerts[:]

	// Apply pagination
	if offset > 0 {
		if offset >= int64(len(alerts)) {
			alerts = nil
		} else {
			alerts = alerts[offset:]
		}
	}

	if limit > 0 && limit < int64(len(alerts)) {
		alerts = alerts[:limit]
	}

	var rows []string
	for _, alert := range alerts {
		raw, err := json.Marshal(alert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal alert")
		}
		rows = append(rows, string(raw))
	}

	return &action.Result{
		Message: "Retrieved alerts",
		Type:    action.ResultTypeJSON,
		Rows:    rows,
	}, nil
}

func (x *Base) exit(ctx context.Context, args map[string]any) (*action.Result, error) {
	conclusion, ok := args["conclusion"].(string)
	if !ok {
		return nil, goerr.New("conclusion is required", goerr.V("args", args))
	}

	return &action.Result{
		Message: conclusion,
		Type:    action.ResultTypeText,
	}, nil
}
