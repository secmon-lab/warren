package memory

import (
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// ExecutionMemory represents learning from task execution
type ExecutionMemory struct {
	ID        types.MemoryID    `json:"id"`
	SchemaID  types.AlertSchema `json:"schema_id"`
	Keep      string            `json:"keep"`
	Change    string            `json:"change"`
	Notes     string            `json:"notes"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Version   int               `json:"version"`
}

// NewExecutionMemory creates a new ExecutionMemory with a unique ID
func NewExecutionMemory(schemaID types.AlertSchema) *ExecutionMemory {
	now := time.Now()
	return &ExecutionMemory{
		ID:        types.NewMemoryID(),
		SchemaID:  schemaID,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
}

// Validate validates the ExecutionMemory
func (m *ExecutionMemory) Validate() error {
	if m.SchemaID == "" {
		return goerr.New("schema_id is required")
	}

	// At least one of Keep, Change, or Notes must be present
	if m.Keep == "" && m.Change == "" && m.Notes == "" {
		return goerr.New("at least one of keep, change, or notes is required")
	}

	if m.Version < 0 {
		return goerr.New("version must be non-negative")
	}

	return nil
}

// IsEmpty returns true if the memory has no content
func (m *ExecutionMemory) IsEmpty() bool {
	return m.Keep == "" && m.Change == "" && m.Notes == ""
}

// String returns a Markdown-formatted representation of the memory
func (m *ExecutionMemory) String() string {
	var parts []string

	if m.Keep != "" {
		parts = append(parts, fmt.Sprintf("*Keep (Successful Patterns):*\n%s", m.Keep))
	}

	if m.Change != "" {
		parts = append(parts, fmt.Sprintf("*Change (Areas for Improvement):*\n%s", m.Change))
	}

	if m.Notes != "" {
		parts = append(parts, fmt.Sprintf("*Notes (Other Insights):*\n%s", m.Notes))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n\n")
}
