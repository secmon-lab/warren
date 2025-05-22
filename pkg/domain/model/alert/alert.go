package alert

import (
	"context"
	_ "embed"
	"encoding/json"
	"math"
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
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

const (
	EmbeddingSize = 256
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
	logger := logging.From(ctx)
	logger.Info("fill metadata", "alert", x.Data)
	prompt, err := prompt.Generate(ctx, alertMetaPrompt, map[string]any{
		"alert":  x.Data,
		"schema": prompt.ToSchema(Metadata{}),
		"lang":   lang.From(ctx).Name(),
	})
	if err != nil {
		return err
	}

	resp, err := llm.Ask(ctx, llmClient, prompt, llm.WithValidate(func(v Metadata) error {
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

	if x.Metadata.Title == "" {
		x.Metadata.Title = resp.Title
	}

	if x.Metadata.Description == "" {
		x.Metadata.Description = resp.Description
	}

	for _, resAttr := range resp.Attributes {
		found := false
		for _, aAttr := range x.Metadata.Attributes {
			if aAttr.Value == resAttr.Value {
				found = true
				break
			}
		}
		if !found {
			resAttr.Auto = true
			x.Metadata.Attributes = append(x.Metadata.Attributes, resAttr)
		}
	}

	rawData, err := json.Marshal(x.Data)
	if err != nil {
		return goerr.Wrap(err, "failed to marshal alert data")
	}
	embedding, err := llmClient.GenerateEmbedding(ctx, EmbeddingSize, []string{string(rawData)})
	if err != nil {
		return err
	}
	if len(embedding) != 1 {
		return goerr.New("failed to generate embedding", goerr.V("embedding.length", len(embedding)))
	}

	x.Embedding = make([]float32, len(embedding[0]))
	for idx, emb := range embedding[0] {
		x.Embedding[idx] = float32(emb)
	}

	return nil
}
