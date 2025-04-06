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
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/urfave/cli/v3"
)

type Base struct {
	alerts    alert.Alerts
	repo      interfaces.Repository
	sessionID types.SessionID
	policies  map[string]string
}

func New(repo interfaces.Repository, alerts alert.Alerts, policies map[string]string, sessionID types.SessionID) *Base {
	return &Base{
		alerts:    alerts,
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
			Name:        "base.memo.get",
			Description: "Get the memos of the agent session. You can use to save summary of the analysis.",
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
		{
			Name:        "base.memo.put",
			Description: "Put a memo to the agent session. You can use to save summary of the analysis.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"note": {
						Type:        genai.TypeString,
						Description: "The memo to put",
					},
				},
			},
		},
		{
			Name:        "base.policy.list",
			Description: "Get the list of policy files for the alert detection in Rego",
		},
		{
			Name:        "base.policy.get",
			Description: "Get the policy of the alert detection in Rego",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"name": {
						Type:        genai.TypeString,
						Description: "The name of the policy file",
					},
				},
				Required: []string{"name"},
			},
		},
	}
}

func (x *Base) Execute(ctx context.Context, name string, args map[string]any) (*action.Result, error) {
	switch name {
	case "base.alerts":
		return x.getAlerts(ctx, args)
	case "base.note.get":
		return x.getNotes(ctx, args)
	case "base.note.put":
		return x.putNote(ctx, args)
	case "base.policy.list":
		return x.listPolicies(ctx, args)
	case "base.policy.get":
		return x.getPolicy(ctx, args)
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
		Name: "base.alerts",
		Data: map[string]any{
			"alerts": rows,
		},
	}, nil
}

func (x *Base) putNote(ctx context.Context, args map[string]any) (*action.Result, error) {
	noteRaw, ok := args["note"]
	if !ok {
		return nil, goerr.New("note is required")
	}

	note, ok := noteRaw.(string)
	if !ok {
		return nil, goerr.New("note is not a string")
	}

	data := session.NewNote(x.sessionID, note)

	if err := x.repo.PutNote(ctx, data); err != nil {
		return nil, goerr.Wrap(err, "failed to put note")
	}

	return &action.Result{
		Name: "base.note.put",
		Data: map[string]any{
			"note": note,
		},
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
		Name: "base.note.get",
		Data: map[string]any{
			"notes":  rows,
			"count":  len(notes),
			"offset": offset,
			"limit":  limit,
		},
	}, nil
}

func (x *Base) listPolicies(_ context.Context, _ map[string]any) (*action.Result, error) {
	var rows []string
	for name := range x.policies {
		rows = append(rows, name)
	}
	return &action.Result{
		Name: "base.policy.list",
		Data: map[string]any{
			"policies": rows,
		},
	}, nil
}

func (x *Base) getPolicy(_ context.Context, args map[string]any) (*action.Result, error) {
	errResp := func(msg string) (*action.Result, error) {
		return &action.Result{
			Name: "base.policy.get",
			Data: map[string]any{
				"policy": "",
			},
		}, goerr.New(msg)
	}

	argName, ok := args["name"]
	if !ok {
		return errResp("Policy name is required")
	}

	name, ok := argName.(string)
	if !ok {
		return errResp("Policy name is not a string")
	}

	policy, ok := x.policies[name]
	if !ok {
		return errResp("Policy not found")
	}

	return &action.Result{
		Name: "base.policy.get",
		Data: map[string]any{
			"policy": policy,
		},
	}, nil
}
