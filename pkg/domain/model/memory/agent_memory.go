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

	// SuccessDescription is a natural language description of what worked well (K: Keep)
	// Example: "Successfully executed 3 tool calls to retrieve login error data"
	SuccessDescription string `json:"success_description,omitempty"`

	// SuccessResult contains metadata about successful execution (NOT the actual result data)
	// Example: {"tool_call_count": 3, "tools_used": ["bigquery_query", "bigquery_result"]}
	SuccessResult map[string]any `json:"success_result,omitempty"`

	// Problems is a list of issues encountered during execution (P: Problem)
	// Example: ["Query exceeded scan size limit", "Table schema mismatch"]
	Problems []string `json:"problems,omitempty"`

	// Improvements is a list of suggestions for future executions (T: Try)
	// Example: ["Add WHERE clause to reduce scan size", "Verify table schema before querying"]
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
