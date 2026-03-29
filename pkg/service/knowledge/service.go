package knowledge

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	knowledgeModel "github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Service provides knowledge management operations.
type Service struct {
	repo            interfaces.Repository
	embeddingModel  interfaces.EmbeddingClient
	traceRepository trace.Repository
}

// ServiceOption configures a Service.
type ServiceOption func(*Service)

// WithTraceRepository sets the trace repository for introspection tracing.
func WithTraceRepository(repo trace.Repository) ServiceOption {
	return func(s *Service) {
		s.traceRepository = repo
	}
}

// New creates a new knowledge service.
func New(repo interfaces.Repository, embeddingModel interfaces.EmbeddingClient, opts ...ServiceOption) *Service {
	s := &Service{
		repo:           repo,
		embeddingModel: embeddingModel,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// SaveKnowledge creates or updates a knowledge entry, recording a log.
// If k.ID is empty, a new ID is generated (create).
// All tag IDs are validated to exist before saving.
func (s *Service) SaveKnowledge(ctx context.Context, k *knowledgeModel.Knowledge, message string, ticketID types.TicketID) error {
	now := time.Now()

	isNew := k.ID == ""
	if isNew {
		k.ID = types.NewKnowledgeID()
		k.CreatedAt = now
	}
	k.UpdatedAt = now

	// Validate tag existence
	if err := s.validateTags(ctx, k.Tags); err != nil {
		return err
	}

	// Generate embedding from Title + Claim
	embedding, err := s.generateEmbedding(ctx, k.Title+" "+k.Claim)
	if err != nil {
		return goerr.Wrap(err, "failed to generate embedding")
	}
	k.Embedding = embedding

	// Validate
	if err := k.Validate(); err != nil {
		return goerr.Wrap(err, "invalid knowledge")
	}

	// Save knowledge
	if err := s.repo.PutKnowledge(ctx, k); err != nil {
		return goerr.Wrap(err, "failed to save knowledge")
	}

	// Record log
	log := &knowledgeModel.KnowledgeLog{
		ID:          types.NewKnowledgeLogID(),
		KnowledgeID: k.ID,
		Title:       k.Title,
		Claim:       k.Claim,
		Embedding:   k.Embedding,
		Author:      k.Author,
		TicketID:    ticketID,
		Message:     message,
		CreatedAt:   now,
	}
	if err := s.repo.PutKnowledgeLog(ctx, log); err != nil {
		return goerr.Wrap(err, "failed to save knowledge log")
	}

	return nil
}

// DeleteKnowledge records a deletion log then physically deletes the knowledge.
func (s *Service) DeleteKnowledge(ctx context.Context, id types.KnowledgeID, reason string, author types.UserID, ticketID types.TicketID) error {
	k, err := s.repo.GetKnowledge(ctx, id)
	if err != nil {
		return goerr.Wrap(err, "failed to get knowledge for deletion")
	}
	if k == nil {
		return goerr.New("knowledge not found", goerr.V("id", id))
	}

	now := time.Now()

	// Record deletion log (orphan log — persists after knowledge is deleted)
	log := &knowledgeModel.KnowledgeLog{
		ID:          types.NewKnowledgeLogID(),
		KnowledgeID: id,
		Title:       k.Title,
		Claim:       k.Claim,
		Embedding:   k.Embedding,
		Author:      author,
		TicketID:    ticketID,
		Message:     "Deleted: " + reason,
		CreatedAt:   now,
	}
	if err := s.repo.PutKnowledgeLog(ctx, log); err != nil {
		return goerr.Wrap(err, "failed to save deletion log")
	}

	// Physically delete
	if err := s.repo.DeleteKnowledge(ctx, id); err != nil {
		return goerr.Wrap(err, "failed to delete knowledge")
	}

	return nil
}

// SearchKnowledge performs hybrid search (cosine similarity + BM25 + RRF)
// on knowledges filtered by category and tags.
func (s *Service) SearchKnowledge(ctx context.Context, category types.KnowledgeCategory, tagIDs []types.KnowledgeTagID, query string, limit int) ([]*knowledgeModel.Knowledge, error) {
	if len(tagIDs) == 0 {
		return nil, goerr.New("at least one tag is required for knowledge search")
	}

	// Fetch candidates from Firestore (category + tag filter)
	candidates, err := s.repo.ListKnowledgesByCategoryAndTags(ctx, category, tagIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list knowledges")
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// If no query text, return candidates as-is (no ranking needed)
	if query == "" {
		if limit > 0 && len(candidates) > limit {
			candidates = candidates[:limit]
		}
		return candidates, nil
	}

	// Generate query embedding
	queryEmbedding, err := s.generateEmbedding(ctx, query)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate query embedding")
	}

	// Hybrid search + RRF
	results := hybridSearch(candidates, queryEmbedding, query, limit)
	return results, nil
}

// GetKnowledge retrieves a specific knowledge by ID.
func (s *Service) GetKnowledge(ctx context.Context, id types.KnowledgeID) (*knowledgeModel.Knowledge, error) {
	return s.repo.GetKnowledge(ctx, id)
}

// ListKnowledgeLogs retrieves change history for a knowledge.
func (s *Service) ListKnowledgeLogs(ctx context.Context, knowledgeID types.KnowledgeID) ([]*knowledgeModel.KnowledgeLog, error) {
	return s.repo.ListKnowledgeLogs(ctx, knowledgeID)
}

// Tag management

// ListTags returns all knowledge tags.
func (s *Service) ListTags(ctx context.Context) ([]*knowledgeModel.KnowledgeTag, error) {
	return s.repo.ListKnowledgeTags(ctx)
}

// CreateTag creates a new tag. Returns existing tag if name already exists (case-insensitive).
func (s *Service) CreateTag(ctx context.Context, name, description string) (*knowledgeModel.KnowledgeTag, error) {
	normalizedName := strings.ToLower(strings.TrimSpace(name))

	// Check for duplicate name
	existing, err := s.repo.ListKnowledgeTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tags for duplicate check")
	}
	for _, t := range existing {
		if strings.ToLower(t.Name) == normalizedName {
			return t, nil // Return existing tag instead of creating duplicate
		}
	}

	now := time.Now()
	tag := &knowledgeModel.KnowledgeTag{
		ID:          types.NewKnowledgeTagID(),
		Name:        normalizedName,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := tag.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid tag")
	}
	if err := s.repo.PutKnowledgeTag(ctx, tag); err != nil {
		return nil, goerr.Wrap(err, "failed to create tag")
	}
	return tag, nil
}

// UpdateTag updates an existing tag.
func (s *Service) UpdateTag(ctx context.Context, id types.KnowledgeTagID, name, description string) (*knowledgeModel.KnowledgeTag, error) {
	tag, err := s.repo.GetKnowledgeTag(ctx, id)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get tag")
	}
	if tag == nil {
		return nil, goerr.New("tag not found", goerr.V("id", id))
	}

	tag.Name = name
	tag.Description = description
	tag.UpdatedAt = time.Now()

	if err := tag.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid tag")
	}
	if err := s.repo.PutKnowledgeTag(ctx, tag); err != nil {
		return nil, goerr.Wrap(err, "failed to update tag")
	}
	return tag, nil
}

// DeleteTag deletes a tag and removes it from all knowledges that reference it.
func (s *Service) DeleteTag(ctx context.Context, id types.KnowledgeTagID) error {
	tag, err := s.repo.GetKnowledgeTag(ctx, id)
	if err != nil {
		return goerr.Wrap(err, "failed to get tag")
	}
	if tag == nil {
		return goerr.New("tag not found", goerr.V("id", id))
	}

	// Delete the tag
	if err := s.repo.DeleteKnowledgeTag(ctx, id); err != nil {
		return goerr.Wrap(err, "failed to delete tag")
	}

	return nil
}

// MergeTags merges oldID into newID: replaces oldID with newID in all knowledges, then deletes oldID.
func (s *Service) MergeTags(ctx context.Context, oldID, newID types.KnowledgeTagID) error {
	// Validate both tags exist
	oldTag, err := s.repo.GetKnowledgeTag(ctx, oldID)
	if err != nil {
		return goerr.Wrap(err, "failed to get old tag")
	}
	if oldTag == nil {
		return goerr.New("old tag not found", goerr.V("id", oldID))
	}

	newTag, err := s.repo.GetKnowledgeTag(ctx, newID)
	if err != nil {
		return goerr.Wrap(err, "failed to get new tag")
	}
	if newTag == nil {
		return goerr.New("new tag not found", goerr.V("id", newID))
	}

	// Find all knowledges with oldID by searching both categories
	for _, category := range []types.KnowledgeCategory{types.KnowledgeCategoryFact, types.KnowledgeCategoryTechnique} {
		knowledges, err := s.repo.ListKnowledgesByCategoryAndTags(ctx, category, []types.KnowledgeTagID{oldID})
		if err != nil {
			return goerr.Wrap(err, "failed to list knowledges for tag merge")
		}

		for _, k := range knowledges {
			changed := false
			hasNew := false
			newTags := make([]types.KnowledgeTagID, 0, len(k.Tags))
			for _, t := range k.Tags {
				if t == oldID {
					changed = true
					continue // remove old
				}
				if t == newID {
					hasNew = true
				}
				newTags = append(newTags, t)
			}
			if changed {
				if !hasNew {
					newTags = append(newTags, newID)
				}
				k.Tags = newTags
				if err := s.repo.PutKnowledge(ctx, k); err != nil {
					return goerr.Wrap(err, "failed to update knowledge during tag merge", goerr.V("knowledgeID", k.ID))
				}
			}
		}
	}

	// Delete old tag
	if err := s.repo.DeleteKnowledgeTag(ctx, oldID); err != nil {
		return goerr.Wrap(err, "failed to delete old tag after merge")
	}

	return nil
}

// validateTags checks that all tag IDs exist.
func (s *Service) validateTags(ctx context.Context, tagIDs []types.KnowledgeTagID) error {
	for _, id := range tagIDs {
		tag, err := s.repo.GetKnowledgeTag(ctx, id)
		if err != nil {
			return goerr.Wrap(err, "failed to get tag", goerr.V("tagID", id))
		}
		if tag == nil {
			return goerr.New("tag not found", goerr.V("tagID", id))
		}
	}
	return nil
}

const embeddingDimension = 256

// generateEmbedding generates an embedding vector for the given text.
func (s *Service) generateEmbedding(ctx context.Context, text string) (firestore.Vector32, error) {
	if s.embeddingModel == nil {
		return nil, nil
	}
	embeddings, err := s.embeddingModel.Embeddings(ctx, []string{text}, embeddingDimension)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to embed text")
	}
	if len(embeddings) == 0 {
		return nil, nil
	}
	return firestore.Vector32(embeddings[0]), nil
}
