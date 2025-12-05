package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Knowledge represents a user-instructed memory with versioning
type Knowledge struct {
	// Slug uniquely identifies the knowledge within a topic
	Slug types.KnowledgeSlug `json:"slug"`

	// Name is a human-readable short name (max 100 chars)
	Name string `json:"name"`

	// Topic is the namespace for this knowledge
	Topic types.KnowledgeTopic `json:"topic"`

	// Content is the actual knowledge text
	Content string `json:"content"`

	// CommitID is the SHA256 hash version identifier
	CommitID string `json:"commit_id"`

	// Author is who created/updated this knowledge
	Author types.UserID `json:"author"`

	// CreatedAt is the timestamp when this slug was first created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when this version was created
	UpdatedAt time.Time `json:"updated_at"`

	// State indicates if this knowledge is active or archived
	State types.KnowledgeState `json:"state"`
}

// SlugInfo contains slug and name pair for listing
type SlugInfo struct {
	Slug types.KnowledgeSlug `json:"slug"`
	Name string              `json:"name"`
}

// GenerateCommitID generates SHA256 hash from updatedAt + author + content
func GenerateCommitID(updatedAt time.Time, author types.UserID, content string) string {
	data := fmt.Sprintf("%d%s%s", updatedAt.UnixNano(), author.String(), content)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Validate checks if the Knowledge is valid
func (k *Knowledge) Validate() error {
	if err := k.Slug.Validate(); err != nil {
		return goerr.Wrap(err, "invalid slug")
	}
	if k.Name == "" {
		return goerr.New("name is required")
	}
	if len(k.Name) > 100 {
		return goerr.New("name must be 100 characters or less")
	}
	if err := k.Topic.Validate(); err != nil {
		return goerr.Wrap(err, "invalid topic")
	}
	if k.Content == "" {
		return goerr.New("content is required")
	}
	if k.CommitID == "" {
		return goerr.New("commit_id is required")
	}
	if err := k.Author.Validate(); err != nil {
		return goerr.Wrap(err, "invalid author")
	}
	if err := k.State.Validate(); err != nil {
		return goerr.Wrap(err, "invalid state")
	}
	return nil
}

// Size returns the content size in bytes (for quota calculation)
func (k *Knowledge) Size() int {
	return len(k.Content)
}
