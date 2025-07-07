package bigquery

import (
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// RunbookEntry represents a single SQL runbook entry
type RunbookEntry struct {
	ID          types.RunbookID `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	SQLContent  string          `json:"sql_content"`
}

// RunbookEntries is a slice of RunbookEntry
type RunbookEntries []*RunbookEntry
