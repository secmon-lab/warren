package policy

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// EnrichTask represents a prompt-based enrichment task definition
type EnrichTask struct {
	ID       string                   `json:"id"`
	Template string                   `json:"template,omitempty"` // Template file path
	Params   map[string]any           `json:"params,omitempty"`   // Template parameters
	Inline   string                   `json:"inline,omitempty"`   // Inline prompt
	Format   types.GenAIContentFormat `json:"format"`             // "text" or "json"
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
	// Exactly one of Template or Inline must be specified
	if (t.Template == "" && t.Inline == "") || (t.Template != "" && t.Inline != "") {
		return goerr.New("exactly one of template or inline must be specified")
	}

	return nil
}

// HasTemplateFile returns true if task uses a template file
func (t *EnrichTask) HasTemplateFile() bool {
	return t.Template != ""
}

// GetPromptContent returns the prompt content (inline or template path)
func (t *EnrichTask) GetPromptContent() string {
	if t.Inline != "" {
		return t.Inline
	}
	return t.Template
}

// EnrichPolicyResult represents the result of enrich policy evaluation
type EnrichPolicyResult struct {
	Prompts []EnrichTask `json:"prompts"`
}

// TaskCount returns the total number of tasks
func (r *EnrichPolicyResult) TaskCount() int {
	return len(r.Prompts)
}

// EnsureTaskIDs ensures all tasks have IDs
func (r *EnrichPolicyResult) EnsureTaskIDs() {
	for i := range r.Prompts {
		r.Prompts[i].EnsureID()
	}
}

// EnrichResult represents a single enrichment task result with metadata
type EnrichResult struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"` // Actual prompt text used
	Result any    `json:"result"` // Task execution result
}

// EnrichResults represents all enrichment task results as an array
type EnrichResults []EnrichResult
