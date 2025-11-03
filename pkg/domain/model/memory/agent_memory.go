package memory

import (
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// AgentMemory represents a memory record for Sub-Agent execution history
// This model follows the KPT (Keep/Problem/Try) format for storing execution experiences
type AgentMemory struct {
	// ID is a unique identifier for this memory record
	ID types.AgentMemoryID `json:"id"`

	// AgentID identifies which agent this memory belongs to (e.g., "bigquery")
	AgentID string `json:"agent_id"`

	// TaskQuery is the original natural language query that triggered this execution
	TaskQuery string `json:"task_query"`

	// QueryEmbedding is the vector embedding of TaskQuery for semantic search
	// Generated automatically by Memory Service when saving
	QueryEmbedding []float32 `json:"query_embedding,omitempty"`

	// Timestamp records when this task was executed
	Timestamp time.Time `json:"timestamp"`

	// Duration records how long the task took to complete
	Duration time.Duration `json:"duration"`

	// Successes is a list of successful patterns observed (K: Keep in KPT format)
	// Contains domain knowledge about what worked: field semantics, data formats, query patterns
	// Example: ["Login failures identified by severity='ERROR' AND action='login'. User ID is user.email (STRING) not user.id (INT64)"]
	Successes []string `json:"successes,omitempty"`

	// Problems is a list of issues encountered during execution (P: Problem in KPT format)
	// Contains domain knowledge mistakes: wrong field assumptions, unexpected data formats
	// Example: ["Expected 'timestamp' field but actual field is 'event_time' (TIMESTAMP type). user_id is INT64 not STRING email"]
	Problems []string `json:"problems,omitempty"`

	// Improvements is a list of suggestions for future executions (T: Try in KPT format)
	// Contains specific domain knowledge to apply: which fields to use, expected formats, search patterns
	// Example: ["For user searches: use user.email (STRING) not user_id (INT64). For errors: check error_code field values"]
	Improvements []string `json:"improvements,omitempty"`
}

// Validate checks if the AgentMemory is valid
func (m *AgentMemory) Validate() error {
	if m.ID == "" {
		return goerr.New("agent memory ID is required")
	}
	if m.AgentID == "" {
		return goerr.New("agent ID is required")
	}
	if m.TaskQuery == "" {
		return goerr.New("task query is required")
	}
	return nil
}
