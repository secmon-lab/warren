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
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/urfave/cli/v3"
)

type Base struct {
	alerts    alert.Alerts
	repo      interfaces.Repository
	sessionID types.SessionID
}

func New(repo interfaces.Repository, alerts alert.Alerts, sessionID types.SessionID) *Base {
	return &Base{
		alerts:    alerts,
		repo:      repo,
		sessionID: sessionID,
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
			Name:        "base.alerts",
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
			Name:        "base.notes",
			Description: "Get the notes of the agent session",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"limit": {
						Type:        genai.TypeInteger,
						Description: "Maximum number of notes to return. Default is 10.",
					},
					"offset": {
						Type:        genai.TypeInteger,
						Description: "Number of notes to skip. Default is 0.",
					},
				},
			},
		},
	}
}

func (x *Base) Execute(ctx context.Context, name string, args map[string]any) (*action.Result, error) {
	switch name {
	case "exit":
		return x.exit(ctx, args)
	case "base.alerts":
		return x.getAlerts(ctx, args)
	case "base.notes":
		return x.getNotes(ctx, args)
	default:
		return nil, goerr.New("unknown function", goerr.V("name", name))
	}
}

func (x *Base) getAlerts(_ context.Context, args map[string]any) (*action.Result, error) {
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

func (x *Base) exit(_ context.Context, args map[string]any) (*action.Result, error) {
	conclusion, ok := args["conclusion"].(string)
	if !ok {
		return nil, goerr.New("conclusion is required", goerr.V("args", args))
	}

	return &action.Result{
		Message: conclusion,
		Type:    action.ResultTypeText,
	}, nil
}

func (x *Base) getNotes(ctx context.Context, args map[string]any) (*action.Result, error) {
	var limit, offset int64

	if limitVal, ok := args["limit"].(float64); ok {
		limit = int64(limitVal)
	}
	if offsetVal, ok := args["offset"].(float64); ok {
		offset = int64(offsetVal)
	}

	notes, err := x.repo.GetNotes(ctx, x.sessionID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get notes")
	}

	// Apply pagination
	if offset > 0 {
		if offset >= int64(len(notes)) {
			notes = nil
		} else {
			notes = notes[offset:]
		}
	}

	if limit > 0 && limit < int64(len(notes)) {
		notes = notes[:limit]
	}

	var rows []string
	for _, note := range notes {
		rows = append(rows, note.Note)
	}

	return &action.Result{
		Message: "Retrieved notes",
		Type:    action.ResultTypeText,
		Rows:    rows,
	}, nil
}
