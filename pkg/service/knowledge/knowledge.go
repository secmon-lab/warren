package knowledge

import (
	"context"
	"fmt"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

const (
	DefaultKnowledgeTopic     = "default"
	DefaultKnowledgeSizeLimit = 10 * 1024 // 10KB
)

type Service struct {
	repo      interfaces.Repository
	sizeLimit int
}

func New(repo interfaces.Repository) *Service {
	return &Service{
		repo:      repo,
		sizeLimit: DefaultKnowledgeSizeLimit,
	}
}

// normalizeKnowledgeTopic converts empty topic to default topic
func normalizeKnowledgeTopic(topic types.KnowledgeTopic) types.KnowledgeTopic {
	if topic == "" {
		return types.KnowledgeTopic(DefaultKnowledgeTopic)
	}
	return topic
}

<<<<<<< HEAD
=======
// normalizeAndValidateTopic normalizes and validates the topic
func normalizeAndValidateTopic(topic types.KnowledgeTopic) (types.KnowledgeTopic, error) {
	normalized := normalizeKnowledgeTopic(topic)
	if err := normalized.Validate(); err != nil {
		return "", goerr.Wrap(err, "invalid topic")
	}
	return normalized, nil
}

>>>>>>> fix/knowledge-empty-topic-handling
// GetKnowledges retrieves all active knowledges for a topic
func (s *Service) GetKnowledges(ctx context.Context, topic types.KnowledgeTopic) ([]*knowledge.Knowledge, error) {
	topic = normalizeKnowledgeTopic(topic)
	return s.repo.GetKnowledges(ctx, topic)
}

// GetKnowledge retrieves a specific knowledge by slug
func (s *Service) GetKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) (*knowledge.Knowledge, error) {
	topic = normalizeKnowledgeTopic(topic)
	return s.repo.GetKnowledge(ctx, topic, slug)
}

// ListSlugs returns all slugs and names for a topic
func (s *Service) ListSlugs(ctx context.Context, topic types.KnowledgeTopic) ([]*knowledge.SlugInfo, error) {
	topic = normalizeKnowledgeTopic(topic)
	return s.repo.ListKnowledgeSlugs(ctx, topic)
}

// SaveKnowledge saves a new version of a knowledge with quota check
func (s *Service) SaveKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug, name, content string, author types.UserID) (string, error) {
<<<<<<< HEAD
	// Normalize topic before validation
	topic = normalizeKnowledgeTopic(topic)

	// Validate input
	if err := topic.Validate(); err != nil {
		return "", goerr.Wrap(err, "invalid topic")
=======
	// Normalize and validate topic
	var err error
	topic, err = normalizeAndValidateTopic(topic)
	if err != nil {
		return "", err
>>>>>>> fix/knowledge-empty-topic-handling
	}
	if err := slug.Validate(); err != nil {
		return "", goerr.Wrap(err, "invalid slug")
	}
	if name == "" {
		return "", goerr.New("name is required")
	}
	if len(name) > 100 {
		return "", goerr.New("name must be 100 characters or less")
	}
	if content == "" {
		return "", goerr.New("content is required")
	}
	if err := author.Validate(); err != nil {
		return "", goerr.Wrap(err, "invalid author")
	}

	// Calculate current size (excluding this slug if it exists)
	currentSize, err := s.repo.CalculateKnowledgeSize(ctx, topic)
	if err != nil {
		return "", goerr.Wrap(err, "failed to calculate current knowledge size")
	}

	// Get existing knowledge to subtract its size if updating
	existing, err := s.repo.GetKnowledge(ctx, topic, slug)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get existing knowledge")
	}
	if existing != nil {
		currentSize -= existing.Size()
	}

	// Check if adding new content would exceed limit
	newSize := currentSize + len(content)
	if newSize > s.sizeLimit {
		exceeded := newSize - s.sizeLimit
		return "", goerr.Wrap(errs.ErrKnowledgeQuotaExceeded, "knowledge quota exceeded",
			goerr.V("current_size", currentSize),
			goerr.V("new_content_size", len(content)),
			goerr.V("total_size", newSize),
			goerr.V("limit", s.sizeLimit),
			goerr.V("exceeded_by", exceeded),
			goerr.V("suggestion", fmt.Sprintf("Please archive existing knowledges or reduce content size by %d bytes", exceeded)))
	}

	// Generate commit ID
	now := time.Now()
	commitID := knowledge.GenerateCommitID(now, author, content)

	// Create Knowledge object
	k := &knowledge.Knowledge{
		Slug:      slug,
		Name:      name,
		Topic:     topic,
		Content:   content,
		CommitID:  commitID,
		Author:    author,
		CreatedAt: now,
		UpdatedAt: now,
		State:     types.KnowledgeStateActive,
	}

	// If existing knowledge, preserve the original creation time
	if existing != nil {
		k.CreatedAt = existing.CreatedAt
	}

	// Save knowledge
	if err := s.repo.PutKnowledge(ctx, k); err != nil {
		return "", goerr.Wrap(err, "failed to save knowledge")
	}

	return commitID, nil
}

// ArchiveKnowledge archives a knowledge
func (s *Service) ArchiveKnowledge(ctx context.Context, topic types.KnowledgeTopic, slug types.KnowledgeSlug) error {
<<<<<<< HEAD
	// Normalize topic before validation
	topic = normalizeKnowledgeTopic(topic)

	if err := topic.Validate(); err != nil {
		return goerr.Wrap(err, "invalid topic")
=======
	// Normalize and validate topic
	var err error
	topic, err = normalizeAndValidateTopic(topic)
	if err != nil {
		return err
>>>>>>> fix/knowledge-empty-topic-handling
	}
	if err := slug.Validate(); err != nil {
		return goerr.Wrap(err, "invalid slug")
	}

	return s.repo.ArchiveKnowledge(ctx, topic, slug)
}
