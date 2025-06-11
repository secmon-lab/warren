package bigquery

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// RunbookEntry represents a single SQL runbook entry
type RunbookEntry struct {
	ID          types.RunbookID `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	FilePath    string          `json:"file_path"`
	SQLContent  string          `json:"sql_content"`
	Hash        string          `json:"hash"`
	Embedding   []float64       `json:"embedding"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// RunbookEntries is a slice of RunbookEntry
type RunbookEntries []*RunbookEntry

// RunbookSearchResult represents a search result with similarity score
type RunbookSearchResult struct {
	Entry      *RunbookEntry `json:"entry"`
	Similarity float64       `json:"similarity"`
}

// RunbookSearchResults is a slice of RunbookSearchResult
type RunbookSearchResults []*RunbookSearchResult
