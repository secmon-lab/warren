package alert

import (
	"context"
	_ "embed"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/embedding"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

const (
	DefaultAlertTitle       = "(no title)"
	DefaultAlertDescription = "(no description)"
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
	TagIDs      map[string]bool    `json:"tag_ids"`
}

// HasSlackThread returns true if the alert has a valid Slack thread
func (a *Alert) HasSlackThread() bool {
	return a.SlackThread != nil && a.SlackThread.ThreadID != ""
}

// GetTagNames returns tag names for external API compatibility
func (a *Alert) GetTagNames(ctx context.Context, tagGetter func(context.Context, []string) ([]*tag.Tag, error)) ([]string, error) {
	return tag.ConvertIDsToNames(ctx, a.TagIDs, tagGetter)
}

type Alerts []*Alert

type QueryOutput struct {
	Alert []Metadata `json:"alert"`
}

type Metadata struct {
	Title             string       `json:"title"`
	Description       string       `json:"description"`
	Attributes        []Attribute  `json:"attributes"`
	TitleSource       types.Source `json:"title_source"`
	DescriptionSource types.Source `json:"description_source"`
	// Tags field is used temporarily during policy processing to pass tag names
	// These are converted to TagIDs and not persisted in this field
	Tags []string `json:"tags,omitempty"`
	// GenAI field specifies LLM processing configuration
	GenAI *GenAIConfig `json:"genai,omitempty"`
}

// GenAIConfig configures LLM processing for alerts
type GenAIConfig struct {
	Prompt string                  `json:"prompt"` // Prompt template file name
	Format types.GenAIContentFormat `json:"format"` // Response format: "text" | "json" (default: "text")
}

// GenAIResponse represents the LLM response for display purposes
type GenAIResponse struct {
	Data   any                      `json:"data"`   // Raw response data
	Format types.GenAIContentFormat `json:"format"` // Response format for formatting
}

func New(ctx context.Context, schema types.AlertSchema, data any, metadata Metadata) Alert {
	newAlert := Alert{
		ID:        types.NewAlertID(),
		TicketID:  types.EmptyTicketID,
		Schema:    schema,
		CreatedAt: clock.Now(ctx),
		Metadata:  metadata,
		Data:      data,
		TagIDs:    make(map[string]bool),
	}

	if newAlert.Metadata.Title == "" {
		newAlert.Metadata.Title = DefaultAlertTitle
	}
	if newAlert.Metadata.Description == "" {
		newAlert.Metadata.Description = DefaultAlertDescription
	}

	// Set default sources if not specified
	if newAlert.Metadata.TitleSource == "" {
		newAlert.Metadata.TitleSource = types.SourceHuman // Default alert metadata is human-provided
	}
	if newAlert.Metadata.DescriptionSource == "" {
		newAlert.Metadata.DescriptionSource = types.SourceHuman // Default alert metadata is human-provided
	}

	return newAlert
}

type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Link  string `json:"link"`
	Auto  bool   `json:"auto"`
}

// Finding is the conclusion of the alert. This is set by the AI.
type Finding struct {
	Severity       types.AlertSeverity `json:"severity"`
	Summary        string              `json:"summary"`
	Reason         string              `json:"reason"`
	Recommendation string              `json:"recommendation"`
}

func (x *Finding) Validate() error {
	if err := x.Severity.Validate(); err != nil {
		return goerr.Wrap(err, "invalid severity")
	}
	return nil
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
	if x.Metadata.Title == DefaultAlertTitle || x.Metadata.Title == "" {
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

		if x.Metadata.Title == "" || x.Metadata.Title == DefaultAlertTitle {
			x.Metadata.Title = resp.Title
			x.Metadata.TitleSource = types.SourceAI
		}

		if x.Metadata.Description == "" || x.Metadata.Description == DefaultAlertDescription {
			x.Metadata.Description = resp.Description
			x.Metadata.DescriptionSource = types.SourceAI
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
	}

	embedding, err := embedding.Generate(ctx, llmClient, x.Data)
	if err != nil {
		return err
	}
	x.Embedding = embedding

	return nil
}
