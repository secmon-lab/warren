package ticket

import (
	"context"
	_ "embed"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/embedding"
)

type Ticket struct {
	ID             types.TicketID  `json:"id"`
	AlertIDs       []types.AlertID `json:"-"`
	SlackThread    *slack.Thread   `json:"slack_thread"`
	SlackMessageID string          `json:"slack_message_id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`

	Metadata

	Status     types.TicketStatus    `json:"status"`
	Conclusion types.AlertConclusion `json:"conclusion"`
	Reason     string                `json:"reason"`

	Finding  *Finding    `json:"finding"`
	Assignee *slack.User `json:"assignee"`

	IsTest bool `json:"is_test"`

	Embedding firestore.Vector32 `json:"-"`
}

func (x *Ticket) Validate() error {
	if err := x.ID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid ticket ID")
	}
	if err := x.Status.Validate(); err != nil {
		return goerr.Wrap(err, "invalid status")
	}
	if err := x.Metadata.Validate(); err != nil {
		return goerr.Wrap(err, "invalid metadata")
	}
	if err := x.Finding.Validate(); err != nil {
		return goerr.Wrap(err, "invalid finding")
	}
	return nil
}

type Metadata struct {
	// Title is the title of the ticket for human readability.
	Title string `json:"title"`
	// Description is the description of the ticket for human readability.
	Description string `json:"description"`
	// Summary is the summary of the ticket for AI analysis.
	Summary string `json:"summary"`
	// TitleSource indicates the source of the title (human, ai, inherited)
	TitleSource types.Source `json:"title_source"`
	// DescriptionSource indicates the source of the description (human, ai, inherited)
	DescriptionSource types.Source `json:"description_source"`
}

func (x *Metadata) Validate() error {
	if x.Title == "" {
		return goerr.New("title is required")
	}
	// Description is optional - it may be empty for manually created tickets
	// Summary is optional - it may be empty for manually created tickets

	// Set default sources if not specified
	if x.TitleSource == "" {
		x.TitleSource = types.SourceAI
	}
	if x.DescriptionSource == "" {
		x.DescriptionSource = types.SourceAI
	}

	// Validate source fields
	if err := x.TitleSource.Validate(); err != nil {
		return goerr.Wrap(err, "invalid title source")
	}
	if err := x.DescriptionSource.Validate(); err != nil {
		return goerr.Wrap(err, "invalid description source")
	}

	return nil
}

func New(ctx context.Context, alertIDs []types.AlertID, slackThread *slack.Thread) Ticket {
	return Ticket{
		ID:          types.NewTicketID(),
		AlertIDs:    alertIDs,
		SlackThread: slackThread,
		Status:      types.TicketStatusOpen,
		CreatedAt:   clock.Now(ctx),
		UpdatedAt:   clock.Now(ctx),
		Metadata: Metadata{
			TitleSource:       types.SourceAI, // Default to AI source
			DescriptionSource: types.SourceAI, // Default to AI source
		},
	}
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

type AlertRepository interface {
	BatchGetAlerts(ctx context.Context, alertIDs []types.AlertID) (alert.Alerts, error)
}

//go:embed prompt/ticket_meta.md
var ticketMetaPrompt string

func (x *Ticket) FillMetadata(ctx context.Context, llmClient gollem.LLMClient, alertRepo AlertRepository) error {
	// Only generate metadata for fields that have SourceAI
	needsTitle := x.Metadata.TitleSource == types.SourceAI
	needsDescription := x.Metadata.DescriptionSource == types.SourceAI

	// If no AI generation is needed, return early
	if !needsTitle && !needsDescription {
		return nil
	}

	alerts, err := alertRepo.BatchGetAlerts(ctx, x.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	summaryPrompt, err := prompt.Generate(ctx, ticketMetaPrompt, map[string]any{})
	if err != nil {
		return goerr.Wrap(err, "failed to generate summary prompt")
	}

	summary, err := llm.Summary(ctx, llmClient, summaryPrompt, alerts)
	if err != nil {
		return goerr.Wrap(err, "failed to generate summary")
	}

	metaPrompt, err := prompt.Generate(ctx, ticketMetaPrompt, map[string]any{
		"summary": summary,
		"schema":  prompt.ToSchema(Metadata{}),
		"lang":    lang.From(ctx),
	})
	if err != nil {
		return goerr.Wrap(err, "failed to generate meta prompt")
	}

	meta, err := llm.Ask(ctx, llmClient, metaPrompt, llm.WithValidate(func(meta Metadata) error {
		// Validate that generated content has required fields
		if meta.Title == "" {
			return goerr.New("title is required")
		}
		if meta.Description == "" {
			return goerr.New("description is required")
		}
		// Set sources for validation
		if meta.TitleSource == "" {
			meta.TitleSource = types.SourceAI
		}
		if meta.DescriptionSource == "" {
			meta.DescriptionSource = types.SourceAI
		}
		// Validate source fields only
		if err := meta.TitleSource.Validate(); err != nil {
			return goerr.Wrap(err, "invalid title source")
		}
		if err := meta.DescriptionSource.Validate(); err != nil {
			return goerr.Wrap(err, "invalid description source")
		}
		return nil
	}))
	if err != nil {
		return goerr.Wrap(err, "failed to generate meta")
	}
	if meta == nil {
		return goerr.New("failed to generate meta")
	}

	// Only update fields that were marked for AI generation
	if needsTitle {
		x.Metadata.Title = meta.Title
	}
	if needsDescription {
		x.Metadata.Description = meta.Description
	}
	// Keep the existing Summary as it's always generated
	x.Metadata.Summary = meta.Summary

	return nil
}

// CalculateEmbedding calculates the ticket embedding using a weighted average approach.
// It combines embeddings from title/description (weight 0.3) and alerts (weight 0.7).
func (x *Ticket) CalculateEmbedding(ctx context.Context, llmClient gollem.LLMClient, alertRepo AlertRepository) error {
	var metadataEmbedding firestore.Vector32
	var alertEmbedding firestore.Vector32

	// Calculate embedding from title and description if title exists
	// Description can be empty - this is explicitly allowed
	if x.Metadata.Title != "" {
		embeddingText := x.Metadata.Title
		if x.Metadata.Description != "" {
			embeddingText += " " + x.Metadata.Description
		}
		vector32, err := embedding.Generate(ctx, llmClient, embeddingText)
		if err != nil {
			return goerr.Wrap(err, "failed to generate embedding from metadata")
		}
		metadataEmbedding = vector32
	}

	// Calculate average embedding from alerts if they exist
	if len(x.AlertIDs) > 0 {
		alerts, err := alertRepo.BatchGetAlerts(ctx, x.AlertIDs)
		if err != nil {
			return goerr.Wrap(err, "failed to get alerts")
		}

		embeddings := make([]firestore.Vector32, 0, len(alerts))
		for _, alert := range alerts {
			if len(alert.Embedding) > 0 {
				embeddings = append(embeddings, alert.Embedding)
			}
		}

		if len(embeddings) > 0 {
			alertEmbedding = embedding.Average(embeddings)
		}
	}

	// Calculate weighted average if both embeddings exist
	if len(metadataEmbedding) > 0 && len(alertEmbedding) > 0 {
		weightedEmbedding, err := embedding.WeightedAverage(
			[]firestore.Vector32{metadataEmbedding, alertEmbedding},
			[]float32{0.3, 0.7},
		)
		if err != nil {
			return goerr.Wrap(err, "failed to calculate weighted average embedding")
		}
		x.Embedding = weightedEmbedding
	} else if len(metadataEmbedding) > 0 {
		// Only metadata embedding available
		x.Embedding = metadataEmbedding
	} else if len(alertEmbedding) > 0 {
		// Only alert embedding available
		x.Embedding = alertEmbedding
	}
	// If no embeddings are available, x.Embedding remains empty

	return nil
}
