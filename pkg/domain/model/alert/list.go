package alert

import (
	"context"
	_ "embed"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/embedding"
)

type ListStatus string

const (
	ListStatusUnbound ListStatus = "unbound"
	ListStatusBound   ListStatus = "bound"
)

func (s ListStatus) String() string {
	return string(s)
}

func (s ListStatus) Icon() string {
	switch s {
	case ListStatusBound:
		return "🔗"
	case ListStatusUnbound:
		return "📋"
	default:
		return "❓"
	}
}

func (s ListStatus) DisplayName() string {
	switch s {
	case ListStatusBound:
		return "Bound to ticket"
	case ListStatusUnbound:
		return "Unbound"
	default:
		return "Unknown"
	}
}

type List struct {
	ID             types.AlertListID `json:"id"`
	AlertIDs       []types.AlertID   `json:"alert_ids"`
	SlackThread    *slack.Thread     `json:"slack_thread"`
	SlackMessageID string            `json:"slack_message_id"`
	Status         ListStatus        `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	CreatedBy      *slack.User       `json:"created_by"`

	Metadata
	Embedding firestore.Vector32 `json:"-"`

	alertsMutex sync.RWMutex
	alerts      Alerts `json:"-"`
}

func NewList(ctx context.Context, thread slack.Thread, createdBy *slack.User, alerts Alerts) *List {
	list := List{
		ID:          types.NewAlertListID(),
		SlackThread: &thread,
		Status:      ListStatusUnbound,
		CreatedAt:   clock.Now(ctx),
		CreatedBy:   createdBy,
	}
	for _, alert := range alerts {
		list.AlertIDs = append(list.AlertIDs, alert.ID)
		list.alerts = append(list.alerts, alert)
	}

	embeddings := make([]firestore.Vector32, len(alerts))
	for i, alert := range alerts {
		embeddings[i] = alert.Embedding
	}
	list.Embedding = embedding.Average(embeddings)

	return &list
}

//go:embed prompt/list_summary.md
var listSummaryPrompt string

//go:embed prompt/list_meta.md
var listMetaPrompt string

type AlertListRepository interface {
	BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (Alerts, error)
}

func matchIDs(ids []types.AlertID, alerts Alerts) bool {
	if len(ids) != len(alerts) {
		return false
	}

	idSet := make(map[types.AlertID]struct{})
	for _, id := range ids {
		idSet[id] = struct{}{}
	}

	for _, alert := range alerts {
		if _, ok := idSet[alert.ID]; !ok {
			return false
		}
		delete(idSet, alert.ID)
	}

	return len(idSet) == 0
}

func (x *List) Alerts() (Alerts, error) {
	x.alertsMutex.RLock()
	defer x.alertsMutex.RUnlock()

	if matchIDs(x.AlertIDs, x.alerts) {
		return x.alerts, nil
	}

	return nil, goerr.New("alerts are not matched, need to call GetAlerts first")
}

func (x *List) GetAlerts(ctx context.Context, repo AlertListRepository) (Alerts, error) {
	x.alertsMutex.Lock()
	defer x.alertsMutex.Unlock()

	if matchIDs(x.AlertIDs, x.alerts) {
		return x.alerts, nil
	}

	alerts, err := repo.BatchGetAlerts(ctx, x.AlertIDs)
	if err != nil {
		return nil, err
	}

	x.alerts = alerts
	x.AlertIDs = make([]types.AlertID, len(alerts))
	for i, alert := range alerts {
		x.AlertIDs[i] = alert.ID
	}

	return alerts, nil
}

func (x *List) FillMetadata(ctx context.Context, llmClient gollem.LLMClient) error {
	if len(x.alerts) != len(x.AlertIDs) {
		return goerr.New("alert IDs and alerts are not matched",
			goerr.V("alert_ids", len(x.AlertIDs)),
			goerr.V("alerts", len(x.alerts)),
		)
	}

	if len(x.alerts) == 0 {
		x.Metadata = Metadata{
			Title:       "(no alerts)",
			Description: "",
		}
		return nil
	}

	embeddings := make([]firestore.Vector32, len(x.alerts))
	for i, alert := range x.alerts {
		embeddings[i] = alert.Embedding
	}
	x.Embedding = embedding.Average(embeddings)

	summary, err := llm.Summary(ctx, llmClient, listSummaryPrompt, x.alerts)
	if err != nil {
		return err
	}

	p, err := prompt.Generate(ctx, listMetaPrompt, map[string]any{
		"summary": summary,
		"schema":  prompt.ToSchema(Metadata{}),
		"lang":    lang.From(ctx).Name(),
	})
	if err != nil {
		return err
	}

	resp, err := llm.Ask(ctx, llmClient, p, llm.WithValidate(func(v Metadata) error {
		if v.Title == "" {
			return goerr.New("title is required")
		}
		if v.Description == "" {
			return goerr.New("description is required")
		}
		return nil
	}))
	if err != nil {
		return err
	}

	x.Metadata = *resp
	// Mark metadata as AI-generated since it was filled by LLM
	x.TitleSource = types.SourceAI
	x.DescriptionSource = types.SourceAI

	return nil
}
