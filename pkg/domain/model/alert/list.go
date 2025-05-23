package alert

import (
	"context"
	_ "embed"
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

type List struct {
	ID          types.AlertListID `json:"id"`
	AlertIDs    []types.AlertID   `json:"alert_ids"`
	SlackThread *slack.Thread     `json:"slack_thread"`
	CreatedAt   time.Time         `json:"created_at"`
	CreatedBy   *slack.User       `json:"created_by"`

	Metadata
	Embedding firestore.Vector32 `json:"-"`

	Alerts Alerts `firestore:"-"`
}

func NewList(ctx context.Context, thread slack.Thread, createdBy *slack.User, alerts Alerts) List {
	list := List{
		ID:          types.NewAlertListID(),
		SlackThread: &thread,
		CreatedAt:   clock.Now(ctx),
		CreatedBy:   createdBy,
	}
	for _, alert := range alerts {
		list.AlertIDs = append(list.AlertIDs, alert.ID)
		list.Alerts = append(list.Alerts, alert)
	}

	return list
}

//go:embed prompt/list_summary.md
var listSummaryPrompt string

//go:embed prompt/list_meta.md
var listMetaPrompt string

func (x *List) FillMetadata(ctx context.Context, llmClient gollem.LLMClient) error {
	if len(x.Alerts) != len(x.AlertIDs) {
		return goerr.New("alert IDs and alerts are not matched",
			goerr.V("alert_ids", len(x.AlertIDs)),
			goerr.V("alerts", len(x.Alerts)),
		)
	}

	if len(x.Alerts) == 0 {
		x.Metadata = Metadata{
			Title:       "(no alerts)",
			Description: "",
		}
		return nil
	}

	embeddings := make([]firestore.Vector32, len(x.Alerts))
	for i, alert := range x.Alerts {
		embeddings[i] = alert.Embedding
	}
	x.Embedding = embedding.Averate(embeddings)

	summary, err := llm.Summary(ctx, llmClient, listSummaryPrompt, x.Alerts)
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

	return nil
}
