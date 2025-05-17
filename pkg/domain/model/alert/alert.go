package alert

import (
	"context"
	_ "embed"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// Alert represents an event of a potential security incident. This model is designed to be immutable. An Alert can be linked to at most one ticket.
type Alert struct {
	ID       types.AlertID     `json:"id"`
	TicketID types.TicketID    `json:"ticket_id"`
	Schema   types.AlertSchema `json:"schema"`
	Data     any               `json:"data"`

	Metadata

	CreatedAt   time.Time          `json:"created_at"`
	SlackThread *slack.Thread      `json:"slack_thread"`
	Embedding   firestore.Vector32 `json:"-"`
}

type Alerts []*Alert

type QueryOutput struct {
	Alert []Metadata `json:"alert"`
}

type Metadata struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Attributes  []Attribute `json:"attributes"`
}

func New(ctx context.Context, schema types.AlertSchema, data any, metadata Metadata) Alert {
	return Alert{
		ID:        types.NewAlertID(),
		Schema:    schema,
		CreatedAt: clock.Now(ctx),
		Metadata:  metadata,
		Data:      data,
	}
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Link  string `json:"link"`
	Auto  bool   `json:"auto"`
}

func (x *Alert) CosineSimilarity(other []float32) float64 {
	if len(x.Embedding) == 0 || len(other) == 0 {
		return 0
	}

	var dotProduct float64
	var magnitudeA, magnitudeB float64

	for i := range x.Embedding {
		dotProduct += float64(x.Embedding[i]) * float64(other[i])
		magnitudeA += float64(x.Embedding[i]) * float64(x.Embedding[i])
		magnitudeB += float64(other[i]) * float64(other[i])
	}

	return dotProduct / (math.Sqrt(magnitudeA) * math.Sqrt(magnitudeB))
}

//go:embed prompt/alert_meta.md
var alertMetaPrompt string

func (x *Alert) FillMetadata(ctx context.Context, llmClient gollem.LLMClient) error {
	prompt, err := prompt.Generate(ctx, alertMetaPrompt, map[string]any{
		"alert":  x.Data,
		"schema": prompt.ToSchema(Metadata{}),
		"lang":   lang.From(ctx).Name(),
	})
	if err != nil {
		return err
	}

	resp, err := llm.Ask[Metadata](ctx, llmClient, prompt)
	if err != nil {
		return err
	}

	x.Metadata = *resp

	return nil
}
