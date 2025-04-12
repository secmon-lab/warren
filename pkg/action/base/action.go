package base

import (
	"context"
	"log/slog"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/action"
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
			Name:        "base.alert.search",
			Description: "Search the alerts by the given query. You can specify the path as Firestore path, and the operation and value to filter the alerts.",
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"path": {
						Type:        genai.TypeString,
						Description: "The path of the alert",
					},
					"op": {
						Type:        genai.TypeString,
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
						Type:        genai.TypeString,
						Description: "The value of the alert",
					},
					"limit": {
						Type:        genai.TypeInteger,
						Description: "Maximum number of alerts to return. Default is 25.",
					},
					"offset": {
						Type:        genai.TypeInteger,
						Description: "Number of alerts to skip. Default is 0.",
					},
				},
				Required: []string{"path", "op", "value"},
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
						Description: "The conclusion of the alert. Intended means the alert is intended, Unaffected means the alert is actual attack or vulnerability, but the system is not affected, FalsePositive is the alert is a false positive. If you conclude as TruePositive, you should not resolve the alert.",
						Enum: []string{
							types.AlertConclusionIntended.String(),
							types.AlertConclusionUnaffected.String(),
							types.AlertConclusionFalsePositive.String(),
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
		"base.alert.search":  x.searchAlerts,
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
