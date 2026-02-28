package memory

import (
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// AgentMemory represents a single claim-based memory record for agent learning
// This model stores individual insights (claims) learned during agent execution,
// with quality scoring and usage tracking for adaptive memory management
type AgentMemory struct {
	// ID is a unique identifier for this memory record
	ID types.AgentMemoryID

	// AgentID identifies which agent this memory belongs to (e.g., "bigquery")
	AgentID string

	// Query is the task query that led to this insight
	// Used for semantic search to find relevant memories
	Query string

	// QueryEmbedding is the vector embedding of Query for semantic search
	// Generated automatically by Memory Service when saving
	// Must use firestore.Vector32 type for Firestore vector search to work
	QueryEmbedding firestore.Vector32 `json:"-"`

	// Claim is a specific, self-contained insight learned from execution
	// MUST be understandable without knowing the original query or task context
	// Examples:
	// - "BigQuery table 'project.dataset.events' has field 'user_id' as INT64 type, not STRING - attempting to filter with email addresses like 'user@example.com' will cause type mismatch errors, use numeric IDs only"
	// - "Slack search requires both 'from:@username' AND 'in:#channel' syntax for filtering by user in specific channel - using only 'from:' searches all channels and may return irrelevant results"
	Claim string

	// Score represents the usefulness of this memory (-10.0 to +10.0)
	// - Positive: Helpful memory (higher is better)
	// - 0.0: Neutral or newly created (default)
	// - Negative: Harmful/misleading memory (lower is worse)
	// Updated using EMA (Exponential Moving Average) based on reflection feedback
	Score float64

	// CreatedAt records when this memory was first created
	CreatedAt time.Time

	// LastUsedAt records when this memory was last retrieved/used
	// Used for recency calculation and pruning decisions
	// Zero value indicates never used since creation
	LastUsedAt time.Time
}

// Validate checks if the AgentMemory is valid
func (m *AgentMemory) Validate() error {
	if m.ID == "" {
		return goerr.New("agent memory ID is required")
	}
	if m.AgentID == "" {
		return goerr.New("agent ID is required")
	}
	if m.Query == "" {
		return goerr.New("query is required")
	}
	if m.Claim == "" {
		return goerr.New("claim is required")
	}
	if m.Score < -10.0 || m.Score > 10.0 {
		return goerr.Wrap(goerr.New("score must be between -10.0 and +10.0"), "invalid score", goerr.V("score", m.Score))
	}
	if m.CreatedAt.IsZero() {
		return goerr.New("created_at is required")
	}
	return nil
}
