package knowledge

import (
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Knowledge represents a topic-based knowledge entry in the Living Fact Store.
// Each Knowledge holds a collection of facts (or techniques) about a single topic,
// structured as Markdown in the Claim field.
type Knowledge struct {
	ID        types.KnowledgeID       `json:"id" firestore:"id"`
	Category  types.KnowledgeCategory `json:"category" firestore:"category"`
	Title     string                  `json:"title" firestore:"title"`
	Claim     string                  `json:"claim" firestore:"claim"`
	Embedding firestore.Vector32      `json:"-" firestore:"embedding"`
	Tags      []types.KnowledgeTagID  `json:"tags" firestore:"tags"`

	Author types.UserID `json:"author" firestore:"author"`

	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" firestore:"updated_at"`
}

// Validate checks if the Knowledge is valid
func (k *Knowledge) Validate() error {
	if err := k.ID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid knowledge ID")
	}
	if err := k.Category.Validate(); err != nil {
		return goerr.Wrap(err, "invalid category")
	}
	if k.Title == "" {
		return goerr.New("title is required")
	}
	if k.Claim == "" {
		return goerr.New("claim is required")
	}
	if err := k.Author.Validate(); err != nil {
		return goerr.Wrap(err, "invalid author")
	}
	if k.CreatedAt.IsZero() {
		return goerr.New("created_at is required")
	}
	if k.UpdatedAt.IsZero() {
		return goerr.New("updated_at is required")
	}
	for i, tagID := range k.Tags {
		if err := tagID.Validate(); err != nil {
			return goerr.Wrap(err, "invalid tag ID", goerr.V("index", i))
		}
	}
	return nil
}

// KnowledgeLog records a change to a Knowledge entry.
// Logs persist even after the Knowledge is physically deleted (orphan logs for audit trail).
type KnowledgeLog struct {
	ID          types.KnowledgeLogID `json:"id" firestore:"id"`
	KnowledgeID types.KnowledgeID    `json:"knowledge_id" firestore:"knowledge_id"`

	// Snapshot after change
	Title     string             `json:"title" firestore:"title"`
	Claim     string             `json:"claim" firestore:"claim"`
	Embedding firestore.Vector32 `json:"-" firestore:"embedding"`

	// Change metadata
	Author   types.UserID   `json:"author" firestore:"author"`
	TicketID types.TicketID `json:"ticket_id,omitempty" firestore:"ticket_id"`
	Message  string         `json:"message" firestore:"message"`

	CreatedAt time.Time `json:"created_at" firestore:"created_at"`
}

// Validate checks if the KnowledgeLog is valid
func (l *KnowledgeLog) Validate() error {
	if err := l.ID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid knowledge log ID")
	}
	if err := l.KnowledgeID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid knowledge ID")
	}
	if l.Title == "" {
		return goerr.New("title is required")
	}
	if l.Claim == "" {
		return goerr.New("claim is required")
	}
	if err := l.Author.Validate(); err != nil {
		return goerr.Wrap(err, "invalid author")
	}
	if l.Message == "" {
		return goerr.New("message is required")
	}
	if l.CreatedAt.IsZero() {
		return goerr.New("created_at is required")
	}
	return nil
}

// KnowledgeTag represents a managed tag for knowledge classification.
// Tags are stored in a separate Firestore collection (knowledge_tags).
type KnowledgeTag struct {
	ID          types.KnowledgeTagID `json:"id" firestore:"id"`
	Name        string               `json:"name" firestore:"name"`
	Description string               `json:"description" firestore:"description"`
	CreatedAt   time.Time            `json:"created_at" firestore:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at" firestore:"updated_at"`
}

// Validate checks if the KnowledgeTag is valid
func (t *KnowledgeTag) Validate() error {
	if err := t.ID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid knowledge tag ID")
	}
	if t.Name == "" {
		return goerr.New("name is required")
	}
	if t.CreatedAt.IsZero() {
		return goerr.New("created_at is required")
	}
	if t.UpdatedAt.IsZero() {
		return goerr.New("updated_at is required")
	}
	return nil
}
