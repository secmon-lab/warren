package memory

import (
	"fmt"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// TicketMemory represents organizational knowledge from ticket resolution
type TicketMemory struct {
	ID        types.MemoryID    `json:"id" firestore:"id"`
	SchemaID  types.AlertSchema `json:"schema_id" firestore:"schema_id"`
	Insights  string            `json:"insights" firestore:"insights"`
	CreatedAt time.Time         `json:"created_at" firestore:"created_at"`
	UpdatedAt time.Time         `json:"updated_at" firestore:"updated_at"`
	Version   int               `json:"version" firestore:"version"`
}

// NewTicketMemory creates a new TicketMemory with a unique ID
func NewTicketMemory(schemaID types.AlertSchema) *TicketMemory {
	now := time.Now()
	return &TicketMemory{
		ID:        types.NewMemoryID(),
		SchemaID:  schemaID,
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}
}

// Validate validates the TicketMemory
func (m *TicketMemory) Validate() error {
	if m.SchemaID == "" {
		return goerr.New("schema_id is required")
	}

	if m.Insights == "" {
		return goerr.New("insights is required")
	}

	if m.Version < 0 {
		return goerr.New("version must be non-negative")
	}

	return nil
}

// IsEmpty returns true if the memory has no content
func (m *TicketMemory) IsEmpty() bool {
	return m.Insights == ""
}

// String returns a Markdown-formatted representation of the memory
func (m *TicketMemory) String() string {
	if m.Insights == "" {
		return ""
	}

	return fmt.Sprintf("*Organizational Knowledge:*\n%s", m.Insights)
}
