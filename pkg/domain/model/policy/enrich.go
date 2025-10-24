package policy

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// TaskType represents the type of enrichment task
type TaskType string

const (
	TaskTypeQuery TaskType = "query"
	TaskTypeAgent TaskType = "agent"
)

// EnrichTask represents a generic enrichment task definition
type EnrichTask struct {
	ID     string                   `json:"id"`
	Prompt string                   `json:"prompt,omitempty"` // File path
	Inline string                   `json:"inline,omitempty"` // Inline prompt
	Format types.GenAIContentFormat `json:"format"`           // "text" or "json"
}

// generateTaskID generates a random task ID in format "task_XXXXXXXX"
func generateTaskID() string {
	b := make([]byte, 4) // 4 bytes = 8 hex characters
	_, _ = rand.Read(b)  // crypto/rand.Read always returns len(b), nil
	return "task_" + hex.EncodeToString(b)
}

// EnsureID ensures the task has an ID, generating one if necessary
func (t *EnrichTask) EnsureID() {
	if t.ID == "" {
		t.ID = generateTaskID()
	}
}

// Validate checks if the task configuration is valid
func (t *EnrichTask) Validate() error {
	// Exactly one of Prompt or Inline must be specified
	if (t.Prompt == "" && t.Inline == "") || (t.Prompt != "" && t.Inline != "") {
		return goerr.New("exactly one of prompt or inline must be specified")
	}

	return nil
}

// HasPromptFile returns true if task uses a prompt file
func (t *EnrichTask) HasPromptFile() bool {
	return t.Prompt != ""
}

// GetPromptContent returns the prompt content (inline or file path)
func (t *EnrichTask) GetPromptContent() string {
	if t.Inline != "" {
		return t.Inline
	}
	return t.Prompt
}

// QueryTask represents a query-type enrichment task
type QueryTask struct {
	EnrichTask
}

// AgentTask represents an agent-type enrichment task
type AgentTask struct {
	EnrichTask
}

// EnrichPolicyResult represents the result of enrich policy evaluation
type EnrichPolicyResult struct {
	Query []QueryTask `json:"query"`
	Agent []AgentTask `json:"agent"`
}

// TaskCount returns the total number of tasks
func (r *EnrichPolicyResult) TaskCount() int {
	return len(r.Query) + len(r.Agent)
}

// EnsureTaskIDs ensures all tasks have IDs
func (r *EnrichPolicyResult) EnsureTaskIDs() {
	for i := range r.Query {
		r.Query[i].EnsureID()
	}
	for i := range r.Agent {
		r.Agent[i].EnsureID()
	}
}

// EnrichResults represents all enrichment task results
type EnrichResults map[string]any // key: task ID, value: result data
