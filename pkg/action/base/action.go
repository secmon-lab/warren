package base

import (
	"context"
	"encoding/json"
	"log/slog"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
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

func (x *Base) Tools() []*genai.FunctionDeclaration {
	return []*genai.FunctionDeclaration{
		{
			Name:        "base.alerts.get",
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
			Name:        "base.alert.resolve",
			Description: "Resolve the alert",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"alert_id": {
						Type:        genai.TypeString,
						Description: "The ID of the alert to resolve",
					},
					"conclusion": {
						Type:        genai.TypeString,
						Description: "The conclusion of the alert. Intended means the alert is intended, Unaffected means the alert is actual attack or vulnerability, but the system is not affected, FalsePositive is the alert is a false positive, TruePositive is the alert is a true positive.",
						Enum: []string{
							types.AlertConclusionIntended.String(),
							types.AlertConclusionUnaffected.String(),
							types.AlertConclusionFalsePositive.String(),
							types.AlertConclusionTruePositive.String(),
						},
					},
					"reason": {
						Type:        genai.TypeString,
						Description: "The reason of the conclusion",
					},
				},
				Required: []string{"alert_id", "conclusion", "reason"},
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
	actionMap := map[string]func(context.Context, map[string]any) (*action.Result, error){
		"base.alerts.get":    x.getAlerts,
		"base.alert.resolve": x.resolveAlert,
		"base.note.get":      x.getNotes,
		"base.note.put":      x.putNote,
		"base.policy.list":   x.listPolicies,
		"base.policy.get":    x.getPolicy,
	}

	action, ok := actionMap[name]
	if !ok {
		return nil, goerr.New("unknown function", goerr.V("name", name))
	}

	return action(ctx, args)
}

func (x *Base) getAlerts(ctx context.Context, args map[string]any) (*action.Result, error) {
	var limit, offset int64

	if limitVal, ok := args["limit"].(float64); ok {
		limit = int64(limitVal)
	}
	if offsetVal, ok := args["offset"].(float64); ok {
		offset = int64(offsetVal)
	}

	alerts, err := x.repo.BatchGetAlerts(ctx, x.alertIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alerts")
	}

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

func (x *Base) resolveAlert(ctx context.Context, args map[string]any) (*action.Result, error) {
	strAlertID, err := getArg[string](args, "alert_id")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert_id")
	}
	alertID := types.AlertID(strAlertID)

	strConclusion, err := getArg[string](args, "conclusion")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get conclusion")
	}
	conclusion := types.AlertConclusion(strConclusion)

	reason, err := getArg[string](args, "reason")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get reason")
	}

	if err := conclusion.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid conclusion", goerr.V("conclusion", strConclusion))
	}

	alert, err := x.repo.GetAlert(ctx, types.AlertID(alertID))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alert")
	}
	if alert == nil {
		return nil, goerr.New("alert not found", goerr.V("alert_id", alertID))
	}

	alert.Conclusion = types.AlertConclusion(conclusion)
	alert.Reason = reason

	if err := x.repo.PutAlert(ctx, *alert); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert")
	}

	return nil, nil
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
